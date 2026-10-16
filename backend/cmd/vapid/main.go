// Generates a VAPID keypair for web push and prints it for the deployment
// environment. Run locally once: go run ./cmd/vapid — then set
// VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY (optionally VAPID_SUBJECT) in the
// environment. Never commit the keys or .env files.
package main

import (
	"notorious-backend/internal/services"
)

func main() {
	if err := services.GenerateVAPIDKeys(); err != nil {
		panic(err)
	}
}
