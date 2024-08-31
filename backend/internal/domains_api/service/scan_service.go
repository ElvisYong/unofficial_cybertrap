package service

import (
	"github.com/rs/zerolog/log"
	r "github.com/shannevie/unofficial_cybertrap/backend/internal/domains_api/repository"
	"github.com/shannevie/unofficial_cybertrap/backend/internal/rabbitmq"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ScansService struct {
	scansRepo *r.ScansRepository
	mqClient  *rabbitmq.RabbitMQClient
}

// NewUserUseCase creates a new instance of userUseCase
func NewScansService(repository *r.ScansRepository, mqClient *rabbitmq.RabbitMQClient) *ScansService {
	return &ScansService{
		scansRepo: repository,
		mqClient:  mqClient,
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

// TODO: Send the id and template ids to the scanner service
func (s *ScansService) ScanDomain(domainIdStr string, templateIds []string) error {
	// TODO: Check if the domain and the template ids are valid before sending to the scanner

	// TODO: upload to mongodb this scan ID with a scan status of pending
	ScanID := primitive.NewObjectID().Hex()

	scanModel := models.Scan{
		ID:          primitive.NewObjectID(),
		DomainID:    domainIdStr,
		TemplateIDs: templateIds,
		Status:      "Pending",
	}

	// This will send to rabbitmq to be picked up by the scanner
	// Create a new scan record in the database
	messageJson := rabbitmq.ScanMessage{
		ScanID:      ScanID,
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
	err := s.mqClient.Publish(messageJson)
	if err != nil {
		log.Error().Err(err).Msg("Error sending scan message to queue")
		return err
	}

	// TODO: Return the scan ID to the client so they can track the scan

	return nil
}

// // TO DO : Multi-select
// func (s *ScansService) ScanDomain(domainId string, templateIds []string) error {

// }
