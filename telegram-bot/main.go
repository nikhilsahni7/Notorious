package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

type update struct {
	ID            int64 `json:"update_id"`
	CallbackQuery *struct {
		ID   string `json:"id"`
		Data string `json:"data"`
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
		Message *struct {
			Chat struct {
				ID   int64  `json:"id"`
				Type string `json:"type"`
			} `json:"chat"`
		} `json:"message"`
	} `json:"callback_query"`
	Message *struct {
		Text string `json:"text"`
		Chat struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
		} `json:"chat"`
		From struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
	} `json:"message"`
}
type app struct {
	db                                            *pgxpool.Pool
	token, webhookSecret, searchSecret, searchURL string
	admins                                        map[int64]bool
	client                                        *http.Client
	historyKey                                    []byte
	mu                                            sync.Mutex
	visitors                                      map[int64]visitor
	recoveryWake                                  chan struct{}
}
type visitor struct {
	at    time.Time
	count int
}

func (a *app) allow(user int64) bool {
	if a.admins[user] {
		return true
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.visitors == nil {
		a.visitors = make(map[int64]visitor)
	}
	now := time.Now()
	if len(a.visitors) > 10000 {
		for id, v := range a.visitors {
			if now.Sub(v.at) > time.Minute {
				delete(a.visitors, id)
			}
		}
		if len(a.visitors) > 10000 {
			return false
		}
	}
	v := a.visitors[user]
	if now.Sub(v.at) >= time.Minute {
		v = visitor{at: now}
	}
	v.count++
	a.visitors[user] = v
	return v.count <= 20
}

type document struct {
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
}
type searchResult struct {
	Partial bool `json:"partial"`
	Hits    struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source document `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

var phone = regexp.MustCompile(`^[0-9]{10,15}$`)
var master = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,64}$`)

func main() {
	token, secret, apiSecret, url, dburl := os.Getenv("BOT_TOKEN"), os.Getenv("WEBHOOK_SECRET"), os.Getenv("TELEGRAM_SEARCH_SECRET"), os.Getenv("SEARCH_API_URL"), os.Getenv("DATABASE_URL")
	historyKey, keyErr := base64.RawURLEncoding.DecodeString(os.Getenv("HISTORY_ENCRYPTION_KEY"))
	if keyErr != nil || len(historyKey) != 32 {
		log.Fatal("HISTORY_ENCRYPTION_KEY must be a stable 32-byte base64url key")
	}
	if token == "" || len(secret) < 32 || len(apiSecret) < 32 || url == "" || dburl == "" {
		log.Fatal("BOT_TOKEN, DATABASE_URL, SEARCH_API_URL and secrets of at least 32 characters are required")
	}
	admins := map[int64]bool{}
	for _, s := range strings.Split(os.Getenv("ADMIN_IDS"), ",") {
		if n, e := strconv.ParseInt(strings.TrimSpace(s), 10, 64); e == nil && n > 0 {
			admins[n] = true
		}
	}
	if len(admins) == 0 {
		log.Fatal("at least one ADMIN_IDS entry is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, dburl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(ctx, schema); err != nil {
		log.Fatal("database initialization: ", err)
	}
	a := &app{db: db, token: token, webhookSecret: secret, searchSecret: apiSecret, searchURL: url, admins: admins, historyKey: historyKey, client: &http.Client{Timeout: 15 * time.Second}, recoveryWake: make(chan struct{}, 1)}
	go a.recoveryLoop()
	a.wakeRecovery()
	if os.Getenv("BOT_MODE") == "polling" {
		log.Fatal(a.poll())
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("/telegram/webhook", a.webhook)
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8091"
	}
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 40 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("bot listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}

func (a *app) webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Telegram-Bot-Api-Secret-Token")), []byte(a.webhookSecret)) != 1 {
		http.Error(w, "unauthorized", 401)
		return
	}
	var u update
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&u); err != nil {
		http.Error(w, "invalid", 400)
		return
	}
	if u.CallbackQuery != nil {
		q := u.CallbackQuery
		if q.Message == nil || q.Message.Chat.Type != "private" || q.Message.Chat.ID != q.From.ID || !a.allow(q.From.ID) {
			w.WriteHeader(200)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		if err := a.handleCallback(ctx, q.ID, q.From.ID, q.Data); err != nil {
			log.Printf("report callback failed: %v", err)
			http.Error(w, "temporary error", 503)
			return
		}
		w.WriteHeader(200)
		return
	}
	if u.Message == nil || u.Message.Chat.Type != "private" || u.Message.From.ID != u.Message.Chat.ID {
		w.WriteHeader(200)
		return
	}
	if !a.allow(u.Message.From.ID) {
		w.WriteHeader(200)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 32*time.Second)
	defer cancel()
	if err := a.handle(ctx, u); err != nil {
		log.Printf("update processing failed: %v", err)
		http.Error(w, "temporary error", 503)
		return
	}
	w.WriteHeader(200)
}
func (a *app) send(ctx context.Context, chat int64, msg string) error {
	return a.sendMessage(ctx, chat, msg, false)
}
func (a *app) sendMessage(ctx context.Context, chat int64, msg string, menu bool) error {
	payload := map[string]interface{}{"chat_id": chat, "text": premiumHTML(msg), "parse_mode": "HTML", "protect_content": true}
	if menu {
		payload["reply_markup"] = map[string]interface{}{"keyboard": [][]map[string]string{{{"text": "Search phone"}, {"text": "Search email"}}, {{"text": "Search document number"}, {"text": "My account"}}, {{"text": "Plans"}, {"text": "Payment status"}}, {{"text": "History"}, {"text": "Help"}}}, "resize_keyboard": true}
	}
	return a.sendPayload(ctx, payload)
}

// Format only trusted structural labels; escape all content supplied by users or search records.
func premiumHTML(text string) string {
	lines := strings.Split(text, "\n")
	labels := map[string]bool{"Name": true, "Father's name": true, "Phone": true, "Alternate": true, "Email": true, "Document number": true, "Address": true, "Alternate address": true}
	for i, line := range lines {
		if i == 0 || (strings.HasPrefix(line, "Result ") && len(strings.Fields(line)) == 2) || strings.Contains(line, " remaining of ") {
			lines[i] = "<b>" + html.EscapeString(line) + "</b>"
		} else if label, value, ok := strings.Cut(line, ": "); ok && labels[label] {
			lines[i] = "<b>" + html.EscapeString(label) + ":</b> " + html.EscapeString(value)
		} else {
			lines[i] = html.EscapeString(line)
		}
	}
	return strings.Join(lines, "\n")
}
func (a *app) sendPayload(ctx context.Context, payload map[string]interface{}) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.telegram.org/bot"+a.token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return errors.New("telegram request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("telegram response %d", resp.StatusCode)
	}
	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result); err != nil || !result.OK {
		return errors.New("telegram did not confirm message delivery")
	}
	return nil
}
func (a *app) handle(ctx context.Context, u update) error {
	m := u.Message
	uid := m.From.ID
	_, err := a.db.Exec(ctx, `INSERT INTO bot_users (telegram_id,username) VALUES ($1,$2) ON CONFLICT (telegram_id) DO UPDATE SET username=EXCLUDED.username`, uid, m.From.Username)
	if err != nil {
		return err
	}
	parts := strings.Fields(m.Text)
	selected := map[string]string{"Search phone": "phone", "Search email": "email", "Search document number": "id", "Search master ID": "id"}
	if kind, ok := selected[m.Text]; ok {
		if _, err := a.db.Exec(ctx, `UPDATE bot_users SET pending_search=$2,pending_payment=false,pending_ref_order=NULL,pending_ref_value=NULL WHERE telegram_id=$1`, uid, kind); err != nil {
			return err
		}
		prompt := map[string]string{"phone": "Phone search\n\nEnter a phone number using 10–15 digits.\nExample: 9876543210", "email": "Email search\n\nEnter the full email address.\nExample: name@example.com", "id": "Document number search\n\nEnter the complete document number."}[kind]
		return a.send(ctx, uid, prompt+"\n\nNo-result searches are free. Send /cancel to go back.")
	}
	if strings.HasPrefix(m.Text, "Buy ") {
		parts = []string{"/buy", strings.TrimSpace(strings.TrimPrefix(m.Text, "Buy "))}
	}
	if m.Text == "Submit payment" {
		parts = []string{"/paid"}
	}
	if command, ok := map[string]string{"My account": "/account", "Plans": "/plans", "Payment status": "/mypayments", "History": "/history", "Help": "/help", "Home": "/start"}[m.Text]; ok {
		parts = []string{command}
	}
	if len(parts) == 1 && !strings.HasPrefix(parts[0], "/") {
		var pending string
		var payment bool
		var editOrder *int64
		if err := a.db.QueryRow(ctx, `SELECT pending_search,pending_payment,pending_ref_order FROM bot_users WHERE telegram_id=$1`, uid).Scan(&pending, &payment, &editOrder); err != nil {
			return err
		}
		if editOrder != nil {
			return a.stageReferenceEdit(ctx, uid, *editOrder, parts[0])
		} else if payment {
			parts = []string{"/paid", parts[0]}
		} else if pending != "" {
			parts = []string{"/search", pending, parts[0]}
		} else if kind := detectSearchType(parts[0]); kind != "" {
			parts = []string{"/search", kind, parts[0]}
		}
	}
	if len(parts) == 0 {
		return nil
	}
	cmd := strings.Split(strings.ToLower(parts[0]), "@")[0]
	if a.admins[uid] && cmd == "/admin" {
		return a.send(ctx, uid, "Admin commands:\n/users [PAGE] — list users\n/user USER_ID — account profile\n/setplan NAME PRICE_INR DAILY_LIMIT DAYS\n/assign USER_ID PLAN\n/block USER_ID\n/check USER_ID\n/resetquota USER_ID — reset today\n/userhistory USER_ID [PAGE] — search history\n/pending — payment reviews\n/reviewpayment ORDER_ID — lock reference\n/orderplan ORDER_ID PLAN — correct a pending plan\n/approvepayment ORDER_ID\n/upgradepayment ORDER_ID — confirm plan change\n/rejectpayment ORDER_ID REASON\nVerify receipt in your bank before approval.")
	}
	if a.admins[uid] && cmd == "/users" {
		return a.adminUsers(ctx, uid, parts)
	}
	if a.admins[uid] && cmd == "/user" {
		return a.adminUser(ctx, uid, parts)
	}
	if a.admins[uid] && cmd == "/resetquota" {
		return a.resetQuota(ctx, u.ID, uid, parts)
	}
	if a.admins[uid] && cmd == "/userhistory" {
		return a.adminHistory(ctx, uid, parts)
	}
	if a.admins[uid] && (cmd == "/setplan" || cmd == "/assign") {
		return a.configure(ctx, u.ID, uid, cmd, parts)
	}
	if a.admins[uid] && (cmd == "/pending" || cmd == "/reviewpayment" || cmd == "/orderplan" || cmd == "/approvepayment" || cmd == "/upgradepayment" || cmd == "/rejectpayment") {
		return a.paymentAdmin(ctx, u.ID, uid, cmd, parts)
	}
	if a.admins[uid] && (cmd == "/block" || cmd == "/check") {
		return a.admin(ctx, u.ID, uid, cmd, parts)
	}
	if cmd == "/start" || cmd == "/help" || cmd == "/plans" || cmd == "/account" || cmd == "/balance" || cmd == "/history" || cmd == "/mypayments" || cmd == "/support" {
		if _, err := a.db.Exec(ctx, `UPDATE bot_users SET pending_search='',pending_payment=false,pending_ref_order=NULL,pending_ref_value=NULL WHERE telegram_id=$1`, uid); err != nil {
			return err
		}
	}
	switch cmd {
	case "/start":
		if a.admins[uid] {
			return a.sendMessage(ctx, uid, "Welcome to SHAKIL INTEL X\n\nAdministrator access is active. Use the search buttons below or /help for commands. Your searches do not use credits.\n\nUse /admin to manage users, plans and payments.", true)
		}
		return a.sendMessage(ctx, uid, "Welcome to SHAKIL INTEL X\n\nChoose a search below and enter a phone number, email or document number. We’ll show up to 5 linked records.\n\nYour searches\n• Searches with results use one search, including repeats.\n• No-result searches are free.\n• Your allowance resets at midnight IST.\n\nYou get 3 free trial searches with results. Open Plans for paid access. Use My account to check your balance, or History to see recent searches.\n\nUse /cancel to leave any step.", true)
	case "/help":
		return a.sendMessage(ctx, uid, helpText(a.admins[uid]), true)

	case "/plans":
		return a.listPlans(ctx, uid)
	case "/buy":
		return a.buy(ctx, u.ID, uid, parts)
	case "/paid":
		if len(parts) == 1 {
			if _, err := a.db.Exec(ctx, `UPDATE bot_users SET pending_payment=true,pending_search='',pending_ref_order=NULL,pending_ref_value=NULL WHERE telegram_id=$1`, uid); err != nil {
				return err
			}
			return a.send(ctx, uid, "Submit payment\n\nEnter the transaction reference from your UPI app.\n\nSend /cancel to go back.")
		}
		if len(parts) == 2 {
			if _, ok := normalizeReference(parts[1]); !ok {
				return a.send(ctx, uid, "That reference doesn’t look right. Enter 6–64 letters or numbers, or send /cancel.")
			}
		}
		return a.submitPayment(ctx, uid, parts)
	case "/cancel":
		if _, err := a.db.Exec(ctx, `UPDATE bot_users SET pending_payment=false,pending_search='',pending_ref_order=NULL,pending_ref_value=NULL WHERE telegram_id=$1`, uid); err != nil {
			return err
		}
		return a.sendMessage(ctx, uid, "Cancelled\n\nChoose an option below to continue.", true)
	case "/mypayments":
		return a.myPayments(ctx, uid)
	case "/support":
		return a.send(ctx, uid, "Need help?\n\nContact the administrator who gave you access to SHAKIL INTEL X. For payment issues, include your transaction reference.")
	case "/balance", "/account":
		return a.account(ctx, uid)
	case "/history":
		return a.history(ctx, uid, parts)

	case "/search":
		if len(parts) != 3 {
			return a.send(ctx, uid, "Use /search phone NUMBER, /search email EMAIL or /search id DOCUMENTNUMBER.")
		}
		kind := strings.ToLower(parts[1])
		value := parts[2]
		if kind == "id" {
			kind = "master_id"
		}
		valid := false
		switch kind {
		case "phone", "auto":
			valid = phone.MatchString(value)
		case "master_id":
			valid = master.MatchString(value)
		case "email":
			v, e := mail.ParseAddress(value)
			valid = e == nil && v.Address == value && len(value) <= 254
		}
		if !valid {
			return a.send(ctx, uid, "That search doesn’t look right. Check the value and try again, or open Help for examples.")
		}
		if _, err := a.db.Exec(ctx, `UPDATE bot_users SET pending_search='' WHERE telegram_id=$1`, uid); err != nil {
			return err
		}
		state, region, e := a.reserve(ctx, u.ID, uid, kind, value)
		if e != nil {
			return e
		}
		switch state {
		case "duplicate":
			return nil
		case "denied":
			return a.send(ctx, uid, "Search unavailable\n\nYour free searches or today’s plan allowance may be used. Open My account to check your status, or Plans to review options.")
		case "busy":
			return a.send(ctx, uid, "A search is already in progress. Wait for it to finish, then try again.")
		case "rate_limited":
			return a.send(ctx, uid, "Search limit reached for now. Please wait before trying again. Your daily credits have not changed.")
		}
		a.wakeRecovery()
		result, code, e := a.search(ctx, kind, value, region, a.admins[uid])
		if e != nil || code/100 != 2 {
			if _, err = a.db.Exec(ctx, `UPDATE search_requests SET state='released',outcome='failed' WHERE update_id=$1 AND state='reserved'`, u.ID); err != nil {
				return err
			}
			a.wakeRecovery()
			return a.send(ctx, uid, "Search temporarily unavailable\n\nNo credit was used. Try again in a moment.")
		}
		queued, err := a.queueSearchDelivery(ctx, uid, u.ID, state, result)
		if err != nil {
			return err
		}
		a.wakeRecovery()
		if !queued {
			return a.send(ctx, uid, "Your account is no longer active. No credit was used.")
		}
		return nil
	default:
		return a.send(ctx, uid, "Use /help to see available commands.")
	}
}

