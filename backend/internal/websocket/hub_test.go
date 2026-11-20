package websocket

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"notorious-backend/internal/models"
)

func TestSendToEveryConnectionAndUnregister(t *testing.T) {
	user := uuid.New()
	first := &Client{userID: user, send: make(chan []byte, 1)}
	second := &Client{userID: user, send: make(chan []byte, 1)}
	hub := NewHub(nil, nil)
	hub.clients[user] = map[*Client]struct{}{first: {}, second: {}}
	message := &models.WSEnvelope{Type: models.WSTypeNewMessage, Payload: json.RawMessage(`{}`)}
	if !hub.SendToUser(user, message) {
		t.Fatal("expected delivery")
	}
	for _, client := range []*Client{first, second} {
		select {
		case <-client.send:
		default:
			t.Fatal("a connected tab did not receive the message")
		}
	}
	hub.Unregister(first)
	if !hub.IsOnline(user) {
		t.Fatal("user went offline while another tab was connected")
	}
	if !hub.SendToUser(user, message) {
		t.Fatal("remaining tab did not receive message")
	}
	hub.Unregister(second)
	if hub.IsOnline(user) {
		t.Fatal("user remains online with no connections")
	}
	if hub.SendToUser(user, message) {
		t.Fatal("delivered to an offline user")
	}
}
