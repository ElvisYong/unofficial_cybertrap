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
)

// Helper function to format error details
func formatErrorDetails(err error, context string) map[string]interface{} {
	return map[string]interface{}{
		"error":     err.Error(),
		"context":   context,
		"timestamp": time.Now(),
	}
}

func main() {
	ctx := context.Background()

	// Initialize logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load configuration
	config, err := appConfig.LoadNucleiConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load app config")
	}

	// Initialize RabbitMQ first
	rabbitClient, err := rabbitmq.NewRabbitMQClient(config.RabbitMqUri)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create RabbitMQ client")
	}

	// Get message from queue
	msg, ok, err := rabbitClient.Get()
	if err != nil {
		rabbitClient.Close()
		log.Error().Err(err).Msg("Failed to get message from RabbitMQ")
		os.Exit(1)
	}
	if !ok {
		rabbitClient.Close()
		log.Info().Msg("No message available, exiting normally")
		os.Exit(0)
	}

	// Acknowledge message and close connection
	msg.Ack(false)
	rabbitClient.Close()
	log.Info().Msg("RabbitMQ message acknowledged and connection closed")

	// Parse the message
	var scanMsg rabbitmq.ScanMessage
	if err := json.Unmarshal(msg.Body, &scanMsg); err != nil {
		log.Fatal().Err(err).Msg("Failed to unmarshal message")
	}
	log.Info().Msgf("Processing scan message for domain: %s", scanMsg.Domain)

	// Initialize MongoDB
	mongoClient, err := helpers.NewMongoClient(ctx, config.MongoDbUri)
	if err != nil {
		log.Error().Err(err).Str("scanId", scanMsg.ScanId.Hex()).Msg("MongoDB initialization failed")
		return
	}
	defer mongoClient.Disconnect(ctx)

	mongoHelper := helpers.NewMongoHelper(mongoClient, config.MongoDbName)

	// Initialize S3
	awsCfg, err := awsConfig.LoadDefaultConfig(
		ctx,
		awsConfig.WithRegion("ap-southeast-1"),
		awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				config.AwsAccessKeyId,
				config.AwsSecretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		handleError(ctx, err, "Failed to load AWS config", scanMsg.ScanId, mongoHelper)
		return
	}

	s3Helper, err := helpers.NewS3Helper(awsCfg, config.TemplatesBucketName, config.ScanResultsBucketName)
	if err != nil {
		handleError(ctx, err, "Failed to initialize S3", scanMsg.ScanId, mongoHelper)
		return
	}

	// Create NucleiHelper
	nh := helpers.NewNucleiHelper(s3Helper, mongoHelper)

	// Update scan status to in-progress
	if err = mongoHelper.UpdateScanStartTime(ctx, scanMsg.ScanId, time.Now()); err != nil {
		handleError(ctx, err, "Failed to update scan start time", scanMsg.ScanId, mongoHelper)
		return
	}

	// Setup template directory
	templateDir := filepath.Join(os.TempDir(), "nuclei-templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		handleError(ctx, err, "Failed to create template directory", scanMsg.ScanId, mongoHelper)
		return
	}
	nuclei.DefaultConfig.TemplatesDirectory = templateDir

	// Download and process templates
	templateFilePaths, err := downloadTemplates(ctx, templateDir, scanMsg.TemplateIds, mongoHelper, s3Helper)
	if err != nil {
		handleError(ctx, err, "Template processing failed", scanMsg.ScanId, mongoHelper)
		return
	}

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
		handleError(ctx, err, "Nuclei scan failed", scanMsg.ScanId, mongoHelper)
		return
	}

	log.Info().Msg("Scan completed successfully")
	// Clean up template directory
	if err := os.RemoveAll(templateDir); err != nil {
		log.Error().Err(err).Msg("Failed to clean up template directory")
	}
	os.Exit(0)
}

func downloadTemplates(ctx context.Context, templateDir string, templateIds []primitive.ObjectID, mongoHelper *helpers.MongoHelper, s3Helper *helpers.S3Helper) ([]string, error) {
	var wg sync.WaitGroup
	templateFilePaths := make([]string, 0, len(templateIds))
	errChan := make(chan error, len(templateIds))
	pathChan := make(chan string, len(templateIds))

	for _, templateId := range templateIds {
		wg.Add(1)
		go func(templateId primitive.ObjectID) {
			defer wg.Done()

			template, err := mongoHelper.FindTemplateByID(ctx, templateId)
			if err != nil {
				errChan <- fmt.Errorf("failed to find template %s: %w", templateId.Hex(), err)
				return
			}

			templateFilePath := filepath.Join(templateDir, fmt.Sprintf("template-%s.yaml", templateId.Hex()))
			if err = s3Helper.DownloadFileFromURL(template.S3URL, templateFilePath); err != nil {
				errChan <- fmt.Errorf("failed to download template %s: %w", templateId.Hex(), err)
				return
			}

			pathChan <- templateFilePath
		}(templateId)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(errChan)
		close(pathChan)
	}()

	// Collect paths and check for errors
	for path := range pathChan {
		templateFilePaths = append(templateFilePaths, path)
	}

	var errors []string
	for err := range errChan {
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("template download errors: %v", errors)
	}

	return templateFilePaths, nil
}

func handleError(ctx context.Context, err error, context string, scanID primitive.ObjectID, mongoHelper *helpers.MongoHelper) {
	errorDetails := map[string]interface{}{
		"message":   err.Error(),
		"context":   context,
		"timestamp": time.Now(),
	}

	log.Error().
		Err(err).
		Str("scanID", scanID.Hex()).
		Interface("details", errorDetails).
		Msg(context)

	if updateErr := mongoHelper.UpdateScanStatus(ctx, scanID, "failed", errorDetails); updateErr != nil {
		log.Error().
			Err(updateErr).
			Str("scanID", scanID.Hex()).
			Msg("Failed to update scan error status")
	}
}
