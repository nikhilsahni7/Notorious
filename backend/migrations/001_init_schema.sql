-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'user')),
    daily_search_limit INT NOT NULL DEFAULT 100,
    searches_used_today INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_reset_date DATE NOT NULL DEFAULT CURRENT_DATE
);

-- User registration requests table
CREATE TABLE IF NOT EXISTS user_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    requested_searches_per_day INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    admin_notes TEXT
);

-- Search history table
CREATE TABLE IF NOT EXISTS search_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    query TEXT NOT NULL,
    total_results INT NOT NULL,
    top_results JSONB,
    searched_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create default admin user (password: admin123)
-- In production, change this immediately!
INSERT INTO users (email, password_hash, name, phone, role, daily_search_limit, is_active)
VALUES (
    'admin@notorious.com',
    '$2a$12$vT4Q7iVUMJ.NLg8tySU29OKBzoLUb0qd6lsTFUIOH2MJznb7uFAsa',
    'Administrator',
    '',
    'admin',
    999999,
    true
) ON CONFLICT (email) DO NOTHING;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_user_requests_status ON user_requests(status);
CREATE INDEX IF NOT EXISTS idx_search_history_user_id ON search_history(user_id, searched_at);
CREATE INDEX IF NOT EXISTS idx_search_history_searched_at ON search_history(searched_at);
