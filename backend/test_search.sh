#!/bin/bash

# Test script for search API

echo "Testing search API..."

# Start the server in background
cd backend
go run main.go &
SERVER_PID=$!

# Wait for server to start
sleep 5

# Test search endpoint
echo "Testing search for 'Sunny'..."
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Sunny",
    "fields": ["name"],
    "and_or": "OR",
    "size": 5
  }'

echo -e "\n\nTesting multi-field search..."
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "9990369041",
    "fields": ["mobile", "name", "address"],
    "and_or": "OR",
    "size": 10
  }'

echo -e "\n\nTesting address search..."
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Faridabad",
    "fields": ["address"],
    "and_or": "OR",
    "size": 5
  }'

echo -e "\n\nTesting ID search..."
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "488398401752",
    "fields": ["id"],
    "and_or": "OR",
    "size": 5
  }'

echo -e "\n\nTesting alternate number search..."
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "9711416113",
    "fields": ["alt"],
    "and_or": "OR",
    "size": 5
  }'

# Kill the server
kill $SERVER_PID

echo -e "\n\nSearch tests completed!"
