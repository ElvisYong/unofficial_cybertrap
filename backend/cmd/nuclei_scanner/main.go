package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	nuclei "github.com/projectdiscovery/nuclei/v3/lib"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	appConfig "github.com/shannevie/unofficial_cybertrap/backend/configs"
	helpers "github.com/shannevie/unofficial_cybertrap/backend/internal/nuclei_scanner/helpers"
	"github.com/shannevie/unofficial_cybertrap/backend/internal/rabbitmq"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Start logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load application config
	config, err := appConfig.LoadNucleiConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load app config")
	}

	// Initialize MongoDB client
	clientOpts := options.Client().ApplyURI(config.MongoDbUri)
	mongoClient, err := mongo.Connect(context.Background(), clientOpts)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}
	defer mongoClient.Disconnect(context.Background())

	// Initialize MongoDB repository
	mongoHelper := helpers.NewMongoHelper(mongoClient, config.MongoDbName)
	log.Info().Msg("MongoDB client initialized")

	// Initialize RabbitMQ client
	rabbitClient, err := rabbitmq.NewRabbitMQClient(config.RabbitMqUri)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create rabbitmq client")
	}
	log.Info().Msg("RabbitMQ client initialized")
	defer rabbitClient.Close()

	// Initialize S3 helper
	awsCfg, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithRegion("ap-southeast-1"), awsConfig.WithCredentialsProvider(
		credentials.NewStaticCredentialsProvider(config.AwsAccessKeyId, config.AwsSecretAccessKey, ""),
	))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load AWS config")
	}

	s3Helper, err := helpers.NewS3Helper(awsCfg, config.TemplatesBucketName, config.ScanResultsBucketName)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create S3 helper")
	}

	templateDir := filepath.Join(os.TempDir(), "nuclei-templates")
	nuclei.DefaultConfig.TemplatesDirectory = templateDir

	// Get single message from RabbitMQ
	msg, ok, err := rabbitClient.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get message from RabbitMQ")
	}
	if !ok {
		log.Info().Msg("No message available, exiting...")
		os.Exit(0)
	}

	// Acknowledge message immediately
	msg.Ack(false)

	var scanMsg rabbitmq.ScanMessage
	if err := json.Unmarshal(msg.Body, &scanMsg); err != nil {
		log.Fatal().Err(err).Msg("Failed to unmarshal message")
	}

	log.Info().Msgf("Processing message: %s", msg.Body)

	nh := helpers.NewNucleiHelper(s3Helper, mongoHelper)

	// Update scan status to "in-progress"
	log.Info().Msgf("Updating scan status to in-progress")
	err = mongoHelper.UpdateScanStatus(context.Background(), scanMsg.ScanId, "in-progress")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to update scan status")
	}

	// Download templates
	var wg sync.WaitGroup
	templateFilePaths := make([]string, 0, len(scanMsg.TemplateIds))
	errChan := make(chan error, len(scanMsg.TemplateIds))

	log.Info().Msgf("Downloading templates")
	for _, templateId := range scanMsg.TemplateIds {
		wg.Add(1)
		go func(templateId primitive.ObjectID) {
			defer wg.Done()

			template, err := mongoHelper.FindTemplateByID(context.Background(), templateId)
			if err != nil {
				errChan <- fmt.Errorf("failed to find template by ID: %s, error: %w", templateId.Hex(), err)
				return
			}

			templateFilePath := filepath.Join(templateDir, fmt.Sprintf("template-%s.yaml", templateId.Hex()))
			log.Info().Msgf("Downloading template %s to %s", template.S3URL, templateFilePath)

			err = s3Helper.DownloadFileFromURL(template.S3URL, templateFilePath)
			if err != nil {
				errChan <- fmt.Errorf("failed to download template file from S3 for ID: %s, error: %w", templateId.Hex(), err)
				return
			}

			templateFilePaths = append(templateFilePaths, templateFilePath)
		}(templateId)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		log.Error().Err(err).Msg("Error occurred during template processing")
		os.Exit(1)
	}

	log.Info().Msg("Successfully downloaded templates")

	// Perform the scan
	nh.ScanWithNuclei(scanMsg.MultiScanId, scanMsg.ScanId, scanMsg.Domain, scanMsg.DomainId, templateFilePaths, scanMsg.TemplateIds, scanMsg.ScanAllNuclei, config.Debug)

	log.Info().Msg("Scan completed successfully, exiting...")
	os.Exit(0)
}
