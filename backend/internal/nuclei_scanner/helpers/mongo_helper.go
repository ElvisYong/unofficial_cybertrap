package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoHelper struct {
	client   *mongo.Client
	database string
}

const (
	ScansCollection           = "scans"
	DomainsCollection         = "domains"
	NucleiTemplatesCollection = "nucleiTemplates"
	MultiScansCollection      = "multiScans"
)

func NewMongoHelper(client *mongo.Client, database string) *MongoHelper {
	return &MongoHelper{
		client:   client,
		database: database,
	}
}

func NewMongoClient(ctx context.Context, uri string) (*mongo.Client, error) {
	clientOpts := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(5).
		SetMinPoolSize(1).
		SetMaxConnecting(2).
		SetRetryReads(true).
		SetRetryWrites(true).
		SetTimeout(10 * time.Second)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client, nil
}

func (r *MongoHelper) InsertScan(ctx context.Context, scan models.Scan) (primitive.ObjectID, error) {
	collection := r.client.Database(r.database).Collection(ScansCollection)
	scan.ScanDate = time.Now()
	scan.Status = "pending"

	result, err := collection.InsertOne(ctx, scan)
	if err != nil {
		log.Error().Err(err).Msg("Failed to insert scan")
		return primitive.NilObjectID, err
	}

	return result.InsertedID.(primitive.ObjectID), nil
}

// UpdateScanResult updates the scan model with the new scan result and optional duration
func (r *MongoHelper) UpdateScanResult(ctx context.Context, scan models.Scan) error {
	collection := r.client.Database(r.database).Collection(ScansCollection)
	filter := bson.M{"_id": scan.ID}

	// Convert the entire scan object to a BSON document
	update := bson.M{
		"$set": scan,
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update scan result")
		return err
	}

	return nil
}

func (r *MongoHelper) UpdateScanStatus(ctx context.Context, scanID primitive.ObjectID, status string, errorInfo interface{}) error {
	collection := r.client.Database(r.database).Collection(ScansCollection)
	filter := bson.M{"_id": scanID}
	update := bson.M{
		"$set": bson.M{
			"status": status,
			"error":  errorInfo,
		},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Error().
			Err(err).
			Str("scanID", scanID.Hex()).
			Str("status", status).
			Interface("errorInfo", errorInfo).
			Msg("Failed to update scan status")
		return err
	}

	return nil
}

func (r *MongoHelper) FindScanByID(ctx context.Context, scanID primitive.ObjectID) (models.Scan, error) {
	collection := r.client.Database(r.database).Collection(ScansCollection)
	var scan models.Scan
	err := collection.FindOne(ctx, bson.M{"_id": scanID}).Decode(&scan)
	if err != nil {
		log.Error().Err(err).Msg("Failed to find scan by ID")
		return scan, err
	}

	return scan, nil
}

func (r *MongoHelper) FindDomainByID(ctx context.Context, domainID primitive.ObjectID) (models.Domain, error) {
	collection := r.client.Database(r.database).Collection(DomainsCollection)
	var domain models.Domain
	err := collection.FindOne(ctx, bson.M{"_id": domainID}).Decode(&domain)
	if err != nil {
		log.Error().Err(err).Msg("Failed to find domain by ID")
		return domain, err
	}

	return domain, nil
}

func (r *MongoHelper) FindTemplateByID(ctx context.Context, templateID primitive.ObjectID) (models.Template, error) {
	collection := r.client.Database(r.database).Collection(NucleiTemplatesCollection)
	var template models.Template
	err := collection.FindOne(ctx, bson.M{"_id": templateID}).Decode(&template)
	if err != nil {
		log.Error().Err(err).Msg("Failed to find template by ID")
		return template, err
	}

	return template, nil
}

func (r *MongoHelper) UpdateMultiScanStatus(ctx context.Context, multiScanId primitive.ObjectID, status string, completedScanID, failedScanID *primitive.ObjectID) error {
	collection := r.client.Database(r.database).Collection(MultiScansCollection)
	filter := bson.M{"_id": multiScanId}
	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	if completedScanID != nil {
		update["$push"] = bson.M{"completed_scans": *completedScanID}
	}

	if failedScanID != nil {
		update["$push"] = bson.M{"failed_scans": *failedScanID}
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update multi scan status")
		return err
	}

	return nil
}

func (r *MongoHelper) FindMultiScanByID(ctx context.Context, multiScanId primitive.ObjectID) (models.MultiScan, error) {
	collection := r.client.Database(r.database).Collection(MultiScansCollection)
	var multiScan models.MultiScan
	err := collection.FindOne(ctx, bson.M{"_id": multiScanId}).Decode(&multiScan)
	if err != nil {
		log.Error().Err(err).Msg("Failed to find multi scan by ID")
		return multiScan, err
	}

	return multiScan, nil
}

// Single source of truth for all MongoDB operations
func (mh *MongoHelper) UpdateScanError(ctx context.Context, scanID primitive.ObjectID, status string, errorInfo interface{}, duration int64) error {
	collection := mh.client.Database(mh.database).Collection(ScansCollection)

	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"error":     errorInfo,
			"scan_took": duration,
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": scanID}, update)
	if err != nil {
		log.Error().Err(err).
			Str("scanID", scanID.Hex()).
			Str("status", status).
			Interface("errorInfo", errorInfo).
			Int64("duration", duration).
			Msg("Failed to update scan error status")
	}
	return err
}

func (mh *MongoHelper) UpdateScanStartTime(ctx context.Context, scanID primitive.ObjectID, startTime time.Time) error {
	collection := mh.client.Database(mh.database).Collection(ScansCollection)

	update := bson.M{
		"$set": bson.M{
			"scan_date": startTime,
			"status":    "in-progress",
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": scanID}, update)
	return err
}

func (mh *MongoHelper) UpdateMultiScanCompletion(ctx context.Context, multiScanID primitive.ObjectID, status string, duration int64) error {
	collection := mh.client.Database(mh.database).Collection(MultiScansCollection)

	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"scan_took": duration,
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": multiScanID}, update)
	if err != nil {
		log.Error().Err(err).Msg("Error updating multi-scan completion in MongoDB")
		return err
	}

	return nil
}

func (mh *MongoHelper) UpdateMultiScanTiming(ctx context.Context, multiScanID primitive.ObjectID, status string, duration int64) error {
	collection := mh.client.Database(mh.database).Collection(MultiScansCollection)

	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"scan_took": duration,
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": multiScanID}, update)
	return err
}

func (mh *MongoHelper) UpdateScanWithDuration(ctx context.Context, scanID primitive.ObjectID, status string, duration int64) error {
	collection := mh.client.Database(mh.database).Collection(ScansCollection)

	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"scan_took": duration,
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": scanID}, update)
	if err != nil {
		log.Error().Err(err).
			Str("scanID", scanID.Hex()).
			Str("status", status).
			Int64("duration", duration).
			Msg("Failed to update scan duration")
		return err
	}

	return nil
}
