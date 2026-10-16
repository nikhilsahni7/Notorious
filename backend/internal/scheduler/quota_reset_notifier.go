package scheduler

import (
	"context"
	"log"
	"time"

	"notorious-backend/internal/repository"
	"notorious-backend/internal/services"
)

// QuotaResetNotifier informs opted-in users once per day about their reset
// quota, at midnight IST (00:00), the same window in which limits reset.
type QuotaResetNotifier struct {
	userRepo         *repository.UserRepository
	notificationRepo *repository.NotificationRepository
	notifSvc         *services.NotificationService
	istLocation      *time.Location
}

func NewQuotaResetNotifier(userRepo *repository.UserRepository, notificationRepo *repository.NotificationRepository, notifSvc *services.NotificationService) *QuotaResetNotifier {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Printf("Warning: Could not load Asia/Kolkata timezone: %v. Falling back to Local.", err)
		ist = time.Local
	}
	return &QuotaResetNotifier{
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		notifSvc:         notifSvc,
		istLocation:      ist,
	}
}

// Start ticks hourly like SearchLimitResetter and delivers at the first
// tick after 00:00 IST each day, the same window in which the daily limits
// are reset. The loop runs in a goroutine so Start returns immediately and
// never blocks main before the router binds.
func (n *QuotaResetNotifier) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	log.Println("Quota reset notifier started (delivery at midnight IST)")

	go func() {
		// First check immediately (covers restarts mid-window).
		n.deliverIfDue()

		for {
			select {
			case <-ctx.Done():
				log.Println("Quota reset notifier stopped")
				return
			case <-ticker.C:
				n.deliverIfDue()
			}
		}
	}()
}

func (n *QuotaResetNotifier) deliverIfDue() {
	now := time.Now().In(n.istLocation)

	// Fire in the 00:00-00:05 IST window (same slice SearchLimitResetter
	// uses for the limit reset). With an hourly ticker the exact offset
	// depends on boot time; HasTypeToday makes every tick within the window
	// idempotent — delivery happens exactly once per IST day.
	if now.Hour() != 0 || now.Minute() >= 5 {
		return
	}

	// Generous budget for the whole batch; each user's delivery gets its own
	// shorter timeout inside the service.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	userIDs, err := n.userRepo.GetQuotaResetOptInIDs(ctx)
	if err != nil {
		log.Printf("Failed to list quota-reset opt-ins: %v", err)
		return
	}
	if len(userIDs) == 0 {
		return
	}

	if err := n.notifSvc.NotifyQuotaResetBatch(ctx, userIDs); err != nil {
		log.Printf("Quota-reset batch delivery failed: %v", err)
		return
	}

	log.Printf("Delivered quota-reset notifications to %d opt-in user(s) at %s IST (midnight reset)", len(userIDs), now.Format("2006-01-02 15:04:05"))
}
