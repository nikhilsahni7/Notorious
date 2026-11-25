package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opensearch-project/opensearch-go/v3/opensearchapi"
)

// TelegramSearch only expands direct matches through exact, valid master IDs.
// It deliberately does not follow names, addresses or alternate-number links.
func (s *OpenSearchService) TelegramSearch(ctx context.Context, kind, value, region string) (*SearchResponse, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	var query map[string]interface{}
	switch kind {
	case "phone":
		query = map[string]interface{}{"term": map[string]interface{}{"mobile": value}}
	case "email":
		query = map[string]interface{}{"term": map[string]interface{}{"email": value}}
	case "master_id":
		query = map[string]interface{}{"term": map[string]interface{}{"id": value}}
	default:
		return nil, fmt.Errorf("invalid search type")
	}
	search := func(q map[string]interface{}, size int) (*opensearchapi.SearchResp, error) {
		body, err := json.Marshal(map[string]interface{}{"query": addRegionFilter(q, region), "size": size, "track_total_hits": true, "timeout": "5s"})
		if err != nil {
			return nil, err
		}
		resp, err := s.api.Search(ctx, &opensearchapi.SearchReq{Indices: s.cfg.OpenSearchIndices, Body: bytes.NewReader(body)})
		if err != nil {
			return nil, err
		}
		if resp.Timeout || resp.Shards.Failed > 0 {
			return nil, fmt.Errorf("search returned incomplete results")
		}
		return resp, nil
	}
	initial, err := search(query, 100)
	if err != nil {
		return nil, err
	}
	// Refuse incomplete expansion: silently dropping master IDs would be misleading.
	if initial.Hits.Total.Value > len(initial.Hits.Hits) {
		return nil, fmt.Errorf("too many direct matches")
	}
	ids := make(map[string]bool)
	for _, hit := range initial.Hits.Hits {
		var doc Document
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			return nil, err
		}
		if isValidMasterID(doc.ID) {
			ids[strings.ToLower(doc.ID)] = true
		}
	}
	if len(ids) == 0 {
		return &SearchResponse{}, nil
	}
	values := make([]string, 0, len(ids))
	for id := range ids {
		values = append(values, id)
	}
	expanded, err := search(map[string]interface{}{"terms": map[string]interface{}{"id": values}}, 100)
	if err != nil {
		return nil, err
	}
	return s.convertToSearchResponse(expanded)
}
