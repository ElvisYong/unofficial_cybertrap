package repository

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ScheduledScanRepository struct {
	mongoClient    *mongo.Client
	mongoDbName    string
	collectionName string
}

// NewUserRepository creates a new instance of UserRepository
func NewScheduledScanRepository(mongoClient *mongo.Client, mongoDbName string) *ScheduledScanRepository {
	return &ScheduledScanRepository{
		mongoClient:    mongoClient,
		mongoDbName:    mongoDbName,
		collectionName: "scheduledScans",
	}
}

func (r *ScheduledScanRepository) GetAllScheduledScans() ([]models.ScheduleScan, error) {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Error().Err(err).Msg("Error fetching scans from MongoDB")
		return nil, err
	}

	var scans []models.ScheduleScan

	if err = cursor.All(context.Background(), &scans); err != nil {
		log.Error().Err(err).Msg("Error populating scans from MongoDB cursor")
		return nil, err
	}

	return scans, nil
}

func (r *ScheduledScanRepository) InsertSingleScheduledScan(scan models.Scan) error {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)

	_, err := collection.InsertOne(context.Background(), scan)

	if err != nil {
		log.Error().Err(err).Msg("Error inserting scans into MongoDB")
		return err
	}

	return nil

}

func (r *ScheduledScanRepository) CreateScheduleScanRecord(scheduledscan models.ScheduleScan) error {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)

	_, err := collection.InsertOne(context.Background(), scheduledscan)

	if err != nil {
		log.Error().Err(err).Msg("Error inserting scheduled scans into MongoDB")
		return err
	}

	return nil

}

func (r *ScheduledScanRepository) DeleteScheduledScanByID(id string) error {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)

	objectId, err := primitive.ObjectIDFromHex(id) // converting to mongodb object id
	if err != nil {
		log.Error().Err(err).Msg("Error converting domain ID to Object")
		return err
	}

	_, err = collection.DeleteOne(context.Background(), bson.M{"_id": objectId})
	if err != nil {
		log.Error().Err(err).Msg("Error deleting domain from MongoDB")
		return err
	}

	return nil
}
