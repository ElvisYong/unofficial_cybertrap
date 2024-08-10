package repository

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DomainsRepository struct {
	mongoClient    *mongo.Client
	mongoDbName    string
	collectionName string
}

// NewUserRepository creates a new instance of UserRepository
func NewDomainsRepository(mongoClient *mongo.Client, mongoDbName string) *DomainsRepository {
	return &DomainsRepository{
		mongoClient:    mongoClient,
		mongoDbName:    mongoDbName,
		collectionName: "domains",
	}
}

func (r *DomainsRepository) GetAllDomains() ([]models.Domain, error) {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Error().Err(err).Msg("Error fetching domains from MongoDB")
		return nil, err
	}

	var domains []models.Domain

	if err = cursor.All(context.Background(), &domains); err != nil {
		log.Error().Err(err).Msg("Error populating domains from MongoDB cursor")
		return nil, err
	}

	return domains, nil
}

// InsertDomains inserts multiple domains into the MongoDB collection
// Note if there is a duplicate domain we will not insert it
func (r *DomainsRepository) InsertDomains(domains []models.Domain) error {
	collection := r.mongoClient.Database(r.mongoDbName).Collection("domains")
	var documents []interface{}
	for _, domain := range domains {
		documents = append(documents, domain)
	}

	_, err := collection.InsertMany(context.Background(), documents)
	if err != nil {
		log.Error().Err(err).Msg("Error inserting domains into MongoDB")
		return err
	}

	return nil
}
