package service

import (
	"bufio"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"

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
				ID:         primitive.NewObjectID(),
				Domain:     domain,
				UploadedAt: time.Now(),
				UserID:     "temp_user", // For now we will hardcode the user_id as temp_user until auth is done
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Insert the domains into the database
	err := s.domainsRepo.InsertDomains(domains)
	if err != nil {
		log.Error().Err(err).Msg("Error inserting domains into the database")
		return err
	}

	return nil
}
