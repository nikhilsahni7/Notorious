#!/bin/bash

# Test AND logic - should find people named Sunny with father name Avinash
echo "Testing AND logic (name:Sunny AND fname:Avinash):"
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "name:Sunny AND fname:Avinash",
    "and_or": "AND",
    "size": 5
  }' | jq '.total, .results[0].name, .results[0].fname'

echo -e "\n---\n"

# Test OR logic - should find people named Sunny OR with father name Kishan
echo "Testing OR logic (name:Sunny OR fname:Kishan):"
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "name:Sunny OR fname:Kishan",
    "and_or": "OR",
    "size": 5
  }' | jq '.total'

echo -e "\n---\n"

# Test single field
echo "Testing single field (name:Sunny):"
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "name:Sunny",
    "and_or": "OR",
    "size": 5
  }' | jq '.total, .results[0].name'


