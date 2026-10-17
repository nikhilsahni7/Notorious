package services

import (
	"context"
	"log"
	"time"

	"notorious-backend/internal/models"
	"notorious-backend/internal/repository"

	"github.com/google/uuid"
)

const (
	// maxUserPushesPerDay caps OS-level pushes per user per IST day (non-exempt types).
	maxUserPushesPerDay = 2
	// lastSeenThreshold determines presence: a user with a session active in the
	// last 5 minutes is considered to have the app open, so OS push is skipped
	// and only the in-app bell row surfaces the notification.
	lastSeenThresholdSeconds = 5 * 60
	// perUserTimeout bounds a single user's delivery (row write + pushes).
	perUserTimeout = 20 * time.Second
)

// NotificationService writes in-app notification rows and, when appropriate,
// delivers them as OS-level web pushes.
//
// Delivery rules:
//   - The in-app row is always written (the bell is the fallback channel).
//   - OS push is skipped when: user is online (active session), push is
//     disabled by the user's prefs, VAPID is not configured, the user has no
//     push subscriptions, or the daily push cap is exhausted.
//   - quota_reset is a single scheduled push and is exempt from the cap.
//
// TODO(deferred triggers) — wire these into deliver() once their features land:
//
//   - Trigger B (watchlist_match): no saved_searches table exists yet, so the
//     nightly re-run is intentionally NOT implemented here (no half state).
//     Design: a scheduler job analogous to QuotaResetNotifier iterates active
//     opt-in users' saved searches at ~05:00 IST, executes each through the
//     existing OpenSearch service path, and creates ONE batched
//     watchlist_match row per user per day (payload: {url: "/search?q=…"}),
//     then delivers through deliver() → presence/cap/push rules apply.
//
//   - Trigger C (export_ready): there is no async export/bulk job today
//     (ExportEODReport is synchronous CSV). When an async export lands, the
//     worker that finalizes the file should call deliver() with
//     NotificationTypeExportReady and payload {url: <download_url>}.
type NotificationService struct {
	notificationRepo *repository.NotificationRepository
	pushSubRepo      *repository.PushSubscriptionRepository
	sessionRepo      *repository.SessionRepository
	pushSender       *PushSender
	istLocation      *time.Location
}

func NewNotificationService(
	notificationRepo *repository.NotificationRepository,
	pushSubRepo *repository.PushSubscriptionRepository,
	sessionRepo *repository.SessionRepository,
	pushSender *PushSender,
) *NotificationService {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Printf("Warning: Could not load Asia/Kolkata timezone: %v. Falling back to Local.", err)
		ist = time.Local
	}
	return &NotificationService{
		notificationRepo: notificationRepo,
		pushSubRepo:      pushSubRepo,
		sessionRepo:      sessionRepo,
		pushSender:       pushSender,
		istLocation:      ist,
	}
}

// deepLink returns the payload URL for the notification.
func deepLink(payload map[string]interface{}) string {
	if payload == nil {
		return "/search"
	}
	if url, ok := payload["url"].(string); ok && url != "" {
		return url
	}
	return "/search"
}

// NotifyQuotaReset delivers a quota-reset notification to a single user.
// Used for ad-hoc drives; the scheduler uses NotifyQuotaResetBatch.
// It is exempt from the daily push cap (single scheduled push) but still
// respects in-session suppression.
func (s *NotificationService) NotifyQuotaReset(ctx context.Context, userID uuid.UUID) error {
	return s.NotifyQuotaResetBatch(ctx, []uuid.UUID{userID})
}

