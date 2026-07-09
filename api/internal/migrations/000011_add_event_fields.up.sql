-- Add missing columns to events table
ALTER TABLE events 
ADD COLUMN description TEXT,
ADD COLUMN organizer VARCHAR(255),
ADD COLUMN category VARCHAR(100),
ADD COLUMN tags JSONB,
ADD COLUMN capacity INTEGER;
