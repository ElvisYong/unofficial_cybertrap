package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MultiScan struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	ScanIDs        []string           `bson:"scan_ids"`
	TotalScans     int                `bson:"total_scans"`
	CompletedScans int                `bson:"completed_scans"`
	FailedScans    int                `bson:"failed_scans"`
	Status         string             `bson:"status"`
	ScanDate       time.Time          `bson:"scan_date,omitempty"`
}
