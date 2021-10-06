package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	constants2 "github.com/crawlab-team/crawlab-core/constants"
	"github.com/crawlab-team/crawlab-core/controllers"
	"github.com/crawlab-team/crawlab-core/interfaces"
	models2 "github.com/crawlab-team/crawlab-core/models/models"
	"github.com/crawlab-team/crawlab-db/mongo"
	grpc "github.com/crawlab-team/crawlab-grpc"
	"github.com/crawlab-team/go-trace"
	"github.com/crawlab-team/plugin-dependency/constants"
	"github.com/crawlab-team/plugin-dependency/entity"
	"github.com/crawlab-team/plugin-dependency/models"
	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PythonService struct {
	parent *Service
	api    *gin.Engine
	chMap  sync.Map
}

func (svc *PythonService) Init() {
	svc.api.GET("/python", svc.getList)
	svc.api.POST("/python/update", svc.update)
	svc.api.POST("/python/install", svc.install)
	svc.api.POST("/python/uninstall", svc.uninstall)
}

func (svc *PythonService) getList(c *gin.Context) {
	installed, _ := strconv.ParseBool(c.Query("installed"))
	if installed {
		svc._getInstalledPythonList(c)
	} else {
		svc._getPypiPythonList(c)
	}
}

func (svc *PythonService) _getPypiPythonList(c *gin.Context) {
	// query
	query := c.Query("query")
	pagination := controllers.MustGetPagination(c)

	// validate
	if query == "" {
		controllers.HandleErrorBadRequest(c, errors.New("empty query"))
		return
	}

	// request session
	reqSession := req.New()

	// set timeout
	reqSession.SetTimeout(15 * time.Second)

	// user agent
	ua := req.Header{"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.61 Safari/537.36"}

	// request url
	requestUrl := fmt.Sprintf("https://pypi.org/search?page=%d&q=%s", pagination.Page, url.QueryEscape(query))

	// perform request
	res, err := reqSession.Get(requestUrl, ua)
	if err != nil {
		if res != nil {
			_, _ = c.Writer.Write(res.Bytes())
			_ = c.AbortWithError(res.Response().StatusCode, err)
			return
		}
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// response bytes
	data, err := res.ToBytes()
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}
	buf := bytes.NewBuffer(data)

	// parse html
	doc, err := goquery.NewDocumentFromReader(buf)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// dependencies
	var deps []models.Dependency
	var depNames []string
	doc.Find(".left-layout__main > form ul > li").Each(func(i int, s *goquery.Selection) {
		d := models.Dependency{
			Name:        s.Find(".package-snippet__name").Text(),
			Version:     s.Find(".package-snippet__version").Text(),
			Description: s.Find(".package-snippet__description").Text(),
		}
		deps = append(deps, d)
		depNames = append(depNames, d.Name)
	})

	// total
	totalStr := doc.Find(".left-layout__main .split-layout p > strong").Text()
	totalStr = strings.ReplaceAll(totalStr, ",", "")
	total, _ := strconv.Atoi(totalStr)

	// dependencies in db
	var depsResults []entity.DependencyResult
	pipelines := mongo2.Pipeline{
		{{
			"$match",
			bson.M{
				"type": constants.DependencyTypePython,
				"name": bson.M{
					"$in": depNames,
				},
			},
		}},
		{{
			"$group",
			bson.M{
				"_id": "$name",
				"node_ids": bson.M{
					"$push": "$node_id",
				},
				"versions": bson.M{
					"$addToSet": "$version",
				},
			},
		}},
		{{
			"$project",
			bson.M{
				"name":     "$_id",
				"node_ids": "$node_ids",
				"versions": "$versions",
			},
		}},
	}
	if err := svc.parent.colD.Aggregate(pipelines, nil).All(&depsResults); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// dependencies map
	depsResultsMap := map[string]entity.DependencyResult{}
	for _, dr := range depsResults {
		depsResultsMap[dr.Name] = dr
	}

	// iterate dependencies
	for i, d := range deps {
		dr, ok := depsResultsMap[d.Name]
		if ok {
			deps[i].Result = dr
		}
	}

	controllers.HandleSuccessWithListData(c, deps, total)
}

