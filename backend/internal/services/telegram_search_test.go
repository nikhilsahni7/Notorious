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
func TestTelegramSearchExpandsAlternatePhoneAndIDs(t *testing.T) {
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
	if result.Hits.Total.Value != 1 || len(queries) != 4 {
		t.Fatalf("unexpected result: %+v, %d queries", result, len(queries))
	}
	if !strings.Contains(queries[0], `"alt"`) || !strings.Contains(queries[0], `"mobile"`) {
		t.Fatalf("phone lookup must include mobile and alt: %s", queries[0])
	}
	if !strings.Contains(queries[1], `"terms":{"id":["master123"]}`) || !strings.Contains(queries[2], "5555555555") || !strings.Contains(queries[3], "5555555555") {
		t.Fatalf("alternate expansion missing: %v", queries)
	}
	if !strings.Contains(queries[3], "delhi-ncr") {
		t.Fatalf("region missing: %s", queries[3])
	}

}

func TestTelegramSearchTypesAndEmptyResults(t *testing.T) {
	for _, tc := range []struct{ kind, value, field string }{{"phone", "9876543210", "mobile"}, {"auto", "123456789012", "id"}, {"email", "user@example.com", "email"}, {"master_id", "master123", "id"}} {
		t.Run(tc.kind, func(t *testing.T) {
			calls := 0
			transport := mockTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				data, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(data), `"`+tc.field+`":"`+tc.value+`"`) {
					t.Errorf("wrong initial query: %s", data)
				}
				if tc.kind == "auto" && (!strings.Contains(string(data), `"mobile":"`+tc.value+`"`) || !strings.Contains(string(data), `"alt":"`+tc.value+`"`)) {
					t.Errorf("automatic lookup missing phone fields: %s", data)
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

func TestTelegramSearchHandlesLargeDirectMatches(t *testing.T) {
	calls := 0
	transport := mockTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if calls == 1 {
			if body["size"] != float64(500) {
				t.Fatalf("wrong scan size: %v", body["size"])
			}
			result := `{"hits":{"total":{"value":501,"relation":"eq"},"hits":[{"_score":1,"_source":{"id":"master123"}},{"_score":1,"_source":{"id":123}}]}}`
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(result))}, nil
		}
		if calls == 2 {
			result := `{"hits":{"total":{"value":1,"relation":"eq"},"hits":[{"_score":1,"_source":{"id":"master123"}}]}}`
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(result))}, nil
		}
		if body["size"] != float64(5) {
			t.Fatalf("wrong output size: %v", body["size"])
		}
		result := `{"hits":{"total":{"value":10,"relation":"eq"},"hits":[{"_score":1,"_source":{"id":"master123","name":"Test"}}]}}`
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(result))}, nil
	})
	client, err := opensearchapi.NewClient(opensearchapi.Config{Client: opensearch.Config{Addresses: []string{"http://localhost:9200"}, Transport: transport}})
	if err != nil {
		t.Fatal(err)
	}
	svc := &OpenSearchService{api: client, cfg: &config.Config{OpenSearchIndices: []string{"people"}}}
	result, err := svc.TelegramSearch(context.Background(), "email", "test@example.com", "pan-india")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Partial || result.Hits.Total.Value != 10 || calls != 3 {
		t.Fatalf("unexpected result %+v calls %d", result, calls)
	}
}

func TestTelegramSearchMoreThanHundredMatchingRecords(t *testing.T) {
	calls := 0
	transport := mockTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var body string
		if calls == 1 {
			records := make([]string, 101)
			for i := range records {
				records[i] = `{"_score":1,"_source":{"id":"master123"}}`
			}
			body = `{"hits":{"total":{"value":101,"relation":"eq"},"hits":[` + strings.Join(records, ",") + `]}}`
		} else {
			body = `{"hits":{"total":{"value":1,"relation":"eq"},"hits":[{"_score":1,"_source":{"id":"master123"}}]}}`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	client, err := opensearchapi.NewClient(opensearchapi.Config{Client: opensearch.Config{Addresses: []string{"http://localhost:9200"}, Transport: transport}})
	if err != nil {
		t.Fatal(err)
	}
	svc := &OpenSearchService{api: client, cfg: &config.Config{OpenSearchIndices: []string{"people"}}}
	result, err := svc.TelegramSearch(context.Background(), "email", "test@example.com", "pan-india")
	if err != nil || result.Partial || result.Hits.Total.Value != 1 {
		t.Fatalf("search over 100 failed: %+v, %v", result, err)
	}
}
