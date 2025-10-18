-- Migration: Add refinement tracking to search_history
-- Description: Adds columns to track whether a search is a refinement and its base search

-- Add is_refinement column to track if this is a refinement search
ALTER TABLE search_history
ADD COLUMN IF NOT EXISTS is_refinement BOOLEAN DEFAULT FALSE;

-- Add base_search_id to track the original search this refines
ALTER TABLE search_history
ADD COLUMN IF NOT EXISTS base_search_id UUID REFERENCES search_history(id) ON DELETE SET NULL;

-- Add comment for documentation
COMMENT ON COLUMN search_history.is_refinement IS 'True if this search is a refinement of another search';
COMMENT ON COLUMN search_history.base_search_id IS 'Reference to the original search being refined';

-- Create index for better query performance
CREATE INDEX IF NOT EXISTS idx_search_history_is_refinement ON search_history(is_refinement);
CREATE INDEX IF NOT EXISTS idx_search_history_base_search_id ON search_history(base_search_id);