func detectSearchType(value string) string {
	if phone.MatchString(value) {
		return "auto"
	}
	if addr, err := mail.ParseAddress(value); err == nil && addr.Address == value && len(value) <= 254 {
		return "email"
	}
	if master.MatchString(value) {
		return "id"
	}
	return ""
}

func helpText(admin bool) string {
	text := "SHAKIL INTEL X · Help\n\nSearch using the buttons or commands:\n/search phone 9876543210\n/search email name@example.com\n/search id DOCUMENTNUMBER\n\nUp to 5 linked records are shown. Each search with results uses one credit, including repeats. No-result searches are free. Paid credits reset at midnight IST.\n\nPlans & payments\n/plans — available plans\n/buy PLAN — choose a plan\n/paid REFERENCE — submit your UPI reference\n/mypayments — payment status\n\nAccount\n/account — plan and remaining searches\n/history — recent searches\n/history 2 — next history page\n/cancel — leave the current step\n/support — get help"
	if admin {
		text += "\n\nAdministrator\n/admin — users, plans, payment reviews and other admin commands\nYour administrator searches do not use credits."
	}
	return text
}
func (a *app) reserve(ctx context.Context, updateID, uid int64, kind, value string) (state, region string, err error) {
	tx, e := a.db.Begin(ctx)
	if e != nil {
		return "", "", e
	}
	defer tx.Rollback(ctx)
	var approved, blocked bool
	var limit int
	e = tx.QueryRow(ctx, `SELECT (approved AND COALESCE(plan_expires_at > now(),false)),blocked,daily_limit,region FROM bot_users WHERE telegram_id=$1 FOR UPDATE`, uid).Scan(&approved, &blocked, &limit, &region)
	if e != nil {
		return "", "", e
	}
	var exists bool
	e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM search_requests WHERE update_id=$1)`, updateID).Scan(&exists)
	if e != nil {
		return "", "", e
	}
	if exists {
		return "duplicate", "", tx.Commit(ctx)
	}
	isAdmin := a.admins[uid]
	if blocked {
		return "denied", "", tx.Commit(ctx)
	}
	if !isAdmin {
		var busy bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM search_requests WHERE telegram_id=$1 AND state='reserved')`, uid).Scan(&busy)
		if e != nil {
			return "", "", e
		}
		if busy {
			return "busy", "", tx.Commit(ctx)
		}
		// Count all accepted searches, including repeats, empty results and failures.
		// This workload limit is separate from paid credits and survives restarts.
		var lastMinute, lastHour int
		e = tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE created_at > clock_timestamp()-interval '1 minute'), count(*) FROM search_requests WHERE telegram_id=$1 AND created_at > clock_timestamp()-interval '1 hour'`, uid).Scan(&lastMinute, &lastHour)
		if e != nil {
			return "", "", e
		}
		minuteLimit, hourLimit := searchWorkloadLimits()
		if lastMinute >= minuteLimit || lastHour >= hourLimit {
			return "rate_limited", "", tx.Commit(ctx)
		}
	}
	key := a.searchKey(uid, kind, value, region)
	billing := "paid"
	if isAdmin {
		billing = "admin"
	} else if !approved {
		billing = "trial"
	}
	var used int
	if billing == "trial" {
		e = tx.QueryRow(ctx, `SELECT count(*) FROM search_requests WHERE telegram_id=$1 AND billing_kind='trial' AND state IN ('reserved','charged','uncertain')`, uid).Scan(&used)
		limit = 3
	} else {
		e = tx.QueryRow(ctx, `SELECT count(*) FROM search_requests r WHERE telegram_id=$1 AND billing_kind='paid' AND quota_day=(clock_timestamp() AT TIME ZONE 'Asia/Kolkata')::date AND state IN ('reserved','charged','uncertain') AND created_at > COALESCE((SELECT reset_at FROM quota_resets WHERE telegram_id=$1 AND quota_day=r.quota_day),'-infinity'::timestamptz)`, uid).Scan(&used)
	}
	if e != nil {
		return "", "", e
	}
	if !isAdmin && used >= limit {
		return "denied", "", tx.Commit(ctx)
	}
	encrypted, e := a.encryptHistory(uid, updateID, kind, value)
	if e != nil {
		return "", "", e
	}
	_, e = tx.Exec(ctx, `INSERT INTO search_requests(update_id,telegram_id,quota_day,state,search_type,masked_value,search_key,encrypted_value,billing_kind,created_at) VALUES($1,$2,(clock_timestamp() AT TIME ZONE 'Asia/Kolkata')::date,'reserved',$3,$4,$5,$6,$7,clock_timestamp())`, updateID, uid, kind, maskSearch(kind, value), key, encrypted, billing)
	if e != nil {
		return "", "", e
	}
	if isAdmin {
		return "admin", region, tx.Commit(ctx)
	}
	if billing == "trial" {
		return "trial", region, tx.Commit(ctx)
	}
	return "reserved", region, tx.Commit(ctx)
}
func searchWorkloadLimits() (int, int) {
	parse := func(name string, fallback int) int {
		n, err := strconv.Atoi(os.Getenv(name))
		if err != nil || n < 1 || n > 10000 {
			return fallback
		}
		return n
	}
	return parse("SEARCH_WORKLOAD_PER_MINUTE", 10), parse("SEARCH_WORKLOAD_PER_HOUR", 500)
}
func (a *app) searchKey(uid int64, kind, value, region string) string {
	m := hmac.New(sha256.New, []byte(a.webhookSecret))
	fmt.Fprintf(m, "search-v1\x00%d\x00%s\x00%s\x00%s", uid, kind, strings.ToLower(strings.TrimSpace(value)), region)
	return hex.EncodeToString(m.Sum(nil))
}
func (a *app) search(ctx context.Context, kind, value, region string, admin bool) (searchResult, int, error) {
	var out searchResult
	body, _ := json.Marshal(map[string]string{"type": kind, "value": value, "region": region})
	url := a.searchURL
	if admin {
		url = strings.TrimSuffix(url, "/search") + "/admin/search"
	}
	for attempt := 0; attempt < 2; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, 14*time.Second)
		req, err := http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewReader(body))
		if err != nil {
			cancel()
			return out, 0, err
		}
		req.Header.Set("Authorization", "Bearer "+a.searchSecret)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Shakil-Intel-Service/1.0")
		resp, err := a.client.Do(req)
		if err != nil {
			cancel()
			if attempt == 0 && ctx.Err() == nil {
				continue
			}
			return out, 0, err
		}
		code := resp.StatusCode
		if code/100 == 2 {
			var envelope json.RawMessage
			err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&envelope)
			resp.Body.Close()
			cancel()
			if err != nil {
				return out, code, err
			}
			var shape struct {
				Hits *struct {
					Total *struct {
						Value *int `json:"value"`
					} `json:"total"`
				} `json:"hits"`
			}
			if json.Unmarshal(envelope, &shape) != nil || shape.Hits == nil || shape.Hits.Total == nil || shape.Hits.Total.Value == nil || *shape.Hits.Total.Value < 0 {
				return out, code, errors.New("invalid search response")
			}
			if err = json.Unmarshal(envelope, &out); err != nil {
				return out, code, err
			}
			if out.Hits.Total.Value > 0 && len(out.Hits.Hits) == 0 {
				return out, code, errors.New("incomplete search response")
			}
			return out, code, nil
		}
		resp.Body.Close()
		cancel()
		log.Printf("search integration returned HTTP %d", code)
		if attempt == 0 && (code == 502 || code == 503 || code == 504) {
			select {
			case <-time.After(300 * time.Millisecond):
				continue
			case <-ctx.Done():
				return out, 0, ctx.Err()
			}
		}
		return out, code, nil
	}
	return out, 0, errors.New("search unavailable")
}

// formatResults separates records into Telegram-sized messages without interpreting
// data as markup. Values are bounded so even an unusually large field is safe.
func formatResults(r searchResult, repeat bool) []string {
	if r.Hits.Total.Value == 0 {
		if r.Partial {
			return []string{"No linked results found in the records checked. More matches may exist.\n\nNo credit was used."}
		}
		return []string{"No results found\n\nCheck the value and try again. No credit was used."}
	}
	if len(r.Hits.Hits) == 0 {
		return []string{"Results are temporarily unavailable. Contact support about this search."}
	}
	const maxBody = 3200
	var bodies []string
	var b strings.Builder
	for i, hit := range r.Hits.Hits {
		if i >= 5 {
			break
		}
		d := hit.Source
		var entry strings.Builder
		fmt.Fprintf(&entry, "Result %d\n", i+1)
		fields := []struct{ label, value string }{{"Name", d.Name}, {"Father's name", d.Fname}, {"Phone", d.Mobile}, {"Alternate", d.Alt}, {"Email", d.Email}, {"Document number", d.ID}, {"Address", d.Address}, {"Alternate address", d.AltAddress}}
		for _, f := range fields {
			if v := cleanResultField(f.value); v != "" {
				fmt.Fprintf(&entry, "%s: %s\n", f.label, v)
			}
		}
		entry.WriteString("\n")
		if b.Len() > 0 && b.Len()+entry.Len() > maxBody {
			bodies = append(bodies, b.String())
			b.Reset()
		}
		b.WriteString(entry.String())
	}
	if b.Len() > 0 {
		bodies = append(bodies, b.String())
	}
	pages := make([]string, len(bodies))
	for i, body := range bodies {
		charge := "1 credit after delivery"
		if repeat {
			charge = "Repeat search · no credit used"
		}
		header := fmt.Sprintf("Search results\n\nUp to 5 records · %s\n", charge)
		if r.Partial {
			header += "More matches may exist.\n"
		}
		if len(bodies) > 1 {
			header += fmt.Sprintf("Page %d of %d\n", i+1, len(bodies))
		}
		pages[i] = header + "\n" + body
	}
	return pages
}
func cleanResultField(value string) string {
	value = strings.Join(strings.Fields(strings.ToValidUTF8(value, "")), " ")
	var b strings.Builder
	for _, ch := range value {
		if ch < 32 || ch == 127 {
			continue
		}
		if b.Len()+len(string(ch)) > 300 {
			b.WriteString("…")
			break
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func (a *app) admin(ctx context.Context, updateID, actor int64, cmd string, p []string) error {
	if len(p) < 2 {
		return a.send(ctx, actor, "Provide a numeric Telegram user ID.")
	}
	target, e := strconv.ParseInt(p[1], 10, 64)
	if e != nil || target <= 0 {
		return a.send(ctx, actor, "Invalid user ID.")
	}
	if cmd == "/check" {
		var approved, blocked bool
		var limit int
		var plan string
		e = a.db.QueryRow(ctx, `SELECT approved,blocked,daily_limit,plan_name FROM bot_users WHERE telegram_id=$1`, target).Scan(&approved, &blocked, &limit, &plan)
		if errors.Is(e, pgx.ErrNoRows) {
			return a.send(ctx, actor, "User must start the bot first.")
		}
		if e != nil {
			return e
		}
		return a.send(ctx, actor, fmt.Sprintf("User %d: approved=%t blocked=%t plan=%s daily limit=%d", target, approved, blocked, plan, limit))
	}
	tx, e := a.db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	insert, e := tx.Exec(ctx, `INSERT INTO admin_updates(update_id) VALUES($1) ON CONFLICT DO NOTHING`, updateID)
	if e != nil {
		return e
	}
	if insert.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	var tag int64
	switch cmd {
	case "/approve":
		t, err := tx.Exec(ctx, `UPDATE bot_users SET approved=true,blocked=false WHERE telegram_id=$1`, target)
		e = err
		tag = t.RowsAffected()
	case "/block":
		t, err := tx.Exec(ctx, `UPDATE bot_users SET blocked=true WHERE telegram_id=$1`, target)
		e = err
		tag = t.RowsAffected()
	case "/grant":
		if len(p) != 5 {
			return a.send(ctx, actor, "Use /grant USER_ID DAILY_LIMIT DAYS PLAN_NAME")
		}
		limit, x := strconv.Atoi(p[2])
		days, y := strconv.Atoi(p[3])
		if x != nil || y != nil || limit < 1 || limit > 10000 || days < 1 || days > 3650 || len(p[4]) > 40 {
			return a.send(ctx, actor, "Invalid plan parameters.")
		}
		t, err := tx.Exec(ctx, `UPDATE bot_users SET approved=true,blocked=false,daily_limit=$2,plan_name=$3,plan_expires_at=now()+($4 * interval '1 day') WHERE telegram_id=$1`, target, limit, p[4], days)
		e = err
		tag = t.RowsAffected()
	}
	if e != nil {
		return e
	}
	if tag == 0 {
		return a.send(ctx, actor, "User must start the bot first.")
	}
	_, e = tx.Exec(ctx, `INSERT INTO admin_audit(admin_id,target_id,action,detail) VALUES($1,$2,$3,$4)`, actor, target, cmd, strings.Join(p[2:], " "))
	if e != nil {
		return e
	}
	if e = tx.Commit(ctx); e != nil {
		return e
	}
	return a.send(ctx, actor, "Action completed.")
}

func (a *app) listPlans(ctx context.Context, user int64) error {
	rows, err := a.db.Query(ctx, `SELECT name,price_paise,daily_limit,days FROM plans WHERE active ORDER BY price_paise,name LIMIT 30`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var b strings.Builder
	b.WriteString("<b>Choose your plan</b>\n\nPhone, email and document number searches are included.\n\n")
	count := 0
	keyboard := [][]map[string]string{}
	for rows.Next() {
		var name string
		var price int64
		var limit, days int
		if err := rows.Scan(&name, &price, &limit, &days); err != nil {
			return err
		}
		fmt.Fprintf(&b, "<b>%s</b> · ₹%d.%02d · %d/day · %d days\n", html.EscapeString(name), price/100, price%100, limit, days)
		if count%2 == 0 {
			keyboard = append(keyboard, []map[string]string{})
		}
		keyboard[len(keyboard)-1] = append(keyboard[len(keyboard)-1], map[string]string{"text": "Buy " + name})
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count == 0 {
		b.WriteString("Contact your administrator for available plans.\n")
	}
	b.WriteString("\nSelect a plan below. Your payment will be reviewed by an admin.")
	keyboard = append(keyboard, []map[string]string{{"text": "Payment status"}, {"text": "Home"}})
	return a.sendPayload(ctx, map[string]interface{}{"chat_id": user, "text": b.String(), "parse_mode": "HTML", "protect_content": true, "reply_markup": map[string]interface{}{"keyboard": keyboard, "resize_keyboard": true}})
}
func (a *app) configure(ctx context.Context, updateID, actor int64, cmd string, p []string) error {
	if (cmd == "/setplan" && len(p) != 5) || (cmd == "/assign" && len(p) != 3) {
		return a.send(ctx, actor, "Use /setplan NAME PRICE_INR DAILY_LIMIT DAYS or /assign USER_ID PLAN")
	}
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var target int64
	if cmd == "/setplan" {
		name := p[1]
		price, e1 := strconv.ParseInt(p[2], 10, 32)
		limit, e2 := strconv.Atoi(p[3])
		days, e3 := strconv.Atoi(p[4])
		if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`).MatchString(name) || e1 != nil || e2 != nil || e3 != nil || price < 0 || limit < 1 || limit > 100000 || days < 1 || days > 3650 {
			return a.send(ctx, actor, "Invalid plan. Use a one-word name, whole INR price, daily limit and duration in days.")
		}
		tag, e := tx.Exec(ctx, `INSERT INTO admin_updates(update_id) VALUES($1) ON CONFLICT DO NOTHING`, updateID)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return tx.Commit(ctx)
		}
		_, err = tx.Exec(ctx, `INSERT INTO plans(name,price_paise,daily_limit,days) VALUES($1,$2,$3,$4) ON CONFLICT(name) DO UPDATE SET price_paise=excluded.price_paise,daily_limit=excluded.daily_limit,days=excluded.days,active=true,updated_at=now()`, name, price*100, limit, days)
	} else {
		target, err = strconv.ParseInt(p[1], 10, 64)
		if err != nil || target <= 0 {
			return a.send(ctx, actor, "Invalid user ID.")
		}
		tag, e := tx.Exec(ctx, `INSERT INTO admin_updates(update_id) VALUES($1) ON CONFLICT DO NOTHING`, updateID)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return tx.Commit(ctx)
		}
		res, e := tx.Exec(ctx, `UPDATE bot_users SET approved=true,blocked=false,region=$3,daily_limit=p.daily_limit,plan_name=p.name,plan_expires_at=now()+(p.days*interval '1 day') FROM plans p WHERE telegram_id=$1 AND p.name=$2 AND p.active`, target, p[2], "pan-india")
		err = e
		if err == nil && res.RowsAffected() == 0 {
			return a.send(ctx, actor, "User must start the bot first, and the plan must exist.")
		}
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO admin_audit(admin_id,target_id,action,detail) VALUES($1,$2,$3,$4)`, actor, target, cmd, strings.Join(p[1:], " "))
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return a.send(ctx, actor, "Saved.")
}

// poll is intended for local development and a single bot instance. A webhook
// must be removed explicitly before Telegram will deliver getUpdates responses.
func (a *app) poll() error {
	client := &http.Client{Timeout: 35 * time.Second}
	var offset int64
	for {
		payload, _ := json.Marshal(map[string]interface{}{"offset": offset, "timeout": 20, "allowed_updates": []string{"message"}})
		req, err := http.NewRequest("POST", "https://api.telegram.org/bot"+a.token+"/getUpdates", bytes.NewReader(payload))
		if err != nil {
			return errors.New("invalid telegram request")
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			log.Print("Telegram polling unavailable; retrying")
			time.Sleep(3 * time.Second)
			continue
		}
		var data struct {
			OK     bool     `json:"ok"`
			Result []update `json:"result"`
		}
		err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&data)
		resp.Body.Close()
		if err != nil || !data.OK || resp.StatusCode/100 != 2 {
			log.Print("Telegram polling failed; check whether a webhook is still configured")
			time.Sleep(5 * time.Second)
			continue
		}
		for _, u := range data.Result {
			if q := u.CallbackQuery; q != nil {
				if q.Message != nil && q.Message.Chat.Type == "private" && q.Message.Chat.ID == q.From.ID && a.allow(q.From.ID) {
					ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
					err = a.handleCallback(ctx, q.ID, q.From.ID, q.Data)
					cancel()
					if err != nil {
						break
					}
				}
				offset = u.ID + 1
				continue
			}
			if u.Message == nil || u.Message.Chat.Type != "private" || u.Message.From.ID != u.Message.Chat.ID || !a.allow(u.Message.From.ID) {
				offset = u.ID + 1
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 32*time.Second)
			err = a.handle(ctx, u)
			cancel()
			if err != nil {
				log.Print("Update failed; retrying")
				time.Sleep(3 * time.Second)
				break
			}
			offset = u.ID + 1
		}
	}
}

// Only a partial identifier is kept; search results and full queries are not stored.
func maskSearch(kind, value string) string {
	if kind == "email" {
		parts := strings.SplitN(value, "@", 2)
		if len(parts) == 2 {
			return "***@" + parts[1]
		}
	}
	r := []rune(value)
	if len(r) <= 4 {
		return "****"
	}
	return "****" + string(r[len(r)-4:])
}
func (a *app) account(ctx context.Context, uid int64) error {
	var active, blocked bool
	var limit, used int
	var name, region, expires string
	err := a.db.QueryRow(ctx, `SELECT (approved AND COALESCE(plan_expires_at>now(),false)),blocked,daily_limit,plan_name,region,COALESCE(to_char(plan_expires_at AT TIME ZONE 'Asia/Kolkata','DD Mon YYYY HH24:MI') || ' IST','Not assigned'),(SELECT count(*) FROM search_requests r WHERE telegram_id=$1 AND billing_kind='paid' AND quota_day=(clock_timestamp() AT TIME ZONE 'Asia/Kolkata')::date AND state IN ('reserved','charged','uncertain') AND created_at > COALESCE((SELECT reset_at FROM quota_resets WHERE telegram_id=$1 AND quota_day=r.quota_day),'-infinity'::timestamptz)) FROM bot_users WHERE telegram_id=$1`, uid).Scan(&active, &blocked, &limit, &name, &region, &expires, &used)
	if err != nil {
		return err
	}
	if a.admins[uid] && !blocked {
		return a.send(ctx, uid, "My account\n\nAdministrator access\nSearches: unlimited · no credits used\n\n/history — search history\n/admin — management commands")
	}
	if !active && !blocked {
		var trialUsed int
		if err := a.db.QueryRow(ctx, `SELECT count(*) FROM search_requests WHERE telegram_id=$1 AND billing_kind='trial' AND state IN ('reserved','charged','uncertain')`, uid).Scan(&trialUsed); err != nil {
			return err
		}
		return a.send(ctx, uid, fmt.Sprintf("My account\n\nFree trial searches remaining: %d of 3\n\nResults use one trial search; no-result searches are free. Open Plans to choose paid access.", max(0, 3-trialUsed)))
	}
	status := "Active"
	if !active || blocked {
		limit = 0
		used = 0
	}
	if blocked {
		status = "Blocked"
	} else if !active {
		status = "Inactive"
	}
	return a.send(ctx, uid, fmt.Sprintf("My account\n\n%s · %s plan\nValid until: %s\n\nToday’s searches\n%d remaining of %d  ·  %d used\nResets at midnight IST\n\n/history — search history\n/mypayments — payment status\n/support — get help", status, name, expires, max(0, limit-used), limit, used))
}
func (a *app) history(ctx context.Context, uid int64, parts []string) error {
	page := 1
	if len(parts) > 2 {
		return a.send(ctx, uid, "Use /history or /history PAGE.")
	}
	if len(parts) == 2 {
		var err error
		page, err = strconv.Atoi(parts[1])
		if err != nil || page < 1 || page > 10000000 {
			return a.send(ctx, uid, "Enter a valid history page number, for example /history 2.")
		}
	}
	rows, err := a.db.Query(ctx, `SELECT search_type,masked_value,state,outcome,to_char(created_at AT TIME ZONE 'Asia/Kolkata','DD Mon YYYY HH24:MI'),result_count FROM search_requests WHERE telegram_id=$1 ORDER BY created_at DESC,update_id DESC LIMIT 11 OFFSET $2`, uid, (page-1)*10)
	if err != nil {
		return err
	}
	defer rows.Close()
	var b strings.Builder
	fmt.Fprintf(&b, "Search history · page %d\n\n", page)
	count := 0
	more := false
	for rows.Next() {
		var kind, value, state, outcome, date string
		var resultCount *int
		if err := rows.Scan(&kind, &value, &state, &outcome, &date, &resultCount); err != nil {
			return err
		}
		if count == 10 {
			more = true
			break
		}
		if kind == "" {
			kind = "Search"
		}
		if kind == "master_id" {
			kind = "Document number"
		}
		if kind == "auto" {
			kind = "Phone or document number"
		}
		if value == "" {
			value = "(details unavailable)"
		}
		label := "Results not recorded"
		if state == "reserved" {
			label = "Search in progress"
		} else if outcome == "failed" || outcome == "cancelled" {
			label = "Search not completed"
		} else if outcome == "no_results" {
			label = "No results found"
		} else if resultCount != nil {
			if *resultCount == 0 {
				label = "No results found"
			} else {
				word := "results"
				if *resultCount == 1 {
					word = "result"
				}
				label = fmt.Sprintf("%d %s shown", *resultCount, word)
			}
		}
		fmt.Fprintf(&b, "%s IST\n%s · %s\n%s\n\n", date, kind, value, label)
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count == 0 {
		b.WriteString("No searches found on this page.")
	} else if more {
		fmt.Fprintf(&b, "\nNext page: /history %d", page+1)
	}
	return a.send(ctx, uid, b.String())
}

func historyAAD(uid, updateID int64, kind string) []byte {
	return []byte(fmt.Sprintf("search-history-v1:%d:%d:%s", uid, updateID, kind))
}
func (a *app) historyCipher() (cipher.AEAD, error) {
	if len(a.historyKey) != 32 {
		return nil, errors.New("history encryption key unavailable")
	}
	block, err := aes.NewCipher(a.historyKey)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
func (a *app) encryptHistory(uid, updateID int64, kind, value string) ([]byte, error) {
	gcm, err := a.historyCipher()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(value), historyAAD(uid, updateID, kind)), nil
}
func (a *app) decryptHistory(uid, updateID int64, kind string, data []byte) (string, error) {
	gcm, err := a.historyCipher()
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted history")
	}
	plain, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], historyAAD(uid, updateID, kind))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (a *app) adminHistory(ctx context.Context, actor int64, p []string) error {
	if !a.admins[actor] {
		return a.send(ctx, actor, "This action is not available.")
	}
	if len(p) < 2 || len(p) > 3 {
		return a.send(ctx, actor, "Use /userhistory USER_ID or /userhistory USER_ID PAGE.")
	}
	target, err := strconv.ParseInt(p[1], 10, 64)
	if err != nil || target <= 0 {
		return a.send(ctx, actor, "Enter a valid Telegram user ID.")
	}
	page := 1
	if len(p) == 3 {
		page, err = strconv.Atoi(p[2])
		if err != nil || page < 1 || page > 10000000 {
			return a.send(ctx, actor, "Enter a valid history page number.")
		}
	}
	rows, err := a.db.Query(ctx, `SELECT update_id,search_type,masked_value,encrypted_value,state,outcome,to_char(created_at AT TIME ZONE 'Asia/Kolkata','DD Mon YYYY HH24:MI') FROM search_requests WHERE telegram_id=$1 ORDER BY created_at DESC,update_id DESC LIMIT 11 OFFSET $2`, target, (page-1)*10)
	if err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "User %d · search history · page %d\n\n", target, page)
	n := 0
	hasMore := false
	for rows.Next() {
		var updateID int64
		var kind, masked, state, outcome, date string
		var encrypted []byte
		if err = rows.Scan(&updateID, &kind, &masked, &encrypted, &state, &outcome, &date); err != nil {
			rows.Close()
			return err
		}
		if n == 10 {
			hasMore = true
			break
		}
		if outcome == "" {
			outcome = state
		}
		if kind == "" {
			kind = "search"
		}
		value := "(full value unavailable for older search)"
		if len(encrypted) > 0 {
			full, e := a.decryptHistory(target, updateID, kind, encrypted)
			if e != nil {
				rows.Close()
				return errors.New("history could not be decrypted; check encryption key")
			}
			value = cleanResultField(full)
		} else if masked != "" {
			value = masked + " (older masked record)"
		}
		label := kind
		if label == "master_id" {
			label = "Document number"
		}
		if label == "auto" {
			label = "Phone or document number"
		}
		fmt.Fprintf(&b, "%s IST · %s\n%s · %s\n\n", date, label, value, outcome)
		n++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if n == 0 {
		b.WriteString("No searches on this page.")
	} else if hasMore {
		fmt.Fprintf(&b, "Next: /userhistory %d %d", target, page+1)
	}
	if _, err = a.db.Exec(ctx, `INSERT INTO admin_audit(admin_id,target_id,action,detail) VALUES($1,$2,'/userhistory',$3)`, actor, target, fmt.Sprintf("page=%d", page)); err != nil {
		return err
	}
	return a.send(ctx, actor, b.String())
}

func (a *app) resetQuota(ctx context.Context, updateID, actor int64, p []string) error {
	if len(p) != 2 {
		return a.send(ctx, actor, "Use /resetquota USER_ID to restore today's searches.")
	}
	target, err := strconv.ParseInt(p[1], 10, 64)
	if err != nil || target <= 0 {
		return a.send(ctx, actor, "Enter a valid Telegram user ID.")
	}
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var exists int64
	err = tx.QueryRow(ctx, `SELECT telegram_id FROM bot_users WHERE telegram_id=$1 FOR UPDATE`, target).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return a.send(ctx, actor, "User not found.")
	}
	if err != nil {
		return err
	}
	var busy bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM search_requests WHERE telegram_id=$1 AND state='reserved')`, target).Scan(&busy)
	if err != nil {
		return err
	}
	if busy {
		return a.send(ctx, actor, "This user has a search in progress. Try again when it finishes.")
	}
	tag, err := tx.Exec(ctx, `INSERT INTO admin_updates(update_id) VALUES($1) ON CONFLICT DO NOTHING`, updateID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `INSERT INTO quota_resets(telegram_id,quota_day,reset_at,admin_id) VALUES($1,(clock_timestamp() AT TIME ZONE 'Asia/Kolkata')::date,clock_timestamp(),$2) ON CONFLICT(telegram_id,quota_day) DO UPDATE SET reset_at=excluded.reset_at,admin_id=excluded.admin_id`, target, actor)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO admin_audit(admin_id,target_id,action,detail) VALUES($1,$2,'/resetquota','daily quota reset')`, actor, target)
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return a.send(ctx, actor, "Today's search allowance has been reset. Search history is preserved.")
}
