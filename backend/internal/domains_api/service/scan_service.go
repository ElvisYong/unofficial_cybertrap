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
	multiScanRepo     *r.MultiScanRepository
	templatesRepo     *r.TemplatesRepository
	scheduledScanRepo *r.ScheduledScanRepository
	mqClient          *rabbitmq.RabbitMQClient
}

// NewUserUseCase creates a new instance of userUseCase
func NewScansService(repository *r.ScansRepository, domainsRepo *r.DomainsRepository, multiScanRepo *r.MultiScanRepository, templatesRepo *r.TemplatesRepository, scheduledScanRepo *r.ScheduledScanRepository, mqClient *rabbitmq.RabbitMQClient) *ScansService {
	return &ScansService{
		scansRepo:         repository,
		domainsRepo:       domainsRepo,
		multiScanRepo:     multiScanRepo,
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
		templateIds := make([]string, 0)
		for _, templateId := range scan.TemplateIDs {
			templateIds = append(templateIds, templateId.Hex())
		}

		scansResponse = append(scansResponse, dto.GetAllScansResponse{
			ID:          scan.ID.Hex(),
			DomainId:    scan.DomainId.Hex(),
			Domain:      scan.Domain,
			TemplateIds: templateIds,
			ScanDate:    scan.ScanDate.Format("2006-01-02"),
			Status:      scan.Status,
			S3ResultURL: scan.S3ResultURL,
			ScanTook:    scan.ScanTook,
		})
	}

	return scansResponse, nil
}

func (s *ScansService) ScanDomains(domainIds []primitive.ObjectID, templateIds []primitive.ObjectID, scanAllNuclei bool) error {
	multiScanId := primitive.NewObjectID()

	// Create and insert the multiscan object first
	multiScan := models.MultiScan{
		ID:             multiScanId,
		Name:           fmt.Sprintf("Scan %s", multiScanId.Hex()),
		TotalScans:     len(domainIds),
		CompletedScans: make([]primitive.ObjectID, 0),
		FailedScans:    make([]primitive.ObjectID, 0),
		Status:         "pending",
		ScanDate:       time.Now(),
	}

	err := s.multiScanRepo.CreateMultiScan(multiScan)
	if err != nil {
		log.Error().Err(err).Msg("Error inserting multiscan into the database")
		return err
	}

	// Get all domains at once
	domains, err := s.domainsRepo.GetDomainsByIDs(domainIds)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching domains from the database")
		return err
	}

	// Create a map for quick domain lookup
	domainMap := make(map[string]*models.Domain)
	for i := range domains {
		domainMap[domains[i].Id.Hex()] = &domains[i]
	}

	for _, domainId := range domainIds {
		scanId := primitive.NewObjectID()

		domain, exists := domainMap[domainId.Hex()]
		if !exists {
			log.Error().Str("domainId", domainId.Hex()).Msg("Domain not found in fetched domains")
			continue
		}

		scanModel := models.Scan{
			ID:          scanId,
			DomainId:    domain.Id,
			Domain:      domain.Domain,
			TemplateIDs: templateIds,
			MultiScanID: multiScanId,
			Status:      "Pending",
		}

		// Insert the scan into the database
		errscan := s.scansRepo.InsertSingleScan(scanModel)
		if errscan != nil {
			log.Error().Err(errscan).Str("domainId", domainId.Hex()).Msg("Error inserting single scan into the database")
			continue
		}

		// Create a new scan message for RabbitMQ
		messageJson := rabbitmq.ScanMessage{
			MultiScanId:   multiScanId,
			ScanId:        scanId,
			TemplateIds:   templateIds,
			DomainId:      domain.Id,
			Domain:        domain.Domain,
			ScanAllNuclei: scanAllNuclei,
		}

		// Send the message to the queue
		err = s.mqClient.Publish(messageJson)
		if err != nil {
			log.Error().Err(err).Str("domainId", domainId.Hex()).Msg("Error sending scan message to queue")
			continue
		}

		log.Info().Str("scanId", scanId.Hex()).Str("domainId", domainId.Hex()).Msg("Scan created and sent to queue")
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
	templateIds := make([]primitive.ObjectID, 0)
	for _, template := range templates {
		templateIds = append(templateIds, template.ID)
	}

	// Create the MultiScanId and the scan ids for each domain and then send to rabbitmq
	multiScanId := primitive.NewObjectID()

	// Create a multi scan data with in-progress status
	multiScanModel := models.MultiScan{
		ID:             multiScanId,
		ScanIDs:        make([]primitive.ObjectID, 0),
		Name:           "Scheduled Scan",
		TotalScans:     len(domainIds),
		CompletedScans: make([]primitive.ObjectID, 0),
		FailedScans:    make([]primitive.ObjectID, 0),
		Status:         "in-progress",
	}

	for _, domain := range domainIds {
		scanId := primitive.NewObjectID()

		scanModel := models.Scan{
			ID:          scanId,
			DomainId:    domain.Id,
			Domain:      domain.Domain,
			TemplateIDs: templateIds,
			MultiScanID: multiScanId,
			Status:      "Pending",
		}

		multiScanModel.ScanIDs = append(multiScanModel.ScanIDs, scanId)

		// Insert the scan into the database
		errscan := s.scansRepo.InsertSingleScan(scanModel)
		if errscan != nil {
			log.Error().Err(errscan).Str("domainId", domain.Id.Hex()).Msg("Error inserting single scan into the database")
			continue
		}

		messageJson := rabbitmq.ScanMessage{
			MultiScanId:   multiScanId,
			ScanId:        scanId,
			TemplateIds:   templateIds,
			DomainId:      domain.Id,
			Domain:        domain.Domain,
			ScanAllNuclei: true, // Since we're scanning with all templates
		}

		// Send the message to the queue
		err = s.mqClient.Publish(messageJson)
		if err != nil {
			log.Error().Err(err).Str("domainId", domain.Id.Hex()).Msg("Error sending scan message to queue")
			continue
		}

		log.Info().Str("scanId", scanId.Hex()).Str("domainId", domain.Id.Hex()).Msg("Scan created and sent to queue")
	}

	// Insert the multi scan into the database
	err = s.multiScanRepo.CreateMultiScan(multiScanModel)
	if err != nil {
		log.Error().Err(err).Msg("Error inserting multi scan into the database")
		return err
	}

	return nil
}

