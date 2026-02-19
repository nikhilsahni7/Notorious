package repository

import (
	"context"
	"fmt"
	"notorious-backend/internal/database"
	"notorious-backend/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ChatRepository struct {
	db *database.DB
}

func NewChatRepository(db *database.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

// GetOrCreateConversation finds or creates a conversation between two users.
// Participants are normalized (smaller UUID = participant_a) to ensure uniqueness.
func (r *ChatRepository) GetOrCreateConversation(ctx context.Context, userA, userB uuid.UUID) (*models.ChatConversation, error) {
	// Normalize order: smaller UUID is always participant_a
	if userA.String() > userB.String() {
		userA, userB = userB, userA
	}

	var conv models.ChatConversation
	query := `
		INSERT INTO chat_conversations (participant_a, participant_b)
		VALUES ($1, $2)
		ON CONFLICT (participant_a, participant_b) DO UPDATE SET id = chat_conversations.id
		RETURNING id, participant_a, participant_b, last_message_id, last_message_at, created_at
	`
	err := r.db.Pool.QueryRow(ctx, query, userA, userB).Scan(
		&conv.ID, &conv.ParticipantA, &conv.ParticipantB,
		&conv.LastMessageID, &conv.LastMessageAt, &conv.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create conversation: %w", err)
	}
	return &conv, nil
}

// SaveMessage persists a new chat message and updates the conversation's last message.
// Uses a server-assigned sequence number for guaranteed ordering.
func (r *ChatRepository) SaveMessage(ctx context.Context, conversationID, senderID, receiverID uuid.UUID, content string) (*models.ChatMessage, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var msg models.ChatMessage

	// Insert message with server-assigned sequence number
	insertQuery := `
		INSERT INTO chat_messages (conversation_id, sender_id, receiver_id, content, sequence_num)
		VALUES ($1, $2, $3, $4, nextval('chat_message_seq'))
		RETURNING id, conversation_id, sender_id, receiver_id, content, sequence_num, status, sent_at
	`
	err = tx.QueryRow(ctx, insertQuery, conversationID, senderID, receiverID, content).Scan(
		&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.ReceiverID,
		&msg.Content, &msg.SequenceNum, &msg.Status, &msg.SentAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert message: %w", err)
	}

	// Update conversation's last message
	updateQuery := `
		UPDATE chat_conversations
		SET last_message_id = $1, last_message_at = $2
		WHERE id = $3
	`
	_, err = tx.Exec(ctx, updateQuery, msg.ID, msg.SentAt, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &msg, nil
}

// GetMessages returns paginated messages for a conversation using cursor-based pagination.
// If beforeSeq is 0, returns the latest messages.
func (r *ChatRepository) GetMessages(ctx context.Context, conversationID uuid.UUID, beforeSeq int64, limit int) ([]models.ChatMessage, error) {
	var query string
	var args []interface{}

	if beforeSeq > 0 {
		query = `
			SELECT id, conversation_id, sender_id, receiver_id, content, sequence_num,
			       status, sent_at, delivered_at, read_at
			FROM chat_messages
			WHERE conversation_id = $1 AND sequence_num < $2
			ORDER BY sequence_num DESC
			LIMIT $3
		`
		args = []interface{}{conversationID, beforeSeq, limit}
	} else {
		query = `
			SELECT id, conversation_id, sender_id, receiver_id, content, sequence_num,
			       status, sent_at, delivered_at, read_at
			FROM chat_messages
			WHERE conversation_id = $1
			ORDER BY sequence_num DESC
			LIMIT $2
		`
		args = []interface{}{conversationID, limit}
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var m models.ChatMessage
		if err := rows.Scan(
			&m.ID, &m.ConversationID, &m.SenderID, &m.ReceiverID,
			&m.Content, &m.SequenceNum, &m.Status, &m.SentAt,
			&m.DeliveredAt, &m.ReadAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, m)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, rows.Err()
}

// GetConversations returns all conversations for a user with details.
func (r *ChatRepository) GetConversations(ctx context.Context, userID uuid.UUID) ([]models.ConversationWithDetails, error) {
	query := `
		SELECT
			c.id,
			CASE WHEN c.participant_a = $1 THEN c.participant_b ELSE c.participant_a END AS other_user_id,
			u.name AS other_user_name,
			u.email AS other_user_email,
			u.role AS other_user_role,
			m.content AS last_message,
			c.last_message_at,
			m.sender_id AS last_sender_id,
			COALESCE((
				SELECT COUNT(*)
				FROM chat_messages cm
				WHERE cm.conversation_id = c.id
				AND cm.receiver_id = $1
				AND cm.status != 'read'
			), 0) AS unread_count
		FROM chat_conversations c
		JOIN users u ON u.id = CASE WHEN c.participant_a = $1 THEN c.participant_b ELSE c.participant_a END
		LEFT JOIN chat_messages m ON m.id = c.last_message_id
		WHERE c.participant_a = $1 OR c.participant_b = $1
		ORDER BY c.last_message_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations: %w", err)
	}
	defer rows.Close()

	var conversations []models.ConversationWithDetails
	for rows.Next() {
		var conv models.ConversationWithDetails
		if err := rows.Scan(
			&conv.ID, &conv.OtherUserID, &conv.OtherUserName,
			&conv.OtherUserEmail, &conv.OtherUserRole,
			&conv.LastMessage, &conv.LastMessageAt,
			&conv.LastSenderID, &conv.UnreadCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conversations = append(conversations, conv)
	}

	return conversations, rows.Err()
}

// GetUnreadCount returns the total number of unread messages for a user (DMs + broadcasts).
func (r *ChatRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, int, error) {
	var dmCount, broadcastCount int

	// Count unread DMs
	dmQuery := `SELECT COUNT(*) FROM chat_messages WHERE receiver_id = $1 AND status != 'read'`
	err := r.db.Pool.QueryRow(ctx, dmQuery, userID).Scan(&dmCount)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count unread DMs: %w", err)
	}

	// Count unread broadcasts
	bcQuery := `
		SELECT COUNT(*)
		FROM chat_broadcasts b
		WHERE b.sender_id != $1
		AND NOT EXISTS (
			SELECT 1 FROM chat_broadcast_reads br
			WHERE br.broadcast_id = b.id AND br.user_id = $1
		)
	`
	err = r.db.Pool.QueryRow(ctx, bcQuery, userID).Scan(&broadcastCount)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count unread broadcasts: %w", err)
	}

	return dmCount, broadcastCount, nil
}

// MarkAsDelivered marks all 'sent' messages from a sender to a receiver as 'delivered'.
func (r *ChatRepository) MarkAsDelivered(ctx context.Context, receiverID, senderID uuid.UUID) (int64, error) {
	now := time.Now()
	query := `
		UPDATE chat_messages
		SET status = 'delivered', delivered_at = $1
		WHERE receiver_id = $2 AND sender_id = $3 AND status = 'sent'
	`
	result, err := r.db.Pool.Exec(ctx, query, now, receiverID, senderID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark as delivered: %w", err)
	}
	return result.RowsAffected(), nil
}

// MarkAsRead marks all unread messages from a sender to a receiver as 'read'.
func (r *ChatRepository) MarkAsRead(ctx context.Context, receiverID, senderID uuid.UUID) (int64, error) {
	now := time.Now()
	query := `
		UPDATE chat_messages
		SET status = 'read', read_at = $1,
		    delivered_at = COALESCE(delivered_at, $1)
		WHERE receiver_id = $2 AND sender_id = $3 AND status IN ('sent', 'delivered')
	`
	result, err := r.db.Pool.Exec(ctx, query, now, receiverID, senderID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark as read: %w", err)
	}
	return result.RowsAffected(), nil
}

// GetConversationByUsers looks up conversationID for two users.
func (r *ChatRepository) GetConversationByUsers(ctx context.Context, userA, userB uuid.UUID) (*models.ChatConversation, error) {
	// Normalize order
	if userA.String() > userB.String() {
		userA, userB = userB, userA
	}

	var conv models.ChatConversation
	query := `
		SELECT id, participant_a, participant_b, last_message_id, last_message_at, created_at
		FROM chat_conversations
		WHERE participant_a = $1 AND participant_b = $2
	`
	err := r.db.Pool.QueryRow(ctx, query, userA, userB).Scan(
		&conv.ID, &conv.ParticipantA, &conv.ParticipantB,
		&conv.LastMessageID, &conv.LastMessageAt, &conv.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	return &conv, nil
}

// SaveBroadcast persists an admin broadcast message.
func (r *ChatRepository) SaveBroadcast(ctx context.Context, senderID uuid.UUID, content string) (*models.ChatBroadcast, error) {
	var bc models.ChatBroadcast
	query := `
		INSERT INTO chat_broadcasts (sender_id, content)
		VALUES ($1, $2)
		RETURNING id, sender_id, content, sent_at
	`
	err := r.db.Pool.QueryRow(ctx, query, senderID, content).Scan(
		&bc.ID, &bc.SenderID, &bc.Content, &bc.SentAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save broadcast: %w", err)
	}
	return &bc, nil
}

// GetBroadcasts returns broadcasts with read status for a specific user.
func (r *ChatRepository) GetBroadcasts(ctx context.Context, userID uuid.UUID, limit int) ([]models.ChatBroadcastWithStatus, error) {
	query := `
		SELECT
			b.id, b.sender_id, u.name AS sender_name, b.content, b.sent_at,
			CASE WHEN br.user_id IS NOT NULL THEN true ELSE false END AS is_read,
			br.read_at
		FROM chat_broadcasts b
		JOIN users u ON u.id = b.sender_id
		LEFT JOIN chat_broadcast_reads br ON br.broadcast_id = b.id AND br.user_id = $1
		ORDER BY b.sent_at DESC
		LIMIT $2
	`
	rows, err := r.db.Pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query broadcasts: %w", err)
	}
	defer rows.Close()

	var broadcasts []models.ChatBroadcastWithStatus
	for rows.Next() {
		var bc models.ChatBroadcastWithStatus
		if err := rows.Scan(
			&bc.ID, &bc.SenderID, &bc.SenderName, &bc.Content, &bc.SentAt,
			&bc.IsRead, &bc.ReadAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan broadcast: %w", err)
		}
		broadcasts = append(broadcasts, bc)
	}
	return broadcasts, rows.Err()
}

// MarkBroadcastRead records that a user has read a broadcast.
func (r *ChatRepository) MarkBroadcastRead(ctx context.Context, broadcastID, userID uuid.UUID) error {
	query := `
		INSERT INTO chat_broadcast_reads (broadcast_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (broadcast_id, user_id) DO NOTHING
	`
	_, err := r.db.Pool.Exec(ctx, query, broadcastID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark broadcast as read: %w", err)
	}
	return nil
}

// CleanupOldMessages deletes messages and broadcasts older than the specified number of days.
func (r *ChatRepository) CleanupOldMessages(ctx context.Context, days int) (int64, int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)

	// Delete old messages
	msgResult, err := r.db.Pool.Exec(ctx,
		`DELETE FROM chat_messages WHERE sent_at < $1`, cutoff)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to cleanup messages: %w", err)
	}

	// Delete old broadcasts (cascade deletes broadcast_reads)
	bcResult, err := r.db.Pool.Exec(ctx,
		`DELETE FROM chat_broadcasts WHERE sent_at < $1`, cutoff)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to cleanup broadcasts: %w", err)
	}

	// Delete conversations with no messages
	_, err = r.db.Pool.Exec(ctx, `
		DELETE FROM chat_conversations c
		WHERE NOT EXISTS (
			SELECT 1 FROM chat_messages m WHERE m.conversation_id = c.id
		)
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to cleanup empty conversations: %w", err)
	}

	return msgResult.RowsAffected(), bcResult.RowsAffected(), nil
}

// GetUnreadCountPerUser returns unread message counts grouped by sender for a specific receiver.
// Used by admin to show per-user unread badges.
func (r *ChatRepository) GetUnreadCountPerUser(ctx context.Context, receiverID uuid.UUID) (map[uuid.UUID]int, error) {
	query := `
		SELECT sender_id, COUNT(*)
		FROM chat_messages
		WHERE receiver_id = $1 AND status != 'read'
		GROUP BY sender_id
	`
	rows, err := r.db.Pool.Query(ctx, query, receiverID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unread counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[uuid.UUID]int)
	for rows.Next() {
		var senderID uuid.UUID
		var count int
		if err := rows.Scan(&senderID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan unread count: %w", err)
		}
		counts[senderID] = count
	}
	return counts, rows.Err()
}
