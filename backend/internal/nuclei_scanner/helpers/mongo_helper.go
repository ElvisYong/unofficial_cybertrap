package helpers

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoHelper struct {
	client   *mongo.Client
	database string
	logger   zerolog.Logger
}

const (
	ScansCollection           = "scans"
	DomainsCollection         = "domains"
	NucleiTemplatesCollection = "nucleiTemplates"
)

func NewMongoHelper(client *mongo.Client, database string, logger zerolog.Logger) *MongoHelper {
	return &MongoHelper{
		client:   client,
		database: database,
		logger:   logger,
	}
}

func (r *MongoHelper) InsertScan(ctx context.Context, scan models.Scan) (primitive.ObjectID, error) {
	collection := r.client.Database(r.database).Collection(ScansCollection)
	scan.ScanDate = time.Now()
	scan.Status = "pending"

	result, err := collection.InsertOne(ctx, scan)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to insert scan")
		return primitive.NilObjectID, err
	}

	return result.InsertedID.(primitive.ObjectID), nil
}

// UpdateScanResult overwrites the scan model with the new scan result
func (r *MongoHelper) UpdateScanResult(ctx context.Context, scan models.Scan) error {
	collection := r.client.Database(r.database).Collection(ScansCollection)
	filter := bson.M{"_id": scan.ID}
	update := bson.M{"$set": scan}

	_, err := collection.ReplaceOne(ctx, filter, update)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to update scan result")
		return err
	}

	return nil
}


func (r *MongoHelper) UpdateScanStatus(ctx context.Context, scanID primitive.ObjectID, status string) error {
	collection := r.client.Database(r.database).Collection(ScansCollection)
	filter := bson.M{"_id": scanID}
	update := bson.M{"$set": bson.M{"status": status}}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to update scan status")
		return err
	}

	return nil
}

func (r *MongoHelper) FindScanByID(ctx context.Context, scanID primitive.ObjectID) (models.Scan, error) {
	collection := r.client.Database(r.database).Collection(ScansCollection)
	var scan models.Scan
	err := collection.FindOne(ctx, bson.M{"_id": scanID}).Decode(&scan)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to find scan by ID")
		return scan, err
	}

	return scan, nil
}

func (r *MongoHelper) FindDomainByID(ctx context.Context, domainID primitive.ObjectID) (models.Domain, error) {
	collection := r.client.Database(r.database).Collection(DomainsCollection)
	var domain models.Domain
	err := collection.FindOne(ctx, bson.M{"_id": domainID}).Decode(&domain)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to find domain by ID")
		return domain, err
	}

	return domain, nil
}

func (r *MongoHelper) FindTemplateByID(ctx context.Context, templateID primitive.ObjectID) (models.Template, error) {
	collection := r.client.Database(r.database).Collection(NucleiTemplatesCollection)
	var template models.Template
	err := collection.FindOne(ctx, bson.M{"_id": templateID}).Decode(&template)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to find template by ID")
		return template, err
	}

	return template, nil
}
