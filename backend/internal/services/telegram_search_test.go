package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/opensearch-project/opensearch-go/v3"
	"github.com/opensearch-project/opensearch-go/v3/opensearchapi"
	"notorious-backend/internal/config"
)

type mockTransport func(*http.Request) (*http.Response, error)

func (m mockTransport) RoundTrip(r *http.Request) (*http.Response, error) { return m(r) }
func TestTelegramSearchExpandsOnlyExactMasterIDs(t *testing.T) {
	var queries []string
	transport := mockTransport(func(r *http.Request) (*http.Response, error) {
		var q map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			t.Error(err)
		}
		b, _ := json.Marshal(q)
		queries = append(queries, string(b))
		result := `{"took":1,"hits":{"total":{"value":1,"relation":"eq"},"hits":[{"_score":1,"_source":{"id":"master123"}}]}}`
		if len(queries) == 1 {
			result = `{"took":1,"hits":{"total":{"value":1,"relation":"eq"},"hits":[{"_score":1,"_source":{"id":"master123","mobile":"1234567890","alt":"5555555555"}}]}}`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(result))}, nil
	})
	client, err := opensearchapi.NewClient(opensearchapi.Config{Client: opensearch.Config{Addresses: []string{"http://localhost:9200"}, Transport: transport}})
	if err != nil {
		t.Fatal(err)
	}
	svc := &OpenSearchService{api: client, cfg: &config.Config{OpenSearchIndices: []string{"people"}}}
	result, err := svc.TelegramSearch(context.Background(), "phone", "1234567890", "delhi-ncr")
	if err != nil {
		t.Fatal(err)
	}
	if result.Hits.Total.Value != 1 || len(queries) != 2 {
		t.Fatalf("unexpected result: %+v, %d queries", result, len(queries))
	}
	if strings.Contains(queries[0], `"alt"`) {
		t.Fatalf("initial phone search must not match alternate numbers: %s", queries[0])
	}
	if !strings.Contains(queries[1], `"terms":{"id":["master123"]}`) || strings.Contains(queries[1], "5555555555") || strings.Contains(queries[1], `"prefix"`) {
		t.Fatalf("unexpected expansion query: %s", queries[1])
	}
	if !strings.Contains(queries[1], "delhi-ncr") {
		t.Fatalf("region missing: %s", queries[1])
	}
}

func TestTelegramSearchTypesAndEmptyResults(t *testing.T) {
	for _, tc := range []struct{ kind, value, field string }{{"phone", "9876543210", "mobile"}, {"email", "user@example.com", "email"}, {"master_id", "master123", "id"}} {
		t.Run(tc.kind, func(t *testing.T) {
			calls := 0
			transport := mockTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				data, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(data), `"`+tc.field+`":"`+tc.value+`"`) {
					t.Errorf("wrong initial query: %s", data)
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`))}, nil
			})
			client, err := opensearchapi.NewClient(opensearchapi.Config{Client: opensearch.Config{Addresses: []string{"http://localhost:9200"}, Transport: transport}})
			if err != nil {
				t.Fatal(err)
			}
			svc := &OpenSearchService{api: client, cfg: &config.Config{OpenSearchIndices: []string{"people"}}}
			result, err := svc.TelegramSearch(context.Background(), tc.kind, tc.value, "pan-india")
			if err != nil || result.Hits.Total.Value != 0 || calls != 1 {
				t.Fatalf("unexpected: result=%v err=%v calls=%d", result, err, calls)
			}
		})
	}
}
func TestTelegramSearchRejectsIncompleteOpenSearch(t *testing.T) {
	for _, body := range []string{`{"timed_out":true,"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`, `{"_shards":{"failed":1},"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`} {
		transport := mockTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		client, err := opensearchapi.NewClient(opensearchapi.Config{Client: opensearch.Config{Addresses: []string{"http://localhost:9200"}, Transport: transport}})
		if err != nil {
			t.Fatal(err)
		}
		svc := &OpenSearchService{api: client, cfg: &config.Config{OpenSearchIndices: []string{"people"}}}
		if _, err = svc.TelegramSearch(context.Background(), "phone", "9876543210", "pan-india"); err == nil {
			t.Fatal("accepted partial search")
		}
	}
}