func (s *ScansService) ScheduleScan(req *dto.ScheduleScanRequest) error {
	convertedStartScanToTimeFormat, error := time.Parse("2006-01-02", req.ScheduledDate)
	log.Info().Msgf("Converted start scan to time format: %v", convertedStartScanToTimeFormat)
	year, month, day := convertedStartScanToTimeFormat.Date()
	log.Info().Msgf("Date: %d-%02d-%02d", year, int(month), day)

	if error != nil {
		fmt.Println(error)
	}

	if req.ScanAll {
		scheduledScanModel := models.ScheduleScan{
			ID:            primitive.NewObjectID(),
			DomainIds:     []primitive.ObjectID{},
			TemplatesIDs:  []primitive.ObjectID{},
			ScanAll:       req.ScanAll,
			ScheduledDate: convertedStartScanToTimeFormat,
		}

		s.scheduledScanRepo.CreateScheduleScanRecord(scheduledScanModel)
	} else {
		domainsObjectIds := make([]primitive.ObjectID, 0)
		for _, domainId := range req.DomainIds {
			domainObjectID, err := primitive.ObjectIDFromHex(domainId)
			if err != nil {
				log.Error().Err(err).Str("domainId", domainId).Msg("Error converting domain ID to ObjectID")
				return err
			}
			domainsObjectIds = append(domainsObjectIds, domainObjectID)
		}

		// Simply to check if the domains exist
		_, err := s.domainsRepo.GetDomainsByIDs(domainsObjectIds)
		if err != nil {
			log.Error().Err(err).Msg("Error fetching domains from the database")
			return err
		}

		templatesObjectIds := make([]primitive.ObjectID, 0)
		for _, templateId := range req.TemplateIds {
			templateObjectID, err := primitive.ObjectIDFromHex(templateId)
			if err != nil {
				log.Error().Err(err).Str("templateId", templateId).Msg("Error converting template ID to ObjectID")
				return err
			}
			templatesObjectIds = append(templatesObjectIds, templateObjectID)
		}

		// Simply to check if the templates exist
		_, err = s.templatesRepo.GetTemplatesByIDs(templatesObjectIds)
		if err != nil {
			log.Error().Err(err).Msg("Error fetching templates from the database")
			return err
		}

		scheduledScanModel := models.ScheduleScan{
			ID:            primitive.NewObjectID(),
			DomainIds:     domainsObjectIds,
			TemplatesIDs:  templatesObjectIds,
			ScanAll:       req.ScanAll,
			ScheduledDate: convertedStartScanToTimeFormat,
		}

		s.scheduledScanRepo.CreateScheduleScanRecord(scheduledScanModel)
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
		domains, err := s.domainsRepo.GetDomainsByIDs(scan.DomainIds)
		if err != nil {
			log.Error().Err(err).Msg("Error fetching domains from the database")
			return nil, err
		}

		domainsResponse := make([]dto.GetDomainResponse, 0)
		for _, domain := range domains {
			domainsResponse = append(domainsResponse, dto.GetDomainResponse{
				ID:         domain.Id.Hex(),
				Domain:     domain.Domain,
				UploadedAt: domain.UploadedAt.Format(time.RFC3339),
				UserID:     domain.UserId,
			})
		}

		templates, err := s.templatesRepo.GetTemplatesByIDs(scan.TemplatesIDs)
		if err != nil {
			log.Error().Err(err).Msg("Error fetching templates from the database")
			return nil, err
		}

		templatesResponse := make([]dto.GetTemplatesResponse, 0)
		for _, template := range templates {
			templatesResponse = append(templatesResponse, dto.GetTemplatesResponse{
				ID:          template.ID.Hex(),
				TemplateID:  template.TemplateID,
				Name:        template.Name,
				Description: template.Description,
				S3URL:       template.S3URL,
				Metadata:    template.Metadata,
				Type:        template.Type,
				CreatedAt:   template.CreatedAt.Format(time.RFC3339),
			})
		}

		scheduledScans = append(scheduledScans, dto.ScheduleScanResponse{
			ID:            scan.ID.Hex(),
			Domains:       domainsResponse,
			Templates:     templatesResponse,
			ScanAll:       scan.ScanAll,
			ScheduledDate: scan.ScheduledDate.Format("2006-01-02"),
		})
	}

	return scheduledScans, nil
}

