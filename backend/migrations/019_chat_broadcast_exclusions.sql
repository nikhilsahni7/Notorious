-- Recipients excluded by the sender must not receive or list a broadcast.
CREATE TABLE IF NOT EXISTS chat_broadcast_exclusions (
    broadcast_id UUID NOT NULL REFERENCES chat_broadcasts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (broadcast_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_chat_broadcast_exclusions_user
    ON chat_broadcast_exclusions (user_id, broadcast_id);
