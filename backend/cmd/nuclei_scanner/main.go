package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	appConfig "github.com/shannevie/unofficial_cybertrap/backend/configs"
	helpers "github.com/shannevie/unofficial_cybertrap/backend/internal/nuclei_scanner/helpers"
	"github.com/shannevie/unofficial_cybertrap/backend/internal/rabbitmq"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// Create error channel for goroutine errors
	errChan := make(chan error, 1)

	// Start main processing in a goroutine
	go func() {
		errChan <- processScans(ctx)
	}()

	// Wait for either completion or signal
	select {
	case err := <-errChan:
		if err != nil {
			log.Error().Err(err).Msg("Scan processing failed")
			os.Exit(1)
		}
	case sig := <-sigChan:
		log.Info().Msgf("Received signal: %v", sig)
		cancel() // Trigger graceful shutdown
		// Wait for cleanup with timeout
		cleanup := make(chan bool)
		go func() {
			// Wait for processing to finish
			<-errChan
			cleanup <- true
		}()

		select {
		case <-cleanup:
			log.Info().Msg("Graceful shutdown completed")
		case <-time.After(1 * time.Minute): // Increased from 30s to 1m
			log.Warn().Msg("Graceful shutdown timed out")
		}
	}
}

func processScans(ctx context.Context) error {
	// Set up memory monitoring
	memoryThreshold := float64(0.90) // 90% memory usage threshold
	go monitorMemory(memoryThreshold)

	// Force garbage collection before starting
	debug.FreeOSMemory()

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
	defer rabbitClient.Close()
	log.Info().Msg("RabbitMQ client initialized")

	// Get message from queue
	msg, ok, err := rabbitClient.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get message from RabbitMQ after all retries")
		return err
	}
	if !ok {
		log.Info().Msg("No message available, exiting...")
		return nil
	}

	// Parse the message into array of scan messages
	var scanMsgs []rabbitmq.ScanMessage
	if err := json.Unmarshal(msg.Body, &scanMsgs); err != nil {
		log.Fatal().Err(err).Msg("Failed to unmarshal message")
		msg.Nack(false, true)
		return err
	}

	// Create error channel and wait group for concurrent processing
	errChan := make(chan error, len(scanMsgs))
	var wg sync.WaitGroup

	// Process each scan message concurrently
	for _, scanMsg := range scanMsgs {
		wg.Add(1)
		go func(scan rabbitmq.ScanMessage) {
			defer wg.Done()

			// Initialize MongoDB for this goroutine
			mongoClient, err := helpers.NewMongoClient(ctx, config.MongoDbUri)
			if err != nil {
				errChan <- fmt.Errorf("MongoDB initialization failed for scan %s: %w", scan.ScanId, err)
				return
			}
			defer mongoClient.Disconnect(ctx)

			// Initialize mongoHelper for this goroutine
			mongoHelper := helpers.NewMongoHelper(mongoClient, config.MongoDbName)

			// Initialize S3 for this goroutine
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
				errChan <- fmt.Errorf("AWS config failed for scan %s: %w", scan.ScanId, err)
				return
			}

			s3Helper, err := helpers.NewS3Helper(awsCfg, config.TemplatesBucketName, config.ScanResultsBucketName)
			if err != nil {
				errChan <- fmt.Errorf("S3 initialization failed for scan %s: %w", scan.ScanId, err)
				return
			}

			// Create NucleiHelper for this goroutine
			nh := helpers.NewNucleiHelper(s3Helper, mongoHelper)

			// Setup template directory for this scan
			templateDir := filepath.Join(os.TempDir(), fmt.Sprintf("nuclei-templates-%s", scan.ScanId))
			if err := os.MkdirAll(templateDir, 0755); err != nil {
				errChan <- fmt.Errorf("failed to create template directory for scan %s: %w", scan.ScanId, err)
				return
			}
			defer os.RemoveAll(templateDir)

			// Download and process templates
			templateFilePaths, err := downloadTemplates(ctx, templateDir, scan.TemplateIds, mongoHelper, s3Helper)
			if err != nil {
				errChan <- fmt.Errorf("template processing failed for scan %s: %w", scan.ScanId, err)
				return
			}

			// Perform the scan
			if err = nh.ScanWithNuclei(
				ctx,
				scan.MultiScanId,
				scan.ScanId,
				scan.Domain,
				scan.DomainId,
				templateFilePaths,
				scan.TemplateIds,
				scan.ScanAllNuclei,
				config.Debug,
			); err != nil {
				errChan <- fmt.Errorf("nuclei scan failed for scan %s: %w", scan.ScanId, err)
				return
			}
		}(scanMsg)
	}

	// Wait for all scans to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Check for any errors
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
			log.Error().Err(err).Msg("Scan processing error")
		}
	}

	// Handle message acknowledgment based on results
	if len(errors) > 0 {
		msg.Nack(false, true) // Requeue on errors
		return fmt.Errorf("multiple scan errors occurred: %v", errors)
	}

	msg.Ack(false) // Acknowledge message if all scans completed successfully
	log.Info().Msgf("Successfully processed %d scans", len(scanMsgs))
	return nil
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

