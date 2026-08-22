-- Migration: 065_add_category_to_songs
-- Purpose: Let an admin group the song catalog into the sections the mobile
-- Chants screen shows.
--
-- The screen used to head its sections by chant schedule ("Todays chants",
-- "Upcoming chants"), which tied the browsing list to whatever an admin had
-- scheduled for the live match. Sections now come from this free-text column
-- instead, so the catalog reads the same whether or not a match is on. An
-- empty category is the norm, not an error: those songs fall into the
-- catch-all section.

ALTER TABLE songs
    ADD COLUMN IF NOT EXISTS category VARCHAR(100) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_songs_category ON songs(category);
