package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"notorious-backend/internal/models"

	"github.com/SherClockHolmes/webpush-go"
)

// PushSender delivers VAPID web push messages (RFC 8291/8030) using
// webpush-go. Keys come from VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY env vars —
// never hardcoded, and never committed.
type PushSender struct {
	publicKey  string
	privateKey string
	client     *http.Client
}

func NewPushSender() *PushSender {
	return &PushSender{
		publicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		privateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		client:     &http.Client{Timeout: 15 * time.Second},
	}
}

// Enabled reports whether web push is configured on this server. When it is
// not, notification rows are still written (bell stays functional) and
// subscribe attempts fail gracefully at the /api/push/public-key endpoint.
func (s *PushSender) Enabled() bool {
	return s.publicKey != "" && s.privateKey != ""
}

// PublicKey exposes the VAPID public key to the browser for PushManager.subscribe.
func (s *PushSender) PublicKey() string {
	return s.publicKey
}

// sendPayload carries the web push message to the service worker.
type sendPayload struct {
	Title   string                 `json:"title"`
	Body    string                 `json:"body"`
	URL     string                 `json:"url"`
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// SendTo delivers an OS notification to one subscription.
// Returns stale=true when the endpoint is gone (410 Gone / 404) so the
// caller can clean up the row.
func (s *PushSender) Send(ctx context.Context, sub *models.PushSubscription, title, body, url, ntype string, payload map[string]interface{}) (stale bool, err error) {
	if !s.Enabled() {
		return false, fmt.Errorf("web push not configured")
	}

	data, err := json.Marshal(sendPayload{Title: title, Body: body, URL: url, Type: ntype, Payload: payload})
	if err != nil {
		return false, fmt.Errorf("failed to marshal push payload: %w", err)
	}

	resp, err := webpush.SendNotificationWithContext(ctx, data, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}, &webpush.Options{
		HTTPClient:      s.client,
		Subscriber:      s.subscriber(),
		TTL:             24 * 60 * 60, // keep at the push service for a day
		Urgency:         webpush.UrgencyHigh,
		VAPIDPublicKey:  s.publicKey,
		VAPIDPrivateKey: s.privateKey,
	})
	if err != nil {
		// webpush-go surfaces non-2xx as errors; classify by response if present
		if resp != nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
				return true, nil
			}
		}
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		return true, nil
	}
	if resp.StatusCode >= 400 {
		return false, fmt.Errorf("push service returned status %d", resp.StatusCode)
	}
	return false, nil
}

// subscriber returns the VAPID "sub" claim contact for the JWT.
func (s *PushSender) subscriber() string {
	if subject := os.Getenv("VAPID_SUBJECT"); subject != "" {
		return subject
	}
	return "mailto:admin@knotorious.us"
}

// GenerateVAPIDKeys prints a freshly generated VAPID keypair. Intent: an
// operator runs this once locally (e.g. `go run ./cmd/vapid`) and stores the
// values in the deployment env. Keys are never committed.
func GenerateVAPIDKeys() error {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return err
	}
	log.Printf("VAPID_PUBLIC_KEY=%s", publicKey)
	log.Printf("VAPID_PRIVATE_KEY=%s", privateKey)
	log.Println("Store these in the deployment environment (.env is git-ignored). Do not commit them.")
	return nil
}
