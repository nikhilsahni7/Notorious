-- Add region field to users table
-- Default region is 'pan-india' for existing users
-- Regions: 'pan-india' (access all), 'delhi-ncr' (only Delhi data)

ALTER TABLE users ADD COLUMN IF NOT EXISTS region VARCHAR(50) NOT NULL DEFAULT 'pan-india';

-- Create index for region filtering
CREATE INDEX IF NOT EXISTS idx_users_region ON users(region);

-- Update admin to have pan-india access
UPDATE users SET region = 'pan-india' WHERE role = 'admin';

COMMENT ON COLUMN users.region IS 'User region access: pan-india (all data), delhi-ncr (Delhi only), etc.';
