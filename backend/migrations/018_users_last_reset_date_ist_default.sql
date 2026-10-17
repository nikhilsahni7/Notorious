-- Daily-limit reset marker was defaulting to the DB session date (UTC) when a
-- user was created. Align the default with the IST day used everywhere else so
-- the stored marker never starts a day behind the IST calendar.
--
-- The migration runner replays every file on startup, so this is idempotent.
ALTER TABLE users
    ALTER COLUMN last_reset_date SET DEFAULT (now() AT TIME ZONE 'Asia/Kolkata')::date;
