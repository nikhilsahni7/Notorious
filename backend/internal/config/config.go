package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AWSRegion                 string
	OpenSearchEndpoint        string
	OpenSearchIndex           string
	OpenSearchMasterUser      string
	OpenSearchMasterPass      string
	S3UploadBucket            string
	S3UploadPrefix            string
	AWSAccessKeyID            string
	AWSSecretAccessKey        string
	OpenSearchBulkMaxAttempts int
	OpenSearchBulkRetryBase   time.Duration
	IngestBatchSize           int
	IngestWorkerMultiplier    int
}

func Load() *Config {
	return &Config{
		AWSRegion:                 getEnv("AWS_REGION", "us-east-1"),
		OpenSearchEndpoint:        getEnv("OPENSEARCH_ENDPOINT", ""),
		OpenSearchIndex:           getEnv("OPENSEARCH_INDEX", "people-dev-0001"),
		OpenSearchMasterUser:      getEnv("OPENSEARCH_MASTER_USER", ""),
		OpenSearchMasterPass:      getEnv("OPENSEARCH_MASTER_PASSWORD", ""),
		S3UploadBucket:            getEnv("S3_UPLOAD_BUCKET", ""),
		S3UploadPrefix:            getEnv("S3_UPLOAD_PREFIX", "ingest/raw/"),
		AWSAccessKeyID:            getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:        getEnv("AWS_SECRET_ACCESS_KEY", ""),
		OpenSearchBulkMaxAttempts: getEnvInt("OPENSEARCH_BULK_MAX_ATTEMPTS", 5),
		OpenSearchBulkRetryBase:   getEnvDuration("OPENSEARCH_BULK_RETRY_BASE", 2*time.Second),
		IngestBatchSize:           clampInt(getEnvInt("INGEST_BATCH_SIZE", 7500), 1000, 20000),
		IngestWorkerMultiplier:    clampInt(getEnvInt("INGEST_WORKER_MULTIPLIER", 2), 1, 8),
	}
}

func clampInt(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
