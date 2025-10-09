package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"notorious-backend/internal/config"

	opensearch "github.com/opensearch-project/opensearch-go/v3"
	"github.com/opensearch-project/opensearch-go/v3/opensearchapi"
)

type OpenSearchService struct {
	client *opensearch.Client
	api    *opensearchapi.Client
	cfg    *config.Config
}

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

type Document struct {
	Mobile             string `json:"mobile"`
	Name               string `json:"name"`
	Fname              string `json:"fname"`
	Address            string `json:"address"`
	AltAddress         string `json:"alt_address"`
	Alt                string `json:"alt"`
	ID                 string `json:"id"`
	Email              string `json:"email"`
	YearOfRegistration int    `json:"year_of_registration"`
}

type SearchRequest struct {
	Query  string   `json:"query"`
	Fields []string `json:"fields"`
	AndOr  string   `json:"and_or"` // "AND" or "OR"
	Size   int      `json:"size"`
}

type SearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source Document `json:"_source"`
			Score  float64  `json:"_score"`
		} `json:"hits"`
	} `json:"hits"`
	Took int `json:"took"`
}

func NewOpenSearchService(cfg *config.Config) *OpenSearchService {
	// Create OpenSearch client with basic auth
	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{cfg.OpenSearchEndpoint},
		Username:  cfg.OpenSearchMasterUser,
		Password:  cfg.OpenSearchMasterPass,
	})
	if err != nil {
		log.Fatalf("Error creating OpenSearch client: %v", err)
	}

	apiClient, err := opensearchapi.NewClient(opensearchapi.Config{Client: opensearch.Config{
		Addresses: []string{cfg.OpenSearchEndpoint},
		Username:  cfg.OpenSearchMasterUser,
		Password:  cfg.OpenSearchMasterPass,
	}})
	if err != nil {
		log.Fatalf("Error creating OpenSearch API client: %v", err)
	}

	return &OpenSearchService{
		client: client,
		api:    apiClient,
		cfg:    cfg,
	}
}

func (s *OpenSearchService) ApplyIndexTemplate() error {
	templatePath := filepath.Join("templates", "people_v1.json")

	templateJSON, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read index template %s: %w", templatePath, err)
	}

	req := opensearchapi.IndexTemplateCreateReq{
		IndexTemplate: "people_v1",
		Body:          bytes.NewReader(templateJSON),
	}

	resp, err := s.api.IndexTemplate.Create(context.Background(), req)
	if err != nil {
		var apiErr opensearchapi.Error
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusBadRequest && strings.Contains(apiErr.Err.Reason, "already exists") {
			log.Printf("Index template people_v1 already exists; skipping creation")
			return nil
		}
		return fmt.Errorf("error applying index template: %w", err)
	}

	log.Printf("Index template applied successfully: acknowledged=%t", resp.Acknowledged)
	return nil
}

func (s *OpenSearchService) CreateIndex() error {
	indexSettings := `{
		"settings": {
			"number_of_shards": 6,
			"number_of_replicas": 0,
			"refresh_interval": "-1"
		}
	}`

	resp, err := s.api.Indices.Create(
		context.Background(),
		opensearchapi.IndicesCreateReq{
			Index: s.cfg.OpenSearchIndex,
			Body:  strings.NewReader(indexSettings),
		},
	)
	if err != nil {
		var apiErr opensearchapi.Error
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusBadRequest &&
			strings.Contains(apiErr.Err.Type, "resource_already_exists_exception") {
			log.Printf("Index %s already exists; skipping creation", s.cfg.OpenSearchIndex)
			return nil
		}
		return fmt.Errorf("error creating index: %w", err)
	}

	log.Printf("Index created successfully: index=%s acknowledged=%t", resp.Index, resp.Acknowledged)
	return nil
}

func (s *OpenSearchService) BulkIndex(documents []Document) error {
	if len(documents) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, doc := range documents {
		// Create index action
		indexAction := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": s.cfg.OpenSearchIndex,
			},
		}

		indexActionJSON, _ := json.Marshal(indexAction)
		buf.Write(indexActionJSON)
		buf.WriteString("\n")

		// Add document
		docJSON, _ := json.Marshal(doc)
		buf.Write(docJSON)
		buf.WriteString("\n")
	}

	resp, err := s.api.Bulk(
		context.Background(),
		opensearchapi.BulkReq{
			Body: bytes.NewReader(buf.Bytes()),
		},
	)
	if err != nil {
		return fmt.Errorf("error bulk indexing: %v", err)
	}

	log.Printf("Bulk indexed %d documents (errors=%t)", len(documents), resp.Errors)
	return nil
}

func (s *OpenSearchService) TransformDocument(rawDoc map[string]interface{}) Document {
	// Generate random year of registration
	year := 2022 + seededRand.Intn(3) // 2022, 2023, or 2024

	doc := Document{
		YearOfRegistration: year,
	}

	// Map fields, dropping _id and circle
	if val, ok := rawDoc["mobile"].(string); ok {
		doc.Mobile = val
	}
	if val, ok := rawDoc["name"].(string); ok {
		doc.Name = val
	}
	if val, ok := rawDoc["fname"].(string); ok {
		doc.Fname = val
	}
	if val, ok := rawDoc["address"].(string); ok {
		doc.Address = val
		doc.AltAddress = val // Copy address to alt_address
	}
	if val, ok := rawDoc["alt"].(string); ok {
		doc.Alt = val
	}
	if val, ok := rawDoc["id"].(string); ok {
		doc.ID = val
	}
	if val, ok := rawDoc["email"].(string); ok {
		doc.Email = val
	}

	return doc
}

