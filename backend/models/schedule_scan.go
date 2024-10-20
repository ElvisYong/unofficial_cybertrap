package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ScheduleScan struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty"`
	DomainIds     []primitive.ObjectID `bson:"domain_ids"`
	TemplatesIDs  []primitive.ObjectID `bson:"template_ids"`
	ScanAll       bool                 `bson:"scan_all"` // If true we don't need domain_ids or
	ScheduledDate time.Time            `bson:"scheduled_date"`
}
