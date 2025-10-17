package services

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
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
	OID                string `json:"oid"`
	Email              string `json:"email"`
	YearOfRegistration int    `json:"year_of_registration"`
	InternalID         string `json:"-"`
}

type SearchRequest struct {
	Query  string   `json:"query"`
	Fields []string `json:"fields"`
	AndOr  string   `json:"and_or"` // "AND" or "OR"
	Size   int      `json:"size"`
	From   int      `json:"from"` // Pagination offset
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
		docID := doc.InternalID
		if docID == "" {
			docID = generateDocumentID(doc)
		}

		indexAction := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": s.cfg.OpenSearchIndex,
				"_id":    docID,
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

	var lastErr error
	maxAttempts := int(math.Max(1, float64(s.cfg.OpenSearchBulkMaxAttempts)))

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := s.api.Bulk(
			context.Background(),
			opensearchapi.BulkReq{
				Body: bytes.NewReader(buf.Bytes()),
			},
		)
		if err != nil {
			lastErr = fmt.Errorf("bulk request failed on attempt %d/%d: %w", attempt, maxAttempts, err)
		} else {
			if resp.Errors {
				if bulkErr := s.inspectBulkErrors(resp); bulkErr != nil {
					lastErr = fmt.Errorf("bulk request returned item errors on attempt %d/%d: %w", attempt, maxAttempts, bulkErr)
				} else {
					log.Printf("Bulk indexed %d documents with recoverable errors on attempt %d", len(documents), attempt)
					return nil
				}
			} else {
				log.Printf("Bulk indexed %d documents on attempt %d", len(documents), attempt)
				return nil
			}
		}

		if attempt < maxAttempts {
			backoff := s.cfg.OpenSearchBulkRetryBase * time.Duration(1<<uint(attempt-1))
			jitter := time.Duration(rand.Int63n(int64(time.Second)))
			wait := backoff + jitter
			log.Printf("Retrying bulk index (attempt %d/%d) after %s due to error: %v", attempt, maxAttempts, wait, lastErr)
			time.Sleep(wait)
		}
	}

	return lastErr
}

func (s *OpenSearchService) inspectBulkErrors(resp *opensearchapi.BulkResp) error {
	if resp == nil || !resp.Errors {
		return nil
	}

	var failureMessages []string
	failedCount := 0
	for idx, item := range resp.Items {
		for action, result := range item {
			if result.Error != nil {
				failedCount++
				failureMessages = append(failureMessages, fmt.Sprintf("item %d action %s status %d type=%s reason=%s", idx, action, result.Status, result.Error.Type, result.Error.Reason))
			} else if result.Status >= 300 {
				failedCount++
				failureMessages = append(failureMessages, fmt.Sprintf("item %d action %s returned status %d", idx, action, result.Status))
			}
		}
	}

	if failedCount == 0 {
		return nil
	}

	if len(failureMessages) > 5 {
		failureMessages = failureMessages[:5]
	}

	return fmt.Errorf("bulk had %d failed items, sample: %s", failedCount, strings.Join(failureMessages, "; "))
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
	if val, ok := rawDoc["oid"].(string); ok {
		doc.OID = val
	}
	if doc.OID == "" {
		if val, ok := rawDoc["_id"].(string); ok {
			doc.OID = val
		}
	}
	if val, ok := rawDoc["email"].(string); ok {
		doc.Email = val
	}
	if val, ok := rawDoc["_id"].(map[string]interface{}); ok {
		if oid, ok := val["$oid"].(string); ok && oid != "" {
			doc.InternalID = oid
			if doc.OID == "" {
				doc.OID = oid
			}
		}
	}

	return doc
}

