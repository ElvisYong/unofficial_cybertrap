package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Domain struct {
	Id         primitive.ObjectID `bson:"_id,omitempty"`
	Domain     string             `bson:"domain"`
	UploadedAt time.Time          `bson:"uploaded_at"`
	UserId     string             `bson:"user_id,"`
}
