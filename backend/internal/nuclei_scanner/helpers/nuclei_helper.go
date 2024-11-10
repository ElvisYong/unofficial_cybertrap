package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	nuclei "github.com/projectdiscovery/nuclei/v3/lib"
	"github.com/projectdiscovery/nuclei/v3/pkg/model/types/severity"
	"github.com/projectdiscovery/nuclei/v3/pkg/output"
	"github.com/rs/zerolog/log"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NucleiHelper struct {
	s3Helper    *S3Helper
	mongoHelper *MongoHelper
}

func NewNucleiHelper(s3Helper *S3Helper, mongoHelper *MongoHelper) *NucleiHelper {
	return &NucleiHelper{
		s3Helper:    s3Helper,
		mongoHelper: mongoHelper,
	}
}

// Create a helper function to format detailed error messages
func formatErrorDetails(err error, context string) string {
	return fmt.Sprintf("%s: %v\nTimestamp: %s",
		context,
		err,
		time.Now().Format(time.RFC3339),
	)
}

func (nh *NucleiHelper) ScanWithNuclei(
	multiScanId primitive.ObjectID,
	scanID primitive.ObjectID,
	domain string,
	domainId primitive.ObjectID,
	templateFilePaths []string,
	templateIDs []primitive.ObjectID,
	scanAllNuclei bool,
	debug bool,
) error {
	scanStartTime := time.Now()
	timeoutDuration := 2 * time.Hour

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	// Update scan with start time
	if err := nh.mongoHelper.UpdateScanStartTime(ctx, scanID, scanStartTime); err != nil {
		errorMsg := formatErrorDetails(err, "Failed to update scan start time")
		nh.handleScanError(ctx, scanID, multiScanId, errorMsg, scanStartTime)
		return fmt.Errorf(errorMsg)
	}

	// Check the length of templateFiles
	templateSources := nuclei.TemplateSources{
		Templates: templateFilePaths,
	}

	var ne *nuclei.NucleiEngine
	var err error

	options := []nuclei.NucleiSDKOptions{
		nuclei.WithNetworkConfig(nuclei.NetworkConfig{
			DisableMaxHostErr: true,  // This probably doesn't work from what I can see
			MaxHostError:      10000, // Using a larger number to avoid host errors dying in 30 tries dropping the domain
		}),
		nuclei.WithTemplatesOrWorkflows(templateSources),
	}

	if scanAllNuclei {
		options = append(options, nuclei.WithTemplateUpdateCallback(true, func(newVersion string) {
			log.Info().Msgf("New template version available: %s", newVersion)
		}))
	}

	ne, err = nuclei.NewNucleiEngineCtx(ctx, options...)
	if err != nil {
		errorMsg := formatErrorDetails(err, "Failed to create nuclei engine")
		nh.handleScanError(ctx, scanID, multiScanId, errorMsg, scanStartTime)
		return fmt.Errorf(errorMsg)
	}

	// Disable host errors
	ne.Options().Severities = []severity.Severity{severity.Info, severity.Low, severity.Medium, severity.High, severity.Critical}
	ne.Options().StatsJSON = true
	ne.Engine().ExecuterOptions().Options.NoHostErrors = true
	ne.GetExecuterOptions().Options.NoHostErrors = true
	ne.Options().StatsJSON = true
	ne.Options().Verbose = true
	ne.Options().Debug = debug

	// Load all templates
	err = ne.LoadAllTemplates()
	if err != nil {
		errorMsg := formatErrorDetails(err, "Failed to load templates")
		nh.handleScanError(ctx, scanID, multiScanId, errorMsg, scanStartTime)
		return fmt.Errorf(errorMsg)
	}

	// Load the targets from the domain fetched from MongoDB
	targets := []string{domain}
	ne.LoadTargets(targets, false)
	log.Info().Msg("Successfully loaded targets into nuclei engine")
	log.Info().Msg("Starting scan")

	// Execute the scan with timeout context
	scanResults := []output.ResultEvent{}
	err = ne.ExecuteCallbackWithCtx(ctx, func(event *output.ResultEvent) {
		scanResults = append(scanResults, *event)
	})

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		errorMsg := formatErrorDetails(
			ctx.Err(),
			fmt.Sprintf("Scan timed out after %s. Templates: %d, Results so far: %d",
				timeoutDuration,
				len(templateFilePaths),
				len(scanResults)),
		)
		nh.handleScanError(context.Background(), scanID, multiScanId, errorMsg, scanStartTime)
		return fmt.Errorf(errorMsg)
	}

	if err != nil {
		errorMsg := formatErrorDetails(err, fmt.Sprintf(
			"Scan execution failed. Templates: %d, Results: %d, Duration: %s",
			len(templateFilePaths),
			len(scanResults),
			time.Since(scanStartTime),
		))
		nh.handleScanError(context.Background(), scanID, multiScanId, errorMsg, scanStartTime)
		nh.mongoHelper.UpdateScanStatus(context.Background(), scanID, "failed", map[string]interface{}{
			"message":   err.Error(),
			"timestamp": time.Now(),
		})
		return fmt.Errorf(errorMsg)
	}
	log.Info().Msg("Scan completed")

	log.Info().Msgf("There are %d results", len(scanResults))

	// Loop the scan results and parse them into a json
	scanResultUrls := []string{}

	for _, result := range scanResults {
		// Convert the result to a json
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal result")
			continue
		}

		// Upload the results onto s3 into the following structure
		// scanID/templateID.json
		// Once uploaded take the url and update the scan results
		multipartFile := bytes.NewReader(resultJSON)

		// Get current timestamp in millis
		currentTime := time.Now()
		currentTimeMillis := currentTime.UnixNano() / int64(time.Millisecond)
		fileName := result.TemplateID + "_" + result.Host + "_" + strconv.FormatInt(currentTimeMillis, 10) + ".json"

		s3URL, err := nh.s3Helper.UploadScanResultsS3(multipartFile, fileName)
		if err != nil {
			log.Error().Err(err).Msg("Failed to upload result to s3 for scanID, templateID: " + scanID.Hex() + ", " + result.TemplateID)
			continue
		}

		scanResultUrls = append(scanResultUrls, s3URL)

		// Write the result to a local temporary file
		// tempDir := os.TempDir()
		// tempFile, err := os.CreateTemp(tempDir, "scan_result_.json")
		// if err != nil {
		// 	log.Error().Err(err).Msg("Failed to create temporary file")
		// 	return
		// }
		// defer tempFile.Close()

		// _, err = tempFile.Write(resultJSON)
		// if err != nil {
		// 	log.Error().Err(err).Msg("Failed to write result to temporary file")
		// 	return
		// }

		// log.Info().Str("file", tempFile.Name()).Msg("Scan result written to temporary file")

	}
	// Update the scan result with the s3 url
	scan := models.Scan{
		ID:          scanID,
		DomainId:    domainId,
		Domain:      domain,
		TemplateIDs: templateIDs,
		Error:       nil,
		S3ResultURL: scanResultUrls,
		ScanDate:    time.Now(),
		Status:      "completed",
	}

	scanDuration := time.Since(scanStartTime).Milliseconds()
	scan.ScanTook = scanDuration

	log.Info().Msg("Updating scan results to completed")
	err = nh.mongoHelper.UpdateScanResult(context.Background(), scan)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update scan result")
		nh.mongoHelper.UpdateScanStatus(context.Background(), scanID, "failed", map[string]interface{}{
			"message":   err.Error(),
			"timestamp": time.Now(),
		})
		return fmt.Errorf("failed to update scan result: %w", err)
	}

	// First, update the completed scans for this multi-scan
	log.Info().Msg("Updating multi scan status to completed")
	err = nh.mongoHelper.UpdateMultiScanStatus(context.Background(), multiScanId, "", &scanID, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update multi scan status")
		return fmt.Errorf("failed to update multi scan status: %w", err)
	}

	// Then, fetch the updated multi-scan to check if all scans are completed
	multiScan, err := nh.mongoHelper.FindMultiScanByID(context.Background(), multiScanId)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch multi scan")
		return fmt.Errorf("failed to fetch multi scan: %w", err)
	}

	if len(multiScan.CompletedScans) == multiScan.TotalScans {
		// All scans are completed, update the final status and duration
		scanDuration := time.Since(multiScan.ScanDate).Milliseconds()
		err = nh.mongoHelper.UpdateMultiScanCompletion(context.Background(), multiScanId, "completed", scanDuration)
		if err != nil {
			log.Error().Err(err).
				Str("multiScanId", multiScanId.Hex()).
				Int64("duration", scanDuration).
				Msg("Failed to update multi scan completion")
		}
	}

	return nil
}

