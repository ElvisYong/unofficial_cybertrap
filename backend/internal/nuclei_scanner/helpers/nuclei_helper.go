package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
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
	ctx context.Context,
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

	// Create a child context with a 2-hour timeout specifically for the scan
	scanCtx, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()

	// Create cleanup channel
	done := make(chan struct{})
	defer close(done)

	// Handle context cancellation
	go func() {
		select {
		case <-scanCtx.Done():
			errorMsg := formatErrorDetails(scanCtx.Err(), "Scan interrupted")
			nh.handleScanError(context.Background(), scanID, multiScanId, errorMsg, time.Now())
		case <-done:
			return
		}
	}()

	// Update multi-scan status to in-progress at the start
	if err := nh.mongoHelper.UpdateMultiScanStatus(ctx, multiScanId, "in-progress", nil, nil); err != nil {
		errorMsg := formatErrorDetails(err, "Failed to update multi-scan status to in-progress")
		log.Error().Err(err).Msg(errorMsg)
		return fmt.Errorf(errorMsg)
	}

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

	ne, err = nuclei.NewNucleiEngineCtx(scanCtx, options...)
	if err != nil {
		errorMsg := formatErrorDetails(err, "Failed to create nuclei engine")
		nh.handleScanError(ctx, scanID, multiScanId, errorMsg, scanStartTime)
		return fmt.Errorf(errorMsg)
	}

	// Configure Nuclei engine options
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

	// Execute the scan with the scan-specific context
	scanResults := []output.ResultEvent{}
	err = ne.ExecuteCallbackWithCtx(scanCtx, func(event *output.ResultEvent) {
		scanResults = append(scanResults, *event)
	})

	// Check context again after scan
	if err := scanCtx.Err(); err != nil {
		errorMsg := formatErrorDetails(err, "Scan context error after execution")
		nh.handleScanError(context.Background(), scanID, multiScanId, errorMsg, scanStartTime)
		return fmt.Errorf(errorMsg)
	}

	// Check for timeout
	if scanCtx.Err() == context.DeadlineExceeded {
		errorMsg := formatErrorDetails(
			scanCtx.Err(),
			fmt.Sprintf("Scan timed out after 2 hours. Templates: %d, Results so far: %d",
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

	log.Info().Msg("Adding scan to completed scans in multi-scan")
	err = nh.mongoHelper.UpdateMultiScanStatus(context.Background(), multiScanId, "in-progress", &scanID, nil)
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
	// Use the provided context directly without a timeout
	duration := time.Since(startTime).Milliseconds()

	errorInfo := map[string]interface{}{
		"message":   errorMsg,
		"timestamp": time.Now(),
		"duration":  duration,
		"scanId":    scanID.Hex(),
	}

	// Use WaitGroup to ensure all cleanup operations complete
	var wg sync.WaitGroup
	wg.Add(2)

	// Update scan status
	go func() {
		defer wg.Done()
		if err := nh.mongoHelper.UpdateScanError(ctx, scanID, "failed", errorInfo, duration); err != nil {
			log.Error().Err(err).Str("scanID", scanID.Hex()).Msg("Failed to update scan error status")
		}
	}()

	// Update multi-scan status
	go func() {
		defer wg.Done()
		if err := nh.mongoHelper.UpdateMultiScanStatus(ctx, multiScanId, "failed", nil, &scanID); err != nil {
			log.Error().Err(err).Str("multiScanId", multiScanId.Hex()).Msg("Failed to update multi scan status")
		}
	}()

	// Wait for cleanup to complete
	wg.Wait()
	log.Info().Msg("Error handling completed successfully")
}
