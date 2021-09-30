package main

import (
	"github.com/apex/log"
	"github.com/crawlab-team/crawlab-core/interfaces"
	"github.com/crawlab-team/crawlab-core/node/config"
	mongo2 "github.com/crawlab-team/crawlab-db/mongo"
	grpc "github.com/crawlab-team/crawlab-grpc"
	plugin "github.com/crawlab-team/crawlab-plugin"
	"github.com/crawlab-team/go-trace"
	"github.com/crawlab-team/plugin-dependency/constants"
	"github.com/crawlab-team/plugin-dependency/models"
	"github.com/crawlab-team/plugin-dependency/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/dig"
	"io"
	"strings"
	"time"
)

type Service struct {
	*plugin.Internal
	col    *mongo2.Col // dependency settings
	cfgSvc interfaces.NodeConfigService

	// api
	settingSvc *services.SettingService
	pythonSvc  *services.PythonService
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

		// start api
		go svc.StartApi()
	}

	// handle events
	svc.handleEvents()

	return nil
}

func (svc *Service) Stop() (err error) {
	svc.StopApi()
	return nil
}

func (svc *Service) initData() (err error) {
	total, err := svc.col.Count(nil)
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
	_, err = svc.col.InsertMany(data)
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) handleEvents() {
	log.Infof("start handling events")

	// get stream
	log.Infof("attempt to obtain grpc stream...")
	var stream grpc.PluginService_SubscribeClient
	for {
		stream = svc.Internal.GetEventService().GetStream()
		if stream == nil {
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
	log.Infof("obtained grpc stream, start receiving messages...")

	for {
		// receive stream message
		msg, err := stream.Recv()

		if err != nil {
			// end
			if strings.HasSuffix(err.Error(), io.EOF.Error()) {
				// TODO: implement
				log.Infof("received EOF signal, re-connecting...")
				//svc.GetGrpcClient().Restart()
			}

			trace.PrintError(err)
			time.Sleep(1 * time.Second)
			continue
		}

		//var data entity.GrpcEventServiceMessage
		switch msg.Code {
		//case grpc.StreamMessageCode_SEND_EVENT:
		//	// data
		//	if err := json.Unmarshal(msg.Data, &data); err != nil {
		//		return
		//	}
		default:
			continue
		}
	}
}

func NewService() *Service {
	// service
	svc := &Service{
		Internal: plugin.NewInternal(),
		col:      mongo2.GetMongoCol(constants.DependencySettingsColName),
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
	svc.settingSvc = services.NewSettingService(svc.GetApi())
	svc.pythonSvc = services.NewPythonService(svc.GetApi())

	// initialize
	if err := svc.Init(); err != nil {
		panic(err)
	}

	return svc
}
