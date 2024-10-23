package service

import (
	"bufio"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/shannevie/unofficial_cybertrap/backend/internal/domains_api/dto"
	r "github.com/shannevie/unofficial_cybertrap/backend/internal/domains_api/repository"
	"github.com/shannevie/unofficial_cybertrap/backend/models"
)

type DomainsService struct {
	domainsRepo *r.DomainsRepository
}

// NewUserUseCase creates a new instance of userUseCase
func NewDomainsService(repository *r.DomainsRepository) *DomainsService {
	return &DomainsService{
		domainsRepo: repository,
	}
}

func (s *DomainsService) GetAllDomains() ([]dto.GetDomainResponse, error) {
	domains, err := s.domainsRepo.GetAllDomains()
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

	return domainsResponse, nil
}

func (s *DomainsService) GetDomainById(id string) (*dto.GetDomainResponse, error) {
	domain, err := s.domainsRepo.GetDomainByID(id)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching domain from the database")
		return nil, err
	}

	domainResponse := dto.GetDomainResponse{
		ID:         domain.Id.Hex(),
		Domain:     domain.Domain,
		UploadedAt: domain.UploadedAt.Format(time.RFC3339),
		UserID:     domain.UserId,
	}

	return &domainResponse, nil
}

func (s *DomainsService) DeleteDomainById(id string) error {
	err := s.domainsRepo.DeleteDomainById(id)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting domain from the database")
		return err
	}

	return nil
}

// ProcessDomainsFile reads the file content and inserts all domains into the database
func (s *DomainsService) ProcessDomainsFile(file multipart.File, file_header *multipart.FileHeader) error {
	// Check if the file is a txt
	ext := strings.ToLower(filepath.Ext(file_header.Filename))
	if ext != ".txt" {
		return ErrInvalidFileType
	}

	// Read the file content
	var domains []models.Domain
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		domain := strings.TrimSpace(scanner.Text())
		if domain != "" {
			domains = append(domains, models.Domain{
				Id:         primitive.NewObjectID(),
				Domain:     domain,
				UploadedAt: time.Now(),
				UserId:     "temp_user", // For now we will hardcode the user_id as temp_user until auth is done
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Insert the domains into the database``
	err := s.domainsRepo.InsertDomains(domains)
	if err != nil {
		log.Error().Err(err).Msg("Error inserting domains into the database")
		return err
	}

	return nil
}

// ProcessDomainsFile reads the file content and inserts all domains into the database
func (s *DomainsService) ProcessDomains(domainQuery string) error {

	domainModel := models.Domain{
		Id:         primitive.NewObjectID(),
		Domain:     domainQuery,
		UploadedAt: time.Now(),
		UserId:     "temp_user", // For now we will hardcode the user_id as temp_user until auth is done
	}

	// Insert the domains into the database
	err := s.domainsRepo.InsertSingleDomain(domainModel)
	if err != nil {
		log.Error().Err(err).Msg("Error inserting domains into the database")
		return err
	}

	return nil
}
