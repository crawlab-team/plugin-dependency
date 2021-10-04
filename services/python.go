package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
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
	"time"
)

type PythonService struct {
	parent *Service
	api    *gin.Engine
	colD   *mongo.Col // dependencies
}

func (svc *PythonService) Init() {
	svc.api.GET("/python", svc.getPythonList)
	svc.api.POST("/python/update", svc.updatePythonList)
}

func (svc *PythonService) updateDependencyList() {
	// installed dependencies
	deps, err := svc.getDependencies()
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

	// node service
	nodeModelSvc, err := svc.parent.GetModelService().NewBaseServiceDelegate(interfaces.ModelIdNode)
	if err != nil {
		trace.PrintError(err)
		return
	}

	// master node
	doc, err := nodeModelSvc.Get(bson.M{"is_master": true}, nil)
	if err != nil {
		trace.PrintError(err)
		return
	}
	masterNode, ok := doc.(interfaces.Node)
	if !ok {
		trace.PrintError(errors.New("invalid type"))
		return
	}

	// stream message
	msg := &grpc.StreamMessage{
		Code:    grpc.StreamMessageCode_SEND,
		NodeKey: svc.parent.n.GetKey(),
		From:    "plugin:" + svc.parent.n.GetKey(),
		To:      "plugin:" + masterNode.GetKey(),
		Data:    msgData,
		Error:   "",
	}

	// send message
	if err := svc.parent.msgStream.Send(msg); err != nil {
		trace.PrintError(err)
		return
	}
}

func (svc *PythonService) saveDependencyList(msg *grpc.StreamMessage, msgData entity.MessageData) {
	// dependencies
	var deps []models.Dependency
	if err := json.Unmarshal(msgData.Data, &deps); err != nil {
		trace.PrintError(err)
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
		return
	}

	// node
	doc, err := nodeModelSvc.Get(bson.M{"key": msg.NodeKey}, nil)
	if err != nil {
		trace.PrintError(err)
		return
	}
	n, ok := doc.(interfaces.Node)
	if !ok {
		trace.PrintError(errors.New("invalid type"))
		return
	}

	// run transaction to update dependencies
	err = mongo.RunTransaction(func(ctx mongo2.SessionContext) (err error) {
		// remove non-existing dependencies
		if err := svc.colD.Delete(bson.M{
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
		if err := svc.colD.Find(query, nil).All(&depsDb); err != nil {
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
		if _, err := svc.colD.InsertMany(depsNew); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		trace.PrintError(err)
		return
	}
}

func (svc *PythonService) getPythonList(c *gin.Context) {
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
	if err := svc.colD.Aggregate(pipelines, nil).All(&depsResults); err != nil {
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
	if err := svc.colD.Aggregate(pipelines, nil).All(&depsResults); err != nil {
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
	if err := svc.colD.Aggregate(pipelinesTotal, nil).One(&depsResultTotal); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}
	total := depsResultTotal.Count

	controllers.HandleSuccessWithListData(c, depsResults, total)
}

func (svc *PythonService) updatePythonList(c *gin.Context) {
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
	for _, d := range list.Values() {
		n, ok := d.(models2.Node)
		if !ok {
			continue
		}
		data, err := json.Marshal(&entity.MessageData{Code: constants.MessageCodeUpdatePython})
		if err != nil {
			trace.PrintError(err)
			return
		}
		msg := &grpc.StreamMessage{
			Code:    grpc.StreamMessageCode_SEND,
			NodeKey: svc.parent.n.GetKey(),
			From:    "plugin:" + svc.parent.n.GetKey(),
			To:      "plugin:" + n.GetKey(),
			Data:    data,
		}
		if err := svc.parent.msgStream.Send(msg); err != nil {
			trace.PrintError(err)
			continue
		}
	}
	controllers.HandleSuccess(c)
}

func (svc *PythonService) getDependencies() (deps []models.Dependency, err error) {
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

func NewPythonService(parent *Service) (svc *PythonService) {
	svc = &PythonService{
		parent: parent,
		api:    parent.GetApi(),
		colD:   mongo.GetMongoCol(constants.DependenciesColName),
	}

	return svc
}
