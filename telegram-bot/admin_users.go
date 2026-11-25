package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func userState(approved, blocked, active bool) string {
	if blocked {
		return "Blocked"
	}
	if !approved {
		return "Pending"
	}
	if !active {
		return "Expired"
	}
	return "Active"
}
func (a *app) adminUsers(ctx context.Context, actor int64, p []string) error {
	if !a.admins[actor] {
		return a.send(ctx, actor, "This action is not available.")
	}
	if len(p) > 2 {
		return a.send(ctx, actor, "Use /users or /users PAGE.")
	}
	page := 1
	var err error
	if len(p) == 2 {
		page, err = strconv.Atoi(p[1])
		if err != nil || page < 1 || page > 10000000 {
			return a.send(ctx, actor, "Enter a valid page number.")
		}
	}
	rows, err := a.db.Query(ctx, `SELECT telegram_id,username,plan_name,region,approved,blocked,COALESCE(plan_expires_at>now(),false) FROM bot_users ORDER BY created_at DESC,telegram_id DESC LIMIT 11 OFFSET $1`, (page-1)*10)
	if err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Users · page %d\n\n", page)
	n := 0
	more := false
	for rows.Next() {
		var id int64
		var username, plan, region string
		var approved, blocked, active bool
		if err = rows.Scan(&id, &username, &plan, &region, &approved, &blocked, &active); err != nil {
			rows.Close()
			return err
		}
		if n == 10 {
			more = true
			break
		}
		if username == "" {
			username = "No username"
		} else {
			username = "@" + cleanResultField(username)
		}
		fmt.Fprintf(&b, "%d · %s\n%s · %s\n/user %d\n\n", id, username, plan, userState(approved, blocked, active), id)
		n++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if n == 0 {
		b.WriteString("No users on this page.")
	} else if more {
		fmt.Fprintf(&b, "Next: /users %d", page+1)
	}
	if _, err = a.db.Exec(ctx, `INSERT INTO admin_audit(admin_id,target_id,action,detail) VALUES($1,0,'/users',$2)`, actor, fmt.Sprintf("page=%d", page)); err != nil {
		return err
	}
	return a.send(ctx, actor, b.String())
}
func (a *app) adminUser(ctx context.Context, actor int64, p []string) error {
	if !a.admins[actor] {
		return a.send(ctx, actor, "This action is not available.")
	}
	if len(p) != 2 {
		return a.send(ctx, actor, "Use /user USER_ID.")
	}
	target, err := strconv.ParseInt(p[1], 10, 64)
	if err != nil || target <= 0 {
		return a.send(ctx, actor, "Enter a valid Telegram user ID.")
	}
	var username, plan, region, created, expires string
	var approved, blocked, active bool
	var limit, used, total, pending int
	err = a.db.QueryRow(ctx, `SELECT username,plan_name,region,approved,blocked,COALESCE(plan_expires_at>now(),false),daily_limit,to_char(created_at AT TIME ZONE 'Asia/Kolkata','DD Mon YYYY HH24:MI'),COALESCE(to_char(plan_expires_at AT TIME ZONE 'Asia/Kolkata','DD Mon YYYY HH24:MI'),'Not assigned'),(SELECT count(*) FROM search_requests r WHERE r.telegram_id=$1 AND r.quota_day=(clock_timestamp() AT TIME ZONE 'Asia/Kolkata')::date AND r.state IN ('reserved','charged','uncertain') AND r.created_at>COALESCE((SELECT reset_at FROM quota_resets q WHERE q.telegram_id=$1 AND q.quota_day=r.quota_day),'-infinity'::timestamptz)),(SELECT count(*) FROM search_requests WHERE telegram_id=$1),(SELECT count(*) FROM payment_orders WHERE user_id=$1 AND status='pending_review') FROM bot_users WHERE telegram_id=$1`, target).Scan(&username, &plan, &region, &approved, &blocked, &active, &limit, &created, &expires, &used, &total, &pending)
	if errors.Is(err, pgx.ErrNoRows) {
		return a.send(ctx, actor, "User not found.")
	}
	if err != nil {
		return err
	}
	if username == "" {
		username = "Not set"
	} else {
		username = "@" + cleanResultField(username)
	}
	available := max(0, limit-used)
	if blocked || !approved || !active {
		available = 0
	}
	text := fmt.Sprintf("User profile\n\nTelegram ID: %d\nUsername: %s\nStatus: %s\nPlan: %s\nJoined: %s IST\nPlan expires: %s IST\nDaily allowance: %d\nUsed today: %d\nAvailable today: %d\nRecorded searches: %d\nPending payment reviews: %d\n\nSearch history: /userhistory %d\nReset allowance: /resetquota %d\nBlock: /block %d", target, username, userState(approved, blocked, active), plan, created, expires, limit, used, available, total, pending, target, target, target)
	if _, err = a.db.Exec(ctx, `INSERT INTO admin_audit(admin_id,target_id,action) VALUES($1,$2,'/user')`, actor, target); err != nil {
		return err
	}
	return a.send(ctx, actor, text)
}