func generateDocumentID(doc Document) string {
	h := sha1.New()
	bump := doc.OID
	if bump == "" {
		bump = doc.InternalID
	}
	components := []string{
		bump,
		doc.Mobile,
		doc.Name,
		doc.Fname,
		doc.Address,
		doc.Alt,
		doc.ID,
		doc.Email,
	}

	for idx, part := range components {
		if idx > 0 {
			h.Write([]byte("|"))
		}
		h.Write([]byte(strings.ToLower(part)))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// buildFieldQuery creates the appropriate query based on field type
// Uses STRICT EXACT matching - NO fuzzy/partial matches for names
// Phone numbers support prefix for typing partial numbers
func buildFieldQuery(field, value string) map[string]interface{} {
	value = strings.TrimSpace(value)
	valueLower := strings.ToLower(value)

	// Phone number fields (mobile, alt) - exact term or prefix
	if field == "mobile" || field == "alt" {
		// Exact match or prefix for typing partial numbers
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						// Exact match
						"term": map[string]interface{}{
							field: valueLower,
						},
					},
					{
						// Prefix for partial numbers
						"prefix": map[string]interface{}{
							field: valueLower,
						},
					},
				},
				"minimum_should_match": 1,
			},
		}
	}

	// ID field - exact term or prefix
	if field == "id" || field == "oid" {
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							field: valueLower,
						},
					},
					{
						"prefix": map[string]interface{}{
							field: valueLower,
						},
					},
				},
				"minimum_should_match": 1,
			},
		}
	}

	// Email field - exact term or prefix
	if field == "email" {
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							field: valueLower,
						},
					},
					{
						"prefix": map[string]interface{}{
							field: valueLower,
						},
					},
				},
				"minimum_should_match": 1,
			},
		}
	}

	// Name field - exact keyword match with AND token requirement (no fuzziness)
	if field == "name" {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil
		}

		tokens := tokenize(trimmed)
		shouldClauses := make([]map[string]interface{}, 0, 2)
		shouldClauses = append(shouldClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"name.keyword": map[string]interface{}{
					"value":            trimmed,
					"case_insensitive": true,
				},
			},
		})

		if len(tokens) > 0 {
			mustTerms := make([]map[string]interface{}, 0, len(tokens))
			for _, token := range tokens {
				mustTerms = append(mustTerms, map[string]interface{}{
					"term": map[string]interface{}{
						"name.exact": token,
					},
				})
			}
			shouldClauses = append(shouldClauses, map[string]interface{}{
				"bool": map[string]interface{}{
					"must": mustTerms,
				},
			})
		}

		return map[string]interface{}{
			"bool": map[string]interface{}{
				"should":               shouldClauses,
				"minimum_should_match": 1,
			},
		}
	}

	// Father name - exact keyword match with AND token requirement
	if field == "fname" {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil
		}

		tokens := tokenize(trimmed)
		shouldClauses := make([]map[string]interface{}, 0, 2)
		shouldClauses = append(shouldClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"fname.keyword": map[string]interface{}{
					"value":            trimmed,
					"case_insensitive": true,
				},
			},
		})

		if len(tokens) > 0 {
			mustTerms := make([]map[string]interface{}, 0, len(tokens))
			for _, token := range tokens {
				mustTerms = append(mustTerms, map[string]interface{}{
					"term": map[string]interface{}{
						"fname.exact": token,
					},
				})
			}
			shouldClauses = append(shouldClauses, map[string]interface{}{
				"bool": map[string]interface{}{
					"must": mustTerms,
				},
			})
		}

		return map[string]interface{}{
			"bool": map[string]interface{}{
				"should":               shouldClauses,
				"minimum_should_match": 1,
			},
		}
	}

	// Address - keyword or token AND match
	if field == "address" {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil
		}

		tokens := tokenize(trimmed)
		shouldClauses := []map[string]interface{}{
			{
				"term": map[string]interface{}{
					"address.keyword": map[string]interface{}{
						"value":            trimmed,
						"case_insensitive": true,
					},
				},
			},
		}

		if len(tokens) > 0 {
			mustTerms := make([]map[string]interface{}, 0, len(tokens))
			for _, token := range tokens {
				mustTerms = append(mustTerms, map[string]interface{}{
					"term": map[string]interface{}{
						"address.parts": token,
					},
				})
			}
			shouldClauses = append(shouldClauses, map[string]interface{}{
				"bool": map[string]interface{}{
					"must": mustTerms,
				},
			})
		}

		return map[string]interface{}{
			"bool": map[string]interface{}{
				"should":               shouldClauses,
				"minimum_should_match": 1,
			},
		}
	}

	// Default: exact term match
	return map[string]interface{}{
		"term": map[string]interface{}{
			field: map[string]interface{}{
				"value":            valueLower,
				"case_insensitive": true,
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
			if q := buildFieldQuery(field, req.Query); q != nil {
				mustOrShould = append(mustOrShould, q)
			}
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
				if q := buildFieldQuery(field, value); q != nil {
					mustOrShould = append(mustOrShould, q)
				}
			}
		}

		query = map[string]interface{}{
			"bool": map[string]interface{}{
				operator: mustOrShould,
			},
		}
	}

	// Limit results to 50 per page for better performance
	size := req.Size
	if size <= 0 || size > 100 {
		size = 50 // Default to 50 results
	}

	// Pagination offset
	from := req.From
	if from < 0 {
		from = 0
	}

	searchBody := map[string]interface{}{
		"query":   query,
		"size":    size,
		"from":    from, // Pagination offset
		"_source": true,
		"timeout": "5s", // Fail fast if query takes too long
		"sort": []map[string]interface{}{
			{
				"_score": map[string]string{
					"order": "desc",
				},
			},
		},
	}

	bodyJSON, _ := json.Marshal(searchBody)

	// Log the query for debugging performance issues
	log.Printf("Search query: %s", string(bodyJSON))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	startTime := time.Now()
	resp, err := s.api.Search(
		ctx,
		&opensearchapi.SearchReq{
			Indices: []string{s.cfg.OpenSearchIndex},
			Body:    bytes.NewReader(bodyJSON),
			Params: opensearchapi.SearchParams{
				RequestCache: opensearchapi.ToPointer(true), // Enable request cache
			},
		},
	)
	queryDuration := time.Since(startTime)

	if err != nil {
		log.Printf("Search failed after %v: %v", queryDuration, err)
		return nil, fmt.Errorf("error searching: %v", err)
	}

	log.Printf("Search completed in %v (OpenSearch took: %dms, total hits: %d)",
		queryDuration, resp.Took, resp.Hits.Total.Value)

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