func handleError(ctx context.Context, err error, context string, scanID primitive.ObjectID, mongoHelper *helpers.MongoHelper, config appConfig.NucleiConfig) {
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

	// Send Slack notification for the error
	slackMessage := fmt.Sprintf("Scan failed for scanID: %s\nContext: %s\nError: %s", scanID.Hex(), context, err.Error())
	if slackErr := helpers.SendSlackNotification(config.SlackWebhookURL, slackMessage); slackErr != nil {
		log.Error().Err(slackErr).Msg("Failed to send Slack notification for error")
	}

	// Check if mongoHelper is initialized before updating the scan status
	if mongoHelper != nil {
		// Get the multi-scan ID from the scan
		scan, findErr := mongoHelper.FindScanByID(ctx, scanID)
		if findErr != nil {
			log.Error().Err(findErr).Str("scanID", scanID.Hex()).Msg("Failed to fetch scan for multi-scan update")
			return
		}

		// Update scan status
		if updateErr := mongoHelper.UpdateScanStatus(ctx, scanID, "failed", errorDetails); updateErr != nil {
			log.Error().
				Err(updateErr).
				Str("scanID", scanID.Hex()).
				Msg("Failed to update scan error status")
		}

		// Update multi-scan status if applicable
		if scan.MultiScanID != primitive.NilObjectID {
			if updateErr := updateMultiScanStatusInMain(ctx, mongoHelper, scan.MultiScanID, scanID, false); updateErr != nil {
				log.Error().
					Err(updateErr).
					Str("multiScanID", scan.MultiScanID.Hex()).
					Str("scanID", scanID.Hex()).
					Msg("Failed to update multi-scan status")
			}
		}
	} else {
		// Send Slack notification about the inability to update MongoDB
		slackMessage := fmt.Sprintf("MongoHelper is not initialized. Unable to update scan status for scanID: %s", scanID.Hex())
		if slackErr := helpers.SendSlackNotification(config.SlackWebhookURL, slackMessage); slackErr != nil {
			log.Error().Err(slackErr).Msg("Failed to send Slack notification about MongoDB initialization error")
		}
	}
}

// Monitor memory usage
func monitorMemory(threshold float64) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Calculate memory usage percentage
		memoryUsage := float64(m.Alloc) / float64(m.Sys)

		if memoryUsage > threshold {
			log.Warn().
				Float64("usage", memoryUsage*100).
				Msg("High memory usage detected")

			// Force garbage collection
			debug.FreeOSMemory()
		}
	}
}

// Add helper function
func updateMultiScanStatusInMain(ctx context.Context, mongoHelper *helpers.MongoHelper, multiScanID primitive.ObjectID, scanID primitive.ObjectID, isSuccess bool) error {
	multiScan, err := mongoHelper.FindMultiScanByID(ctx, multiScanID)
	if err != nil {
		return fmt.Errorf("failed to fetch multi scan: %w", err)
	}

	// Remove scanID from both arrays if it exists
	completedScans := removeObjectID(multiScan.CompletedScans, scanID)
	failedScans := removeObjectID(multiScan.FailedScans, scanID)

	// Add to appropriate array based on current status
	if isSuccess {
		completedScans = append(completedScans, scanID)
	} else {
		failedScans = append(failedScans, scanID)
	}

	// Update the multi-scan with new arrays
	if err := mongoHelper.UpdateMultiScanArrays(ctx, multiScanID, completedScans, failedScans); err != nil {
		return fmt.Errorf("failed to update multi-scan arrays: %w", err)
	}

	// Check if all scans are processed
	totalProcessed := len(completedScans) + len(failedScans)
	if totalProcessed >= multiScan.TotalScans {
		duration := time.Since(multiScan.ScanDate).Milliseconds()
		finalStatus := "failed"
		if len(completedScans) > 0 {
			finalStatus = "completed"
		}
		if err := mongoHelper.UpdateMultiScanCompletion(ctx, multiScanID, finalStatus, duration); err != nil {
			return fmt.Errorf("failed to update multi-scan completion: %w", err)
		}
	}

	return nil
}

// Helper function to remove an ObjectID from a slice
func removeObjectID(slice []primitive.ObjectID, target primitive.ObjectID) []primitive.ObjectID {
	result := make([]primitive.ObjectID, 0, len(slice))
	for _, id := range slice {
		if id != target {
			result = append(result, id)
		}
	}
	return result
}