// NotifyQuotaResetBatch delivers to many opted-in users with one presence
// lookup. Each user gets a bounded deadline so one slow endpoint cannot
// exhaust the budget for the rest of the batch.
func (s *NotificationService) NotifyQuotaResetBatch(ctx context.Context, userIDs []uuid.UUID) error {
	online := s.onlineSet(ctx)

	for _, userID := range userIDs {
		// Idempotence: skip if a quota_reset notification row already exists
		// for this IST day (covers scheduler restarts and repeated calls).
		exists, err := s.notificationRepo.HasTypeToday(ctx, userID, models.NotificationTypeQuotaReset, s.istLocation)
		if err != nil {
			log.Printf("Failed duplicate check for user %s: %v", userID, err)
			continue
		}
		if exists {
			continue
		}

		userCtx, cancel := context.WithTimeout(ctx, perUserTimeout)
		n := &models.Notification{
			UserID: userID,
			Type:   models.NotificationTypeQuotaReset,
			Title:  "Search quota reset",
			Body:   "Your daily search quota has been reset. You're good to go again!",
			Payload: map[string]interface{}{
				"url": "/search",
			},
		}
		// Presence suppression: skip the OS push when the user is online.
		if err := s.deliver(userCtx, n, true, online); err != nil {
			log.Printf("Failed to deliver quota-reset notification for user %s: %v", userID, err)
		}
		cancel()
	}
	return nil
}

// deliver writes the row and evaluates OS push eligibility. `online` is the
// pre-resolved presence set (nil = no presence data; push allowed).
func (s *NotificationService) deliver(ctx context.Context, n *models.Notification, pushExempt bool, online map[uuid.UUID]bool) error {
	// Always write the in-app row first — the bell is the fallback channel.
	if err := s.notificationRepo.Create(ctx, n); err != nil {
		return err
	}

	if !s.pushSender.Enabled() {
		return nil
	}

	if online != nil && online[n.UserID] {
		return nil // in-session suppression — bell surfaces it
	}

	if !pushExempt {
		count, err := s.notificationRepo.CountPushedToday(ctx, n.UserID, s.istLocation)
		if err != nil {
			log.Printf("notification push cap check failed for user %s: %v", n.UserID, err)
			return nil // in-app row already saved
		}
		if count >= maxUserPushesPerDay {
			return nil // cap reached — in-app only
		}
	}

	subs, err := s.pushSubRepo.ListByUser(ctx, n.UserID)
	if err != nil {
		log.Printf("failed to list push subscriptions for user %s: %v", n.UserID, err)
		return nil // in-app row already saved
	}
	// No subscriptions: bell-only (batched delivery skips these users).
	if len(subs) == 0 {
		return nil
	}

	url := deepLink(n.Payload)
	pushed := false
	for i := range subs {
		sub := &subs[i]
		stale, err := s.pushSender.Send(ctx, sub, n.Title, n.Body, url, n.Type, n.Payload)
		if stale {
			// 410 Gone (endpoint expired, browser uninstalled, etc.) — clean up.
			if err := s.pushSubRepo.DeleteByEndpoint(ctx, sub.Endpoint, nil); err != nil {
				log.Printf("failed to delete stale push subscription %s: %v", trunc(sub.Endpoint, 48), err)
			}
			continue
		}
		if err != nil {
			log.Printf("web push to endpoint for user %s failed: %v", n.UserID, err)
			continue
		}
		pushed = true
	}

	if pushed {
		if err := s.notificationRepo.MarkPushed(ctx, n.ID); err != nil {
			log.Printf("failed to stamp pushed_at on notification %s: %v", n.ID, err)
		}
	}
	return nil
}

// onlineSet resolves the presence map once per delivery run. A user is
// "online" when any of their sessions was active within lastSeenThreshold.
func (s *NotificationService) onlineSet(ctx context.Context) map[uuid.UUID]bool {
	set := make(map[uuid.UUID]bool)
	if s.sessionRepo == nil {
		return nil
	}
	online, err := s.sessionRepo.GetOnlineUserIDs(ctx, lastSeenThresholdSeconds)
	if err != nil {
		// Cannot confirm presence -> treat as "no one online" is unsafe for
		// duplicate avoidance; instead return nil so the caller pushes.
		// (Trigger A runs once a day; a rare double-notify with an open tab
		// is acceptable when presence data is unavailable.)
		return nil
	}
	for _, id := range online {
		set[id] = true
	}
	return set
}

// trunc safely shortens a string for log lines (no runtime-sensitive slicing).
func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
