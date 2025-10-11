-- Add last_search_query field to users table to track duplicate searches
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_search_query TEXT DEFAULT '';
