package helpers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
)

type S3Helper struct {
	client                *s3.Client
	templateBucketName    string
	scanResultsBucketName string
}

func NewS3Helper(cfg aws.Config, templateBucketName string, scanResultsBucketName string) (*S3Helper, error) {
	client := s3.NewFromConfig(cfg)
	return &S3Helper{client: client, templateBucketName: templateBucketName, scanResultsBucketName: scanResultsBucketName}, nil
}

func (s *S3Helper) DownloadFileFromURL(s3URL, dest string) error {
	parsedURL, err := url.Parse(s3URL)
	if err != nil {
		return fmt.Errorf("invalid S3 URL: %w", err)
	}

	bucket := parsedURL.Host[:strings.Index(parsedURL.Host, ".")]
	key := parsedURL.Path[1:]

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to download file from S3: %w", err)
	}
	defer result.Body.Close()

	// Ensure the directory exists
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, result.Body)
	if err != nil {
		return fmt.Errorf("failed to write file to local destination: %w", err)
	}

	return nil
}

func (s *S3Helper) UploadScanResultsS3(file *bytes.Reader, filename string) (string, error) {
	uploader := manager.NewUploader(s.client)

	log.Info().Msgf("Uploading file to S3: %s", filename)

	result, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: &s.scanResultsBucketName,
		Key:    &filename,
		Body:   file,
	})

	if err != nil {
		log.Error().Err(err).Msg("Error uploading file to S3")
		return "", err
	}

	return result.Location, nil
}