func tokenize(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	lower := strings.ToLower(trimmed)
	split := strings.FieldsFunc(lower, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})

	if len(split) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(split))
	for _, token := range split {
		if token == "" {
			continue
		}
		normalized = append(normalized, token)
	}
	return normalized
}

// isValidMasterID checks if a Master ID is valid (not masked with 'x' characters)
// Valid: "402371432105", "6802357444f7c329baa9993"
// Invalid: "xxxxxxxx2105", "xxxx1234", "xxx"
func isValidMasterID(masterID string) bool {
	if masterID == "" {
		return false
	}

	// Count the number of 'x' characters (case-insensitive)
	xCount := 0
	totalChars := len(masterID)

	for _, ch := range strings.ToLower(masterID) {
		if ch == 'x' {
			xCount++
		}
	}

	// If more than 30% of the ID is 'x' characters, consider it masked/invalid
	// This handles cases like "xxxxxxxx2105" (8 out of 12 chars = 66%)
	if totalChars > 0 && float64(xCount)/float64(totalChars) > 0.3 {
		return false
	}

	// Additional check: if ID starts with multiple 'x' characters, it's likely masked
	if totalChars >= 4 && strings.HasPrefix(strings.ToLower(masterID), "xxxx") {
		return false
	}

	// Must have at least some alphanumeric content
	if totalChars < 8 {
		return false
	}

	return true
}