// Enhanced error handling function
func (nh *NucleiHelper) handleScanError(ctx context.Context, scanID, multiScanId primitive.ObjectID, errorMsg string, startTime time.Time) {
	duration := time.Since(startTime).Milliseconds()

	// Create detailed error information
	errorInfo := map[string]interface{}{
		"message":   errorMsg,
		"timestamp": time.Now(),
		"duration":  duration,
		"scanId":    scanID.Hex(),
	}

	// Use the consolidated MongoDB helper methods
	if err := nh.mongoHelper.UpdateScanError(ctx, scanID, "failed", errorInfo, duration); err != nil {
		log.Error().Err(err).
			Str("scanID", scanID.Hex()).
			Str("errorMsg", errorMsg).
			Msg("Failed to update scan error status")
	}

	// Handle multi-scan updates
	multiScan, err := nh.mongoHelper.FindMultiScanByID(ctx, multiScanId)
	if err != nil {
		log.Error().Err(err).
			Str("multiScanId", multiScanId.Hex()).
			Msg("Failed to fetch multi scan")
		return
	}

	// Update multi-scan status
	if err := nh.mongoHelper.UpdateMultiScanStatus(ctx, multiScanId, "", nil, &scanID); err != nil {
		log.Error().Err(err).
			Str("multiScanId", multiScanId.Hex()).
			Msg("Failed to update multi scan failed scans")
	}

	// Check if this was the last scan
	totalProcessed := len(multiScan.CompletedScans) + len(multiScan.FailedScans) + 1
	if totalProcessed >= multiScan.TotalScans {
		duration = time.Since(multiScan.ScanDate).Milliseconds()
		if err := nh.mongoHelper.UpdateMultiScanCompletion(ctx, multiScanId, "failed", duration); err != nil {
			log.Error().Err(err).
				Str("multiScanId", multiScanId.Hex()).
				Int64("duration", duration).
				Msg("Failed to update multi scan completion")
		}
	}
}
