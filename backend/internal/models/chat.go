package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Message status constants
const (
	MessageStatusSent      = "sent"
	MessageStatusDelivered = "delivered"
	MessageStatusRead      = "read"
)

// WebSocket event types
const (
	WSTypeChatMessage    = "chat_message"
	WSTypeMessageAck     = "message_ack"
	WSTypeNewMessage     = "new_message"
	WSTypeTypingStart    = "typing_start"
	WSTypeTypingStop     = "typing_stop"
	WSTypeReadReceipt    = "read_receipt"
	WSTypeReadUpdate     = "read_update"
	WSTypeOnlineStatus   = "online_status"
	WSTypeDeliveryUpdate = "delivery_update"
	WSTypeBroadcast      = "broadcast"
	WSTypeBroadcastRead  = "broadcast_read"
	WSTypeError          = "error"
)

// ChatMessage represents a single chat message
type ChatMessage struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	SenderID       uuid.UUID  `json:"sender_id"`
	ReceiverID     uuid.UUID  `json:"receiver_id"`
	Content        string     `json:"content"`
	SequenceNum    int64      `json:"sequence_num"`
	Status         string     `json:"status"`
	SentAt         time.Time  `json:"sent_at"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
}

// ChatConversation represents a conversation between two users
type ChatConversation struct {
	ID            uuid.UUID    `json:"id"`
	ParticipantA  uuid.UUID    `json:"participant_a"`
	ParticipantB  uuid.UUID    `json:"participant_b"`
	LastMessageID *uuid.UUID   `json:"last_message_id,omitempty"`
	LastMessageAt time.Time    `json:"last_message_at"`
	CreatedAt     time.Time    `json:"created_at"`
}

// ConversationWithDetails is used for API responses with joined data
type ConversationWithDetails struct {
	ID            uuid.UUID    `json:"id"`
	OtherUserID   uuid.UUID    `json:"other_user_id"`
	OtherUserName string       `json:"other_user_name"`
	OtherUserEmail string      `json:"other_user_email"`
	OtherUserRole string       `json:"other_user_role"`
	LastMessage   *string      `json:"last_message,omitempty"`
	LastMessageAt time.Time    `json:"last_message_at"`
	LastSenderID  *uuid.UUID   `json:"last_sender_id,omitempty"`
	UnreadCount   int          `json:"unread_count"`
	IsOnline      bool         `json:"is_online"`
}

// ChatBroadcast represents an admin broadcast message
type ChatBroadcast struct {
	ID       uuid.UUID `json:"id"`
	SenderID uuid.UUID `json:"sender_id"`
	Content  string    `json:"content"`
	SentAt   time.Time `json:"sent_at"`
}

// ChatBroadcastWithStatus includes read status for a specific user
type ChatBroadcastWithStatus struct {
	ID         uuid.UUID  `json:"id"`
	SenderID   uuid.UUID  `json:"sender_id"`
	SenderName string     `json:"sender_name"`
	Content    string     `json:"content"`
	SentAt     time.Time  `json:"sent_at"`
	IsRead     bool       `json:"is_read"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
}

// WSEnvelope is the wrapper for all WebSocket messages
type WSEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// WSChatMessage is sent by client to send a new message
type WSChatMessage struct {
	ReceiverID string `json:"receiver_id"`
	Content    string `json:"content"`
	TempID     string `json:"temp_id"` // Client-generated ID for tracking
}

// WSMessageAck is sent by server to confirm message saved
type WSMessageAck struct {
	TempID      string    `json:"temp_id"`
	MessageID   uuid.UUID `json:"message_id"`
	SequenceNum int64     `json:"sequence_num"`
	SentAt      time.Time `json:"sent_at"`
}

// WSNewMessage is sent by server when a new message arrives
type WSNewMessage struct {
	Message ChatMessage `json:"message"`
}

// WSTypingIndicator is sent in both directions
type WSTypingIndicator struct {
	UserID uuid.UUID `json:"user_id"`
}

// WSReadReceipt is sent by client to mark messages as read
type WSReadReceipt struct {
	SenderID string `json:"sender_id"` // The user whose messages we're marking as read
}

// WSReadUpdate is sent by server to notify sender their messages were read
type WSReadUpdate struct {
	ReaderID uuid.UUID `json:"reader_id"`
	ReadAt   time.Time `json:"read_at"`
}

// WSDeliveryUpdate notifies sender that messages were delivered
type WSDeliveryUpdate struct {
	ReceiverID uuid.UUID `json:"receiver_id"`
}

// WSOnlineStatus broadcast when a user comes online/offline
type WSOnlineStatus struct {
	UserID   uuid.UUID `json:"user_id"`
	IsOnline bool      `json:"is_online"`
}

// WSBroadcastMessage is sent by admin to broadcast to all users
type WSBroadcastMessage struct {
	Content string `json:"content"`
}

// WSBroadcastNotification is sent to users when a broadcast arrives
type WSBroadcastNotification struct {
	Broadcast ChatBroadcastWithStatus `json:"broadcast"`
}

// WSBroadcastReadReceipt is sent by client to mark a broadcast as read
type WSBroadcastReadReceipt struct {
	BroadcastID string `json:"broadcast_id"`
}

// WSError is sent by server on error
type WSError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
