-- Performance indexes for production latency reduction
-- These indexes target the hottest query paths identified by gograph analysis.
-- All use CREATE INDEX IF NOT EXISTS for safe re-runs.
--
-- Already existing indexes (DO NOT duplicate):
--   001: idx_search_history_user_id ON search_history(user_id, searched_at)
--   001: idx_search_history_searched_at ON search_history(searched_at)
--   004: idx_admin_sessions_token_hash ON admin_sessions(token_hash)
--   009: idx_user_sessions_token_hash ON user_sessions(token_hash)

-- Zero-result analytics: partial index only covers total_results = 0 rows.
-- Used by stats endpoints for zero-result percentage calculations.
CREATE INDEX IF NOT EXISTS idx_search_history_zero_results
ON search_history (total_results) WHERE total_results = 0;

-- Session presence tracking: used by GetOnlineUserIDs and admin dashboard polling.
-- Speeds up queries that filter sessions by last_active timestamp.
CREATE INDEX IF NOT EXISTS idx_sessions_last_active
ON user_sessions (last_active);
