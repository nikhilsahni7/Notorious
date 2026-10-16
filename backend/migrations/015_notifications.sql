-- Notifications (in-app inbox) and per-user notification preferences.
-- Rows are always per-user (no broadcasts in this PR).

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL CHECK (type IN ('quota_reset', 'watchlist_match', 'export_ready', 'quota_warning')),
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    payload JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    read_at TIMESTAMP WITH TIME ZONE NULL,
    -- Set when the OS-level (web) push for this row was actually delivered.
    -- Used to enforce the max-2-OS-pushes-per-user-per-day cap without another table.
    pushed_at TIMESTAMP WITH TIME ZONE NULL
);

-- Per-user notification preferences (JSONB), e.g. {"quota_reset": true}.
-- quota_reset explicitly requires opt-in; other keys default to enabled.
ALTER TABLE users ADD COLUMN IF NOT EXISTS notification_prefs JSONB NOT NULL DEFAULT '{}';

-- Recents-first listing per user
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications (user_id, created_at DESC);

-- Fast unread counts and unread filters
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications (user_id, created_at DESC) WHERE read_at IS NULL;
