-- Add metadata fields to user_requests table for tracking signup requests
ALTER TABLE user_requests ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45);
ALTER TABLE user_requests ADD COLUMN IF NOT EXISTS country VARCHAR(100);
ALTER TABLE user_requests ADD COLUMN IF NOT EXISTS city VARCHAR(100);
ALTER TABLE user_requests ADD COLUMN IF NOT EXISTS device_type VARCHAR(50);
ALTER TABLE user_requests ADD COLUMN IF NOT EXISTS browser VARCHAR(100);
ALTER TABLE user_requests ADD COLUMN IF NOT EXISTS os VARCHAR(100);
ALTER TABLE user_requests ADD COLUMN IF NOT EXISTS user_agent TEXT;

