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

func TestBroadcastExceptSkipsEveryConnectionOfExcludedUser(t *testing.T) {
	included, excluded := uuid.New(), uuid.New()
	receiver := &Client{userID: included, send: make(chan []byte, 1)}
	excludedFirst := &Client{userID: excluded, send: make(chan []byte, 1)}
	excludedSecond := &Client{userID: excluded, send: make(chan []byte, 1)}
	hub := NewHub(nil, nil)
	hub.clients[included] = map[*Client]struct{}{receiver: {}}
	hub.clients[excluded] = map[*Client]struct{}{excludedFirst: {}, excludedSecond: {}}
	hub.BroadcastExcept(&models.WSEnvelope{Type: models.WSTypeBroadcast, Payload: json.RawMessage(`{}`)}, map[uuid.UUID]struct{}{excluded: {}})
	select {
	case <-receiver.send:
	default:
		t.Fatal("included user did not receive broadcast")
	}
	for _, client := range []*Client{excludedFirst, excludedSecond} {
		select {
		case <-client.send:
			t.Fatal("excluded user received broadcast")
		default:
		}
	}
}
