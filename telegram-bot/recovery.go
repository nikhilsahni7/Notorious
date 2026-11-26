package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Result pages are encrypted with a separate authenticated context from history.
func (a *app) encryptDelivery(uid, updateID int64, pages []string) ([]byte, error) {
	g, err := a.historyCipher()
	if err != nil {
		return nil, err
	}
	plain, err := json.Marshal(pages)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return g.Seal(nonce, nonce, plain, []byte(fmt.Sprintf("search-delivery-v1:%d:%d", uid, updateID))), nil
}
func (a *app) decryptDelivery(uid, updateID int64, data []byte) ([]string, error) {
	g, err := a.historyCipher()
	if err != nil {
		return nil, err
	}
	if len(data) < g.NonceSize() {
		return nil, errors.New("invalid delivery")
	}
	plain, err := g.Open(nil, data[:g.NonceSize()], data[g.NonceSize():], []byte(fmt.Sprintf("search-delivery-v1:%d:%d", uid, updateID)))
	if err != nil {
		return nil, err
	}
	var pages []string
	if err = json.Unmarshal(plain, &pages); err != nil {
		return nil, err
	}
	if len(pages) == 0 || len(pages) > 3 {
		return nil, errors.New("invalid delivery pages")
	}
	return pages, nil
}
func (a *app) wakeRecovery() {
	if a.recoveryWake != nil {
		select {
		case a.recoveryWake <- struct{}{}:
		default:
		}
	}
}

