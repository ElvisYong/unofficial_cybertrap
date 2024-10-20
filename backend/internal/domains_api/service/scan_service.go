package service

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shannevie/unofficial_cybertrap/backend/internal/domains_api/dto"

	r "github.com/shannevie/unofficial_cybertrap/backend/internal/domains_api/repository"
	"github.com/shannevie/unofficial_cybertrap/backend/internal/rabbitmq"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ScansService struct {
	domainsRepo       *r.DomainsRepository
	scansRepo         *r.ScansRepository
	templatesRepo     *r.TemplatesRepository
	scheduledScanRepo *r.ScheduledScanRepository
	mqClient          *rabbitmq.RabbitMQClient
}

// NewUserUseCase creates a new instance of userUseCase
func NewScansService(repository *r.ScansRepository, domainsRepo *r.DomainsRepository, templatesRepo *r.TemplatesRepository, scheduledScanRepo *r.ScheduledScanRepository, mqClient *rabbitmq.RabbitMQClient) *ScansService {
	return &ScansService{
		scansRepo:         repository,
		domainsRepo:       domainsRepo,
		templatesRepo:     templatesRepo,
		scheduledScanRepo: scheduledScanRepo,
		mqClient:          mqClient,
	}
}

func (s *ScansService) GetAllScans() ([]dto.GetAllScansResponse, error) {
	scans, err := s.scansRepo.GetAllScans()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching scans from the database")
		return nil, err
	}

	scansResponse := make([]dto.GetAllScansResponse, 0)
	for _, scan := range scans {
		scansResponse = append(scansResponse, dto.GetAllScansResponse{
			ID:          scan.ID.Hex(),
			DomainId:    scan.DomainId.Hex(),
			TemplateIds: scan.TemplateIDs,
			ScanDate:    scan.ScanDate.Format("2006-01-02"),
			Status:      scan.Status,
			S3ResultURL: scan.S3ResultURL,
		})
	}

	return scansResponse, nil
}

func (s *ScansService) ScanDomains(domainIdStrs []string, templateIds []string) error {
	multiScanId := primitive.NewObjectID()

	// Get all domains at once
	domains, err := s.domainsRepo.GetDomainsByIDs(domainIdStrs)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching domains from the database")
		return err
	}

	// Create a map for quick domain lookup
	domainMap := make(map[string]*models.Domain)
	for i := range domains {
		domainMap[domains[i].ID.Hex()] = &domains[i]
	}

	for _, domainIdStr := range domainIdStrs {
		scanId := primitive.NewObjectID()

		domain, exists := domainMap[domainIdStr]
		if !exists {
			log.Error().Str("domainId", domainIdStr).Msg("Domain not found in fetched domains")
			continue
		}

		scanModel := models.Scan{
			ID:          scanId,
			DomainId:    domain.ID,
			Domain:      domain.Domain,
			TemplateIDs: templateIds,
			Status:      "Pending",
		}

		// Insert the scan into the database
		errscan := s.scansRepo.InsertSingleScan(scanModel)
		if errscan != nil {
			log.Error().Err(errscan).Str("domainId", domainIdStr).Msg("Error inserting single scan into the database")
			continue
		}

		// Create a new scan message for RabbitMQ
		messageJson := rabbitmq.ScanMessage{
			MultiScanId: multiScanId,
			ScanId:      scanId,
			TemplateIds: templateIds,
			DomainId:    domain.ID,
		}

		// Send the message to the queue
		err = s.mqClient.Publish(messageJson)
		if err != nil {
			log.Error().Err(err).Str("domainId", domainIdStr).Msg("Error sending scan message to queue")
			continue
		}

		log.Info().Str("scanId", scanId.Hex()).Str("domainId", domainIdStr).Msg("Scan created and sent to queue")
	}

	return nil
}

// Retrieve all scheduled scans and scan all of them
func (s *ScansService) ScanAllDomains() error {
	// Get all domains
	domainIds, err := s.domainsRepo.GetAllDomains()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching all domain ids from the database")
		return err
	}

	// Get all templates
	templates, err := s.templatesRepo.GetAllTemplates()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching all templates from the database")
		return err
	}

	// list of template ids strings
	templateIds := make([]string, 0)
	for _, template := range templates {
		templateIds = append(templateIds, template.ID.Hex())
	}

	// Create the MultiScanId and the scan ids for each domain and then send to rabbitmq
	multiScanId := primitive.NewObjectID()

	for _, domain := range domainIds {
		scanId := primitive.NewObjectID()

		scanModel := models.Scan{
			ID:          scanId,
			DomainId:    domain.ID,
			Domain:      domain.Domain,
			TemplateIDs: templateIds,
			Status:      "Pending",
		}

		// Insert the scan into the database
		errscan := s.scansRepo.InsertSingleScan(scanModel)
		if errscan != nil {
			log.Error().Err(errscan).Str("domainId", domain.ID.Hex()).Msg("Error inserting single scan into the database")
			continue
		}

		messageJson := rabbitmq.ScanMessage{
			MultiScanId: multiScanId,
			ScanId:      scanId,
			TemplateIds: templateIds,
			DomainId:    domain.ID,
		}

		// Send the message to the queue
		err = s.mqClient.Publish(messageJson)
		if err != nil {
			log.Error().Err(err).Str("domainId", domain.ID.Hex()).Msg("Error sending scan message to queue")
			continue
		}

		log.Info().Str("scanId", scanId.Hex()).Str("domainId", domain.ID.Hex()).Msg("Scan created and sent to queue")
	}

	return nil
}

func (s *ScansService) GetAllScheduledScans() ([]dto.ScheduleScanResponse, error) {
	scans, err := s.scheduledScanRepo.GetAllScheduledScans()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching scheduled scans from the database")
		return nil, err
	}

	// Convert the scans to the response
	scheduledScans := make([]dto.ScheduleScanResponse, 0)
	for _, scan := range scans {
		scheduledScans = append(scheduledScans, dto.ScheduleScanResponse{
			ID:            scan.ID.Hex(),
			DomainId:      scan.DomainID,
			TemplateIds:   scan.TemplatesIDs,
			ScheduledDate: scan.ScheduledDate.Format("2006-01-02"),
		})
	}

	return scheduledScans, nil
}

func (s *ScansService) CreateScheduleScanRecord(domainid string, scheduledDate string, templateIDs []string) error {
	_, err := s.domainsRepo.GetDomainByID(domainid)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching domain from the database")
		return err
	}

	convertedStartScanToTimeFormat, error := time.Parse("2006-01-02", scheduledDate)
	log.Info().Msgf("Converted start scan to time format: %v", convertedStartScanToTimeFormat)
	year, month, day := convertedStartScanToTimeFormat.Date()
	log.Info().Msgf("Date: %d-%02d-%02d", year, int(month), day)

	if error != nil {
		fmt.Println(error)
	}

	schedulescanModel := models.ScheduleScan{
		ID:            primitive.NewObjectID(),
		DomainID:      domainid,
		TemplatesIDs:  templateIDs,
		ScheduledDate: convertedStartScanToTimeFormat,
	}

	// Insert the domains into the database
	errscan := s.scheduledScanRepo.CreateScheduleScanRecord(schedulescanModel)
	if errscan != nil {
		log.Error().Err(errscan).Msg("Error single scan into the database")
		return errscan
	}

	// TODO: Return the scan ID to the client so they can track the scan

	return nil
}

func (s *ScansService) DeleteScheduledScanRequest(id string) error {
	err := s.scheduledScanRepo.DeleteScheduledScanByID(id)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting domain from the database")
		return err
	}

	return nil
}
