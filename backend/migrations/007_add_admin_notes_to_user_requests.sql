-- Migration: Add admin notes to user_requests table
-- This allows admins to record why they approved/rejected a request

ALTER TABLE user_requests
ADD COLUMN IF NOT EXISTS admin_note TEXT,
ADD COLUMN IF NOT EXISTS reviewed_by UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMP;

-- Create index for reviewed_by for faster queries
CREATE INDEX IF NOT EXISTS idx_user_requests_reviewed_by ON user_requests(reviewed_by);

-- Add comment
COMMENT ON COLUMN user_requests.admin_note IS 'Admin note explaining why request was approved or rejected';
COMMENT ON COLUMN user_requests.reviewed_by IS 'Admin user who reviewed this request';
COMMENT ON COLUMN user_requests.reviewed_at IS 'Timestamp when request was reviewed';