func (svc *PythonService) _getInstalledPythonList(c *gin.Context) {
	// params
	searchQuery := c.Query("query")
	pagination := controllers.MustGetPagination(c)

	// query
	query := bson.M{}
	query["type"] = constants.DependencyTypePython
	if searchQuery != "" {
		query["name"] = primitive.Regex{
			Pattern: searchQuery,
			Options: "i",
		}
	}

	// base pipelines
	basePipelines := mongo2.Pipeline{
		{{
			"$match",
			query,
		}},
		{{
			"$group",
			bson.M{
				"_id": "$name",
				"node_ids": bson.M{
					"$push": "$node_id",
				},
				"versions": bson.M{
					"$addToSet": "$version",
				},
			},
		}},
		{{
			"$project",
			bson.M{
				"name":     "$_id",
				"node_ids": "$node_ids",
				"versions": "$versions",
			},
		}},
		{{"$sort", bson.D{{"name", 1}}}},
	}

	// dependency results
	var depsResults []entity.DependencyResult
	pipelines := basePipelines
	pipelines = append(pipelines, bson.D{{"$skip", (pagination.Page - 1) * pagination.Size}})
	pipelines = append(pipelines, bson.D{{"$limit", pagination.Size}})
	if err := svc.parent.colD.Aggregate(pipelines, nil).All(&depsResults); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// total
	var depsResultTotal entity.DependencyResult
	pipelinesTotal := basePipelines
	pipelinesTotal = append(pipelinesTotal, bson.D{{
		"$group",
		bson.M{
			"_id":   nil,
			"count": bson.M{"$sum": 1},
		},
	}})
	if err := svc.parent.colD.Aggregate(pipelinesTotal, nil).One(&depsResultTotal); err != nil {
		if err.Error() != mongo2.ErrNoDocuments.Error() {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}
	}
	total := depsResultTotal.Count

	controllers.HandleSuccessWithListData(c, depsResults, total)
}

func (svc *PythonService) update(c *gin.Context) {
	// nodes
	nodeModelSvc, err := svc.parent.GetModelService().NewBaseServiceDelegate(interfaces.ModelIdNode)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}
	list, err := nodeModelSvc.GetList(bson.M{"active": true}, nil)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// wait group
	wg := sync.WaitGroup{}
	wg.Add(list.Size())

	// iterate nodes
	for _, d := range list.Values() {
		// node
		n, ok := d.(models2.Node)
		if !ok {
			wg.Done()
			continue
		}

		go func() {
			// notify channel
			ch := svc._getDefaultCh()

			// send message
			data, err := json.Marshal(&entity.MessageData{Code: constants.MessageCodeUpdatePython})
			if err != nil {
				trace.PrintError(err)
				wg.Done()
				return
			}
			msg := &grpc.StreamMessage{
				Code:    grpc.StreamMessageCode_SEND,
				NodeKey: svc.parent.currentNode.GetKey(),
				From:    "plugin:" + svc.parent.currentNode.GetKey(),
				To:      "plugin:" + n.GetKey(),
				Data:    data,
			}
			if err := svc.parent.msgStream.Send(msg); err != nil {
				trace.PrintError(err)
				wg.Done()
				return
			}

			// wait
			<-ch
			wg.Done()
		}()
	}

	// wait for all nodes to finish
	wg.Wait()

	controllers.HandleSuccess(c)
}

