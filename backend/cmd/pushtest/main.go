// Manual web-push tester (local testing + stale-endpoint verification).
//
// Usage (from backend/, with DATABASE_URL + VAPID_* in .env or env):
//
//	go run ./cmd/pushtest -email someone@example.com
//	go run ./cmd/pushtest -email someone@example.com -url "/search?q=test" -list
//
// It sends ONE OS push to every stored subscription of the user and prints
// per-endpoint results. A 410 Gone removes the stale subscription row (same
// cleanup path the scheduler uses), so `... -email ... ; re-run` proves
// cleanup works. Because frequency caps/in-session suppression live in
// services.NotificationService.deliver, this CLI deliberately bypasses them —
// it is a raw pipeline test, not a user-facing notification.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"notorious-backend/internal/database"
	"notorious-backend/internal/repository"
	"notorious-backend/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	email := flag.String("email", "", "user email to send the test push to (required)")
	url := flag.String("url", "/search", "deep link carried by the push payload")
	title := flag.String("title", "Pushtest notification", "push title")
	body := flag.String("body", "Delivered via cmd/pushtest", "push body")
	list := flag.Bool("list", false, "only list the user's stored subscriptions; do not send")
	flag.Parse()

	if *email == "" {
		log.Fatal("usage: go run ./cmd/pushtest -email someone@example.com [-url /search]")
	}
	if err := run(*email, *url, *title, *body, *list); err != nil {
		log.Fatal(err)
	}
}

func run(email, url, title, body string, listOnly bool) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	db, err := database.NewPostgresDB(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Pool.Close()

	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.GetByEmail(context.Background(), email)
	if err != nil {
		return fmt.Errorf("user lookup failed for %s: %w", email, err)
	}

	subRepo := repository.NewPushSubscriptionRepository(db)
	subs, err := subRepo.ListByUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to load subscriptions: %w", err)
	}

	fmt.Printf("user %s (%s): %d stored subscription(s)\n", user.Email, user.ID, len(subs))
	for _, s := range subs {
		fmt.Printf("  endpoint: %s... device=%s created=%s\n", s.Endpoint[:min(48, len(s.Endpoint))], s.DeviceType, s.CreatedAt.Format(time.RFC3339))
	}
	if listOnly {
		return nil
	}

	sender := services.NewPushSender()
	if !sender.Enabled() {
		return fmt.Errorf("web push not configured: set VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY (generate with `go run ./cmd/vapid`)")
	}

	stale := 0
	sent := 0
	for i := range subs {
		sub := &subs[i]
		isStale, err := sender.Send(context.Background(), sub, title, body, url, "quota_warning", map[string]interface{}{
			"url": url,
		})
		if isStale {
			stale++
			fmt.Printf("  STALE (410/404): %s... removing row\n", sub.Endpoint[:min(48, len(sub.Endpoint))])
			userID := user.ID
			if err := subRepo.DeleteByEndpoint(context.Background(), sub.Endpoint, &userID); err != nil {
				return fmt.Errorf("failed to delete stale subscription: %w", err)
			}
			continue
		}
		if err != nil {
			fmt.Printf("  ERROR: %s... %v\n", sub.Endpoint[:min(48, len(sub.Endpoint))], err)
			continue
		}
		sent++
		fmt.Printf("  sent: %s...\n", sub.Endpoint[:min(48, len(sub.Endpoint))])
	}
	fmt.Printf("done: %d sent, %d stale removed, %d errors\n", sent, stale, len(subs)-sent-stale)
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
