package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	nuclei "github.com/projectdiscovery/nuclei/v3/lib"
	amqp "github.com/rabbitmq/amqp091-go"
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

	// Declare exchange and queue
	err = rabbitClient.DeclareExchangeAndQueue()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to declare exchange and queue")
	}

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

	// Signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	templateDir := filepath.Join(os.TempDir(), "nuclei-templates")
	nuclei.DefaultConfig.TemplatesDirectory = templateDir

	// Set the maximum number of concurrent scans
	maxConcurrentScans := config.MaxConcurrentScans
	semaphore := make(chan struct{}, maxConcurrentScans)

	// Process messages from RabbitMQ
	log.Info().Msg("Starting to process messages from RabbitMQ")
	for {
		select {
		case <-signalChan:
			log.Info().Msg("Received shutdown signal. Cleaning up...")
			close(semaphore)
			for i := 0; i < maxConcurrentScans; i++ {
				semaphore <- struct{}{}
			}
			return
		default:
			semaphore <- struct{}{} // Acquire a slot

			msg, ok, err := rabbitClient.Get()
			if err != nil {
				log.Error().Err(err).Msg("Failed to get message from RabbitMQ")
				<-semaphore // Release the slot
				continue
			}
			if !ok {
				log.Debug().Msg("No message available, waiting...")
				<-semaphore                 // Release the slot
				time.Sleep(1 * time.Second) // Wait before trying again
				continue
			}

			go func(msg *amqp.Delivery) {
				defer func() { <-semaphore }() // Release the slot once the goroutine is finished

				msg.Ack(false)

				var scanMsg rabbitmq.ScanMessage
				if err := json.Unmarshal(msg.Body, &scanMsg); err != nil {
					log.Error().Err(err).Msg("Failed to unmarshal message")
					return
				}

				log.Info().Msgf("Processing message: %s", msg.Body)

				// Create a scan ID for the scan which will be used to store the results
				scanID, _ := primitive.ObjectIDFromHex(scanMsg.ScanID)
				nh := helpers.NewNucleiHelper(s3Helper, mongoHelper)

				// Update scan status to "in-progress"
				err = mongoHelper.UpdateScanStatus(context.Background(), scanID, "in-progress")
				if err != nil {
					log.Error().Err(err).Msg("Failed to update scan status")
					return
				}

				// Fetch the domain from MongoDB
				domainID, err := primitive.ObjectIDFromHex(scanMsg.DomainID)
				if err != nil {
					log.Error().Err(err).Msg("Failed to convert domain ID to ObjectID")
					return
				}

				domain, err := mongoHelper.FindDomainByID(context.Background(), domainID)
				if err != nil {
					log.Error().Err(err).Msg("Failed to find domain by ID")
					// TODO: Log the error into the logs db
					return
				}

				// Concurrently download the templates
				// Fetch template and domain from MongoDB
				var wg sync.WaitGroup

				templateFiles := make([]string, 0, len(scanMsg.TemplateIDs))
				errChan := make(chan error, len(scanMsg.TemplateIDs))

				log.Info().Msgf("Downloading templates")
				for _, templateIDStr := range scanMsg.TemplateIDs {
					wg.Add(1)
					go func(idStr string) {
						defer wg.Done()

						templateID, err := primitive.ObjectIDFromHex(idStr)
						if err != nil {
							errChan <- fmt.Errorf("invalid template ID: %s, error: %w", idStr, err)
							return
						}

						template, err := mongoHelper.FindTemplateByID(context.Background(), templateID)
						if err != nil {
							errChan <- fmt.Errorf("failed to find template by ID: %s, error: %w", idStr, err)
							return
						}

						templateFilePath := filepath.Join(templateDir, fmt.Sprintf("template-%s.yaml", idStr))

						log.Info().Msgf("Downloading template %s to %s", template.S3URL, templateFilePath)

						err = s3Helper.DownloadFileFromURL(template.S3URL, templateFilePath)
						if err != nil {
							errChan <- fmt.Errorf("failed to download template file from S3 for ID: %s, error: %w", idStr, err)
							return
						}

						templateFiles = append(templateFiles, templateFilePath)
					}(templateIDStr)
				}

				wg.Wait()
				close(errChan)

				for err := range errChan {
					log.Error().Err(err).Msg("Error occurred during template processing")

					if err != nil {
						log.Error().Err(err).Msg("Failed to download template file from S3")
						// TODO: Log the error into the logs db
						return
					}
				}

				// Ensure all downloaded files are deleted after scan
				defer func() {
					for _, file := range templateFiles {
						// Don't delete TemplatesDirectory
						if file == nuclei.DefaultConfig.TemplatesDirectory {
							continue
						}
						os.Remove(file)
					}
				}()

				log.Info().Msg("Successfully downloaded templates")

				nh.ScanWithNuclei(scanID, domain.Domain, domainID.Hex(), templateFiles)
			}(msg)
		}
	}
}
