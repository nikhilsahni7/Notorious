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
	if _, err = a.db.Exec(ctx, `UPDATE bot_users SET pending_payment=true,pending_search='',pending_ref_order=NULL,pending_ref_value=NULL WHERE telegram_id=$1`, uid); err != nil {
		return err
	}
	text := fmt.Sprintf("Your %s plan\n\n%s  ·  %d searches per day  ·  %d days\n\n1. Pay %s to %s using UPI.\n2. Tap Submit payment and enter the transaction reference.\n3. An admin will review your payment and activate your plan.\n\nOpen Payment status anytime to check your request.", o.Plan, money(o.Amount), o.Limit, o.Days, money(o.Amount), upi)
	if testMode {
		text = fmt.Sprintf("Test · %s plan\n\nSimulated price: %s\n%d searches per day  ·  %d days\n\nDO NOT SEND MONEY. Tap Submit payment and enter TEST123456.\n\nAn admin will review this test request.", o.Plan, money(o.Amount), o.Limit, o.Days)
	}
	if err = a.sendPayload(ctx, map[string]interface{}{"chat_id": uid, "text": premiumHTML(text), "parse_mode": "HTML", "protect_content": true, "reply_markup": map[string]interface{}{"keyboard": [][]map[string]string{{{"text": "Submit payment"}}, {{"text": "Payment status"}, {"text": "Home"}}}, "resize_keyboard": true}}); err != nil {
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
		// The latest selected plan is the customer's current intent. Admins must
		// independently verify the actual payment amount before activation.
		rows, err := a.db.Query(ctx, `SELECT id FROM payment_orders WHERE user_id=$1 AND status='awaiting_reference' AND is_test=$2 AND ($3='' OR lower(plan_name)=lower($3)) ORDER BY id DESC LIMIT 1`, uid, paymentTestMode(), plan)
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
	var status, existing, planName string
	var amount int64
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `UPDATE payment_orders SET transaction_reference=$3,status='pending_review',submitted_at=now() WHERE id=$1 AND user_id=$2 AND status='awaiting_reference' AND is_test=$4 RETURNING status,transaction_reference,plan_name,amount_paise`, id, uid, ref, paymentTestMode()).Scan(&status, &existing, &planName, &amount)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT status,COALESCE(transaction_reference,'') FROM payment_orders WHERE id=$1 AND user_id=$2`, id, uid).Scan(&status, &existing)
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
	tag, err := tx.Exec(ctx, `INSERT INTO payment_reference_ledger(reference_upper,order_id) VALUES($1,$2) ON CONFLICT(reference_upper) DO UPDATE SET order_id=payment_reference_ledger.order_id WHERE payment_reference_ledger.order_id=excluded.order_id`, ref, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return a.send(ctx, uid, "That transaction reference has already been submitted. Contact support if this is a mistake.")
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if _, err = a.db.Exec(ctx, `UPDATE bot_users SET pending_payment=false WHERE telegram_id=$1`, uid); err != nil {
		return err
	}
	// Admins can always recover pending items via /pending; delivery is best effort.
	notice := fmt.Sprintf("Payment request #%d from user %d\nPlan: %s · %s\nReference: %s\nUse /pending and verify the actual bank receipt before approval.", id, uid, planName, money(amount), ref)
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
	return a.send(ctx, uid, fmt.Sprintf("Payment submitted\n\n%s plan · %s\nYour request is waiting for admin review. We’ll verify the payment before activating your plan.\n\nOpen Payment status to check it anytime.", planName, money(amount)))
}
func (a *app) myPayments(ctx context.Context, uid int64) error {
	rows, err := a.db.Query(ctx, `SELECT id,plan_name,amount_paise,status,is_test,review_started_at IS NOT NULL FROM payment_orders WHERE user_id=$1 ORDER BY id DESC LIMIT 10`, uid)
	if err != nil {
		return err
	}
	defer rows.Close()
	var b strings.Builder
	buttons := [][]map[string]string{}
	b.WriteString("Payment status\n\nYour most recent requests:\n\n")
	n := 0
	for rows.Next() {
		var o paymentOrder
		var reviewing bool
		if err = rows.Scan(&o.ID, &o.Plan, &o.Amount, &o.Status, &o.IsTest, &reviewing); err != nil {
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
		if reviewing && o.Status == "pending_review" {
			status = "Review in progress · reference locked"
		}
		fmt.Fprintf(&b, "%s%s · %s\n%s  ·  Ref #%d\n\n", label, o.Plan, money(o.Amount), status, o.ID)
		if o.Status == "pending_review" && !reviewing {
			buttons = append(buttons, []map[string]string{{"text": fmt.Sprintf("Edit %s reference", o.Plan), "callback_data": fmt.Sprintf("editref:%d", o.ID)}})
		}
		n++
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if n == 0 {
		b.WriteString("No payment requests yet. Open Plans to choose a plan.")
	}
	if len(buttons) == 0 {
		return a.send(ctx, uid, b.String())
	}
	return a.sendPayload(ctx, map[string]interface{}{"chat_id": uid, "text": premiumHTML(b.String()), "parse_mode": "HTML", "protect_content": true, "reply_markup": map[string]interface{}{"inline_keyboard": buttons}})
}

func (a *app) paymentAdmin(ctx context.Context, updateID, admin int64, cmd string, p []string) error {
	if cmd == "/reviewpayment" {
		if len(p) != 2 {
			return a.send(ctx, admin, "Use /reviewpayment ORDER_ID to lock the reference before checking the payment.")
		}
		id, e := strconv.ParseInt(p[1], 10, 64)
		if e != nil || id <= 0 {
			return a.send(ctx, admin, "Enter a valid order ID.")
		}
		tx, e := a.db.Begin(ctx)
		if e != nil {
			return e
		}
		defer tx.Rollback(ctx)
		var user int64
		var plan, ref string
		e = tx.QueryRow(ctx, `UPDATE payment_orders SET review_started_at=clock_timestamp() WHERE id=$1 AND status='pending_review' AND review_started_at IS NULL AND is_test=$2 RETURNING user_id,plan_name,transaction_reference`, id, paymentTestMode()).Scan(&user, &plan, &ref)
		if errors.Is(e, pgx.ErrNoRows) {
			return a.send(ctx, admin, "This request is unavailable or already under review.")
		}
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO admin_audit(admin_id,target_id,action,detail) VALUES($1,$2,'/reviewpayment',$3)`, admin, user, fmt.Sprintf("order %d", id)); e != nil {
			return e
		}
		if e = tx.Commit(ctx); e != nil {
			return e
		}
		if e = a.send(ctx, user, "Payment review started\n\nYour reference is now locked while the admin checks the payment. Contact /support if it needs correction."); e != nil {
			log.Printf("review notice failed for order %d", id)
		}
		return a.send(ctx, admin, fmt.Sprintf("Review started · request #%d\n\nPlan: %s\nReference: %s\n\nThe reference is locked. Verify the bank receipt before approval.", id, plan, ref))
	}
	if cmd == "/orderplan" {
		if len(p) != 3 {
			return a.send(ctx, admin, "Use /orderplan ORDER_ID PLAN_NAME after checking the payment amount.")
		}
		id, e := strconv.ParseInt(p[1], 10, 64)
		if e != nil || id <= 0 {
			return a.send(ctx, admin, "Enter a valid order ID.")
		}
		tx, e := a.db.Begin(ctx)
		if e != nil {
			return e
		}
		defer tx.Rollback(ctx)
		tag, e := tx.Exec(ctx, `INSERT INTO admin_updates(update_id) VALUES($1) ON CONFLICT DO NOTHING`, updateID)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return tx.Commit(ctx)
		}
		var user int64
		var name string
		var amount int64
		e = tx.QueryRow(ctx, `UPDATE payment_orders o SET plan_name=p.name,amount_paise=p.price_paise,daily_limit=p.daily_limit,plan_days=p.days FROM plans p WHERE o.id=$1 AND o.status='pending_review' AND o.is_test=$2 AND p.name=$3 AND p.active AND p.price_paise>0 RETURNING o.user_id,o.plan_name,o.amount_paise`, id, paymentTestMode(), p[2]).Scan(&user, &name, &amount)
		if errors.Is(e, pgx.ErrNoRows) {
			return a.send(ctx, admin, "Pending request or active plan not found.")
		}
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO admin_audit(admin_id,target_id,action,detail) VALUES($1,$2,'/orderplan',$3)`, admin, user, fmt.Sprintf("order %d plan %s", id, name)); e != nil {
			return e
		}
		if e = tx.Commit(ctx); e != nil {
			return e
		}
		return a.send(ctx, admin, fmt.Sprintf("Request #%d changed to %s · %s. Verify the bank payment before approving.", id, name, money(amount)))
	}
	if cmd == "/pending" {
		rows, err := a.db.Query(ctx, `SELECT id,user_id,plan_name,amount_paise,daily_limit,plan_days,COALESCE(transaction_reference,''),status,review_started_at IS NOT NULL FROM payment_orders WHERE status IN ('pending_review','awaiting_reference') AND is_test=$1 ORDER BY CASE WHEN status='pending_review' THEN 0 ELSE 1 END, COALESCE(submitted_at,created_at),id LIMIT 20`, paymentTestMode())
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
			var reviewing bool
			if err = rows.Scan(&o.ID, &o.UserID, &o.Plan, &o.Amount, &o.Limit, &o.Days, &o.Reference, &o.Status, &reviewing); err != nil {
				return err
			}
			if o.Status == "awaiting_reference" {
				fmt.Fprintf(&b, "#%d · user %d · %s\n%s · %d/day · %d days\nAwaiting payment reference — cannot approve yet\n\n", o.ID, o.UserID, o.Plan, money(o.Amount), o.Limit, o.Days)
			} else {
				fmt.Fprintf(&b, "#%d · user %d · %s\n%s · %d/day · %d days\nReference: %s\n\n", o.ID, o.UserID, o.Plan, money(o.Amount), o.Limit, o.Days, o.Reference)
				if reviewing {
					b.WriteString("Review in progress · reference locked\n\n")
				}
			}
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
			b.WriteString("/reviewpayment ORDER_ID — lock reference\n/orderplan ORDER_ID PLAN — correct plan\n/approvepayment ORDER_ID\n/upgradepayment ORDER_ID — confirm replacement\n/rejectpayment ORDER_ID REASON")
		}
		return a.send(ctx, admin, b.String())
	}
	if ((cmd == "/approvepayment" || cmd == "/upgradepayment") && len(p) != 2) || (cmd == "/rejectpayment" && len(p) < 3) {
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
	err = tx.QueryRow(ctx, `SELECT id,user_id,plan_name,amount_paise,daily_limit,plan_days,status,is_test,COALESCE(transaction_reference,'') FROM payment_orders WHERE id=$1 FOR UPDATE`, id).Scan(&o.ID, &o.UserID, &o.Plan, &o.Amount, &o.Limit, &o.Days, &o.Status, &o.IsTest, &o.Reference)
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
	if cmd == "/approvepayment" || cmd == "/upgradepayment" {
		var blocked, active bool
		var oldPlan, oldExpiry, oldExpiryRaw string
		var used int
		err = tx.QueryRow(ctx, `SELECT blocked, COALESCE(approved AND plan_expires_at>now(),false),plan_name,COALESCE(to_char(plan_expires_at AT TIME ZONE 'Asia/Kolkata','DD Mon YYYY HH24:MI') || ' IST','Not assigned'),COALESCE(plan_expires_at::text,''),(SELECT count(*) FROM search_requests r WHERE r.telegram_id=$1 AND r.billing_kind='paid' AND r.quota_day=(clock_timestamp() AT TIME ZONE 'Asia/Kolkata')::date AND r.state IN ('reserved','charged','uncertain') AND r.created_at>COALESCE((SELECT reset_at FROM quota_resets WHERE telegram_id=$1 AND quota_day=r.quota_day),'-infinity'::timestamptz)) FROM bot_users WHERE telegram_id=$1 FOR UPDATE`, o.UserID).Scan(&blocked, &active, &oldPlan, &oldExpiry, &oldExpiryRaw, &used)
		if err != nil {
			return err
		}
		if blocked {
			return a.send(ctx, admin, "User is blocked. Resolve the account status before approving.")
		}
		if active && cmd == "/approvepayment" {
			_, err = tx.Exec(ctx, `INSERT INTO payment_upgrade_confirmations(admin_id,order_id,reference,plan_name,amount_paise,daily_limit,plan_days,old_plan,old_expiry) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(admin_id,order_id) DO UPDATE SET reference=excluded.reference,plan_name=excluded.plan_name,amount_paise=excluded.amount_paise,daily_limit=excluded.daily_limit,plan_days=excluded.plan_days,old_plan=excluded.old_plan,old_expiry=excluded.old_expiry,created_at=now()`, admin, id, o.Reference, o.Plan, o.Amount, o.Limit, o.Days, oldPlan, oldExpiryRaw)
			if err != nil {
				return err
			}
			if err = tx.Commit(ctx); err != nil {
				return err
			}
			return a.send(ctx, admin, fmt.Sprintf("Confirm plan replacement\n\nCurrent: %s · expires %s\nNew: %s · %d searches/day · %d days · %s\nReference: %s\nUsed today: %d\n\nThe new plan starts immediately and today’s usage carries over. Verify the payment, then send /upgradepayment %d within 15 minutes.", oldPlan, oldExpiry, o.Plan, o.Limit, o.Days, money(o.Amount), o.Reference, used, id))
		}
		if !active && cmd == "/upgradepayment" {
			return a.send(ctx, admin, "This user has no active plan. Use /approvepayment for this request.")
		}
		if cmd == "/upgradepayment" {
			var valid bool
			err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM payment_upgrade_confirmations WHERE admin_id=$1 AND order_id=$2 AND reference=$3 AND plan_name=$4 AND amount_paise=$5 AND daily_limit=$6 AND plan_days=$7 AND old_plan=$8 AND old_expiry=$9 AND created_at>now()-interval '15 minutes')`, admin, id, o.Reference, o.Plan, o.Amount, o.Limit, o.Days, oldPlan, oldExpiryRaw).Scan(&valid)
			if err != nil {
				return err
			}
			if !valid {
				return a.send(ctx, admin, "The confirmation expired or the request changed. Review it again with /approvepayment.")
			}
			if _, err = tx.Exec(ctx, `DELETE FROM payment_upgrade_confirmations WHERE order_id=$1`, id); err != nil {
				return err
			}
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
	if cmd == "/upgradepayment" {
		notice = fmt.Sprintf("Plan updated\n\nYour %s plan is active now. Today’s used searches carry over. Open My account to see your allowance.", o.Plan)
	}
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
