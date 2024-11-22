package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
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

	// Change the common templates directory to /root/nuclei-templates
	commonTemplateDir := "/root"
	if err := os.MkdirAll(commonTemplateDir, 0755); err != nil {
		log.Fatal().Err(err).Msg("Failed to create common template directory")
	}

	// Initialize S3 before message processing
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
		log.Fatal().Err(err).Msg("Failed to initialize AWS config")
		return err
	}

	s3Helper, err := helpers.NewS3Helper(awsCfg, config.TemplatesBucketName, config.ScanResultsBucketName)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize S3 helper")
		return err
	}

	// Download all templates once before processing any messages
	log.Info().Msg("Starting download of all nuclei templates...")
	startTime := time.Now()
	if err := s3Helper.DownloadAllTemplates(commonTemplateDir); err != nil {
		log.Fatal().Err(err).Msg("Failed to download all templates")
		return err
	}
	downloadDuration := time.Since(startTime)
	log.Info().
		Dur("duration", downloadDuration).
		Str("location", commonTemplateDir).
		Msg("Successfully downloaded all nuclei templates")

	// Move .nuclei-ignore file to the correct location
	nucleiIgnoreSrc := filepath.Join(commonTemplateDir, "nuclei-templates", ".nuclei-ignore")
	nucleiIgnoreDst := filepath.Join("/root", ".config", "nuclei", ".nuclei-ignore")
	
	// Create .config directory if it doesn't exist
	if err := os.MkdirAll(filepath.Join("/root", ".config"), 0755); err != nil {
		log.Fatal().Err(err).Msg("Failed to create .config directory")
		return err
	}
	
	// Move the file
	if err := os.Rename(nucleiIgnoreSrc, nucleiIgnoreDst); err != nil {
		log.Fatal().Err(err).Msg("Failed to move .nuclei-ignore file")
		return err
	}
	log.Info().Msg("Successfully moved .nuclei-ignore file to /root/.config/")

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
	log.Info().Msgf("Received message: %s", string(msg.Body))

	// Parse the message into array of scan messages
	var scanMsgs []rabbitmq.ScanMessage
	if err := json.Unmarshal(msg.Body, &scanMsgs); err != nil {
		log.Error().
			Err(err).
			Str("rawMessage", string(msg.Body)).
			Msg("Failed to unmarshal message")
		msg.Nack(false, true)
		return err
	}
	log.Info().Msgf("Parsed scan messages: %v", scanMsgs)

	// Create error channel and wait group for concurrent processing
	errChan := make(chan error, len(scanMsgs))
	var wg sync.WaitGroup

	// Create a semaphore to control concurrent scans
	maxConcurrent := make(chan struct{}, config.MaxConcurrentScans)
	log.Info().Int("maxConcurrentScans", config.MaxConcurrentScans).Msg("Initialized concurrent scan limiter")

	// Process each scan message with controlled concurrency
	for _, scanMsg := range scanMsgs {
		wg.Add(1)
		go func(scan rabbitmq.ScanMessage) {
			// Acquire semaphore
			maxConcurrent <- struct{}{}
			defer func() {
				// Release semaphore
				<-maxConcurrent
				wg.Done()
			}()

			// Initialize MongoDB for this goroutine
			mongoClient, err := helpers.NewMongoClient(ctx, config.MongoDbUri)
			if err != nil {
				errChan <- fmt.Errorf("MongoDB initialization failed for scan %s: %w", scan.ScanId, err)
				return
			}
			defer mongoClient.Disconnect(ctx)

			// Initialize mongoHelper for this goroutine
			mongoHelper := helpers.NewMongoHelper(mongoClient, config.MongoDbName)

			// Create NucleiHelper for this goroutine
			nh := helpers.NewNucleiHelper(s3Helper, mongoHelper)

			// Use templates from the common directory
			var templateFilePaths []string
			if scan.ScanAllNuclei {
				// Use all templates from common directory
				err := filepath.Walk(commonTemplateDir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
						relativePath, err := filepath.Rel(commonTemplateDir, path)
						if err == nil {
							log.Debug().Str("template", relativePath).Msg("Adding template to scan")
						}
						templateFilePaths = append(templateFilePaths, path)
					}
					return nil
				})
				if err != nil {
					errChan <- fmt.Errorf("failed to walk template directory for scan %s: %w", scan.ScanId, err)
					return
				}
				log.Info().Int("templateCount", len(templateFilePaths)).Msg("Found templates for all-nuclei scan")
			} else {
				// For specific templates, just use the paths from common directory
				for _, templateId := range scan.TemplateIds {
					template, err := mongoHelper.FindTemplateByID(ctx, templateId)
					if err != nil {
						errChan <- fmt.Errorf("failed to find template %s: %w", templateId.Hex(), err)
						return
					}

					// Extract template name from S3 URL and construct path in common directory
					parsedURL, _ := url.Parse(template.S3URL)
					templatePath := filepath.Join(commonTemplateDir, filepath.Base(parsedURL.Path))
					templateFilePaths = append(templateFilePaths, templatePath)
				}
			}

			// Perform the scan with the template paths
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
