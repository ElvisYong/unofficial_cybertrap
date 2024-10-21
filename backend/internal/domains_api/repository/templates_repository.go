package repository

import (
	"context"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TemplatesRepository struct {
	s3Client       *s3.Client
	bucketName     string
	mongoClient    *mongo.Client
	mongoDbName    string
	collectionName string
}

// NewUserRepository creates a new instance of UserRepository
func NewTemplatesRepository(s3Client *s3.Client, bucketName string, mongoClient *mongo.Client, mongoDbName string) *TemplatesRepository {
	return &TemplatesRepository{
		s3Client:       s3Client,
		bucketName:     bucketName,
		mongoClient:    mongoClient,
		mongoDbName:    mongoDbName,
		collectionName: "nucleiTemplates",
	}
}

// Uploads to S3 repository
func (r *TemplatesRepository) UploadToS3(file multipart.File, filename string) (string, error) {
	uploader := manager.NewUploader(r.s3Client)

	result, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: &r.bucketName,
		Key:    &filename,
		Body:   file,
	})

	if err != nil {
		log.Error().Err(err).Msg("Error uploading file to S3")
		return "", err
	}

	return result.Location, nil
}

// UploadToMongo inserts a template into MongoDB
func (r *TemplatesRepository) UploadToMongo(template *models.Template) (string, error) {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)
	result, err := collection.InsertOne(context.Background(), template)

	if err != nil {
		log.Error().Err(err).Msg("Error inserting template into MongoDB")
		return "", err
	}

	insertedID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		log.Error().Msg("Failed to convert inserted ID to ObjectID")
		return "", err
	}

	return insertedID.Hex(), nil
}

// GetTemplatesByIDs retrieves templates from MongoDB by their IDs
func (r *TemplatesRepository) GetTemplatesByIDs(ids []primitive.ObjectID) ([]models.Template, error) {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)

	// Create a filter to match documents with the given IDs
	filter := bson.M{"_id": bson.M{"$in": ids}}

	// Execute the query
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching templates from MongoDB")
		return nil, err
	}
	defer cursor.Close(context.Background())

	var templates []models.Template
	if err = cursor.All(context.Background(), &templates); err != nil {
		log.Error().Err(err).Msg("Error decoding templates from MongoDB cursor")
		return nil, err
	}

	return templates, nil
}

func (r *TemplatesRepository) GetAllTemplates() ([]models.Template, error) {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Error().Err(err).Msg("Error fetching templates from MongoDB")
		return nil, err
	}

	var templates []models.Template

	if err = cursor.All(context.Background(), &templates); err != nil {
		log.Error().Err(err).Msg("Error populating templates from MongoDB cursor")
		return nil, err
	}

	return templates, nil
}

func (r *TemplatesRepository) DeleteTemplateById(id string) error {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)

	objectId, err := primitive.ObjectIDFromHex(id) // converting to mongodb object id
	if err != nil {
		log.Error().Err(err).Msg("Error converting template ID to Object")
		return err
	}

	_, err = collection.DeleteOne(context.Background(), bson.M{"_id": objectId})
	if err != nil {
		log.Error().Err(err).Msg("Error deleting template from MongoDB")
		return err
	}

	return nil
}
