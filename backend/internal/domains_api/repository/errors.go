package repository

import "errors"

// Service Errors
var (
	ErrS3Upload = errors.New("failed to upload to s3 bucket")
)
