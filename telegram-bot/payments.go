package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var referencePattern = regexp.MustCompile(`^[A-Z0-9]{6,64}$`)
var upiPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{2,128}@[A-Za-z0-9.-]{2,64}$`)

type paymentOrder struct {
	ID, UserID, Amount int64
	Plan               string
	Limit, Days        int
	Status, Reference  string
	IsTest             bool
}

// Test orders can only be created or approved by a locally polled bot.
func paymentTestMode() bool {
	return os.Getenv("BOT_MODE") == "polling" && os.Getenv("PAYMENT_TEST_MODE") == "true"
}

func normalizeReference(s string) (string, bool) {
	v := strings.ToUpper(strings.TrimSpace(s))
	return v, referencePattern.MatchString(v)
}
func money(paise int64) string { return fmt.Sprintf("₹%d.%02d", paise/100, paise%100) }

func (a *app) buy(ctx context.Context, updateID, uid int64, p []string) error {
	if len(p) != 2 {
		return a.send(ctx, uid, "Choose a plan with /buy PLAN_NAME. See /plans.")
	}
	upi := strings.TrimSpace(os.Getenv("UPI_ID"))
	testMode := paymentTestMode()
	if !testMode && !upiPattern.MatchString(upi) {
		return a.send(ctx, uid, "Payments are not configured yet. Please contact support.")
	}
	var o paymentOrder
	err := a.db.QueryRow(ctx, `INSERT INTO payment_orders(create_update_id,user_id,plan_name,amount_paise,daily_limit,plan_days,is_test)
 SELECT $1,$2,name,price_paise,daily_limit,days,$4 FROM plans WHERE name=$3 AND active AND price_paise>0
 ON CONFLICT(create_update_id) DO UPDATE SET create_update_id=payment_orders.create_update_id
 WHERE payment_orders.user_id=excluded.user_id
 RETURNING id,user_id,plan_name,amount_paise,daily_limit,plan_days,status,is_test`, updateID, uid, p[1], testMode).Scan(&o.ID, &o.UserID, &o.Plan, &o.Amount, &o.Limit, &o.Days, &o.Status, &o.IsTest)
	if errors.Is(err, pgx.ErrNoRows) {
		return a.send(ctx, uid, "Plan not found or unavailable. See /plans.")
	}
	if err != nil {
		return err
	}
	if o.IsTest != testMode {
		return a.send(ctx, uid, "This order belongs to a different payment mode. Please create a new order.")
	}
	text := fmt.Sprintf("Your %s plan\n\n%s  ·  %d searches per day  ·  %d days\n\n1. Pay %s to %s using UPI.\n2. Tap Submit payment and enter the transaction reference.\n3. An admin will review your payment and activate your plan.\n\nOpen Payment status anytime to check your request.", o.Plan, money(o.Amount), o.Limit, o.Days, money(o.Amount), upi)
	if testMode {
		text = fmt.Sprintf("Test · %s plan\n\nSimulated price: %s\n%d searches per day  ·  %d days\n\nDO NOT SEND MONEY. Tap Submit payment and enter TEST123456.\n\nAn admin will review this test request.", o.Plan, money(o.Amount), o.Limit, o.Days)
	}
	if err = a.sendPayload(ctx, map[string]interface{}{"chat_id": uid, "text": text, "protect_content": true, "reply_markup": map[string]interface{}{"keyboard": [][]map[string]string{{{"text": "Submit payment"}}, {{"text": "Payment status"}, {"text": "Home"}}}, "resize_keyboard": true}}); err != nil {
		return err
	}
	if path := strings.TrimSpace(os.Getenv("UPI_QR_PATH")); path != "" && !testMode {
		if err = a.sendQR(ctx, uid, path, o.ID); err != nil {
			log.Printf("UPI QR unavailable; order %d has UPI ID instructions", o.ID)
		}
	}
	return nil
}

func (a *app) sendQR(ctx context.Context, uid int64, path string, _ int64) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	if !stat.Mode().IsRegular() || stat.Size() > 5<<20 {
		return errors.New("invalid QR file")
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err = w.WriteField("chat_id", strconv.FormatInt(uid, 10)); err != nil {
		return err
	}
	if err = w.WriteField("caption", "UPI QR: check the payee and amount before payment."); err != nil {
		return err
	}
	if err = w.WriteField("protect_content", "true"); err != nil {
		return err
	}
	part, err := w.CreateFormFile("photo", "upi-qr.png")
	if err != nil {
		return err
	}
	if _, err = io.Copy(part, f); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.telegram.org/bot"+a.token+"/sendPhoto", &buf)
	if err != nil {
		return errors.New("telegram request failed")
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := a.client.Do(req)
	if err != nil {
		return errors.New("telegram request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("telegram response %d", resp.StatusCode)
	}
	return nil
}
func numericOrderID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
func (a *app) submitPayment(ctx context.Context, uid int64, p []string) error {
	if len(p) == 2 || (len(p) == 3 && !numericOrderID(p[1])) {
		ref := p[len(p)-1]
		if _, valid := normalizeReference(ref); !valid {
			return a.send(ctx, uid, "Enter a 6–64 character transaction reference using letters and numbers.")
		}
		plan := ""
		if len(p) == 3 {
			plan = p[1]
		}
		// Identical plan snapshots are interchangeable: choose the most recent.
		// Different prices or allowances must never be guessed.
		rows, err := a.db.Query(ctx, `SELECT MAX(id) FROM payment_orders WHERE user_id=$1 AND status='awaiting_reference' AND is_test=$2 AND ($3='' OR lower(plan_name)=lower($3)) GROUP BY plan_name,amount_paise,daily_limit,plan_days ORDER BY MAX(id) DESC LIMIT 2`, uid, paymentTestMode(), plan)
		if err != nil {
			return err
		}
		var ids []int64
		for rows.Next() {
			var id int64
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return a.send(ctx, uid, "No matching unpaid plan was found. Use /mypayments to check your orders.")
		}
		if len(ids) > 1 {
			if plan == "" {
				return a.send(ctx, uid, "You have unpaid orders with different plan terms. Send /paid PLAN_NAME TRANSACTION_REFERENCE to choose the plan.")
			}
			return a.send(ctx, uid, "That plan has unpaid orders with different prices or limits. Use /mypayments and send /paid ORDER_ID TRANSACTION_REFERENCE for the order you paid.")
		}
		p = []string{p[0], strconv.FormatInt(ids[0], 10), ref}
	}
	if len(p) != 3 {
		return a.send(ctx, uid, "Use /paid TRANSACTION_REFERENCE, or /paid PLAN_NAME TRANSACTION_REFERENCE if you have different unpaid plans.")
	}

	id, err := strconv.ParseInt(p[1], 10, 64)
	ref, ok := normalizeReference(p[2])
	if err != nil || id <= 0 || !ok {
		return a.send(ctx, uid, "Enter a valid order ID and 6–64 character letter/number transaction reference.")
	}
	var status, existing string
	err = a.db.QueryRow(ctx, `UPDATE payment_orders SET transaction_reference=$3,status='pending_review',submitted_at=now() WHERE id=$1 AND user_id=$2 AND status='awaiting_reference' AND is_test=$4 RETURNING status,transaction_reference`, id, uid, ref, paymentTestMode()).Scan(&status, &existing)
	if errors.Is(err, pgx.ErrNoRows) {
		err = a.db.QueryRow(ctx, `SELECT status,COALESCE(transaction_reference,'') FROM payment_orders WHERE id=$1 AND user_id=$2`, id, uid).Scan(&status, &existing)
		if errors.Is(err, pgx.ErrNoRows) {
			return a.send(ctx, uid, "Order not found.")
		}
		if err != nil {
			return err
		}
		if status == "pending_review" && existing == ref {
			return a.send(ctx, uid, "This order is already pending review.")
		}
		return a.send(ctx, uid, "This order cannot accept that reference. Check /mypayments or contact support.")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return a.send(ctx, uid, "That transaction reference has already been submitted. Contact support if this is a mistake.")
	}
	if err != nil {
		return err
	}
	// Admins can always recover pending items via /pending; delivery is best effort.
	notice := fmt.Sprintf("Payment order #%d submitted by user %d. Use /pending and verify the bank receipt before approval.", id, uid)
	if paymentTestMode() {
		notice = fmt.Sprintf("TEST payment order #%d submitted by user %d. No real payment was made. Use /pending to review.", id, uid)
	}
	for admin := range a.admins {
		if e := a.send(ctx, admin, notice); e != nil {
			log.Printf("admin payment notification failed for order %d: %v", id, e)
		} else {
			log.Printf("admin payment notification accepted by Telegram for order %d", id)
		}
	}
	if paymentTestMode() {
		return a.send(ctx, uid, "Test request submitted\n\nYour request is waiting for admin review. No real payment was made.\n\nOpen Payment status to check it anytime.")
	}
	return a.send(ctx, uid, "Payment submitted\n\nYour request is waiting for admin review. We’ll verify the payment before activating your plan.\n\nOpen Payment status to check it anytime.")
}
func (a *app) myPayments(ctx context.Context, uid int64) error {
	rows, err := a.db.Query(ctx, `SELECT id,plan_name,amount_paise,status,is_test FROM payment_orders WHERE user_id=$1 ORDER BY id DESC LIMIT 10`, uid)
	if err != nil {
		return err
	}
	defer rows.Close()
	var b strings.Builder
	b.WriteString("Payment status\n\nYour most recent requests:\n\n")
	n := 0
	for rows.Next() {
		var o paymentOrder
		if err = rows.Scan(&o.ID, &o.Plan, &o.Amount, &o.Status, &o.IsTest); err != nil {
			return err
		}
		label := ""
		if o.IsTest {
			label = "TEST · "
		}
		status := map[string]string{"awaiting_reference": "Waiting for reference", "pending_review": "Under review", "approved": "Approved", "rejected": "Not approved"}[o.Status]
		if status == "" {
			status = "Status unavailable"
		}
		fmt.Fprintf(&b, "%s%s · %s\n%s  ·  Ref #%d\n\n", label, o.Plan, money(o.Amount), status, o.ID)
		n++
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if n == 0 {
		b.WriteString("No payment requests yet. Open Plans to choose a plan.")
	}
	return a.send(ctx, uid, b.String())
}

func (a *app) paymentAdmin(ctx context.Context, updateID, admin int64, cmd string, p []string) error {
	if cmd == "/pending" {
		rows, err := a.db.Query(ctx, `SELECT id,user_id,plan_name,amount_paise,daily_limit,plan_days,transaction_reference FROM payment_orders WHERE status='pending_review' AND is_test=$1 ORDER BY submitted_at,id LIMIT 20`, paymentTestMode())
		if err != nil {
			return err
		}
		defer rows.Close()
		var b strings.Builder
		if paymentTestMode() {
			b.WriteString("TEST payment reviews — no real payments (up to 20)\n\n")
		} else {
			b.WriteString("Pending payment reviews (up to 20)\n\n")
		}
		n := 0
		for rows.Next() {
			var o paymentOrder
			if err = rows.Scan(&o.ID, &o.UserID, &o.Plan, &o.Amount, &o.Limit, &o.Days, &o.Reference); err != nil {
				return err
			}
			fmt.Fprintf(&b, "#%d · user %d · %s\n%s · %d/day · %d days\nReference: %s\n\n", o.ID, o.UserID, o.Plan, money(o.Amount), o.Limit, o.Days, o.Reference)
			n++
		}
		if err = rows.Err(); err != nil {
			return err
		}
		if n == 0 {
			b.WriteString("No pending reviews.")
		} else {
			if !paymentTestMode() {
				b.WriteString("Verify the actual bank receipt.\n")
			}
			b.WriteString("/approvepayment ORDER_ID\n/rejectpayment ORDER_ID REASON")
		}
		return a.send(ctx, admin, b.String())
	}
	if (cmd == "/approvepayment" && len(p) != 2) || (cmd == "/rejectpayment" && len(p) < 3) {
		return a.send(ctx, admin, "Use /approvepayment ORDER_ID or /rejectpayment ORDER_ID REASON.")
	}
	id, err := strconv.ParseInt(p[1], 10, 64)
	if err != nil || id <= 0 {
		return a.send(ctx, admin, "Invalid order ID.")
	}
	region, reason := "pan-india", ""
	if cmd == "/rejectpayment" {
		reason = strings.TrimSpace(strings.Join(p[2:], " "))
		if len(reason) < 3 || len(reason) > 200 {
			return a.send(ctx, admin, "Give a rejection reason between 3 and 200 characters.")
		}
	}
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var o paymentOrder
	err = tx.QueryRow(ctx, `SELECT id,user_id,plan_name,amount_paise,daily_limit,plan_days,status,is_test FROM payment_orders WHERE id=$1 FOR UPDATE`, id).Scan(&o.ID, &o.UserID, &o.Plan, &o.Amount, &o.Limit, &o.Days, &o.Status, &o.IsTest)
	if errors.Is(err, pgx.ErrNoRows) {
		return a.send(ctx, admin, "Order not found.")
	}
	if err != nil {
		return err
	}
	if o.IsTest != paymentTestMode() {
		return a.send(ctx, admin, "This order belongs to a different payment mode and cannot be reviewed here.")
	}
	if o.Status != "pending_review" {
		return a.send(ctx, admin, "Order is not pending review; it may already have been processed.")
	}
	if cmd == "/approvepayment" {
		var blocked, active bool
		err = tx.QueryRow(ctx, `SELECT blocked, COALESCE(plan_expires_at>now(),false) FROM bot_users WHERE telegram_id=$1 FOR UPDATE`, o.UserID).Scan(&blocked, &active)
		if err != nil {
			return err
		}
		if blocked {
			return a.send(ctx, admin, "User is blocked. Resolve the account status before approving.")
		}
		if active {
			return a.send(ctx, admin, "User already has an active plan. Resolve the existing plan before approving this order; no changes were made.")
		}
		_, err = tx.Exec(ctx, `UPDATE bot_users SET approved=true,region=$2,daily_limit=$3,plan_name=$4,plan_expires_at=now()+($5*interval '1 day') WHERE telegram_id=$1`, o.UserID, region, o.Limit, o.Plan, o.Days)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE payment_orders SET status='approved',region=$2,review_admin_id=$3,reviewed_at=now() WHERE id=$1`, id, region, admin)
	} else {
		_, err = tx.Exec(ctx, `UPDATE payment_orders SET status='rejected',review_reason=$2,review_admin_id=$3,reviewed_at=now() WHERE id=$1`, id, reason, admin)
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO admin_audit(admin_id,target_id,action,detail) VALUES($1,$2,$3,$4)`, admin, o.UserID, cmd, fmt.Sprintf("order=%d %s", id, region))
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	notice := fmt.Sprintf("Payment approved\n\nYour %s plan is active. Open My account to see your allowance, then choose a search to get started.", o.Plan)
	if o.IsTest {
		notice = "TEST — no payment was made. " + notice
	}
	if cmd == "/rejectpayment" {
		notice = fmt.Sprintf("Payment not approved\n\n%s\n\nContact /support if you need help.", reason)
	}
	if err = a.send(ctx, o.UserID, notice); err != nil {
		log.Printf("payment review notification failed for order %d", id)
	}
	return a.send(ctx, admin, fmt.Sprintf("Order #%d reviewed. Check /pending for remaining orders.", id))
}
