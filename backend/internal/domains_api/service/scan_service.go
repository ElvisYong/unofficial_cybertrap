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
	scheduledScanRepo *r.ScheduledScanRepository
	mqClient          *rabbitmq.RabbitMQClient
}

// NewUserUseCase creates a new instance of userUseCase
func NewScansService(repository *r.ScansRepository, domainsRepo *r.DomainsRepository, scheduledScanRepo *r.ScheduledScanRepository, mqClient *rabbitmq.RabbitMQClient) *ScansService {
	return &ScansService{
		scansRepo:         repository,
		domainsRepo:       domainsRepo,
		scheduledScanRepo: scheduledScanRepo,
		mqClient:          mqClient,
	}
}

func (s *ScansService) GetAllScans() ([]models.Scan, error) {
	scans, err := s.scansRepo.GetAllScans()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching scans from the database")
		return nil, err
	}

	return scans, nil
}

func (s *ScansService) GetAllScheduledScans() ([]models.Scan, error) {
	scans, err := s.scheduledScanRepo.GetAllScheduledScans()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching scheduled scans from the database")
		return nil, err
	}

	return scans, nil
}

func (s *ScansService) ScanDomain(domainIdStr string, templateIds []string) error {
	ScanID := primitive.NewObjectID()

	// Get the domain from database
	domain, err := s.domainsRepo.GetDomainByID(domainIdStr)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching domain from the database")
		return err
	}

	scanModel := models.Scan{
		ID:          ScanID,
		DomainID:    domainIdStr,
		Domain:      domain.Domain,
		TemplateIDs: templateIds,
		Status:      "Pending",
	}

	// This will send to rabbitmq to be picked up by the scanner
	// Create a new scan record in the database
	messageJson := rabbitmq.ScanMessage{
		ScanID:      ScanID.Hex(),
		TemplateIDs: templateIds,
		DomainID:    domainIdStr,
	}

	// Insert the domains into the database
	errscan := s.scansRepo.InsertSingleScan(scanModel)
	if errscan != nil {
		log.Error().Err(errscan).Msg("Error single scan into the database")
		return errscan
	}

	// Send the message to the queue
	err = s.mqClient.Publish(messageJson)
	if err != nil {
		log.Error().Err(err).Msg("Error sending scan message to queue")
		return err
	}

	// TODO: Return the scan ID to the client so they can track the scan

	return nil
}

func (s *ScansService) ScanMultiDomain(scanMultiRequests []dto.ScanDomainRequest) error {

	scanArray := make([]models.Scan, 0)
	for _, scan := range scanMultiRequests {
		scanModel := models.Scan{
			ID:          primitive.NewObjectID(),
			DomainID:    scan.DomainID,
			TemplateIDs: scan.TemplateIDs,
			Status:      "Pending",
		}

		scanArray = append(scanArray, scanModel)
	}

	// Insert the domains into the database
	errscan := s.scansRepo.InsertMultiScan(scanArray)
	if errscan != nil {
		log.Error().Err(errscan).Msg("Error multi scan into the database")
		return errscan
	}

	for _, scan := range scanArray {

		messageJson := rabbitmq.ScanMessage{
			ScanID:      scan.ID.Hex(),
			TemplateIDs: scan.TemplateIDs,
			DomainID:    scan.DomainID,
		}

		// Send the message to the queue
		err := s.mqClient.Publish(messageJson)
		if err != nil {
			log.Error().Err(err).Msg("Error sending scan message to queue")
			return err
		}
	}

	return errscan
}

func (s *ScansService) CreateScheduleScanRecord(domainid string, startScan string, templateIDs []string) error {
	// FIXME: fix this array slice issues
	// _, err := s.domainsRepo.GetDomainByID(domainid)
	// if err != nil {
	// 	log.Error().Err(err).Msg("Error fetching domain from the database")
	// 	return err
	// }

	convertedStartScanToTimeFormat, error := time.Parse("2006-01-02", startScan)

	if error != nil {
		fmt.Println(error)
	}

	schedulescanModel := models.ScheduleScan{
		ID:           primitive.NewObjectID(),
		DomainID:     domainid,
		TemplatesIDs: templateIDs,
		StartScan:    convertedStartScanToTimeFormat,
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
