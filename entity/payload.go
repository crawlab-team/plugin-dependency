package entity

import "go.mongodb.org/mongo-driver/bson/primitive"

type InstallPayload struct {
	Names   []string             `json:"names"`
	Mode    string               `json:"mode"`
	NodeIds []primitive.ObjectID `json:"node_ids"`
}
