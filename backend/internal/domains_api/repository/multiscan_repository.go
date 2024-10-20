package repository

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MultiScanRepository struct {
	mongoClient    *mongo.Client
	mongoDbName    string
	collectionName string
}

// NewUserRepository creates a new instance of UserRepository
func NewMultiScanRepository(mongoClient *mongo.Client, mongoDbName string) *MultiScanRepository {
	return &MultiScanRepository{
		mongoClient:    mongoClient,
		mongoDbName:    mongoDbName,
		collectionName: "multiScans",
	}
}

func (r *MultiScanRepository) GetAllMultiScans() ([]models.MultiScan, error) {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Error().Err(err).Msg("Error fetching scans from MongoDB")
		return nil, err
	}

	var multiScans []models.MultiScan

	if err = cursor.All(context.Background(), &multiScans); err != nil {
		log.Error().Err(err).Msg("Error populating multi scans from MongoDB cursor")
		return nil, err
	}

	return multiScans, nil
}

func (r *MultiScanRepository) CreateMultiScan(scan models.MultiScan) error {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)

	_, err := collection.InsertOne(context.Background(), scan)

	if err != nil {
		log.Error().Err(err).Msg("Error inserting scans into MongoDB")
		return err
	}

	return nil

}
