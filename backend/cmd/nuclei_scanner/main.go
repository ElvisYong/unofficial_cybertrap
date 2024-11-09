package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

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

func connectWithRetry(uri string, maxRetries int) (*mongo.Client, error) {
	var client *mongo.Client
	var err error

	// Configure connection pool
	clientOpts := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(5).           // Limit concurrent connections
		SetMinPoolSize(1).           // Keep at least one connection
		SetMaxConnecting(2).         // Limit new connections being established
		SetRetryReads(true).         // Enable retry for read operations
		SetRetryWrites(true).        // Enable retry for write operations
		SetTimeout(10 * time.Second) // Set operation timeout

	for i := 0; i < maxRetries; i++ {
		client, err = mongo.Connect(context.Background(), clientOpts)
		if err == nil {
			// Test the connection
			err = client.Ping(context.Background(), nil)
			if err == nil {
				return client, nil
			}
		}
		log.Warn().Msgf("Failed to connect to MongoDB (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
	}
	return nil, fmt.Errorf("failed to connect after %d attempts: %v", maxRetries, err)
}

func main() {
	// Start logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load application config
	config, err := appConfig.LoadNucleiConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load app config")
	}

	// Initialize MongoDB client with retry
	mongoClient, err := connectWithRetry(config.MongoDbUri, 5)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MongoDB after retries")
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

	// Function to update scan status on failure
	updateFailureStatus := func(err error) {
		log.Error().Err(err).Msg("Operation failed")
		if updateErr := mongoHelper.UpdateScanStatus(context.Background(), scanMsg.ScanId, "failed"); updateErr != nil {
			log.Error().Err(updateErr).Msg("Failed to update scan status to failed")
		}
		os.Exit(1)
	}

	log.Info().Msgf("Processing message: %s", msg.Body)

	nh := helpers.NewNucleiHelper(s3Helper, mongoHelper)

	// Update scan status to "in-progress"
	log.Info().Msgf("Updating scan status to in-progress")
	if err = mongoHelper.UpdateScanStatus(context.Background(), scanMsg.ScanId, "in-progress"); err != nil {
		updateFailureStatus(fmt.Errorf("failed to update scan status to in-progress: %w", err))
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

	// Check for any errors during template processing
	if err := processErrors(errChan); err != nil {
		updateFailureStatus(fmt.Errorf("template processing failed: %w", err))
	}

	log.Info().Msg("Successfully downloaded templates")

	// Perform the scan
	if err = nh.ScanWithNuclei(
		scanMsg.MultiScanId,
		scanMsg.ScanId,
		scanMsg.Domain,
		scanMsg.DomainId,
		templateFilePaths,
		scanMsg.TemplateIds,
		scanMsg.ScanAllNuclei,
		config.Debug,
	); err != nil {
		updateFailureStatus(fmt.Errorf("nuclei scan failed: %w", err))
	}

	// Update status to completed on success
	if err = mongoHelper.UpdateScanStatus(context.Background(), scanMsg.ScanId, "completed"); err != nil {
		updateFailureStatus(fmt.Errorf("failed to update scan status to completed: %w", err))
	}

	log.Info().Msg("Scan completed successfully, exiting...")
	os.Exit(0)
}

func processErrors(errChan chan error) error {
	var errors []string
	for err := range errChan {
		if err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errors)
	}
	return nil
}
