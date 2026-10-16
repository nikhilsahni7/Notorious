-- Migration 013: composite index for admin user-request listing.
--
-- user_request_repository.ListByStatus queries:
--   WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
-- idx_user_requests_status (status) from migration 001 satisfies the filter
-- but not the sort, forcing Planner-side sort of every pending/rejected row.
--
-- CREATE INDEX CONCURRENTLY cannot run inside a transaction block, and the
-- migration runner executes each file through a single Exec call (multi-statement
-- simple-query runs in an implicit transaction) — therefore this file must
-- contain exactly this one statement.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_requests_status_created_at
ON user_requests (status, created_at DESC);
