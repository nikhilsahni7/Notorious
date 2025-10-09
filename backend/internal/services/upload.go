package services

import (
	"context"
	"fmt"
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

type InitUploadResponse struct {
	UploadID string `json:"upload_id"`
	Bucket   string `json:"bucket"`
	Key      string `json:"key"`
	PartSize int64  `json:"part_size_mb"`
}

func NewUploadService(cfg *config.Config) *UploadService {
	// Load AWS config
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

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UseAccelerate = true
	})

	return &UploadService{
		s3Client: s3Client,
		cfg:      cfg,
	}
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
