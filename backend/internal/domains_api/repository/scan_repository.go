package repository

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ScansRepository struct {
	mongoClient    *mongo.Client
	mongoDbName    string
	collectionName string
}

// NewUserRepository creates a new instance of UserRepository
func NewScansRepository(mongoClient *mongo.Client, mongoDbName string) *ScansRepository {
	return &ScansRepository{
		mongoClient:    mongoClient,
		mongoDbName:    mongoDbName,
		collectionName: "scans",
	}
}

func (r *ScansRepository) GetAllScans() ([]models.Scan, error) {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Error().Err(err).Msg("Error fetching scans from MongoDB")
		return nil, err
	}

	var scans []models.Scan

	if err = cursor.All(context.Background(), &scans); err != nil {
		log.Error().Err(err).Msg("Error populating scans from MongoDB cursor")
		return nil, err
	}

	return scans, nil
}

func (r *ScansRepository) InsertSingleScan(scan models.Scan) error {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)

	_, err := collection.InsertOne(context.Background(), scan)

	if err != nil {
		log.Error().Err(err).Msg("Error inserting scans into MongoDB")
		return err
	}

	return nil

}

func (r *ScansRepository) BatchInsertScans(scans []models.Scan) error {
	collection := r.mongoClient.Database(r.mongoDbName).Collection("scans")
	var documents []interface{}
	for _, scan := range scans {
		documents = append(documents, scan)
	}

	_, err := collection.InsertMany(context.Background(), documents)
	if err != nil {
		log.Error().Err(err).Msg("Error inserting domains into MongoDB")
		return err
	}

	return nil
}

func (r *ScansRepository) GetAllMultiScans() ([]models.MultiScan, error) {
	collection := r.mongoClient.Database(r.mongoDbName).Collection("multi_scans")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Error().Err(err).Msg("Error fetching multi-scans from MongoDB")
		return nil, err
	}

	var multiScans []models.MultiScan

	if err = cursor.All(context.Background(), &multiScans); err != nil {
		log.Error().Err(err).Msg("Error populating multi-scans from MongoDB cursor")
		return nil, err
	}

	return multiScans, nil
}

func (r *ScansRepository) GetScansByIds(scanIds []primitive.ObjectID) ([]models.Scan, error) {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)
	filter := bson.M{"_id": bson.M{"$in": scanIds}}
	
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching scans from MongoDB")
		return nil, err
	}
	defer cursor.Close(context.Background())

	var scans []models.Scan
	if err = cursor.All(context.Background(), &scans); err != nil {
		log.Error().Err(err).Msg("Error decoding scans from cursor")
		return nil, err
	}

	return scans, nil
}

func (r *ScansRepository) UpdateScanWithDuration(ctx context.Context, scanID primitive.ObjectID, status string, duration int64) error {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)
	
	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"scan_took": duration,
		},
	}
	
	_, err := collection.UpdateOne(ctx, bson.M{"_id": scanID}, update)
	if err != nil {
		log.Error().Err(err).Msg("Error updating scan duration in MongoDB")
		return err
	}
	
	return nil
}