func (svc *PythonService) install(c *gin.Context) {
	// payload
	var payload entity.InstallPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// setting
	var s models.Setting
	if err := svc.parent.colS.Find(bson.M{"key": constants.SettingKeyPython}, nil).One(&s); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// node model service
	nodeModelSvc, err := svc.parent.GetModelService().NewBaseServiceDelegate(interfaces.ModelIdNode)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// nodes
	query := bson.M{}
	if payload.Mode == constants.InstallModeAll {
		query["active"] = true
	} else {
		query["node_id"] = bson.M{"$in": payload.NodeIds}
	}
	list, err := nodeModelSvc.GetList(bson.M{"active": true}, nil)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}
	var nodes []models2.Node
	for _, d := range list.Values() {
		n, ok := d.(models2.Node)
		if !ok {
			controllers.HandleErrorInternalServerError(c, errors.New("invalid type"))
			return
		}
		nodes = append(nodes, n)
	}

	// iterate nodes
	for _, n := range nodes {
		// task
		t := &models.Task{
			Id:        primitive.NewObjectID(),
			Status:    constants2.TaskStatusRunning,
			SettingId: s.Id,
			Type:      constants.SettingKeyPython,
			NodeId:    n.Id,
			DepNames:  payload.Names,
			Action:    constants.ActionInstall,
			UpdatedTs: time.Now(),
		}
		if _, err := svc.parent.colT.Insert(t); err != nil {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}

		// params
		params := &entity.InstallParams{
			TaskId: t.Id,
			Names:  payload.Names,
			Proxy:  s.Proxy,
		}

		// data
		data, err := json.Marshal(params)
		if err != nil {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}

		// message data
		msgDataObj := &entity.MessageData{
			Code: constants.MessageCodeInstallPython,
			Data: data,
		}
		msgData, err := json.Marshal(msgDataObj)
		if err != nil {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}

		// stream message
		msg := &grpc.StreamMessage{
			Code:    grpc.StreamMessageCode_SEND,
			NodeKey: svc.parent.currentNode.GetKey(),
			From:    "plugin:" + svc.parent.currentNode.GetKey(),
			To:      "plugin:" + n.GetKey(),
			Data:    msgData,
		}

		// send message
		if err := svc.parent.msgStream.Send(msg); err != nil {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}
	}

	controllers.HandleSuccess(c)
}

func (svc *PythonService) uninstall(c *gin.Context) {
	// TODO: implement
}

func (svc *PythonService) updateDependencyList() {
	// installed dependencies
	deps, err := svc._getDependencies()
	if err != nil {
		trace.PrintError(err)
		return
	}

	// data
	data, err := json.Marshal(deps)
	if err != nil {
		trace.PrintError(err)
		return
	}

	// message data
	msgDataObj := &entity.MessageData{
		Code: constants.MessageCodeSavePython,
		Data: data,
	}
	msgData, err := json.Marshal(msgDataObj)
	if err != nil {
		trace.PrintError(err)
		return
	}

	// stream message
	msg := &grpc.StreamMessage{
		Code:    grpc.StreamMessageCode_SEND,
		NodeKey: svc.parent.currentNode.GetKey(),
		From:    "plugin:" + svc.parent.currentNode.GetKey(),
		To:      "plugin:" + svc.parent.masterNode.GetKey(),
		Data:    msgData,
	}

	// send message
	if err := svc.parent.msgStream.Send(msg); err != nil {
		trace.PrintError(err)
		return
	}
}

