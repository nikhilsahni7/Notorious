-- Migration 014: composite index for admin password-change-request listing.
--
-- password_change_repository.ListByStatus queries:
--   WHERE pcr.status = $1 AND pcr.reason != 'Self-service password change'
--   ORDER BY pcr.created_at DESC LIMIT $2 OFFSET $3
-- No index covers (status, created_at) — ordering requires a sort of all
-- matching rows on every page load.
--
-- CREATE INDEX CONCURRENTLY cannot run inside a transaction block, and the
-- migration runner executes each file through a single Exec call (multi-statement
-- simple-query runs in an implicit transaction) — therefore this file must
-- contain exactly this one statement.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_password_change_requests_status_created_at
ON password_change_requests (status, created_at DESC);
