#!/bin/bash

# Test script for ingesting the 5GB sample file

echo "Starting ingestion test..."

# Load environment variables
source .env

# Check if file exists
FILE_PATH="/home/nikhil-sahni/Downloads/Telegram Desktop/try sample.json"
if [ ! -f "$FILE_PATH" ]; then
    echo "Error: File not found at $FILE_PATH"
    exit 1
fi

echo "File found: $FILE_PATH"
echo "File size: $(du -h "$FILE_PATH" | cut -f1)"

# Initialize Go modules
echo "Initializing Go modules..."
cd backend
go mod tidy

# Run the ingestion
echo "Starting ingestion..."
go run cmd/ingest/main.go "$FILE_PATH"

echo "Ingestion completed!"
