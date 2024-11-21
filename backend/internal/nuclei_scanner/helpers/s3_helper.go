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
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

func (s *S3Helper) DownloadAllTemplates(destDir string) error {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.templateBucketName),
		Prefix: aws.String("nuclei-templates/"),
	}

	// Create a downloader with the S3 client and custom configurations
	downloader := manager.NewDownloader(s.client, func(d *manager.Downloader) {
		d.PartSize = 64 * 1024 * 1024 // 64MB per part
		d.Concurrency = 10            // 10 concurrent downloads
	})

	// Create a wait group to manage concurrent downloads
	var wg sync.WaitGroup
	// Create an error channel to collect errors
	errCh := make(chan error, 100)
	// Create a semaphore to limit concurrent downloads
	sem := make(chan struct{}, 20) // Limit to 20 concurrent operations

	paginator := s3.NewListObjectsV2Paginator(s.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			// Skip if it's a directory
			if strings.HasSuffix(*obj.Key, "/") {
				continue
			}

			wg.Add(1)
			go func(obj types.Object) {
				defer wg.Done()
				sem <- struct{}{}        // Acquire semaphore
				defer func() { <-sem }() // Release semaphore

				// Create the full destination path
				destPath := filepath.Join(destDir, *obj.Key)

				// Ensure the directory exists
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					errCh <- fmt.Errorf("failed to create directory for %s: %w", *obj.Key, err)
					return
				}

				// Create the file
				file, err := os.Create(destPath)
				if err != nil {
					errCh <- fmt.Errorf("failed to create file %s: %w", destPath, err)
					return
				}
				defer file.Close()

				// Download the file
				_, err = downloader.Download(context.TODO(), file, &s3.GetObjectInput{
					Bucket: aws.String(s.templateBucketName),
					Key:    obj.Key,
				})
				if err != nil {
					errCh <- fmt.Errorf("failed to download %s: %w", *obj.Key, err)
					return
				}

				log.Debug().Msgf("Successfully downloaded: %s", *obj.Key)
			}(obj)
		}
	}

	// Wait for all downloads to complete
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Collect any errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple download errors occurred: %v", errors)
	}

	return nil
}
