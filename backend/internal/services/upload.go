package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"notorious-backend/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type UploadService struct {
	s3Client *s3.Client
	cfg      *config.Config
}

type S3StreamService struct {
	s3Client *s3.Client
}

type InitUploadResponse struct {
	UploadID string `json:"upload_id"`
	Bucket   string `json:"bucket"`
	Key      string `json:"key"`
	PartSize int64  `json:"part_size_mb"`
}

func NewUploadService(cfg *config.Config) *UploadService {
	s3Client := createS3Client(cfg)

	return &UploadService{
		s3Client: s3Client,
		cfg:      cfg,
	}
}

func NewS3StreamService(cfg *config.Config) (*S3StreamService, error) {
	client := createS3Client(cfg)
	return &S3StreamService{s3Client: client}, nil
}

func (s *UploadService) InitMultipartUpload(filename string, partSizeMB int64) (*InitUploadResponse, error) {
	key := s.cfg.S3UploadPrefix + filename
	if partSizeMB <= 0 {
		partSizeMB = 64 // 64 MB default
	}

	// Create multipart upload
	input := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(s.cfg.S3UploadBucket),
		Key:    aws.String(key),
	}

	result, err := s.s3Client.CreateMultipartUpload(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("error creating multipart upload: %v", err)
	}

	return &InitUploadResponse{
		UploadID: *result.UploadId,
		Bucket:   s.cfg.S3UploadBucket,
		Key:      key,
		PartSize: partSizeMB,
	}, nil
}

func (s *S3StreamService) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.s3Client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error fetching S3 object %s/%s: %w", bucket, key, err)
	}

	log.Printf("Opened S3 object stream: s3://%s/%s (content-length=%d)", bucket, key, result.ContentLength)
	return result.Body, nil
}

func (s *UploadService) PresignPartUpload(uploadID, key string, partNumber int32) (string, error) {
	presignClient := s3.NewPresignClient(s.s3Client)

	request, err := presignClient.PresignUploadPart(context.TODO(), &s3.UploadPartInput{
		Bucket:     aws.String(s.cfg.S3UploadBucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(partNumber),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 24 * time.Hour
	})

	if err != nil {
		return "", err
	}

	return request.URL, nil
}

func (s *UploadService) CompleteMultipartUpload(uploadID, key string, completedParts []types.CompletedPart) error {
	if len(completedParts) == 0 {
		return fmt.Errorf("no parts provided for completion")
	}

	input := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.cfg.S3UploadBucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}

	_, err := s.s3Client.CompleteMultipartUpload(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("error completing multipart upload: %v", err)
	}

	log.Printf("Upload completed: s3://%s/%s", s.cfg.S3UploadBucket, key)
	return nil
}

func (s *UploadService) AbortMultipartUpload(uploadID, key string) error {
	_, err := s.s3Client.AbortMultipartUpload(context.TODO(), &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s.cfg.S3UploadBucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		return fmt.Errorf("error aborting multipart upload: %v", err)
	}
	return nil
}

func createS3Client(cfg *config.Config) *s3.Client {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(cfg.AWSRegion),
		awsconfig.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.AWSAccessKeyID,
				SecretAccessKey: cfg.AWSSecretAccessKey,
			}, nil
		})),
	)
	if err != nil {
		log.Fatalf("Error loading AWS config: %v", err)
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UseAccelerate = true
	})
}
