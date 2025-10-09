package config

import (
	"os"
)

type Config struct {
	AWSRegion              string
	OpenSearchEndpoint     string
	OpenSearchIndex        string
	OpenSearchMasterUser   string
	OpenSearchMasterPass   string
	S3UploadBucket         string
	S3UploadPrefix         string
	AWSAccessKeyID         string
	AWSSecretAccessKey     string
}

func Load() *Config {
	return &Config{
		AWSRegion:              getEnv("AWS_REGION", "ap-south-1"),
		OpenSearchEndpoint:     getEnv("OPENSEARCH_ENDPOINT", ""),
		OpenSearchIndex:        getEnv("OPENSEARCH_INDEX", "people-dev-0001"),
		OpenSearchMasterUser:   getEnv("OPENSEARCH_MASTER_USER", ""),
		OpenSearchMasterPass:   getEnv("OPENSEARCH_MASTER_PASSWORD", ""),
		S3UploadBucket:         getEnv("S3_UPLOAD_BUCKET", ""),
		S3UploadPrefix:         getEnv("S3_UPLOAD_PREFIX", "ingest/raw/"),
		AWSAccessKeyID:         getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:     getEnv("AWS_SECRET_ACCESS_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
