package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebhookRejectsWrongSecret(t *testing.T) {
	a := &app{webhookSecret: "01234567890123456789012345678901"}
	r := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader(`{}`))
	r.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
	w := httptest.NewRecorder()
	a.webhook(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", w.Code)
	}
}
func TestLimitPerUser(t *testing.T) {
	a := &app{}
	for i := 0; i < 20; i++ {
		if !a.allow(1) {
			t.Fatal("blocked too early")
		}
	}
	if a.allow(1) {
		t.Fatal("limit not enforced")
	}
	if !a.allow(2) {
		t.Fatal("users share limit")
	}
}

func TestMaskSearch(t *testing.T) {
	for _, tc := range []struct{ kind, value, want string }{{"phone", "9876543210", "****3210"}, {"email", "person@example.com", "***@example.com"}, {"master_id", "ABCD1234", "****1234"}} {
		if got := maskSearch(tc.kind, tc.value); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.kind, got, tc.want)
		}
	}
}

func TestFormatResults(t *testing.T) {
	empty := formatResults(searchResult{}, false)
	if len(empty) != 1 || !strings.Contains(empty[0], "No results found") {
		t.Fatalf("unexpected empty result: %v", empty)
	}
	var r searchResult
	r.Hits.Total.Value = 12
	for i := 0; i < 12; i++ {
		var hit struct {
			Source document `json:"_source"`
		}
		hit.Source.Name = strings.Repeat("漢", 200)
		hit.Source.Mobile = "9876543210"
		hit.Source.AltAddress = "Second address"
		r.Hits.Hits = append(r.Hits.Hits, hit)
	}
	pages := formatResults(r, false)
	if len(pages) < 1 {
		t.Fatal("missing results")
	}
	combined := strings.Join(pages, "\n")
	for _, v := range []string{"Result 1\n", "Alternate address: Second address", "1 credit after delivery"} {
		if !strings.Contains(combined, v) {
			t.Errorf("missing %q", v)
		}
	}
	for _, page := range pages {
		if len(page) > 4096 {
			t.Errorf("page too long: %d", len(page))
		}
	}
	if strings.Contains(combined, "Result 4\n") {
		t.Fatal("more than three results shown")
	}
	repeatPages := formatResults(r, true)
	if !strings.Contains(repeatPages[0], "no credit used") {
		t.Fatal("repeat label missing")
	}

}

type telegramResponseTransport struct {
	status int
	body   string
}

func (tr telegramResponseTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: tr.status, Body: io.NopCloser(strings.NewReader(tr.body)), Header: make(http.Header)}, nil
}
func TestSendRequiresTelegramConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		body      string
		wantError bool
	}{{"accepted", 200, `{"ok":true}`, false}, {"rejected", 200, `{"ok":false}`, true}, {"bad response", 200, `not-json`, true}, {"http error", 403, `{"ok":false}`, true}} {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{token: "test", client: &http.Client{Transport: telegramResponseTransport{tc.status, tc.body}}}
			err := a.send(context.Background(), 123, "test")
			if (err != nil) != tc.wantError {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSearchKey(t *testing.T) {
	a := &app{webhookSecret: "test-secret"}
	key := a.searchKey(1, "email", "Person@Example.com", "pan-india")
	if key != a.searchKey(1, "email", "person@example.com", "pan-india") {
		t.Fatal("normalization differs")
	}
	if key == a.searchKey(2, "email", "person@example.com", "pan-india") || key == a.searchKey(1, "email", "person@example.com", "delhi-ncr") {
		t.Fatal("search key not scoped to user and region")
	}
}

type retrySearchTransport struct {
	calls       int
	successBody string
}

func (tr *retrySearchTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	tr.calls++
	if tr.calls == 1 {
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tr.successBody)), Header: make(http.Header)}, nil
}
func TestSearchRetryAndMalformedResponse(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantErr    bool
	}{{"retry", `{"hits":{"total":{"value":0},"hits":[]}}`, false}, {"bad shape", `{"ok":true}`, true}} {
		t.Run(tc.name, func(t *testing.T) {
			tr := &retrySearchTransport{successBody: tc.body}
			a := &app{searchURL: "https://search.example", searchSecret: "secret", client: &http.Client{Transport: tr}}
			_, _, err := a.search(context.Background(), "email", "test@example.com", "pan-india")
			if (err != nil) != tc.wantErr || tr.calls != 2 {
				t.Fatalf("calls=%d err=%v", tr.calls, err)
			}
		})
	}
}

func TestHistoryEncryption(t *testing.T) {
	a := &app{historyKey: []byte("01234567890123456789012345678901")}
	value := "person@example.com"
	first, err := a.encryptHistory(101, 42, "email", value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.encryptHistory(101, 42, "email", value)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(first, []byte(value)) || bytes.Equal(first, second) {
		t.Fatal("plaintext or repeated ciphertext")
	}
	got, err := a.decryptHistory(101, 42, "email", first)
	if err != nil || got != value {
		t.Fatalf("round trip: %v", err)
	}
	if _, err = a.decryptHistory(102, 42, "email", first); err == nil {
		t.Fatal("wrong account accepted")
	}
	if _, err = a.decryptHistory(101, 43, "email", first); err == nil {
		t.Fatal("wrong update accepted")
	}
	first[len(first)-1] ^= 1
	if _, err = a.decryptHistory(101, 42, "email", first); err == nil {
		t.Fatal("tampering accepted")
	}
}

func TestPartialSearchResults(t *testing.T) {
	var r searchResult
	r.Partial = true
	if got := formatResults(r, false)[0]; !strings.Contains(got, "records checked") || !strings.Contains(got, "No credit") {
		t.Fatalf("bad partial empty message: %q", got)
	}
	r.Hits.Total.Value = 9
	var hit struct {
		Source document `json:"_source"`
	}
	hit.Source.ID = "master123"
	r.Hits.Hits = append(r.Hits.Hits, hit)
	got := formatResults(r, false)[0]
	if !strings.Contains(got, "More matches may exist") || strings.Contains(got, "9 match(es)") {
		t.Fatalf("misleading count: %q", got)
	}
}
