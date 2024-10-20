package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Scan struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"` //  Not gonna be used in the code, but only just for MongoDB
	DomainId    primitive.ObjectID `bson:"domain_id"`
	Domain      string             `bson:"domain"`
	TemplateIDs []string           `bson:"template_ids"` // The unique template ID which we use to identify the template
	ScanDate    time.Time          `bson:"scan_date,omitempty"`
	Status      string             `bson:"status"`          // pending, in-progress, completed, failed
	Error       interface{}        `bson:"error,omitempty"` // Store error information if the scan fails
	S3ResultURL []string           `bson:"s3_result_url,"`  // URL to result stored in S3, if applicable
}
