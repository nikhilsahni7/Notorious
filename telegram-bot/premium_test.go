package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPremiumHTMLAlwaysEscapesRecordValues(t *testing.T) {
	got := premiumHTML("Search results\n\nResult 1\nName: <script>&value\nAddress: <b>unsafe</b>\n📞 Phone: <unsafe>")
	if strings.Contains(got, "<script>") || strings.Contains(got, "<b>unsafe</b>") || !strings.Contains(got, "&lt;script&gt;&amp;value") || !strings.Contains(got, "<b>Result 1</b>") || !strings.Contains(got, "<b>📞 Phone:</b> &lt;unsafe&gt;") {
		t.Fatalf("unsafe formatted text: %s", got)
	}
}

func TestPaymentUpgradeAndReferenceEditsPostgres(t *testing.T) {
	url := os.Getenv("PAYMENTS_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set PAYMENTS_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	base, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	name := fmt.Sprintf("premium_test_%d", os.Getpid())
	if _, err = base.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
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
	_, err = db.Exec(ctx, `INSERT INTO bot_users(telegram_id,approved,daily_limit,plan_name,plan_expires_at) VALUES(1,true,5,'Old',now()+interval '10 days'),(2,false,0,'Unassigned',NULL),(3,false,0,'Unassigned',NULL);
      INSERT INTO payment_orders(id,create_update_id,user_id,plan_name,amount_paise,daily_limit,plan_days,status,transaction_reference) VALUES(101,201,1,'New',10000,10,30,'pending_review','OLDREF101'),(102,202,2,'New',10000,10,30,'pending_review','OLDREF102'),(103,203,3,'New',10000,10,30,'pending_review','TAKENREF1');
      INSERT INTO search_requests(update_id,telegram_id,quota_day,state,billing_kind) VALUES(301,1,(clock_timestamp() AT TIME ZONE 'Asia/Kolkata')::date,'charged','paid'),(302,1,(clock_timestamp() AT TIME ZONE 'Asia/Kolkata')::date,'charged','paid'),(303,1,(clock_timestamp() AT TIME ZONE 'Asia/Kolkata')::date,'charged','paid')`)
	if err != nil {
		t.Fatal(err)
	}
	a := &app{db: db, admins: map[int64]bool{999: true}, client: &http.Client{Transport: paymentTestTransport{}, Timeout: time.Second}}
	if err = a.paymentAdmin(ctx, 401, 999, "/approvepayment", []string{"/approvepayment", "101"}); err != nil {
		t.Fatal(err)
	}
	var status, plan string
	if err = db.QueryRow(ctx, `SELECT status FROM payment_orders WHERE id=101`).Scan(&status); err != nil || status != "pending_review" {
		t.Fatalf("preview changed order: %s %v", status, err)
	}
	if _, err = db.Exec(ctx, `UPDATE payment_orders SET transaction_reference='CHANGED101' WHERE id=101`); err != nil {
		t.Fatal(err)
	}
	if err = a.paymentAdmin(ctx, 402, 999, "/upgradepayment", []string{"/upgradepayment", "101"}); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT status FROM payment_orders WHERE id=101`).Scan(&status); err != nil || status != "pending_review" {
		t.Fatal("stale confirmation was accepted", err)
	}
	if err = a.paymentAdmin(ctx, 403, 999, "/approvepayment", []string{"/approvepayment", "101"}); err != nil {
		t.Fatal(err)
	}
	if err = a.paymentAdmin(ctx, 404, 999, "/upgradepayment", []string{"/upgradepayment", "101"}); err != nil {
		t.Fatal(err)
	}
	var limit, used int
	if err = db.QueryRow(ctx, `SELECT plan_name,daily_limit,(SELECT count(*) FROM search_requests WHERE telegram_id=1 AND state='charged' AND billing_kind='paid') FROM bot_users WHERE telegram_id=1`).Scan(&plan, &limit, &used); err != nil || plan != "New" || limit != 10 || used != 3 {
		t.Fatalf("upgrade lost usage: %s %d %d %v", plan, limit, used, err)
	}
	if err = a.handleCallback(ctx, "cb1", 2, "editref:102"); err != nil {
		t.Fatal(err)
	}
	if err = a.stageReferenceEdit(ctx, 2, 102, "newref102"); err != nil {
		t.Fatal(err)
	}
	var ref string
	if err = db.QueryRow(ctx, `SELECT transaction_reference FROM payment_orders WHERE id=102`).Scan(&ref); err != nil || ref != "OLDREF102" {
		t.Fatal("reference changed before confirmation", err)
	}
	if err = a.stageReferenceEdit(ctx, 2, 102, "OTHERREF1"); err != nil {
		t.Fatal(err)
	}
	if err = a.handleCallback(ctx, "stale", 2, "confirmref:102:"+a.referenceConfirmToken(2, 102, "NEWREF102")); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT transaction_reference FROM payment_orders WHERE id=102`).Scan(&ref); err != nil || ref != "OLDREF102" {
		t.Fatal("stale reference confirmation accepted", err)
	}
	if err = a.stageReferenceEdit(ctx, 2, 102, "NEWREF102"); err != nil {
		t.Fatal(err)
	}
	if err = a.handleCallback(ctx, "cb2", 2, "confirmref:102:"+a.referenceConfirmToken(2, 102, "NEWREF102")); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT transaction_reference FROM payment_orders WHERE id=102`).Scan(&ref); err != nil || ref != "NEWREF102" {
		t.Fatal("reference not saved", err)
	}
	var audit int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM payment_reference_audit WHERE order_id=102`).Scan(&audit); err != nil || audit != 1 {
		t.Fatal("reference audit missing", err)
	}
	if err = a.handleCallback(ctx, "old1", 3, "editref:103"); err != nil {
		t.Fatal(err)
	}
	if err = a.stageReferenceEdit(ctx, 3, 103, "OLDREF102"); err != nil {
		t.Fatal(err)
	}
	if err = a.handleCallback(ctx, "old2", 3, "confirmref:103:"+a.referenceConfirmToken(3, 103, "OLDREF102")); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT transaction_reference FROM payment_orders WHERE id=103`).Scan(&ref); err != nil || ref != "TAKENREF1" {
		t.Fatal("previously used reference was reused", err)
	}
	if err = a.handleCallback(ctx, "cb3", 3, "editref:102"); err != nil {
		t.Fatal(err)
	}
	if err = a.handleCallback(ctx, "cb4", 2, "editref:102"); err != nil {
		t.Fatal(err)
	}
	if err = a.stageReferenceEdit(ctx, 2, 102, "TAKENREF1"); err != nil {
		t.Fatal(err)
	}
	if err = a.handleCallback(ctx, "cb5", 2, "confirmref:102:"+a.referenceConfirmToken(2, 102, "TAKENREF1")); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT transaction_reference FROM payment_orders WHERE id=102`).Scan(&ref); err != nil || ref != "NEWREF102" {
		t.Fatal("duplicate reference accepted", err)
	}
	if err = a.stageReferenceEdit(ctx, 2, 102, "UNIQUE102"); err != nil {
		t.Fatal(err)
	}
	if err = a.paymentAdmin(ctx, 405, 999, "/reviewpayment", []string{"/reviewpayment", "102"}); err != nil {
		t.Fatal(err)
	}
	if err = a.handleCallback(ctx, "locked", 2, "confirmref:102:"+a.referenceConfirmToken(2, 102, "UNIQUE102")); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT transaction_reference FROM payment_orders WHERE id=102`).Scan(&ref); err != nil || ref != "NEWREF102" {
		t.Fatal("reference changed after review started", err)
	}
	var pending *int64
	if err = db.QueryRow(ctx, `SELECT pending_ref_order FROM bot_users WHERE telegram_id=2`).Scan(&pending); err != nil || pending != nil {
		t.Fatal("locked review left user stuck in edit mode", err)
	}
	if _, err = db.Exec(ctx, `UPDATE payment_orders SET status='approved' WHERE id=102`); err != nil {
		t.Fatal(err)
	}
	if err = a.handleCallback(ctx, "cb6", 2, "confirmref:102:"+a.referenceConfirmToken(2, 102, "TAKENREF1")); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(ctx, `SELECT transaction_reference FROM payment_orders WHERE id=102`).Scan(&ref); err != nil || ref != "NEWREF102" {
		t.Fatal("approved reference changed", err)
	}
	if _, err = db.Exec(ctx, `INSERT INTO bot_users(telegram_id) VALUES(4),(5); INSERT INTO payment_orders(id,create_update_id,user_id,plan_name,amount_paise,daily_limit,plan_days,status,transaction_reference) VALUES(104,204,4,'New',10000,10,30,'pending_review','ORIGINAL4'),(105,205,5,'New',10000,10,30,'pending_review','ORIGINAL5')`); err != nil {
		t.Fatal(err)
	}
	for _, x := range []struct{ uid, id int64 }{{4, 104}, {5, 105}} {
		if err = a.handleCallback(ctx, "start", x.uid, fmt.Sprintf("editref:%d", x.id)); err != nil {
			t.Fatal(err)
		}
		if err = a.stageReferenceEdit(ctx, x.uid, x.id, "SAME12345"); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, x := range []struct{ uid, id int64 }{{4, 104}, {5, 105}} {
		wg.Add(1)
		go func(uid, id int64) {
			defer wg.Done()
			errs <- a.handleCallback(ctx, "confirm", uid, fmt.Sprintf("confirmref:%d:%s", id, a.referenceConfirmToken(uid, id, "SAME12345")))
		}(x.uid, x.id)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	var shared int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM payment_orders WHERE transaction_reference='SAME12345'`).Scan(&shared); err != nil || shared != 1 {
		t.Fatalf("reference accepted by %d orders: %v", shared, err)
	}
}
