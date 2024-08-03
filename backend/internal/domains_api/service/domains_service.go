package service

import (
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	r "github.com/shannevie/unofficial_cybertrap/backend/internal/domains_api/repository"
)

type DomainsService struct {
	artefactRepo *r.DomainsRepository
}

// NewUserUseCase creates a new instance of userUseCase
func NewDomainsService(repository *r.DomainsRepository) *DomainsService {
	return &DomainsService{
		artefactRepo: repository,
	}
}

// Checks the file if its a valid type
// Accepted file types are:
// .yml .yaml .json
func (s *DomainsService) isValidFileType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".yml" || ext == ".yaml" || ext == ".json"
}

func (s *DomainsService) UploadArtefact(file multipart.File, file_header *multipart.FileHeader) (string, error) {
	filename := file_header.Filename
	// First check the file type
	if !s.isValidFileType(filename) {
		return "", ErrInvalidFileType
	}

	loc, err := s.artefactRepo.Upload(file, filename)
	if err != nil {
		log.Error().Err(err).Msg("Error uploading file")
		return "", r.ErrS3Upload
	}

	return loc, nil

}
