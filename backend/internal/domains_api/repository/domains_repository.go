package repository

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

// GetDomainsByIDs retrieves multiple domains by their IDs
func (r *DomainsRepository) GetDomainsByIDs(ids []primitive.ObjectID) ([]models.Domain, error) {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)

	filter := bson.M{"_id": bson.M{"$in": ids}}
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching domains from MongoDB")
		return nil, err
	}
	defer cursor.Close(context.Background())

	var domains []models.Domain
	if err = cursor.All(context.Background(), &domains); err != nil {
		log.Error().Err(err).Msg("Error decoding domains from MongoDB cursor")
		return nil, err
	}

	return domains, nil
}

// GetDomainByID gets a domain by its ID and return the domain
func (r *DomainsRepository) GetDomainByID(id string) (*models.Domain, error) {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)

	objectId, err := primitive.ObjectIDFromHex(id) // converting to mongodb object id
	if err != nil {
		log.Error().Err(err).Msg("Error converting domain ID to Object")
		return nil, err
	}

	var domain models.Domain
	err = collection.FindOne(context.Background(), bson.M{"_id": objectId}).Decode(&domain)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching domain from MongoDB")
		return nil, err
	}

	return &domain, nil
}

func (r *DomainsRepository) DeleteDomainById(id string) error {
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

// InsertDomains inserts multiple domains into the MongoDB collection
// Note if there is a duplicate domain we will not insert it
func (r *DomainsRepository) InsertDomains(domains []models.Domain) error {
	collection := r.mongoClient.Database(r.mongoDbName).Collection("domains")
	var documents []interface{}

	for _, domain := range domains {
		// Check if the domain already exists
		filter := bson.M{"domain": domain.Domain} // Assuming 'DomainName' is the unique field
		count, err := collection.CountDocuments(context.Background(), filter)
		if err != nil {
			log.Error().Err(err).Msg("Error checking for existing domain")
			return err
		}

		// If the domain does not exist, add it to the documents slice
		if count == 0 {
			documents = append(documents, domain)
		}
	}

	if len(documents) > 0 {
		_, err := collection.InsertMany(context.Background(), documents)
		if err != nil {
			log.Error().Err(err).Msg("Error inserting domains into MongoDB")
			return err
		}
	}

	return nil
}

// InsertDomains inserts multiple domains into the MongoDB collection
// Note if there is a duplicate domain we will not insert it
func (r *DomainsRepository) InsertSingleDomain(domain models.Domain) error {
	collection := r.mongoClient.Database(r.mongoDbName).Collection(r.collectionName)

	_, err := collection.InsertOne(context.Background(), domain)

	if err != nil {
		log.Error().Err(err).Msg("Error inserting domains into MongoDB")
		return err
	}

	return nil

}