// ComprehensiveMobileSearch performs an extensive search when searching by mobile number
// It searches for:
// 1. Direct matches in mobile and alt fields
// 2. All records associated with the master ID (oid) of found records
// 3. Records with matching name, fname, and address from initial results
func (s *OpenSearchService) ComprehensiveMobileSearch(mobileNumber string, size int) (*SearchResponse, error) {
	mobileNumber = strings.TrimSpace(mobileNumber)
	if mobileNumber == "" {
		return nil, fmt.Errorf("mobile number cannot be empty")
	}

	if size <= 0 || size > 100 {
		size = 50
	}

	// Step 1: Search for the mobile number in both mobile and alt fields
	initialQuery := map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []map[string]interface{}{
				{
					"term": map[string]interface{}{
						"mobile": strings.ToLower(mobileNumber),
					},
				},
				{
					"term": map[string]interface{}{
						"alt": strings.ToLower(mobileNumber),
					},
				},
			},
			"minimum_should_match": 1,
		},
	}

	initialSearchBody := map[string]interface{}{
		"query":   initialQuery,
		"size":    size,
		"_source": true,
		"timeout": "5s",
	}

	bodyJSON, _ := json.Marshal(initialSearchBody)
	log.Printf("Comprehensive mobile search - Initial query: %s", string(bodyJSON))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute initial search
	initialResp, err := s.api.Search(
		ctx,
		&opensearchapi.SearchReq{
			Indices: []string{s.cfg.OpenSearchIndex},
			Body:    bytes.NewReader(bodyJSON),
			Params: opensearchapi.SearchParams{
				RequestCache: opensearchapi.ToPointer(true),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("initial mobile search failed: %v", err)
	}

	// Collect unique values for comprehensive search
	masterIDSet := make(map[string]bool)
	nameSet := make(map[string]bool)
	fnameSet := make(map[string]bool)
	addressSet := make(map[string]bool)

	// Track filtered IDs for logging
	invalidMasterIDs := []string{}

	// Parse initial results
	var initialDocs []Document
	for _, hit := range initialResp.Hits.Hits {
		var doc Document
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			continue
		}
		initialDocs = append(initialDocs, doc)

		// Collect master IDs (ID field) - only valid, non-masked IDs
		if doc.ID != "" {
			if isValidMasterID(doc.ID) {
				masterIDSet[doc.ID] = true
			} else {
				// Only add to invalid list if not already there
				alreadyInvalid := false
				for _, inv := range invalidMasterIDs {
					if inv == doc.ID {
						alreadyInvalid = true
						break
					}
				}
				if !alreadyInvalid {
					invalidMasterIDs = append(invalidMasterIDs, doc.ID)
				}
			}
		}

		// Always collect names, father names, and addresses
		// We'll decide later whether to use them based on Master ID availability
		if doc.Name != "" {
			nameSet[strings.ToLower(strings.TrimSpace(doc.Name))] = true
		}
		if doc.Fname != "" {
			fnameSet[strings.ToLower(strings.TrimSpace(doc.Fname))] = true
		}
		if doc.Address != "" {
			addressSet[strings.ToLower(strings.TrimSpace(doc.Address))] = true
		}
	}

	// Log Master ID filtering
	if len(invalidMasterIDs) > 0 {
		log.Printf("Filtered out %d invalid/masked Master IDs (unique): %v", len(invalidMasterIDs), invalidMasterIDs)
	}
	if len(masterIDSet) > 0 {
		// Convert map to slice for logging (deduplicated)
		uniqueMasterIDs := make([]string, 0, len(masterIDSet))
		for id := range masterIDSet {
			uniqueMasterIDs = append(uniqueMasterIDs, id)
		}
		log.Printf("Using %d unique Master ID(s) for comprehensive search: %v", len(uniqueMasterIDs), uniqueMasterIDs)
	}

	// If no initial results, return empty response
	if len(initialDocs) == 0 {
		return &SearchResponse{
			Took: initialResp.Took,
			Hits: struct {
				Total struct {
					Value int `json:"value"`
				} `json:"total"`
				Hits []struct {
					Source Document `json:"_source"`
					Score  float64  `json:"_score"`
				} `json:"hits"`
			}{
				Total: struct {
					Value int `json:"value"`
				}{Value: 0},
				Hits: []struct {
					Source Document `json:"_source"`
					Score  float64  `json:"_score"`
				}{},
			},
		}, nil
	}

	// Step 2: Build comprehensive query with all collected data
	var comprehensiveShould []map[string]interface{}

	// Add original mobile/alt search with highest boost
	comprehensiveShould = append(comprehensiveShould, map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []map[string]interface{}{
				{
					"term": map[string]interface{}{
						"mobile": strings.ToLower(mobileNumber),
					},
				},
				{
					"term": map[string]interface{}{
						"alt": strings.ToLower(mobileNumber),
					},
				},
			},
			"minimum_should_match": 1,
			"boost":                3.0, // Highest boost for direct mobile matches
		},
	})

	// Add master ID searches (using ID field) - this is the most important
	// Use prefix search to handle Master IDs with suffixes (e.g., 718834428718M, 718834428718A)
	if len(masterIDSet) > 0 {
		for masterID := range masterIDSet {
			// For each Master ID, search for exact match OR prefix match
			comprehensiveShould = append(comprehensiveShould, map[string]interface{}{
				"bool": map[string]interface{}{
					"should": []map[string]interface{}{
						{
							"term": map[string]interface{}{
								"id": masterID,
							},
						},
						{
							"prefix": map[string]interface{}{
								"id": masterID, // This will match 718834428718, 718834428718M, 718834428718A, etc.
							},
						},
					},
					"minimum_should_match": 1,
					"boost":                2.0, // High boost for master ID matches
				},
			})
		}

		log.Printf("Master ID search will include prefix matching for suffixes (e.g., M, A, B)")
	}

	// Only add name/fname/address searches if we don't have Master IDs
	// This prevents too many unrelated results
	if len(masterIDSet) == 0 {
		// Add name searches
		if len(nameSet) > 0 {
			for name := range nameSet {
				comprehensiveShould = append(comprehensiveShould, map[string]interface{}{
					"term": map[string]interface{}{
						"name": map[string]interface{}{
							"value":            name,
							"case_insensitive": true,
							"boost":            1.2,
						},
					},
				})
			}
		}

		// Add father name searches
		if len(fnameSet) > 0 {
			for fname := range fnameSet {
				comprehensiveShould = append(comprehensiveShould, map[string]interface{}{
					"term": map[string]interface{}{
						"fname": map[string]interface{}{
							"value":            fname,
							"case_insensitive": true,
							"boost":            1.2,
						},
					},
				})
			}
		}

		// Add address searches
		if len(addressSet) > 0 {
			for address := range addressSet {
				comprehensiveShould = append(comprehensiveShould, map[string]interface{}{
					"term": map[string]interface{}{
						"address": map[string]interface{}{
							"value":            address,
							"case_insensitive": true,
							"boost":            1.1,
						},
					},
				})
			}
		}
	}

	comprehensiveQuery := map[string]interface{}{
		"bool": map[string]interface{}{
			"should":               comprehensiveShould,
			"minimum_should_match": 1,
		},
	}

	// Use a larger size for comprehensive search to ensure we get all Master ID matches
	// OpenSearch can handle up to 10000 results
	comprehensiveSize := 200 // Get up to 200 results for comprehensive search
	if len(masterIDSet) > 0 {
		// If we have Master IDs, we want to get ALL records with those IDs
		comprehensiveSize = 500
	}

	comprehensiveSearchBody := map[string]interface{}{
		"query":   comprehensiveQuery,
		"size":    comprehensiveSize,
		"_source": true,
		"timeout": "10s",
		"sort": []map[string]interface{}{
			{
				"_score": map[string]string{
					"order": "desc",
				},
			},
		},
	}

	comprehensiveBodyJSON, _ := json.Marshal(comprehensiveSearchBody)

	// Log what's actually in the query
	nameCount := 0
	fnameCount := 0
	addressCount := 0
	if len(masterIDSet) == 0 {
		// Only count these if they're actually in the query
		nameCount = len(nameSet)
		fnameCount = len(fnameSet)
		addressCount = len(addressSet)
	}

	log.Printf("Comprehensive mobile search - Query includes: %d Master IDs, %d names, %d fnames, %d addresses (size: %d)",
		len(masterIDSet), nameCount, fnameCount, addressCount, comprehensiveSize)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()

	// Execute comprehensive search
	comprehensiveResp, err := s.api.Search(
		ctx2,
		&opensearchapi.SearchReq{
			Indices: []string{s.cfg.OpenSearchIndex},
			Body:    bytes.NewReader(comprehensiveBodyJSON),
			Params: opensearchapi.SearchParams{
				RequestCache: opensearchapi.ToPointer(true),
			},
		},
	)
	if err != nil {
		log.Printf("Comprehensive search failed, falling back to initial results: %v", err)
		// Fall back to initial results
		return s.convertToSearchResponse(initialResp)
	}

	log.Printf("Comprehensive mobile search completed - returned %d out of %d total matching results",
		len(comprehensiveResp.Hits.Hits), comprehensiveResp.Hits.Total.Value)

	// If we got fewer results than expected with Master ID, log for debugging
	if len(masterIDSet) > 0 && comprehensiveResp.Hits.Total.Value < 60 {
		log.Printf("WARNING: Expected more results with Master ID. Got %d, size was %d",
			comprehensiveResp.Hits.Total.Value, comprehensiveSize)
	}

	return s.convertToSearchResponse(comprehensiveResp)
}

// Helper function to convert opensearchapi response to our SearchResponse
func (s *OpenSearchService) convertToSearchResponse(resp *opensearchapi.SearchResp) (*SearchResponse, error) {
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
