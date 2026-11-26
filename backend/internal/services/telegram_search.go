package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v3/opensearchapi"
)

var (
	ErrTelegramSearchUpstream       = errors.New("upstream search failed")
	ErrTelegramSearchIncomplete     = errors.New("search returned incomplete results")
	ErrTelegramSearchTooManyMatches = errors.New("too many direct matches")
	ErrTelegramSearchInvalidRecord  = errors.New("invalid search record")
)

// TelegramSearch expands through document IDs and one bounded alternate-phone
// hop. It never follows names or addresses and reports truncated scans.
func (s *OpenSearchService) TelegramSearch(ctx context.Context, kind, value, region string) (*SearchResponse, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	var query map[string]interface{}
	switch kind {
	case "phone":
		query = map[string]interface{}{"bool": map[string]interface{}{"should": []interface{}{map[string]interface{}{"term": map[string]interface{}{"mobile": value}}, map[string]interface{}{"term": map[string]interface{}{"alt": value}}}, "minimum_should_match": 1}}
	case "auto":
		query = map[string]interface{}{"bool": map[string]interface{}{"should": []interface{}{map[string]interface{}{"term": map[string]interface{}{"mobile": value}}, map[string]interface{}{"term": map[string]interface{}{"alt": value}}, map[string]interface{}{"term": map[string]interface{}{"id": value}}}, "minimum_should_match": 1}}
	case "email":
		query = map[string]interface{}{"term": map[string]interface{}{"email": value}}
	case "master_id":
		query = map[string]interface{}{"term": map[string]interface{}{"id": value}}
	default:
		return nil, fmt.Errorf("invalid search type")
	}
	search := func(stage string, q map[string]interface{}, size int, source interface{}, track interface{}) (*opensearchapi.SearchResp, error) {
		body, err := json.Marshal(map[string]interface{}{"query": addRegionFilter(q, region), "size": size, "_source": source, "track_total_hits": track, "timeout": "5s"})
		if err != nil {
			return nil, err
		}
		started := time.Now()
		resp, err := s.api.Search(ctx, &opensearchapi.SearchReq{Indices: s.cfg.OpenSearchIndices, Body: bytes.NewReader(body)})
		if err != nil {
			log.Printf("telegram search stage=%s type=%s failed duration=%s", stage, kind, time.Since(started).Round(time.Millisecond))
			return nil, ErrTelegramSearchUpstream
		}
		if resp.Timeout || resp.Shards.Failed > 0 {
			log.Printf("telegram search stage=%s type=%s incomplete duration=%s", stage, kind, time.Since(started).Round(time.Millisecond))
			return nil, ErrTelegramSearchIncomplete
		}
		if time.Since(started) > 2*time.Second {
			log.Printf("telegram search stage=%s type=%s slow duration=%s", stage, kind, time.Since(started).Round(time.Millisecond))
		}
		return resp, nil
	}
	const maxDirectMatches = 500
	initial, err := search("direct", query, maxDirectMatches, []string{"id", "mobile", "alt"}, maxDirectMatches+1)
	if err != nil {
		return nil, err
	}
	// A bounded scan protects the cluster. Tell the caller if more direct
	// matches exist, rather than claiming the resulting count is exhaustive.
	partial := initial.Hits.Total.Value > len(initial.Hits.Hits) || initial.Hits.Total.Relation == "gte"
	ids := make(map[string]bool)
	for _, hit := range initial.Hits.Hits {
		var doc struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			continue
		}
		if isValidMasterID(doc.ID) {
			ids[strings.ToLower(doc.ID)] = true
		}
	}
	if len(initial.Hits.Hits) == 0 {
		return &SearchResponse{Partial: partial}, nil
	}
	values := make([]string, 0, len(ids))
	for id := range ids {
		values = append(values, id)
	}
	sort.Strings(values)
	linked := initial
	if len(values) > 0 {
		linked, err = search("documents", map[string]interface{}{"terms": map[string]interface{}{"id": values}}, maxDirectMatches, []string{"id", "mobile", "alt"}, maxDirectMatches+1)
		if err != nil {
			return nil, err
		}
	}
	partial = partial || linked.Hits.Total.Value > len(linked.Hits.Hits) || linked.Hits.Total.Relation == "gte"
	phones := make(map[string]bool)
	validPhone := regexp.MustCompile(`^[0-9]{10,15}$`)
	for _, hit := range append(initial.Hits.Hits, linked.Hits.Hits...) {
		var doc struct {
			Alt string `json:"alt"`
		}
		if json.Unmarshal(hit.Source, &doc) == nil && validPhone.MatchString(strings.TrimSpace(doc.Alt)) {
			phones[strings.TrimSpace(doc.Alt)] = true
		}
	}
	if len(phones) > 0 {
		numbers := make([]string, 0, len(phones))
		for number := range phones {
			numbers = append(numbers, number)
		}
		sort.Strings(numbers)
		q := map[string]interface{}{"bool": map[string]interface{}{"should": []interface{}{map[string]interface{}{"terms": map[string]interface{}{"mobile": numbers}}, map[string]interface{}{"terms": map[string]interface{}{"alt": numbers}}}, "minimum_should_match": 1}}
		alternate, e := search("alternates", q, maxDirectMatches, []string{"id"}, maxDirectMatches+1)
		if e != nil {
			return nil, e
		}
		partial = partial || alternate.Hits.Total.Value > len(alternate.Hits.Hits) || alternate.Hits.Total.Relation == "gte"
		for _, hit := range alternate.Hits.Hits {
			var doc struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(hit.Source, &doc) == nil && isValidMasterID(doc.ID) {
				ids[strings.ToLower(doc.ID)] = true
			}
		}
	}
	values = values[:0]
	for id := range ids {
		values = append(values, id)
	}
	sort.Strings(values)
	should := []interface{}{query}
	if len(values) > 0 {
		should = append(should, map[string]interface{}{"terms": map[string]interface{}{"id": values}})
	}
	if len(phones) > 0 {
		numbers := make([]string, 0, len(phones))
		for number := range phones {
			numbers = append(numbers, number)
		}
		sort.Strings(numbers)
		should = append(should, map[string]interface{}{"terms": map[string]interface{}{"mobile": numbers}}, map[string]interface{}{"terms": map[string]interface{}{"alt": numbers}})
	}
	expanded, err := search("results", map[string]interface{}{"bool": map[string]interface{}{"should": should, "minimum_should_match": 1}}, 5, true, false)
	if err != nil {
		return nil, err
	}
	result, err := s.convertToSearchResponse(expanded)
	if err != nil {
		return nil, ErrTelegramSearchInvalidRecord
	}
	// The final query deliberately does not count every match. The bot only
	// needs to distinguish zero results from the records actually returned.
	result.Hits.Total.Value = len(result.Hits.Hits)
	result.Partial = partial
	return result, nil
}
