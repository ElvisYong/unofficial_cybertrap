package helpers

import (
	"bytes"
	"context"
	"encoding/json"
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

func (nh *NucleiHelper) ScanWithNuclei(
	multiScanId primitive.ObjectID,
	scanID primitive.ObjectID,
	domain string,
	domainId primitive.ObjectID,
	templateFilePaths []string,
	templateIDs []primitive.ObjectID,
	scanAllNuclei bool,
	debug bool,
) {
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

	ne, err = nuclei.NewNucleiEngineCtx(context.TODO(), options...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute scan")
		// Update scan status to "failed"
		nh.mongoHelper.UpdateScanStatus(context.Background(), scanID, "failed")
		return
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
		log.Error().Err(err).Msg("Failed to load templates")
		// Update scan status to "failed"
		nh.mongoHelper.UpdateScanStatus(context.Background(), scanID, "failed")
		nh.mongoHelper.UpdateMultiScanStatus(context.Background(), multiScanId, "failed", nil, &scanID)
		return
	}

	// Load the targets from the domain fetched from MongoDB
	targets := []string{domain}
	ne.LoadTargets(targets, false)
	log.Info().Msg("Successfully loaded targets into nuclei engine")
	log.Info().Msg("Starting scan")

	// Execute the scan
	scanResults := []output.ResultEvent{}
	err = ne.ExecuteCallbackWithCtx(context.TODO(), func(event *output.ResultEvent) {
		scanResults = append(scanResults, *event)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute scan")
		// Update scan status to "failed"
		nh.mongoHelper.UpdateScanStatus(context.Background(), scanID, "failed")
		nh.mongoHelper.UpdateMultiScanStatus(context.Background(), multiScanId, "failed", &scanID, nil)
		return
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

	err = nh.mongoHelper.UpdateScanResult(context.Background(), scan)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update scan result")
		nh.mongoHelper.UpdateScanStatus(context.Background(), scanID, "failed")
		return
	}

	// First, update the completed scans for this multi-scan
	err = nh.mongoHelper.UpdateMultiScanStatus(context.Background(), multiScanId, "", &scanID, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update multi scan status")
		return
	}

	// Then, fetch the updated multi-scan to check if all scans are completed
	multiScan, err := nh.mongoHelper.FindMultiScanByID(context.Background(), multiScanId)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch multi scan")
		return
	}

	if len(multiScan.CompletedScans) == multiScan.TotalScans {
		// All scans are completed, update the status to "completed"
		err = nh.mongoHelper.UpdateMultiScanStatus(context.Background(), multiScanId, "completed", nil, nil)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update multi scan status to completed")
			return
		}
	}

	log.Info().Msg("Completed scan and updated scan result for scanID: " + scanID.Hex())
}
