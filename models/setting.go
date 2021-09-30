package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Setting struct {
	Id          primitive.ObjectID `json:"_id" bson:"_id"`
	Key         string             `json:"key" bson:"key"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Enabled     bool               `json:"enabled" bson:"enabled"`
}
