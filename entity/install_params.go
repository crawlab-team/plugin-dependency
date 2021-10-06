package entity

import "go.mongodb.org/mongo-driver/bson/primitive"

type InstallParams struct {
	TaskId primitive.ObjectID `json:"task_id"`
	Names  []string           `json:"names"`
	Proxy  string             `json:"proxy"`
}
