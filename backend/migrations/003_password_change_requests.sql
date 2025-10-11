-- Create password change requests table
CREATE TABLE IF NOT EXISTS password_change_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    admin_notes TEXT,
    new_password_hash TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_by UUID REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_password_requests_user ON password_change_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_password_requests_status ON password_change_requests(status);

