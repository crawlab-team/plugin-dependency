package services

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/cenkalti/backoff/v4"
	"github.com/crawlab-team/crawlab-core/interfaces"
	"github.com/crawlab-team/crawlab-core/node/config"
	mongo2 "github.com/crawlab-team/crawlab-db/mongo"
	grpc "github.com/crawlab-team/crawlab-grpc"
	plugin "github.com/crawlab-team/crawlab-plugin"
	"github.com/crawlab-team/go-trace"
	"github.com/crawlab-team/plugin-dependency/constants"
	"github.com/crawlab-team/plugin-dependency/entity"
	"github.com/crawlab-team/plugin-dependency/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/dig"
	"io"
	"time"
)

type Service struct {
	*plugin.Internal

	// internals
	colS      *mongo2.Col // dependency settings
	colD      *mongo2.Col // dependencies
	cfgSvc    interfaces.NodeConfigService
	n         interfaces.Node
	msgStream grpc.MessageService_ConnectClient

	// sub services
	settingSvc *SettingService
	pythonSvc  *PythonService
}

func (svc *Service) Init() (err error) {
	// initialize sub services
	svc.settingSvc.Init()
	svc.pythonSvc.Init()

	return nil
}

func (svc *Service) Start() (err error) {
	if svc.cfgSvc.IsMaster() {
		// initialize data
		if err := svc.initData(); err != nil {
			return err
		}

		// initialize indexes
		if err := svc.initIndexes(); err != nil {
			return err
		}

		// start api
		go svc.StartApi()
	}

	// get current node
	if err := svc.getCurrentNode(); err != nil {
		return err
	}

	// handle stream messages
	svc.handleStreamMessages()

	// wait
	svc.Wait()

	return nil
}

func (svc *Service) Stop() (err error) {
	svc.StopApi()
	return nil
}

func (svc *Service) initData() (err error) {
	total, err := svc.colS.Count(nil)
	if err != nil {
		return err
	}
	if total > 0 {
		return nil
	}

	// data to initialize
	settings := []models.Setting{
		{
			Id:          primitive.NewObjectID(),
			Key:         "python",
			Name:        "Python",
			Description: `Dependencies for Python environment`,
			Enabled:     true,
		},
		{
			Id:          primitive.NewObjectID(),
			Key:         "node",
			Name:        "Node.js",
			Description: `Dependencies for Node.js environment`,
			Enabled:     true,
		},
	}
	var data []interface{}
	for _, s := range settings {
		data = append(data, s)
	}
	_, err = svc.colS.InsertMany(data)
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) initIndexes() (err error) {
	optsColS := &options.IndexOptions{}
	optsColS.SetUnique(true)
	_ = svc.colS.CreateIndexes([]mongo.IndexModel{
		{
			Keys: bson.D{
				{"type", 1},
				{"node_id", 1},
				{"name", 1},
			},
			Options: optsColS,
		},
	})
	return nil
}

func (svc *Service) handleStreamMessages() {
	if err := backoff.Retry(svc._connect, backoff.NewExponentialBackOff()); err != nil {
		panic(err)
	}
	for {
		msg, err := svc.msgStream.Recv()
		if err == io.EOF {
			return
		}

		if err != nil {
			trace.PrintError(err)
			time.Sleep(1 * time.Second)
			continue
		}

		var msgData entity.MessageData
		if err := json.Unmarshal(msg.Data, &msgData); err != nil {
			trace.PrintError(err)
			continue
		}

		switch msgData.Code {
		case constants.MessageCodeUpdatePython:
			go svc.pythonSvc.updateDependencyList()
		case constants.MessageCodeSavePython:
			go svc.pythonSvc.saveDependencyList(msg, msgData)
		}
	}
}

func (svc *Service) _connect() (err error) {
	stream, err := svc.GetGrpcClient().GetMessageClient().Connect(context.Background())
	if err != nil {
		return err
	}
	msg := &grpc.StreamMessage{
		Code:    grpc.StreamMessageCode_CONNECT,
		NodeKey: svc.n.GetKey(),
		Key:     "plugin:" + svc.n.GetKey(),
	}
	if err := stream.Send(msg); err != nil {
		return err
	}
	svc.msgStream = stream
	return nil
}

func (svc *Service) getCurrentNode() (err error) {
	nodeModelSvc, err := svc.GetModelService().NewBaseServiceDelegate(interfaces.ModelIdNode)
	if err != nil {
		return err
	}
	res, err := nodeModelSvc.Get(bson.M{"key": svc.cfgSvc.GetNodeKey()}, nil)
	if err != nil {
		return err
	}
	n, ok := res.(interfaces.Node)
	if !ok {
		return errors.New("invalid type")
	}
	svc.n = n
	return nil
}

func NewService() *Service {
	// service
	svc := &Service{
		Internal: plugin.NewInternal(),
		colS:     mongo2.GetMongoCol(constants.DependencySettingsColName),
		colD:     mongo2.GetMongoCol(constants.DependenciesColName),
	}

	// dependency injection
	c := dig.New()
	if err := c.Provide(config.NewNodeConfigService); err != nil {
		panic(err)
	}
	if err := c.Invoke(func(
		cfgSvc interfaces.NodeConfigService,
	) {
		svc.cfgSvc = cfgSvc
	}); err != nil {
		panic(err)
	}

	// sub services
	svc.settingSvc = NewSettingService(svc)
	svc.pythonSvc = NewPythonService(svc)

	// initialize
	if err := svc.Init(); err != nil {
		panic(err)
	}

	return svc
}
