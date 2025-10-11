package scheduler

import (
	"context"
	"log"
	"time"

	"notorious-backend/internal/repository"
)

type SearchLimitResetter struct {
	userRepo    *repository.UserRepository
	istLocation *time.Location
}

func NewSearchLimitResetter(userRepo *repository.UserRepository) *SearchLimitResetter {
	ist, _ := time.LoadLocation("Asia/Kolkata")
	return &SearchLimitResetter{
		userRepo:    userRepo,
		istLocation: ist,
	}
}

func (s *SearchLimitResetter) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	log.Println("Search limit resetter started")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Search limit resetter stopped")
				return
			case <-ticker.C:
				s.checkAndReset()
			}
		}
	}()

	s.checkAndReset()
}

func (s *SearchLimitResetter) checkAndReset() {
	now := time.Now().In(s.istLocation)
	
	if now.Hour() == 0 && now.Minute() < 5 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.userRepo.ResetAllDailyLimits(ctx); err != nil {
			log.Printf("Failed to reset daily limits: %v", err)
			return
		}

		log.Printf("Successfully reset daily search limits at %s IST", now.Format("2006-01-02 15:04:05"))
	}
}

