-- Demo seed: fake chants data for mobile API testing.
-- Safe to re-run: uses fixed UUIDs with ON CONFLICT DO NOTHING.

BEGIN;

INSERT INTO leagues (id, name, country, provider, provider_league_id, is_active)
VALUES ('a1000000-0000-4000-8000-000000000001', 'ACB League', 'Spain', 'demo', 'acb-1', true)
ON CONFLICT DO NOTHING;

INSERT INTO seasons (id, league_id, name, start_date, end_date, is_active)
VALUES ('a2000000-0000-4000-8000-000000000001', 'a1000000-0000-4000-8000-000000000001', '2025/26', '2025-08-01', '2026-06-30', true)
ON CONFLICT DO NOTHING;

INSERT INTO clubs (id, name, short_name, country, is_active) VALUES
  ('a3000000-0000-4000-8000-000000000001', 'SP Burgos', 'BUR', 'Spain', true),
  ('a3000000-0000-4000-8000-000000000002', 'FC Barcelona', 'BAR', 'Spain', true)
ON CONFLICT DO NOTHING;

INSERT INTO matches (id, league_id, season_id, home_club_id, away_club_id, match_datetime, stadium_name, status)
VALUES (
  'a4000000-0000-4000-8000-000000000001',
  'a1000000-0000-4000-8000-000000000001',
  'a2000000-0000-4000-8000-000000000001',
  'a3000000-0000-4000-8000-000000000001',
  'a3000000-0000-4000-8000-000000000002',
  NOW(),
  'Coliseum Burgos',
  'live'
)
ON CONFLICT DO NOTHING;

INSERT INTO songs (id, title, artist, duration, audio_url, is_active) VALUES
  ('a5000000-0000-4000-8000-000000000001', 'We Will Rock You', 'Queen', 122, 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3', true),
  ('a5000000-0000-4000-8000-000000000002', 'Seven Nation Army', 'The White Stripes', 92, 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-2.mp3', true),
  ('a5000000-0000-4000-8000-000000000003', 'Allez Allez Allez', 'Fan Chant', 75, 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-3.mp3', true),
  ('a5000000-0000-4000-8000-000000000004', 'Never Walk Alone', 'Gerry and the Pacemakers', 180, 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-4.mp3', true)
ON CONFLICT DO NOTHING;

INSERT INTO song_lyrics (song_id, language, lyrics) VALUES
  ('a5000000-0000-4000-8000-000000000001', 'en', E'[00:00.00]Buddy you''re a boy make a big noise\n[00:04.00]Playin'' in the street gonna be a big man some day\n[00:08.00]You got mud on yo face\n[00:10.00]You big disgrace\n[00:12.00]Kickin'' your can all over the place\n[00:16.00]We will we will rock you'),
  ('a5000000-0000-4000-8000-000000000002', 'en', E'[00:00.00]Seven nation army couldn''t hold me back\n[00:04.00]They''re gonna rip it off\n[00:08.00]Taking their time right behind my back'),
  ('a5000000-0000-4000-8000-000000000003', 'en', E'[00:00.00]Allez allez allez\n[00:03.00]We''re the famous Burgos\n[00:06.00]And we''re going to win'),
  ('a5000000-0000-4000-8000-000000000004', 'en', E'[00:00.00]When you walk through a storm\n[00:05.00]Hold your head up high\n[00:10.00]And don''t be afraid of the dark')
ON CONFLICT (song_id, language) DO NOTHING;

INSERT INTO chants (id, match_id, song_id, title, points, duration_seconds, scheduled_at, flash_duration_ms, vibration_duration_ms, is_preview, is_active) VALUES
  ('b1000000-0000-4000-8000-000000000001', 'a4000000-0000-4000-8000-000000000001', 'a5000000-0000-4000-8000-000000000001', 'We will rock you', 200, 120, NOW() - INTERVAL '2 days', 500, 500, false, true),
  ('b1000000-0000-4000-8000-000000000002', 'a4000000-0000-4000-8000-000000000001', 'a5000000-0000-4000-8000-000000000002', 'Seven Nation Army', 150, 92, date_trunc('day', NOW()) + INTERVAL '2 hours', 400, 400, false, true),
  ('b1000000-0000-4000-8000-000000000003', 'a4000000-0000-4000-8000-000000000001', 'a5000000-0000-4000-8000-000000000003', 'Allez Allez Allez', 120, 75, NOW() + INTERVAL '45 minutes', 300, 300, false, true),
  ('b1000000-0000-4000-8000-000000000004', 'a4000000-0000-4000-8000-000000000001', 'a5000000-0000-4000-8000-000000000004', 'Never Walk Alone', 250, 180, date_trunc('day', NOW()) + INTERVAL '1 day', 500, 500, false, true),
  ('b1000000-0000-4000-8000-000000000005', 'a4000000-0000-4000-8000-000000000001', 'a5000000-0000-4000-8000-000000000001', 'We will rock you (Preview)', 50, 30, NOW() + INTERVAL '20 minutes', 200, 200, true, true)
ON CONFLICT DO NOTHING;

COMMIT;
