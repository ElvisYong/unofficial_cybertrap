package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MultiScan struct {
	ID             primitive.ObjectID   `bson:"_id,omitempty"`
	ScanIDs        []primitive.ObjectID `bson:"scan_ids"`
	Name           string               `bson:"name"`
	TotalScans     int                  `bson:"total_scans"`
	CompletedScans []primitive.ObjectID `bson:"completed_scans"`
	FailedScans    []primitive.ObjectID `bson:"failed_scans"`
	Status         string               `bson:"status"` // We will only cater for "in-progress" and "completed"
	ScanDate       time.Time            `bson:"scan_date,omitempty"`
}
