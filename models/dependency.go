package models

import (
	"github.com/crawlab-team/plugin-dependency/entity"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Dependency struct {
	Id          primitive.ObjectID      `json:"_id" bson:"_id"`
	NodeId      primitive.ObjectID      `json:"node_id" bson:"node_id"`
	Type        string                  `json:"type" bson:"type"`
	Name        string                  `json:"name" bson:"name"`
	Version     string                  `json:"version" bson:"version"`
	Description string                  `json:"description" bson:"description"`
	Result      entity.DependencyResult `json:"result" bson:"-"`
}
