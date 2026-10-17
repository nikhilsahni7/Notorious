-- Bug 2: search_history.searched_at was TIMESTAMP WITHOUT TIME ZONE and EOD
-- reports bound the range with IST midnight/end, sending those Go times to
-- Postgres; the driver dropped the +05:30 offset, so the window landed 05:30
-- off — reports before 05:30 IST came back empty and 00:00–05:30 IST searches
-- fell into the wrong day.
--
-- Store the column as an absolute instant (timestamptz) instead.
--
-- Existing values were written by the column default NOW()::timestamp, i.e.
-- the instant rendered in the DB session timezone. The backend and database run
-- in Ohio while users are in India, so the session zone is not guaranteed to be
-- UTC; reinterpreting the naive values with the *current* session timezone
-- (`current_setting('TimeZone')`) recovers the exact original instant. On a
-- cloud Postgres whose default zone is UTC this is identical to using 'UTC'.
--
-- The migration runner replays every file on startup (there is no tracking
-- table), so the type change is guarded to run only while the column is still
-- naive; on later startups it is a no-op.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'search_history'
          AND column_name = 'searched_at'
          AND data_type = 'timestamp without time zone'
    ) THEN
        ALTER TABLE search_history
            ALTER COLUMN searched_at TYPE TIMESTAMPTZ
            USING searched_at AT TIME ZONE current_setting('TimeZone');
    END IF;
END $$;

-- Keep new rows as absolute instants regardless of the session timezone.
ALTER TABLE search_history ALTER COLUMN searched_at SET DEFAULT NOW();
