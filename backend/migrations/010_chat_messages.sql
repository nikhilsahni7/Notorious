-- Migration 010: Chat system tables
-- Supports 1-to-1 messaging and broadcast announcements

-- Conversations table (denormalized for fast listing)
CREATE TABLE IF NOT EXISTS chat_conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    participant_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    participant_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_message_id UUID,
    last_message_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(participant_a, participant_b)
);

-- Messages table
CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    sequence_num BIGINT NOT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'sent'
        CHECK (status IN ('sent', 'delivered', 'read')),
    sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMP,
    read_at TIMESTAMP,
    UNIQUE(conversation_id, sequence_num)
);

-- Broadcast announcements table (admin -> all users)
CREATE TABLE IF NOT EXISTS chat_broadcasts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    sent_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Track which users have read a broadcast
CREATE TABLE IF NOT EXISTS chat_broadcast_reads (
    broadcast_id UUID NOT NULL REFERENCES chat_broadcasts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (broadcast_id, user_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation ON chat_messages(conversation_id, sequence_num DESC);
CREATE INDEX IF NOT EXISTS idx_chat_messages_receiver_unread ON chat_messages(receiver_id, status) WHERE status != 'read';
CREATE INDEX IF NOT EXISTS idx_chat_messages_sent_at ON chat_messages(sent_at);
CREATE INDEX IF NOT EXISTS idx_chat_conversations_participants ON chat_conversations(participant_a, participant_b);
CREATE INDEX IF NOT EXISTS idx_chat_conversations_last_msg ON chat_conversations(last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_broadcasts_sent_at ON chat_broadcasts(sent_at);

-- Sequence for server-assigned message ordering
CREATE SEQUENCE IF NOT EXISTS chat_message_seq;

-- FK for last_message_id (added after messages table exists)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.table_constraints
        WHERE constraint_schema = 'public'
          AND table_name = 'chat_conversations'
          AND constraint_name = 'fk_last_message'
    ) THEN
        ALTER TABLE chat_conversations
            ADD CONSTRAINT fk_last_message
            FOREIGN KEY (last_message_id) REFERENCES chat_messages(id) ON DELETE SET NULL;
    END IF;
END $$;
