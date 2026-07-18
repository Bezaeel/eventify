-- Remove added columns from events table
ALTER TABLE events 
DROP COLUMN IF EXISTS description,
DROP COLUMN IF EXISTS organizer,
DROP COLUMN IF EXISTS category,
DROP COLUMN IF EXISTS tags,
DROP COLUMN IF EXISTS capacity;
