CREATE TABLE IF NOT EXISTS bot_users (
 telegram_id BIGINT PRIMARY KEY,
 username TEXT NOT NULL DEFAULT '',
 approved BOOLEAN NOT NULL DEFAULT FALSE,
 blocked BOOLEAN NOT NULL DEFAULT FALSE,
 daily_limit INT NOT NULL DEFAULT 0 CHECK (daily_limit >= 0),
 plan_name TEXT NOT NULL DEFAULT 'Unassigned',
 plan_expires_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS search_requests (
 update_id BIGINT PRIMARY KEY,
 telegram_id BIGINT NOT NULL REFERENCES bot_users(telegram_id),
 quota_day DATE NOT NULL,
 state TEXT NOT NULL CHECK (state IN ('reserved', 'charged', 'released', 'uncertain')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS search_usage_idx ON search_requests (telegram_id, quota_day, state);
CREATE INDEX IF NOT EXISTS search_reserved_recovery_idx ON search_requests (created_at) WHERE state='reserved';
CREATE INDEX IF NOT EXISTS search_rate_idx ON search_requests (telegram_id, created_at DESC);
CREATE INDEX IF NOT EXISTS search_history_admin_idx ON search_requests (telegram_id, created_at DESC, update_id DESC);
CREATE TABLE IF NOT EXISTS admin_audit (
 id BIGSERIAL PRIMARY KEY, admin_id BIGINT NOT NULL, target_id BIGINT NOT NULL,
 action TEXT NOT NULL, detail TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS admin_updates (update_id BIGINT PRIMARY KEY, created_at TIMESTAMPTZ NOT NULL DEFAULT now());
ALTER TABLE bot_users ADD COLUMN IF NOT EXISTS region TEXT NOT NULL DEFAULT 'delhi-ncr';
CREATE TABLE IF NOT EXISTS plans (
 name TEXT PRIMARY KEY, price_paise BIGINT NOT NULL CHECK(price_paise >= 0),
 daily_limit INT NOT NULL CHECK(daily_limit > 0), days INT NOT NULL CHECK(days > 0),
 active BOOLEAN NOT NULL DEFAULT TRUE, updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE bot_users ADD COLUMN IF NOT EXISTS pending_search TEXT NOT NULL DEFAULT '';
ALTER TABLE bot_users ADD COLUMN IF NOT EXISTS pending_payment BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE bot_users ALTER COLUMN daily_limit SET DEFAULT 0;
ALTER TABLE bot_users ALTER COLUMN plan_name SET DEFAULT 'Unassigned';
ALTER TABLE search_requests ADD COLUMN IF NOT EXISTS search_type TEXT NOT NULL DEFAULT '';
ALTER TABLE search_requests ADD COLUMN IF NOT EXISTS masked_value TEXT NOT NULL DEFAULT '';
ALTER TABLE search_requests ADD COLUMN IF NOT EXISTS search_key TEXT NOT NULL DEFAULT '';
ALTER TABLE search_requests ADD COLUMN IF NOT EXISTS encrypted_value BYTEA;
ALTER TABLE search_requests ADD COLUMN IF NOT EXISTS outcome TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS search_requests_previous_idx ON search_requests (telegram_id, search_key) WHERE state='charged' AND search_key <> '';
CREATE TABLE IF NOT EXISTS quota_resets (
 telegram_id BIGINT NOT NULL REFERENCES bot_users(telegram_id),
 quota_day DATE NOT NULL,
 reset_at TIMESTAMPTZ NOT NULL,
 admin_id BIGINT NOT NULL,
 PRIMARY KEY (telegram_id, quota_day)
);
CREATE TABLE IF NOT EXISTS search_deliveries (
 update_id BIGINT PRIMARY KEY REFERENCES search_requests(update_id),
 telegram_id BIGINT NOT NULL REFERENCES bot_users(telegram_id),
 encrypted_pages BYTEA,
 next_page INT NOT NULL DEFAULT 0,
 chargeable BOOLEAN NOT NULL,
 outcome TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','delivered','cancelled')),
 attempts INT NOT NULL DEFAULT 0,
 next_retry_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX IF NOT EXISTS search_deliveries_pending_idx ON search_deliveries(next_retry_at) WHERE status='pending';

CREATE TABLE IF NOT EXISTS payment_orders (
 id BIGSERIAL PRIMARY KEY,
 create_update_id BIGINT NOT NULL UNIQUE,
 user_id BIGINT NOT NULL REFERENCES bot_users(telegram_id),
 plan_name TEXT NOT NULL,
 amount_paise BIGINT NOT NULL CHECK (amount_paise > 0),
 daily_limit INT NOT NULL CHECK (daily_limit > 0),
 plan_days INT NOT NULL CHECK (plan_days > 0),
 status TEXT NOT NULL DEFAULT 'awaiting_reference' CHECK (status IN ('awaiting_reference','pending_review','approved','rejected')),
 transaction_reference TEXT,
 review_admin_id BIGINT,
 review_reason TEXT,
 region TEXT CHECK (region IS NULL OR region IN ('pan-india','delhi-ncr')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 submitted_at TIMESTAMPTZ,
 reviewed_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS payment_orders_reference_unique ON payment_orders (upper(transaction_reference)) WHERE transaction_reference IS NOT NULL;
CREATE INDEX IF NOT EXISTS payment_orders_review_idx ON payment_orders (status, submitted_at);
CREATE INDEX IF NOT EXISTS payment_orders_user_idx ON payment_orders (user_id, created_at DESC);
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS is_test BOOLEAN NOT NULL DEFAULT FALSE;

-- All bot searches now use the same fixed coverage.
ALTER TABLE bot_users ALTER COLUMN region SET DEFAULT 'pan-india';
UPDATE bot_users SET region='pan-india' WHERE region <> 'pan-india';