func (s *ScansService) DeleteScheduledScanRequest(id string) error {
	err := s.scheduledScanRepo.DeleteScheduledScanByID(id)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting domain from the database")
		return err
	}

	return nil
}

func (s *ScansService) GetAllMultiScans() ([]dto.GetAllMultiScansResponse, error) {
	multiScans, err := s.multiScanRepo.GetAllMultiScans()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching multi-scans from the database")
		return nil, err
	}

	multiScansResponse := make([]dto.GetAllMultiScansResponse, 0)
	for _, multiScan := range multiScans {
		scanIds := make([]string, 0)
		for _, scanId := range multiScan.ScanIDs {
			scanIds = append(scanIds, scanId.Hex())
		}

		multiScansResponse = append(multiScansResponse, dto.GetAllMultiScansResponse{
			ID:             multiScan.ID.Hex(),
			ScanIDs:        scanIds,
			Name:           multiScan.Name,
			TotalScans:     multiScan.TotalScans,
			CompletedScans: len(multiScan.CompletedScans),
			FailedScans:    len(multiScan.FailedScans),
			Status:         multiScan.Status,
		})
	}

	return multiScansResponse, nil
}

func (s *ScansService) GetScansByMultiScanId(multiScanId string) (*dto.GetScansByMultiScanIdResponse, error) {
	// Get multi scan details from multi scan repository
	multiScan, err := s.multiScanRepo.GetMultiScanById(multiScanId)
	if err != nil {
		log.Error().Err(err).Str("multiScanId", multiScanId).Msg("Error fetching multi scan")
		return nil, err
	}

	// Get associated scans from scans repository
	scans, err := s.scansRepo.GetScansByIds(multiScan.ScanIDs)
	if err != nil {
		log.Error().Err(err).Str("multiScanId", multiScanId).Msg("Error fetching associated scans")
		return nil, err
	}

	// Convert scans to response DTOs
	scanResponses := make([]dto.GetAllScansResponse, 0)
	for _, scan := range scans {
		templateIds := make([]string, 0)
		for _, templateId := range scan.TemplateIDs {
			templateIds = append(templateIds, templateId.Hex())
		}

		// Convert the error interface{} to string safely
		var errorStr string
		if scan.Error != nil {
			if err, ok := scan.Error.(string); ok {
				errorStr = err
			} else {
				errorStr = fmt.Sprintf("%v", scan.Error)
			}
		}

		scanResponses = append(scanResponses, dto.GetAllScansResponse{
			ID:          scan.ID.Hex(),
			DomainId:    scan.DomainId.Hex(),
			Domain:      scan.Domain,
			TemplateIds: templateIds,
			ScanDate:    scan.ScanDate.Format("2006-01-02"),
			Status:      scan.Status,
			Error:       errorStr,
			S3ResultURL: scan.S3ResultURL,
			ScanTook:    scan.ScanTook,
		})
	}

	// Create complete response
	response := &dto.GetScansByMultiScanIdResponse{
		MultiScanID:    multiScan.ID.Hex(),
		Name:           multiScan.Name,
		Status:         multiScan.Status,
		TotalScans:     multiScan.TotalScans,
		CompletedScans: len(multiScan.CompletedScans),
		FailedScans:    len(multiScan.FailedScans),
		Scans:          scanResponses,
	}

	return response, nil
}

func (s *ScansService) GetScanById(scanId string) (*dto.GetScanResponse, error) {
	objectId, err := primitive.ObjectIDFromHex(scanId)
	if err != nil {
		log.Error().Err(err).Str("scanId", scanId).Msg("Invalid scan ID")
		return nil, err
	}

	scan, err := s.scansRepo.GetScanById(objectId)
	if err != nil {
		log.Error().Err(err).Str("scanId", scanId).Msg("Error fetching scan")
		return nil, err
	}

	templateIds := make([]string, 0)
	for _, templateId := range scan.TemplateIDs {
		templateIds = append(templateIds, templateId.Hex())
	}

	response := &dto.GetScanResponse{
		ID:          scan.ID.Hex(),
		DomainId:    scan.DomainId.Hex(),
		Domain:      scan.Domain,
		TemplateIds: templateIds,
		ScanDate:    scan.ScanDate.Format("2006-01-02"),
		Status:      scan.Status,
		Error:       fmt.Sprintf("%v", scan.Error),
		S3ResultURL: scan.S3ResultURL,
		ScanTook:    scan.ScanTook,
	}

	return response, nil
}
