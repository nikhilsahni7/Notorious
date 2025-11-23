package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"notorious-backend/internal/database"
	"notorious-backend/internal/models"
	"notorious-backend/internal/repository"
	"notorious-backend/internal/utils"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := database.NewPostgresDB(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.RunMigrations("./migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// Initialize GeoIP (optional)
	utils.InitGeoIP("./GeoLite2-City.mmdb")

	ctx := context.Background()

	// 1. Create Test User
	email := fmt.Sprintf("test_device_%d@example.com", time.Now().Unix())
	user := &models.User{
		Email:            email,
		PasswordHash:     "hash",
		Name:             "Test User",
		Role:             models.RoleUser,
		Region:           "pan-india",
		DailySearchLimit: 100,
		DeviceLimit:      1,
		IsActive:         true,
	}

	log.Printf("Creating test user: %s", email)
	if err := userRepo.Create(ctx, user); err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	defer func() {
		// Cleanup
		log.Println("Cleaning up...")
		sessionRepo.DeleteAllForUser(ctx, user.ID)
		userRepo.Delete(ctx, user.ID)
	}()

	// 2. Simulate Login 1 (Should succeed)
	log.Println("Simulating Login 1...")
	count, _ := sessionRepo.CountActiveSessions(ctx, user.ID)
	if count >= user.DeviceLimit {
		log.Fatalf("Unexpected session count: %d", count)
	}

	session1 := &models.UserSession{
		UserID:     user.ID,
		TokenHash:  "token1",
		DeviceName: "Device 1",
		DeviceOS:   "Linux",
		DeviceType: "Desktop",
		IPAddress:  "127.0.0.1",
		LastActive: time.Now(),
		CreatedAt:  time.Now(),
	}
	if err := sessionRepo.Create(ctx, session1); err != nil {
		log.Fatalf("Failed to create session 1: %v", err)
	}
	log.Println("Login 1 successful")

	// 3. Simulate Login 2 (Should fail check)
	log.Println("Simulating Login 2...")
	count, _ = sessionRepo.CountActiveSessions(ctx, user.ID)
	log.Printf("Active sessions: %d, Limit: %d", count, user.DeviceLimit)

	if count >= user.DeviceLimit {
		log.Println("Login 2 correctly blocked by limit check")
	} else {
		log.Fatalf("Login 2 should have been blocked! Count: %d, Limit: %d", count, user.DeviceLimit)
	}

	// 4. Update Limit to 2
	log.Println("Updating limit to 2...")
	user.DeviceLimit = 2
	if err := userRepo.Update(ctx, user); err != nil {
		log.Fatalf("Failed to update user limit: %v", err)
	}

	// 5. Simulate Login 2 again (Should succeed now)
	log.Println("Simulating Login 2 again...")
	count, _ = sessionRepo.CountActiveSessions(ctx, user.ID)
	if count >= user.DeviceLimit {
		log.Fatalf("Login 2 blocked unexpectedly after limit increase! Count: %d, Limit: %d", count, user.DeviceLimit)
	}

	session2 := &models.UserSession{
		UserID:     user.ID,
		TokenHash:  "token2",
		DeviceName: "Device 2",
		DeviceOS:   "Android",
		DeviceType: "Mobile",
		IPAddress:  "127.0.0.1",
		LastActive: time.Now(),
		CreatedAt:  time.Now(),
	}
	if err := sessionRepo.Create(ctx, session2); err != nil {
		log.Fatalf("Failed to create session 2: %v", err)
	}
	log.Println("Login 2 successful")

	// 6. Verify Session List
	sessions, err := sessionRepo.GetActiveSessions(ctx, user.ID)
	if err != nil {
		log.Fatalf("Failed to get sessions: %v", err)
	}
	log.Printf("Active sessions found: %d", len(sessions))
	if len(sessions) != 2 {
		log.Fatalf("Expected 2 sessions, found %d", len(sessions))
	}

	log.Println("Verification PASSED!")
}
