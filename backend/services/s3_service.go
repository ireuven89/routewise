package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Service struct {
	client     *s3.Client
	bucketName string
}

func NewS3Service() (*S3Service, error) {
	ctx := context.Background()

	// Load AWS config (uses IAM role on EC2, or env vars/credentials file locally)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	return &S3Service{
		client:     s3.NewFromConfig(cfg),
		bucketName: os.Getenv("S3_BUCKET_NAME"),
	}, nil
}

// UploadFile uploads a file to S3 and returns the S3 key
func (s *S3Service) UploadFile(ctx context.Context, file io.Reader, s3Key string, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(s3Key),
		Body:        file,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		fmt.Printf("failed to upload file: %v", err)
		return fmt.Errorf("failed to upload file to S3: %v", err)
	}

	return nil
}

// GetSignedURL generates a presigned URL for downloading a file (valid for 1 hour)
func (s *S3Service) GetSignedURL(ctx context.Context, s3Key string) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Hour * 1 // URL valid for 1 hour
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %v", err)
	}

	return request.URL, nil
}

// DeleteFile deletes a file from S3
func (s *S3Service) DeleteFile(ctx context.Context, s3Key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	})

	if err != nil {
		fmt.Printf("failed to delete file: %v", err)
		return fmt.Errorf("failed to delete file from S3: %v", err)
	}

	return nil
}

// GenerateS3Key creates a unique S3 key for a file
// Format: organizations/{orgID}/projects/{projectID}/{fileType}/{timestamp}_{filename}
func GenerateS3Key(orgID, projectID uint, fileType, filename string) string {
	timestamp := time.Now().Unix()
	safeFilename := filepath.Base(filename) // Prevent path traversal
	return fmt.Sprintf("organizations/%d/projects/%d/%s/%d_%s",
		orgID, projectID, fileType, timestamp, safeFilename)
}