// buildFieldQuery creates the appropriate query based on field type
func buildFieldQuery(field, value string) map[string]interface{} {
	// Keyword fields (exact match) - mobile, alt, id, email
	keywordFields := map[string]bool{
		"mobile": true,
		"alt":    true,
		"id":     true,
		"email":  true,
	}

	if keywordFields[field] {
		// For keyword fields, use wildcard for partial matching
		return map[string]interface{}{
			"wildcard": map[string]interface{}{
				field: map[string]interface{}{
					"value":            "*" + strings.ToLower(value) + "*",
					"case_insensitive": true,
				},
			},
		}
	}

	// Special handling for name - use .exact subfield for precise matching
	if field == "name" {
		return map[string]interface{}{
			"match_phrase": map[string]interface{}{
				"name.exact": map[string]interface{}{
					"query": value,
					"slop":  0,
				},
			},
		}
	}

	// Special handling for fname - use wildcard on .keyword for case-insensitive exact matching
	if field == "fname" {
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						// Try exact match on keyword field (case-sensitive)
						"term": map[string]interface{}{
							"fname.keyword": value,
						},
					},
					{
						// Fallback to wildcard for case-insensitive
						"wildcard": map[string]interface{}{
							"fname.keyword": map[string]interface{}{
								"value":            "*" + value + "*",
								"case_insensitive": true,
							},
						},
					},
				},
				"minimum_should_match": 1,
			},
		}
	}

	// Special handling for address - search on .parts subfield
	if field == "address" {
		return map[string]interface{}{
			"match_phrase": map[string]interface{}{
				"address.parts": map[string]interface{}{
					"query": value,
					"slop":  0,
				},
			},
		}
	}

	// Default: exact phrase match
	return map[string]interface{}{
		"match_phrase": map[string]interface{}{
			field: map[string]interface{}{
				"query": value,
				"slop":  0,
			},
		},
	}
}

// parseFieldQuery parses query string like "name:john AND fname:smith" into field-value pairs
func parseFieldQuery(query string, operator string) []map[string]string {
	result := []map[string]string{}

	// Split by AND or OR
	delimiter := " OR "
	if strings.ToUpper(operator) == "AND" {
		delimiter = " AND "
	}

	parts := strings.Split(query, delimiter)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if it contains field:value syntax
		if strings.Contains(part, ":") {
			colonIdx := strings.Index(part, ":")
			field := strings.TrimSpace(part[:colonIdx])
			value := strings.TrimSpace(part[colonIdx+1:])

			if field != "" && value != "" {
				result = append(result, map[string]string{field: value})
			}
		}
	}

	return result
}

func (s *OpenSearchService) Search(req SearchRequest) (*SearchResponse, error) {
	// Parse query for field:value syntax
	fieldQueries := parseFieldQuery(req.Query, req.AndOr)

	var query map[string]interface{}

	if len(fieldQueries) == 0 {
		// No field:value pairs found, use multi-field search
		var mustOrShould []map[string]interface{}
		operator := "should"
		if strings.ToUpper(req.AndOr) == "AND" {
			operator = "must"
		}

		for _, field := range req.Fields {
			mustOrShould = append(mustOrShould, buildFieldQuery(field, req.Query))
		}

		query = map[string]interface{}{
			"bool": map[string]interface{}{
				operator: mustOrShould,
			},
		}
	} else if len(fieldQueries) == 1 {
		// Single field:value query
		for field, value := range fieldQueries[0] {
			query = buildFieldQuery(field, value)
		}
	} else {
		// Multiple field:value queries with AND/OR
		var mustOrShould []map[string]interface{}
		operator := "should"
		if strings.ToUpper(req.AndOr) == "AND" {
			operator = "must"
		}

		for _, fq := range fieldQueries {
			for field, value := range fq {
				mustOrShould = append(mustOrShould, buildFieldQuery(field, value))
			}
		}

		query = map[string]interface{}{
			"bool": map[string]interface{}{
				operator: mustOrShould,
			},
		}
	}

	searchBody := map[string]interface{}{
		"query":   query,
		"size":    req.Size,
		"_source": true,
	}

	bodyJSON, _ := json.Marshal(searchBody)

	resp, err := s.api.Search(
		context.Background(),
		&opensearchapi.SearchReq{
			Indices: []string{s.cfg.OpenSearchIndex},
			Body:    bytes.NewReader(bodyJSON),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error searching: %v", err)
	}

	// Map the SDK response into our SearchResponse struct
	result := &SearchResponse{
		Took: resp.Took,
	}

	result.Hits.Total.Value = resp.Hits.Total.Value
	for _, hit := range resp.Hits.Hits {
		var doc Document
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			return nil, fmt.Errorf("error decoding search hit: %v", err)
		}
		result.Hits.Hits = append(result.Hits.Hits, struct {
			Source Document `json:"_source"`
			Score  float64  `json:"_score"`
		}{
			Source: doc,
			Score:  float64(hit.Score),
		})
	}

	return result, nil
}

func (s *OpenSearchService) FinalizeIndex() error {
	// Re-enable replicas and refresh
	settings := `{
		"settings": {
			"number_of_replicas": 1,
			"refresh_interval": "1s"
		}
	}`

	resp, err := s.api.Indices.Settings.Put(
		context.Background(),
		opensearchapi.SettingsPutReq{
			Indices: []string{s.cfg.OpenSearchIndex},
			Body:    strings.NewReader(settings),
		},
	)
	if err != nil {
		return fmt.Errorf("error finalizing index: %v", err)
	}

	log.Printf("Index finalized with replicas and refresh enabled: acknowledged=%t", resp.Acknowledged)
	return nil
}