func (svc *PythonService) _saveDependencyList(msg *grpc.StreamMessage, msgData entity.MessageData) {
	// notify channel
	ch := svc._getDefaultCh()

	// dependencies
	var deps []models.Dependency
	if err := json.Unmarshal(msgData.Data, &deps); err != nil {
		trace.PrintError(err)
		ch <- true
		return
	}

	// installed dependency names
	var depNames []string
	for _, d := range deps {
		depNames = append(depNames, d.Name)
	}

	// node model service
	nodeModelSvc, err := svc.parent.GetModelService().NewBaseServiceDelegate(interfaces.ModelIdNode)
	if err != nil {
		trace.PrintError(err)
		ch <- true
		return
	}

	// node
	doc, err := nodeModelSvc.Get(bson.M{"key": msg.NodeKey}, nil)
	if err != nil {
		trace.PrintError(err)
		ch <- true
		return
	}
	n, ok := doc.(interfaces.Node)
	if !ok {
		trace.PrintError(errors.New("invalid type"))
		ch <- true
		return
	}

	// run transaction to update dependencies
	err = mongo.RunTransaction(func(ctx mongo2.SessionContext) (err error) {
		// remove non-existing dependencies
		if err := svc.parent.colD.Delete(bson.M{
			"type":    constants.DependencyTypePython,
			"node_id": n.GetId(),
			"name":    bson.M{"$nin": depNames},
		}); err != nil {
			return err
		}

		// existing dependencies
		query := bson.M{
			"type":    constants.DependencyTypePython,
			"node_id": n.GetId(),
		}
		var depsDb []models.Dependency
		if err := svc.parent.colD.Find(query, nil).All(&depsDb); err != nil {
			return err
		}
		depsDbMap := map[string]models.Dependency{}
		for _, d := range depsDb {
			depsDbMap[d.Name] = d
		}

		// new dependencies
		var depsNew []interface{}
		for _, d := range deps {
			if _, ok := depsDbMap[d.Name]; !ok {
				d.Id = primitive.NewObjectID()
				d.Type = constants.DependencyTypePython
				d.NodeId = n.GetId()
				depsNew = append(depsNew, d)
			}
		}

		// skip if no new dependencies
		if len(depsNew) == 0 {
			return
		}

		// add new dependencies
		if _, err := svc.parent.colD.InsertMany(depsNew); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		trace.PrintError(err)
		ch <- true
		return
	}

	// notify update success
	ch <- true
}

func (svc *PythonService) notifyUpdateDependencyList(msg *grpc.StreamMessage, data entity.MessageData) {
}

func (svc *PythonService) installDependency(msg *grpc.StreamMessage, msgData entity.MessageData) {
	// dependencies
	var params entity.InstallParams
	if err := json.Unmarshal(msgData.Data, &params); err != nil {
		trace.PrintError(err)
		svc.parent._sendTaskStatus(params.TaskId, constants2.TaskStatusError)
		return
	}

	// install
	if err := svc._installDependencies(params); err != nil {
		trace.PrintError(err)
		svc.parent._sendTaskStatus(params.TaskId, constants2.TaskStatusError)
		return
	}

	svc.parent._sendTaskStatus(params.TaskId, constants2.TaskStatusFinished)
}

func (svc *PythonService) _getDependencies() (deps []models.Dependency, err error) {
	cmd := exec.Command("pip", "list", "--format", "json")
	data, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &deps); err != nil {
		return nil, err
	}
	for _, d := range deps {
		d.Type = constants.DependencyTypePython
	}
	return deps, nil
}

func (svc *PythonService) _installDependencies(params entity.InstallParams) (err error) {
	// arguments
	var args []string

	// install
	args = append(args, "install")

	// proxy
	if params.Proxy != "" {
		args = append(args, "-i")
		args = append(args, params.Proxy)
	}

	// dependency names
	for _, depName := range params.Names {
		args = append(args, depName)
	}

	// command
	cmd := exec.Command("pip", args...)

	// logging
	svc.parent._configureLogging(params.TaskId, cmd)

	// start
	if err := cmd.Start(); err != nil {
		return trace.TraceError(err)
	}

	// wait
	if err := cmd.Wait(); err != nil {
		return trace.TraceError(err)
	}

	return nil
}

func (svc *PythonService) _getDefaultCh() (ch chan bool) {
	return svc._getCh(svc.parent.currentNode.GetKey())
}

func (svc *PythonService) _getCh(key string) (ch chan bool) {
	res, ok := svc.chMap.Load(key)
	if ok {
		ch, ok := res.(chan bool)
		if ok {
			return ch
		}
	}
	ch = make(chan bool)
	svc.chMap.Store(key, ch)
	return ch
}

func NewPythonService(parent *Service) (svc *PythonService) {
	svc = &PythonService{
		parent: parent,
		api:    parent.GetApi(),
		chMap:  sync.Map{},
	}

	return svc
}
