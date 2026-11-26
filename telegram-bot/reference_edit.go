package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (a *app) handleCallback(ctx context.Context, callbackID string, uid int64, data string) error {
	if strings.HasPrefix(data, "report:") {
		return a.handleReportCallback(ctx, callbackID, uid, data)
	}
	prefix := "editref:"
	confirm := false
	if strings.HasPrefix(data, "confirmref:") {
		prefix = "confirmref:"
		confirm = true
	}
	if !strings.HasPrefix(data, prefix) {
		return a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": "Action unavailable"})
	}
	parts := strings.Split(strings.TrimPrefix(data, prefix), ":")
	if (!confirm && len(parts) != 1) || (confirm && len(parts) != 2) {
		return a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": "Action unavailable"})
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": "Action unavailable"})
	}
	if !confirm {
		tag, e := a.db.Exec(ctx, `UPDATE bot_users SET pending_ref_order=$2,pending_ref_value=NULL,pending_payment=false,pending_search='' WHERE telegram_id=$1 AND NOT blocked AND EXISTS(SELECT 1 FROM payment_orders WHERE id=$2 AND user_id=$1 AND status='pending_review' AND review_started_at IS NULL AND is_test=$3)`, uid, id, paymentTestMode())
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": "Request cannot be edited"})
		}
		if e = a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID}); e != nil {
			return e
		}
		return a.send(ctx, uid, "Edit payment reference\n\nSend the correct UPI transaction reference. You will review it before confirming.\n\nSend /cancel to go back.")
	}
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var old, status, plan string
	var reviewing bool
	err = tx.QueryRow(ctx, `SELECT COALESCE(transaction_reference,''),status,plan_name,review_started_at IS NOT NULL FROM payment_orders WHERE id=$1 AND user_id=$2 AND is_test=$3 FOR UPDATE`, id, uid, paymentTestMode()).Scan(&old, &status, &plan, &reviewing)
	if errors.Is(err, pgx.ErrNoRows) {
		return a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": "Request unavailable"})
	}
	if err != nil {
		return err
	}
	var candidate *string
	var blocked bool
	err = tx.QueryRow(ctx, `SELECT pending_ref_value,blocked FROM bot_users WHERE telegram_id=$1 AND pending_ref_order=$2 FOR UPDATE`, uid, id).Scan(&candidate, &blocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": "This edit is no longer available"})
	}
	if err != nil {
		return err
	}
	if status != "pending_review" || reviewing || candidate == nil || blocked {
		if _, err = tx.Exec(ctx, `UPDATE bot_users SET pending_ref_order=NULL,pending_ref_value=NULL WHERE telegram_id=$1 AND pending_ref_order=$2`, uid, id); err != nil {
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		return a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": "This edit is no longer available"})
	}
	ref, ok := normalizeReference(*candidate)
	if !ok {
		return errors.New("invalid staged reference")
	}
	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(a.referenceConfirmToken(uid, id, ref))) != 1 {
		return a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": "Reference changed. Please review it again."})
	}
	if old != ref {
		if old != "" {
			if _, e := tx.Exec(ctx, `INSERT INTO payment_reference_ledger(reference_upper,order_id) VALUES(upper($1),$2) ON CONFLICT(reference_upper) DO NOTHING`, old, id); e != nil {
				return e
			}
		}
		tag, e := tx.Exec(ctx, `INSERT INTO payment_reference_ledger(reference_upper,order_id) VALUES($1,$2) ON CONFLICT(reference_upper) DO UPDATE SET order_id=payment_reference_ledger.order_id WHERE payment_reference_ledger.order_id=excluded.order_id`, ref, id)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": "This reference is already in use"})
		}
		_, err = tx.Exec(ctx, `UPDATE payment_orders SET transaction_reference=$2,submitted_at=now() WHERE id=$1`, id, ref)
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == "23505" {
			return a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": "This reference is already in use"})
		}
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO payment_reference_audit(order_id,user_id,previous_reference,new_reference) VALUES($1,$2,$3,$4)`, id, uid, old, ref); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE bot_users SET pending_ref_order=NULL,pending_ref_value=NULL WHERE telegram_id=$1`, uid); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if err = a.telegramMethod(ctx, "answerCallbackQuery", map[string]interface{}{"callback_query_id": callbackID, "text": "Reference updated"}); err != nil {
		return err
	}
	if old != ref {
		for admin := range a.admins {
			_ = a.send(ctx, admin, fmt.Sprintf("Payment reference changed\n\nRequest #%d · user %d · %s\nNew reference: %s\nVerify the new reference before approving.", id, uid, plan, ref))
		}
	}
	return a.send(ctx, uid, "Reference updated\n\nYour request remains under review. We’ll verify the updated reference before activating the plan.")
}

func (a *app) stageReferenceEdit(ctx context.Context, uid, id int64, input string) error {
	ref, ok := normalizeReference(input)
	if !ok {
		return a.send(ctx, uid, "Enter a valid 6–64 character UPI reference using letters and numbers, or send /cancel.")
	}
	tag, err := a.db.Exec(ctx, `UPDATE bot_users SET pending_ref_value=$3 WHERE telegram_id=$1 AND pending_ref_order=$2 AND NOT blocked AND EXISTS(SELECT 1 FROM payment_orders WHERE id=$2 AND user_id=$1 AND status='pending_review' AND review_started_at IS NULL AND is_test=$4)`, uid, id, ref, paymentTestMode())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if _, err = a.db.Exec(ctx, `UPDATE bot_users SET pending_ref_order=NULL,pending_ref_value=NULL WHERE telegram_id=$1 AND pending_ref_order=$2`, uid, id); err != nil {
			return err
		}
		return a.send(ctx, uid, "This request can no longer be edited. Check Payment status.")
	}
	return a.sendPayload(ctx, map[string]interface{}{"chat_id": uid, "text": premiumHTML(fmt.Sprintf("Confirm reference\n\nNew reference: %s\n\nConfirm only if this matches your UPI transaction.", ref)), "parse_mode": "HTML", "protect_content": true, "reply_markup": map[string]interface{}{"inline_keyboard": [][]map[string]string{{{"text": "Confirm reference", "callback_data": fmt.Sprintf("confirmref:%d:%s", id, a.referenceConfirmToken(uid, id, ref)), "style": "primary"}}}}})
}

func (a *app) referenceConfirmToken(uid, id int64, ref string) string {
	m := hmac.New(sha256.New, []byte(a.webhookSecret))
	fmt.Fprintf(m, "reference-confirm-v1\x00%d\x00%d\x00%s", uid, id, ref)
	return hex.EncodeToString(m.Sum(nil)[:8])
}
