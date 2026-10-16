-- Migration 012: expression index for the login email lookup.
--
-- user_repository.GetByEmail (used by POST /auth/login and /auth/revoke-session)
-- queries: WHERE lower(trim(email)) = lower(trim($1))
-- which cannot use idx_users_email (raw email B-tree, from the UNIQUE
-- constraint), so every login performs a sequential scan of users.
--
-- App code normalizes emails on write (strings.ToLower + TrimSpace), so this
-- expression exactly matches all stored values.
--
-- CREATE INDEX CONCURRENTLY cannot run inside a transaction block, and the
-- migration runner executes each file through a single Exec call (multi-statement
-- simple-query runs in an implicit transaction) — therefore this file must
-- contain exactly this one statement.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email_lower_trim
ON users ((lower(trim(email))));
