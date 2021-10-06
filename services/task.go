package services

import (
	"github.com/crawlab-team/crawlab-core/controllers"
	mongo2 "github.com/crawlab-team/crawlab-db/mongo"
	"github.com/crawlab-team/plugin-dependency/constants"
	"github.com/crawlab-team/plugin-dependency/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TaskService struct {
	parent *Service
	api    *gin.Engine
	colT   *mongo2.Col // tasks
	colL   *mongo2.Col // logs
}

func (svc *TaskService) Init() {
	svc.api.GET("/tasks", svc.getList)
	svc.api.GET("/tasks/:id/logs", svc.getLogs)
}

func (svc *TaskService) getList(c *gin.Context) {
	// pagination
	pagination := controllers.MustGetPagination(c)

	// query
	query := bson.M{"type": constants.DependencyTypePython}

	// tasks
	var tasks []models.Task
	opts := &mongo2.FindOptions{
		Skip:  (pagination.Page - 1) * pagination.Size,
		Limit: pagination.Size,
		Sort:  bson.D{{"_id", -1}},
	}
	if err := svc.colT.Find(query, opts).All(&tasks); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// total
	total, err := svc.colT.Count(query)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	controllers.HandleSuccessWithListData(c, tasks, total)
}

func (svc *TaskService) getLogs(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		controllers.HandleErrorBadRequest(c, err)
		return
	}

	var logList []models.Log
	if err := svc.colL.Find(bson.M{"task_id": id}, nil).All(&logList); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	controllers.HandleSuccessWithData(c, logList)
}

func NewTaskService(parent *Service) (svc *TaskService) {
	svc = &TaskService{
		parent: parent,
		api:    parent.GetApi(),
		colT:   mongo2.GetMongoCol(constants.DependencyTasksColName),
		colL:   mongo2.GetMongoCol(constants.DependencyLogsColName),
	}

	return svc
}
