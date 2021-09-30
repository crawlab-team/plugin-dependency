package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Dependency struct {
	Id          primitive.ObjectID `json:"_id" bson:"_id"`
	Type        string             `json:"type" bson:"type"`
	Name        string             `json:"name" bson:"name"`
	Version     string             `json:"version" bson:"version"`
	Description string             `json:"description" bson:"description"`
}
