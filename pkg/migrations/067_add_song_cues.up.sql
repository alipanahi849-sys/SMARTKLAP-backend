-- Migration: 067_add_song_cues
-- Timed vibrate/light markers (seconds from start) authored on the chant edit page.

ALTER TABLE songs
  ADD COLUMN IF NOT EXISTS vibration_cues jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS light_cues jsonb NOT NULL DEFAULT '[]'::jsonb;
