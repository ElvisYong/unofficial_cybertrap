package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-chi/httplog"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	appConfig "github.com/shannevie/unofficial_cybertrap/backend/configs"
	"github.com/shannevie/unofficial_cybertrap/backend/internal/domains_api/repository"
	"github.com/shannevie/unofficial_cybertrap/backend/internal/rabbitmq"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
)

func main() {
	// Start logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	httplog.Configure(httplog.Options{Concise: true, TimeFieldFormat: time.DateTime})

	// load env configurations
	appConfig, err := appConfig.LoadSchedulerConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("unable to load configurations")
	}

	// Prepare external services such as db, cache, etc.

	// Setup mongodb
	clientOpts := options.Client().ApplyURI(appConfig.MongoDbUri)
	mongoClient, err := mongo.Connect(context.Background(), clientOpts)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}
	scansRepo := repository.NewScansRepository(mongoClient, appConfig.MongoDbName)
	multiScanRepo := repository.NewMultiScanRepository(mongoClient, appConfig.MongoDbName)

	// Setup rabbitmq client
	mqClient, err := rabbitmq.NewRabbitMQClient(appConfig.RabbitMqUri)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to RabbitMQ")
	}

	// Use mongo client to get all schedule scans for today
	collection := mongoClient.Database(appConfig.MongoDbName).Collection("ScheduledScans")
	// Get the current date (ignoring the time part)
	today := time.Now()
	justDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	// MongoDB filter to match only documents where the start_scan date is equal to today's date
	filter := bson.M{
		"start_scan": bson.M{
			"$eq": justDate,
		},
	}

	var scheduleScans []models.ScheduleScan

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to find scans for today in MongoDB")
		return
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var result models.ScheduleScan
		if err := cursor.Decode(&result); err != nil {
			log.Fatal().Err(err).Msg("Failed to decode results from MongoDB")
			return
		}
		// Append the result to the results array
		scheduleScans = append(scheduleScans, result)
		fmt.Printf("Scan found: %+v\n", result)
	}

	if err := cursor.Err(); err != nil {
		log.Fatal().Err(err)
		return
	}

	// Now you can work with the 'results' slice which contains all the decoded scans
	fmt.Printf("All Scans: %+v\n", scheduleScans)

	for _, scheduleScan := range scheduleScans {
		// Create a multi scan id
		multiScanId := primitive.NewObjectID()

		// Create a multi scan data with in-progress status
		multiScanModel := models.MultiScan{
			ID:             multiScanId,
			ScanIDs:        make([]primitive.ObjectID, 0),
			Name:           "Scheduled Scan",
			TotalScans:     len(scheduleScan.DomainIds),
			CompletedScans: make([]primitive.ObjectID, 0),
			FailedScans:    make([]primitive.ObjectID, 0),
			Status:         "in-progress",
		}

		scanArray := make([]models.Scan, 0)
		for _, domainId := range scheduleScan.DomainIds {
			scanId := primitive.NewObjectID()
			multiScanModel.ScanIDs = append(multiScanModel.ScanIDs, scanId)
			scanModel := models.Scan{
				ID:          scanId,
				DomainId:    domainId,
				TemplateIDs: scheduleScan.TemplatesIDs,
				Status:      "pending",
			}
			scanArray = append(scanArray, scanModel)
		}

		errscan := scansRepo.BatchInsertScans(scanArray) // update scans to pending
		if errscan != nil {
			log.Error().Err(errscan).Msg("Error multi scan into the database")
			continue
		}

		for _, scan := range scanArray {
			messageJson := rabbitmq.ScanMessage{
				MultiScanId:   multiScanId,
				ScanId:        scan.ID,
				TemplateIds:   scan.TemplateIDs,
				DomainId:      scan.DomainId,
				Domain:        scan.Domain,
				ScanAllNuclei: scheduleScan.ScanAll,
			}

			// Send the message to the queue
			err := mqClient.Publish(messageJson)
			if err != nil {
				log.Error().Err(err).Msg("Error sending scan message to queue")
				return
			}
		}

		errmultiScan := multiScanRepo.CreateMultiScan(multiScanModel)
		if errmultiScan != nil {
			log.Error().Err(errmultiScan).Msg("Error multi scan into the database")
			continue
		}

	}

	log.Log().Msg("Finished publishing to rabbitMQ")
	// for loop all the schedule scans and using mq client to send into mq
	// delete all scheduled scans for today

	// MongoDB filter to match only documents where the start_scan date is equal to today's date

	// Remove all records where the start_scan date is today
	deleteResult, err := collection.DeleteMany(context.Background(), filter)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to delete scans for today in MongoDB")
		return
	}

	fmt.Printf("Number of records deleted: %d\n", deleteResult.DeletedCount)

}
