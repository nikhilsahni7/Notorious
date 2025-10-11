-- Clear old admin sessions that don't have full metadata
DELETE FROM admin_sessions WHERE browser_version IS NULL OR os_version IS NULL;