// queueSearchDelivery commits the result before Telegram delivery. Account row
// locking serializes it with admin blocks and plan changes.
func (a *app) queueSearchDelivery(ctx context.Context, uid, updateID int64, state string, result searchResult) (bool, error) {
	if len(result.Hits.Hits) > 5 {
		result.Hits.Hits = result.Hits.Hits[:5]
	}
	pages := formatResults(result, false)
	if state == "admin" {
		for i := range pages {
			pages[i] = strings.Replace(pages[i], "1 credit after delivery", "Admin search · no credit used", 1)
		}
	}
	if state == "trial" {
		for i := range pages {
			pages[i] = strings.Replace(pages[i], "1 credit after delivery", "1 free trial search after delivery", 1)
		}
	}
	encrypted, err := a.encryptDelivery(uid, updateID, pages)
	if err != nil {
		return false, err
	}
	outcome := "success"
	chargeable := true
	if result.Hits.Total.Value == 0 {
		outcome = "no_results"
		chargeable = false
	} else if state == "admin" {
		outcome = "admin"
		chargeable = false
	}
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var active bool
	err = tx.QueryRow(ctx, `SELECT NOT blocked AND ($2 OR (approved AND COALESCE(plan_expires_at>now(),false)) OR EXISTS(SELECT 1 FROM search_requests WHERE update_id=$3 AND telegram_id=$1 AND billing_kind='trial')) FROM bot_users WHERE telegram_id=$1 FOR UPDATE`, uid, a.admins[uid], updateID).Scan(&active)
	if err != nil {
		return false, err
	}
	if !active {
		_, err = tx.Exec(ctx, `UPDATE search_requests SET state='released',outcome='cancelled' WHERE update_id=$1 AND telegram_id=$2 AND state='reserved'`, updateID, uid)
		if err != nil {
			return false, err
		}
		return false, tx.Commit(ctx)
	}
	if result.Hits.Total.Value > 0 && len(result.Hits.Hits) > 0 {
		data, e := json.Marshal(result)
		if e != nil {
			return false, e
		}
		report, e := a.encryptDelivery(uid, updateID, []string{string(data)})
		if e != nil {
			return false, e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO search_reports(update_id,telegram_id,encrypted_result,expires_at) VALUES($1,$2,$3,now()+interval '24 hours') ON CONFLICT(update_id) DO NOTHING`, updateID, uid, report); e != nil {
			return false, e
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM search_reports WHERE expires_at<now()`); err != nil {
		return false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO search_deliveries(update_id,telegram_id,encrypted_pages,chargeable,outcome) SELECT update_id,telegram_id,$3,$4,$5 FROM search_requests WHERE update_id=$1 AND telegram_id=$2 AND state='reserved' ON CONFLICT(update_id) DO NOTHING`, updateID, uid, encrypted, chargeable, outcome)
	if err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// deliverSearch commits each successful page before moving to the next. Telegram
// does not offer an idempotency key: a crash between send and commit can repeat a
// message, but a search is charged only once after all pages are confirmed.
func (a *app) deliverSearch(ctx context.Context, updateID, uid int64) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var active bool
	err = tx.QueryRow(ctx, `SELECT NOT blocked AND ($2 OR (approved AND COALESCE(plan_expires_at>now(),false)) OR EXISTS(SELECT 1 FROM search_requests WHERE update_id=$3 AND telegram_id=$1 AND billing_kind='trial')) FROM bot_users WHERE telegram_id=$1 FOR UPDATE`, uid, a.admins[uid], updateID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var encrypted []byte
	var page, attempts int
	var chargeable bool
	var outcome, status string
	var due bool
	err = tx.QueryRow(ctx, `SELECT encrypted_pages,next_page,attempts,chargeable,outcome,status,next_retry_at<=clock_timestamp() FROM search_deliveries WHERE update_id=$1 AND telegram_id=$2 FOR UPDATE`, updateID, uid).Scan(&encrypted, &page, &attempts, &chargeable, &outcome, &status, &due)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if status != "pending" || !due {
		return tx.Commit(ctx)
	}
	if !active {
		_, err = tx.Exec(ctx, `UPDATE search_deliveries SET status='cancelled',encrypted_pages=NULL,updated_at=clock_timestamp() WHERE update_id=$1`, updateID)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE search_requests SET state='released',outcome='cancelled' WHERE update_id=$1 AND state='reserved'`, updateID)
		}
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	pages, err := a.decryptDelivery(uid, updateID, encrypted)
	if err != nil {
		return err
	}
	if page < 0 || page > len(pages) {
		return errors.New("invalid delivery progress")
	}
	for page < len(pages) {
		sendCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
		payload := map[string]interface{}{"chat_id": uid, "text": pages[page]}
		if outcome != "no_results" && outcome != "failed" {
			payload["text"] = pages[page] + "\nTap and hold a result to copy it."
			payload["reply_markup"] = map[string]interface{}{"inline_keyboard": [][]map[string]string{{{"text": "Download CSV report", "callback_data": fmt.Sprintf("report:%d", updateID)}}}}
		}
		sendErr := a.sendPayload(sendCtx, payload)
		cancel()
		if sendErr != nil {
			attempts++
			if attempts >= 5 {
				_, err = tx.Exec(ctx, `UPDATE search_deliveries SET status='cancelled',encrypted_pages=NULL,attempts=$2,next_page=$3,updated_at=clock_timestamp() WHERE update_id=$1`, updateID, attempts, page)
				if err == nil {
					_, err = tx.Exec(ctx, `UPDATE search_requests SET state='released',outcome='failed' WHERE update_id=$1 AND state='reserved'`, updateID)
				}
			} else {
				delay := time.Duration(1<<uint(attempts)) * 5 * time.Second
				_, err = tx.Exec(ctx, `UPDATE search_deliveries SET attempts=$2,next_page=$3,next_retry_at=clock_timestamp()+($4*interval '1 second'),updated_at=clock_timestamp() WHERE update_id=$1`, updateID, attempts, page, int(delay.Seconds()))
			}
			if err != nil {
				return err
			}
			return tx.Commit(ctx)
		}
		page++
		// Persist each page before sending another; never log message contents.
		_, err = tx.Exec(ctx, `UPDATE search_deliveries SET next_page=$2,updated_at=clock_timestamp() WHERE update_id=$1`, updateID, page)
		if err != nil {
			return err
		}
		if page < len(pages) {
			if err = tx.Commit(ctx); err != nil {
				return err
			}
			return a.deliverSearch(ctx, updateID, uid)
		}
	}
	state := "released"
	if chargeable {
		state = "charged"
	}
	_, err = tx.Exec(ctx, `UPDATE search_requests SET state=$2,outcome=$3 WHERE update_id=$1 AND state='reserved'`, updateID, state, outcome)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE search_deliveries SET status='delivered',encrypted_pages=NULL,updated_at=clock_timestamp() WHERE update_id=$1`, updateID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// recoverAbandoned releases an interrupted read-only search after its request
// deadline, and queues a notice. It never repeats a search or charges a credit.
func (a *app) recoverAbandoned(ctx context.Context, updateID, uid int64) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var active bool
	err = tx.QueryRow(ctx, `SELECT NOT blocked AND ($2 OR (approved AND COALESCE(plan_expires_at>now(),false)) OR EXISTS(SELECT 1 FROM search_requests WHERE update_id=$3 AND telegram_id=$1 AND billing_kind='trial')) FROM bot_users WHERE telegram_id=$1 FOR UPDATE`, uid, a.admins[uid], updateID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var eligible bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM search_requests r WHERE r.update_id=$1 AND r.telegram_id=$2 AND r.state='reserved' AND r.created_at<clock_timestamp()-interval '2 minutes' AND NOT EXISTS(SELECT 1 FROM search_deliveries d WHERE d.update_id=r.update_id))`, updateID, uid).Scan(&eligible)
	if err != nil {
		return err
	}
	if !eligible {
		return tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `UPDATE search_requests SET state='released',outcome='failed' WHERE update_id=$1 AND state='reserved'`, updateID)
	if err != nil {
		return err
	}
	if active {
		pages := []string{"A previous search was interrupted. No credit was used. You can try again."}
		encrypted, e := a.encryptDelivery(uid, updateID, pages)
		if e != nil {
			return e
		}
		_, err = tx.Exec(ctx, `INSERT INTO search_deliveries(update_id,telegram_id,encrypted_pages,chargeable,outcome) VALUES($1,$2,$3,false,'failed') ON CONFLICT(update_id) DO NOTHING`, updateID, uid, encrypted)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

type recoveryItem struct{ id, uid int64 }

func (a *app) recoveryStep(ctx context.Context) (time.Duration, error) {
	rows, err := a.db.Query(ctx, `SELECT update_id,telegram_id FROM search_deliveries WHERE status='pending' AND next_retry_at<=clock_timestamp() ORDER BY next_retry_at LIMIT 20`)
	if err != nil {
		return 0, err
	}
	var due []recoveryItem
	for rows.Next() {
		var i recoveryItem
		if err = rows.Scan(&i.id, &i.uid); err != nil {
			rows.Close()
			return 0, err
		}
		due = append(due, i)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	for _, i := range due {
		deliveryCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
		err = a.deliverSearch(deliveryCtx, i.id, i.uid)
		cancel()
		if err != nil {
			return 0, err
		}
	}
	rows, err = a.db.Query(ctx, `SELECT r.update_id,r.telegram_id FROM search_requests r WHERE r.state='reserved' AND r.created_at<clock_timestamp()-interval '2 minutes' AND NOT EXISTS (SELECT 1 FROM search_deliveries d WHERE d.update_id=r.update_id) ORDER BY r.created_at LIMIT 20`)
	if err != nil {
		return 0, err
	}
	var stale []recoveryItem
	for rows.Next() {
		var i recoveryItem
		if err = rows.Scan(&i.id, &i.uid); err != nil {
			rows.Close()
			return 0, err
		}
		stale = append(stale, i)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	for _, i := range stale {
		if err = a.recoverAbandoned(ctx, i.id, i.uid); err != nil {
			return 0, err
		}
	}
	if len(due) == 20 || len(stale) > 0 {
		return time.Millisecond, nil
	}
	var seconds *float64
	err = a.db.QueryRow(ctx, `SELECT EXTRACT(EPOCH FROM (MIN(due_at)-clock_timestamp())) FROM (SELECT MIN(next_retry_at) AS due_at FROM search_deliveries WHERE status='pending' UNION ALL SELECT MIN(r.created_at+interval '2 minutes') FROM search_requests r WHERE r.state='reserved' AND NOT EXISTS (SELECT 1 FROM search_deliveries d WHERE d.update_id=r.update_id)) q`).Scan(&seconds)
	if err != nil {
		return 0, err
	}
	if seconds == nil {
		return 0, nil
	}
	wait := time.Duration(*seconds * float64(time.Second))
	if wait < time.Millisecond {
		wait = time.Millisecond
	}
	return wait, nil
}
func (a *app) recoveryLoop() {
	var wait time.Duration
	for {
		if wait == 0 {
			<-a.recoveryWake
		} else {
			timer := time.NewTimer(wait)
			select {
			case <-a.recoveryWake:
				timer.Stop()
			case <-timer.C:
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		next, err := a.recoveryStep(ctx)
		cancel()
		if err != nil {
			log.Printf("search delivery recovery failed: %v", err)
			wait = 30 * time.Second
			continue
		}
		wait = next
	}
}
