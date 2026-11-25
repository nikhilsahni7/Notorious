package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type paymentTestTransport struct{}

func (paymentTestTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
}

type historyCaptureTransport struct{ text string }

func (tr *historyCaptureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	tr.text = body.Text
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
}

type deliveryFailureTransport struct{}

func (deliveryFailureTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
}

type searchFlowTransport struct{}

func (searchFlowTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	body := `{"ok":true}`
	if r.URL.Host == "search.example" {
		var q struct {
			Value string `json:"value"`
		}
		_ = json.NewDecoder(r.Body).Decode(&q)
		body = `{"hits":{"total":{"value":1},"hits":[{"_source":{"name":"Example","id":"master123"}}]}}`
		if q.Value == "noresults@example.com" {
			body = `{"hits":{"total":{"value":0},"hits":[]}}`
		}
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}
func testSearchUpdate(t *testing.T, id, uid int64, text string) update {
	t.Helper()
	var u update
	b, _ := json.Marshal(map[string]interface{}{"update_id": id, "message": map[string]interface{}{"text": text, "chat": map[string]interface{}{"id": uid, "type": "private"}, "from": map[string]interface{}{"id": uid}}})
	if err := json.Unmarshal(b, &u); err != nil {
		t.Fatal(err)
	}
	return u
}
func TestReferenceAndMoney(t *testing.T) {
	for _, tc := range []struct {
		in, want string
		ok       bool
	}{{" ab123456 ", "AB123456", true}, {"abc-123", "ABC-123", false}, {"1", "1", false}} {
		got, ok := normalizeReference(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("%q: %q %v", tc.in, got, ok)
		}
	}
	if got := money(12345); got != "₹123.45" {
		t.Fatal(got)
	}
}

func TestPaymentWorkflowPostgres(t *testing.T) {
	url := os.Getenv("PAYMENTS_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set PAYMENTS_TEST_DATABASE_URL for PostgreSQL integration test")
	}
	ctx := context.Background()
	base, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	name := fmt.Sprintf("payment_test_%d", os.Getpid())
	_, err = base.Exec(ctx, "CREATE SCHEMA "+name)
	if err != nil {
		t.Fatal(err)
	}
	defer base.Exec(ctx, "DROP SCHEMA "+name+" CASCADE")
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = name
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(ctx, schema); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(ctx, `INSERT INTO bot_users(telegram_id) VALUES(101),(102); INSERT INTO plans(name,price_paise,daily_limit,days) VALUES('Plus',50000,100,30)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("UPI_ID", "merchant@upi")
	t.Setenv("UPI_QR_PATH", "")
	a := &app{db: db, token: "fake", historyKey: []byte("01234567890123456789012345678901"), admins: map[int64]bool{999: true}, client: &http.Client{Transport: paymentTestTransport{}, Timeout: time.Second}}
	if err = a.buy(ctx, 1, 101, []string{"/buy", "Plus"}); err != nil {
		t.Fatal(err)
	}
	if err = a.buy(ctx, 2, 102, []string{"/buy", "Plus"}); err != nil {
		t.Fatal(err)
	}
	if err = a.submitPayment(ctx, 101, []string{"/paid", "TXN123456"}); err != nil {
		t.Fatal(err)
	}
	if err = a.submitPayment(ctx, 102, []string{"/paid", "txn123456"}); err != nil {
		t.Fatal(err)
	}
	var st string
	if err = db.QueryRow(ctx, `SELECT status FROM payment_orders WHERE id=2`).Scan(&st); err != nil || st != "awaiting_reference" {
		t.Fatalf("duplicate reference accepted: %s %v", st, err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- a.paymentAdmin(ctx, int64(i+3), 999, "/approvepayment", []string{"/approvepayment", "1"})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	// Concurrent search attempts may reserve at most the available credit.
	if _, err = db.Exec(ctx, `UPDATE bot_users SET daily_limit=1 WHERE telegram_id=101`); err != nil {
		t.Fatal(err)
	}
	states := make(chan string, 2)
	var searchWG sync.WaitGroup
	for i := 0; i < 2; i++ {
		searchWG.Add(1)
		go func(i int) {
			defer searchWG.Done()
			state, region, e := a.reserve(ctx, int64(100+i), 101, "phone", "****3210")
			if e != nil {
				states <- "error:" + e.Error()
				return
			}
			if state == "reserved" && region != "pan-india" {
				states <- "wrong-region"
				return
			}
			states <- state
		}(i)
	}
	searchWG.Wait()
	close(states)
	reserved, other := 0, 0
	for state := range states {
		if state == "reserved" {
			reserved++
		} else if state == "busy" || state == "denied" {
			other++
		} else {
			t.Fatalf("unexpected reservation %s", state)
		}
	}
	if reserved != 1 || other != 1 {
		t.Fatalf("quota race reserved=%d other=%d", reserved, other)
	}
	if _, err = db.Exec(ctx, `UPDATE search_requests SET state='charged' WHERE telegram_id=101`); err != nil {
		t.Fatal(err)
	}
	state, _, err := a.reserve(ctx, 200, 101, "phone", "1234567899")
	if err != nil || state != "denied" {
		t.Fatalf("exceeded quota: %s %v", state, err)
	}
	repeatState, _, err := a.reserve(ctx, 201, 101, "phone", "****3210")
	if err != nil || repeatState != "repeat" {
		t.Fatalf("repeat at limit: %s %v", repeatState, err)
	}
	if _, err = db.Exec(ctx, `UPDATE search_requests SET state='released' WHERE update_id=201`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `UPDATE bot_users SET daily_limit=100 WHERE telegram_id=101`); err != nil {
		t.Fatal(err)
	}
	// A rejected order never grants access, and its reference remains unusable.
	if err = a.submitPayment(ctx, 102, []string{"/paid", "2", "TXN999999"}); err != nil {
		t.Fatal(err)
	}
	if err = a.paymentAdmin(ctx, 10, 999, "/rejectpayment", []string{"/rejectpayment", "2", "payment", "not", "found"}); err != nil {
		t.Fatal(err)
	}
	var rejected string
	var secondApproved bool
	if err = db.QueryRow(ctx, `SELECT status FROM payment_orders WHERE id=2`).Scan(&rejected); err != nil || rejected != "rejected" {
		t.Fatalf("rejection failed %s %v", rejected, err)
	}
	if err = db.QueryRow(ctx, `SELECT approved FROM bot_users WHERE telegram_id=102`).Scan(&secondApproved); err != nil || secondApproved {
		t.Fatal("rejected payment granted access", err)
	}
	// Identical unpaid orders select the newest without asking for an ID.
	if err = a.buy(ctx, 11, 102, []string{"/buy", "Plus"}); err != nil {
		t.Fatal(err)
	}
	if err = a.buy(ctx, 12, 102, []string{"/buy", "Plus"}); err != nil {
		t.Fatal(err)
	}
	if err = a.submitPayment(ctx, 102, []string{"/paid", "TXN777777"}); err != nil {
		t.Fatal(err)
	}
	var matched int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM payment_orders WHERE transaction_reference='TXN777777'`).Scan(&matched); err != nil || matched != 1 {
		t.Fatal("identical orders were not matched", err)
	}
	// Different plans are ambiguous, until the customer names their plan.
	if _, err = db.Exec(ctx, `INSERT INTO plans(name,price_paise,daily_limit,days) VALUES('Pro',90000,200,30)`); err != nil {
		t.Fatal(err)
	}
	if err = a.buy(ctx, 13, 102, []string{"/buy", "Pro"}); err != nil {
		t.Fatal(err)
	}
	if err = a.submitPayment(ctx, 102, []string{"/paid", "TXN888888"}); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT count(*) FROM payment_orders WHERE transaction_reference='TXN888888'`).Scan(&matched); err != nil || matched != 0 {
		t.Fatal("different plans were guessed", err)
	}
	if err = a.submitPayment(ctx, 102, []string{"/paid", "Pro", "TXN888888"}); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT count(*) FROM payment_orders WHERE transaction_reference='TXN888888' AND plan_name='Pro'`).Scan(&matched); err != nil || matched != 1 {
		t.Fatal("named plan was not selected", err)
	}
	// End-to-end handling reserves once, releases duplicate and zero-result searches.
	a.webhookSecret = "test-secret"
	a.searchURL = "https://search.example/search"
	a.client = &http.Client{Transport: searchFlowTransport{}, Timeout: time.Second}
	var before, after int
	countCharged := func() int {
		var n int
		if e := db.QueryRow(ctx, `SELECT count(*) FROM search_requests WHERE telegram_id=101 AND state='charged'`).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	before = countCharged()
	if err = a.handle(ctx, testSearchUpdate(t, 301, 101, "/search email person@example.com")); err != nil {
		t.Fatal(err)
	}
	if countCharged() != before {
		t.Fatal("charged before delivery")
	}
	if err = a.deliverSearch(ctx, 301, 101); err != nil {
		t.Fatal(err)
	}
	after = countCharged()
	if after != before+1 {
		t.Fatal("new search not charged")
	}
	if err = a.handle(ctx, testSearchUpdate(t, 302, 101, "/search email PERSON@example.com")); err != nil {
		t.Fatal(err)
	}
	if err = a.deliverSearch(ctx, 302, 101); err != nil {
		t.Fatal(err)
	}
	if countCharged() != after {
		t.Fatal("repeat search charged")
	}
	if err = a.handle(ctx, testSearchUpdate(t, 303, 101, "/search email noresults@example.com")); err != nil {
		t.Fatal(err)
	}
	if err = a.deliverSearch(ctx, 303, 101); err != nil {
		t.Fatal(err)
	}
	if countCharged() != after {
		t.Fatal("zero-result search charged")
	}
	// Full values are encrypted in storage, readable only using the configured key and audit identity.
	var cipherText []byte
	if err = db.QueryRow(ctx, `SELECT encrypted_value FROM search_requests WHERE update_id=301`).Scan(&cipherText); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cipherText), "person@example.com") {
		t.Fatal("plaintext search value stored")
	}
	plain, err := a.decryptHistory(101, 301, "email", cipherText)
	if err != nil || plain != "person@example.com" {
		t.Fatal("encrypted history did not round-trip", err)
	}
	if err = a.adminHistory(ctx, 999, []string{"/userhistory", "101"}); err != nil {
		t.Fatal(err)
	}
	var historyReads int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM admin_audit WHERE admin_id=999 AND action='/userhistory'`).Scan(&historyReads); err != nil || historyReads != 1 {
		t.Fatal("admin history audit missing", err)
	}
	if err = a.adminHistory(ctx, 102, []string{"/userhistory", "101"}); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT count(*) FROM admin_audit WHERE admin_id=102 AND action='/userhistory'`).Scan(&historyReads); err != nil || historyReads != 0 {
		t.Fatal("non-admin history access accepted", err)
	}
	if err = a.adminUsers(ctx, 999, []string{"/users"}); err != nil {
		t.Fatal(err)
	}
	if err = a.adminUser(ctx, 999, []string{"/user", "101"}); err != nil {
		t.Fatal(err)
	}
	if err = a.adminUsers(ctx, 102, []string{"/users"}); err != nil {
		t.Fatal(err)
	}
	if err = a.adminUser(ctx, 102, []string{"/user", "101"}); err != nil {
		t.Fatal(err)
	}
	var views int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM admin_audit WHERE admin_id=102 AND action IN ('/users','/user')`).Scan(&views); err != nil || views != 0 {
		t.Fatal("unauthorized profile access accepted", err)
	}
	// Quota reset keeps history and permits new searches without double-resetting on replay.
	if _, err = db.Exec(ctx, `UPDATE bot_users SET daily_limit=1 WHERE telegram_id=101`); err != nil {
		t.Fatal(err)
	}
	var historyBefore int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM search_requests WHERE telegram_id=101`).Scan(&historyBefore); err != nil {
		t.Fatal(err)
	}
	if err = a.resetQuota(ctx, 401, 999, []string{"/resetquota", "101"}); err != nil {
		t.Fatal(err)
	}
	rs, _, err := a.reserve(ctx, 402, 101, "phone", "9999999999")
	if err != nil || rs != "reserved" {
		t.Fatalf("reset did not restore allowance: %s %v", rs, err)
	}
	if _, err = db.Exec(ctx, `UPDATE search_requests SET state='charged' WHERE update_id=402`); err != nil {
		t.Fatal(err)
	}
	if err = a.resetQuota(ctx, 401, 999, []string{"/resetquota", "101"}); err != nil {
		t.Fatal(err)
	}
	rs, _, err = a.reserve(ctx, 403, 101, "phone", "9999999998")
	if err != nil || rs != "denied" {
		t.Fatalf("reset replay changed allowance: %s %v", rs, err)
	}
	var historyAfter int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM search_requests WHERE telegram_id=101`).Scan(&historyAfter); err != nil || historyAfter != historyBefore+1 {
		t.Fatal("reset changed history", err)
	}
	// Plan assignment applies the fixed scope.
	if _, err = db.Exec(ctx, `INSERT INTO bot_users(telegram_id,region) VALUES(103,'delhi-ncr')`); err != nil {
		t.Fatal(err)
	}
	if err = a.configure(ctx, 650, 999, "/assign", []string{"/assign", "103", "Plus"}); err != nil {
		t.Fatal(err)
	}
	var assignedRegion string
	var assigned bool
	if err = db.QueryRow(ctx, `SELECT region,approved FROM bot_users WHERE telegram_id=103`).Scan(&assignedRegion, &assigned); err != nil || !assigned || assignedRegion != "pan-india" {
		t.Fatal("fixed scope assignment failed", err)
	}
	// Customer history remains visible through both the command and menu after a reset.
	capture := &historyCaptureTransport{}
	a.client = &http.Client{Transport: capture, Timeout: time.Second}
	if err = a.handle(ctx, testSearchUpdate(t, 510, 101, "/history")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capture.text, "Search history") || !strings.Contains(capture.text, "****9999") || strings.Contains(capture.text, "No searches found") {
		t.Fatalf("history missing after reset: %q", capture.text)
	}
	if err = a.handle(ctx, testSearchUpdate(t, 511, 101, "History")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capture.text, "****9999") {
		t.Fatal("history button did not show records")
	}
	a.client = &http.Client{Transport: searchFlowTransport{}, Timeout: time.Second}
	if _, err = db.Exec(ctx, `UPDATE bot_users SET daily_limit=100 WHERE telegram_id=101`); err != nil {
		t.Fatal(err)
	}
	// Non-admin users cannot reset other accounts or use admin history.
	if err = a.handle(ctx, testSearchUpdate(t, 501, 102, "/resetquota 101")); err != nil {
		t.Fatal(err)
	}
	var unauthorized int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM admin_audit WHERE admin_id=102 AND action='/resetquota'`).Scan(&unauthorized); err != nil || unauthorized != 0 {
		t.Fatal("non-admin reset accepted", err)
	}
	// A blocked account cannot reserve a new search.
	if _, err = db.Exec(ctx, `UPDATE bot_users SET blocked=true WHERE telegram_id=101`); err != nil {
		t.Fatal(err)
	}
	blockedState, _, err := a.reserve(ctx, 502, 101, "phone", "1111111111")
	if err != nil || blockedState != "denied" {
		t.Fatalf("blocked user: %s %v", blockedState, err)
	}
	if _, err = db.Exec(ctx, `UPDATE bot_users SET blocked=false WHERE telegram_id=101`); err != nil {
		t.Fatal(err)
	}
	// Workload limits count searches regardless of credit outcome.
	if _, err = db.Exec(ctx, `INSERT INTO bot_users(telegram_id,approved,daily_limit,plan_expires_at) VALUES(104,true,100,now()+interval '30 days')`); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SEARCH_WORKLOAD_PER_MINUTE", "2")
	for _, id := range []int64{601, 602} {
		st, _, e := a.reserve(ctx, id, 104, "phone", "1111111111")
		if e != nil || st != "reserved" {
			t.Fatalf("reservation %d: %s %v", id, st, e)
		}
		if _, e = db.Exec(ctx, `UPDATE search_requests SET state='released',outcome='no_results' WHERE update_id=$1`, id); e != nil {
			t.Fatal(e)
		}
	}
	limited, _, err := a.reserve(ctx, 603, 104, "phone", "1111111111")
	if err != nil || limited != "rate_limited" {
		t.Fatalf("unmetered workload not limited: %s %v", limited, err)
	}
	t.Setenv("SEARCH_WORKLOAD_PER_MINUTE", "10")
	// A failed Telegram send keeps an encrypted retry and does not charge.
	if _, err = db.Exec(ctx, `INSERT INTO bot_users(telegram_id,approved,daily_limit,plan_expires_at) VALUES(105,true,100,now()+interval '30 days'),(106,true,100,now()+interval '30 days'),(107,true,100,now()+interval '30 days')`); err != nil {
		t.Fatal(err)
	}
	a.client = &http.Client{Transport: searchFlowTransport{}, Timeout: time.Second}
	if err = a.handle(ctx, testSearchUpdate(t, 701, 105, "/search email person@example.com")); err != nil {
		t.Fatal(err)
	}
	var encryptedPages []byte
	var deliveryState, requestState string
	var attempts int
	if err = db.QueryRow(ctx, `SELECT encrypted_pages FROM search_deliveries WHERE update_id=701`).Scan(&encryptedPages); err != nil || strings.Contains(string(encryptedPages), "Example") {
		t.Fatal("result was not encrypted", err)
	}
	a.client = &http.Client{Transport: deliveryFailureTransport{}, Timeout: time.Second}
	if err = a.deliverSearch(ctx, 701, 105); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT d.status,r.state,d.attempts FROM search_deliveries d JOIN search_requests r USING(update_id) WHERE d.update_id=701`).Scan(&deliveryState, &requestState, &attempts); err != nil || deliveryState != "pending" || requestState != "reserved" || attempts != 1 {
		t.Fatalf("failed delivery charged or lost: %s %s %d %v", deliveryState, requestState, attempts, err)
	}
	if _, err = db.Exec(ctx, `UPDATE search_deliveries SET next_retry_at=clock_timestamp() WHERE update_id=701`); err != nil {
		t.Fatal(err)
	}
	a.client = &http.Client{Transport: searchFlowTransport{}, Timeout: time.Second}
	if err = a.deliverSearch(ctx, 701, 105); err != nil {
		t.Fatal(err)
	}
	if err = a.deliverSearch(ctx, 701, 105); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT d.status,r.state,d.attempts FROM search_deliveries d JOIN search_requests r USING(update_id) WHERE d.update_id=701`).Scan(&deliveryState, &requestState, &attempts); err != nil || deliveryState != "delivered" || requestState != "charged" {
		t.Fatalf("retry failed: %s %s %v", deliveryState, requestState, err)
	}
	// Blocking after a result is queued cancels it before delivery and releases the credit.
	if err = a.handle(ctx, testSearchUpdate(t, 702, 106, "/search email person@example.com")); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `UPDATE bot_users SET blocked=true WHERE telegram_id=106`); err != nil {
		t.Fatal(err)
	}
	if err = a.deliverSearch(ctx, 702, 106); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT d.status,r.state FROM search_deliveries d JOIN search_requests r USING(update_id) WHERE d.update_id=702`).Scan(&deliveryState, &requestState); err != nil || deliveryState != "cancelled" || requestState != "released" {
		t.Fatal("blocked result was not cancelled", err)
	}
	// An abandoned reservation is released and a recovery notice is queued.
	if _, _, err = a.reserve(ctx, 703, 107, "phone", "9999999999"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `UPDATE search_requests SET created_at=clock_timestamp()-interval '3 minutes' WHERE update_id=703`); err != nil {
		t.Fatal(err)
	}
	if err = a.recoverAbandoned(ctx, 703, 107); err != nil {
		t.Fatal(err)
	}
	if err = a.deliverSearch(ctx, 703, 107); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT state FROM search_requests WHERE update_id=703`).Scan(&requestState); err != nil || requestState != "released" {
		t.Fatal("abandoned credit not released", err)
	}
	// Recovery processes queued results after startup.
	if _, err = db.Exec(ctx, `INSERT INTO bot_users(telegram_id,approved,daily_limit,plan_expires_at) VALUES(108,true,100,now()+interval '30 days')`); err != nil {
		t.Fatal(err)
	}
	if err = a.handle(ctx, testSearchUpdate(t, 704, 108, "/search email person@example.com")); err != nil {
		t.Fatal(err)
	}
	if _, err = a.recoveryStep(ctx); err != nil {
		t.Fatal("recovery step", err)
	}
	if err = db.QueryRow(ctx, `SELECT state FROM search_requests WHERE update_id=704`).Scan(&requestState); err != nil || requestState != "charged" {
		t.Fatal("recovery worker did not deliver", err)
	}
	var approved bool
	var region string
	var limit, audits int
	if err = db.QueryRow(ctx, `SELECT approved,region,daily_limit FROM bot_users WHERE telegram_id=101`).Scan(&approved, &region, &limit); err != nil {
		t.Fatal(err)
	}
	if !approved || region != "pan-india" || limit != 100 {
		t.Fatal("wrong entitlement")
	}
	if err = db.QueryRow(ctx, `SELECT count(*) FROM admin_audit WHERE action='/approvepayment'`).Scan(&audits); err != nil || audits != 1 {
		t.Fatal("approval not idempotent", strconv.Itoa(audits), err)
	}
}
