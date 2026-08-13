-- Illuminate — Jessie Reyez & Elyanna (FIFA World Cup 2026)
-- Audio: /uploads/chants/illuminate.mp3

INSERT INTO songs (
    id, title, artist, album, duration, audio_url, is_active,
    storage_key, mime_type, duration_ms
) VALUES (
    'a5000000-0000-4000-8000-000000000005',
    'Illuminate',
    'Jessie Reyez & Elyanna',
    'FIFA World Cup 2026 Official Album',
    183,
    '/uploads/chants/illuminate.mp3',
    true,
    'chants/illuminate.mp3',
    'audio/mpeg',
    183248
) ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    artist = EXCLUDED.artist,
    album = EXCLUDED.album,
    duration = EXCLUDED.duration,
    audio_url = EXCLUDED.audio_url,
    storage_key = EXCLUDED.storage_key,
    mime_type = EXCLUDED.mime_type,
    duration_ms = EXCLUDED.duration_ms,
    updated_at = NOW();

INSERT INTO song_lyrics (song_id, language, lyrics) VALUES (
    'a5000000-0000-4000-8000-000000000005',
    'en',
    $lrc$[00:06.80]Oh, can you see our colors in the dark?
[00:09.89]We're shinin' on forever, illuminate
[00:14.45]Oh, can you see our colors in the dark?
[00:18.47]Our spirit on our shoulders, oh yeah, oh yeah, oh yeah
[00:21.58]Illuminate-ate, yeah
[00:28.60]Illuminate-ate, yeah, yeah, ayy (Yeah, mm)
[00:38.17]We walk in the room kinda like Hov
[00:40.72]In a room full of vultures, we ain't gon' (Lose)
[00:42.73]We walk in the room kinda like Luda
[00:44.23]We need some space, baby, you should (Move)
[00:46.25]Kinda like a rose had sex with a bomb and it birthed us
[00:48.75]Baby, 'bout to watch us (Bloom), 'bout to watch us (Boom)
[00:52.30]I feel the limit is the sky
[00:56.33]اوه يا ليل اوه يا ليل اوه يا ليل (Yeah, yeah, yeah)
[01:00.49]'Cause we are champions tonight
[01:04.00]اوه يا ليل اوه يا ليل اوه يا ليل
[01:08.65]Oh, can you see our colors in the dark?
[01:11.16]We're shinin' on forever, illuminate
[01:15.82]Oh, can you see our colors in the dark?
[01:19.33]Our spirit on our shoulders, oh yeah, oh yeah, oh yeah
[01:23.35]Illuminate-ate, yeah (Ayy)
[01:30.88]Illuminate-ate (Ayy), yeah, yeah, ayy
[01:39.55]Shine bright, never dim that light
[01:41.55](تعال تعال تعال)
[01:43.56]Can't stop, going side by side
[01:45.56](تعال تعال تعال)
[01:47.57]سمعني صوتك، الأول بالعالم
[01:51.08]تحية للأبطال
[01:53.64](تعال تعال تعال)
[01:55.13]I feel the limit is the sky
[01:58.15]اوه يا ليل اوه يا ليل اوه يا ليل (Yeah, yeah, yeah)
[02:01.66]'Cause we are champions tonight
[02:05.75]اوه يا ليل اوه يا ليل اوه يا ليل
[02:09.75]Oh, can you see our colors in the dark?
[02:12.76]We're shinin' up forever, illuminate
[02:16.78]Oh, can you see our colors in the dark?
[02:20.41]Our spirit on our shoulders, oh yeah, oh yeah, oh yeah (Oh yeah)
[02:24.42]Illuminate-ate (Oh), yeah (Oh, ayy)
[02:32.44]Illuminate-ate, yeah, yeah, ayy (Oh)
[02:39.96]'Luminate-luminate-luminate, ايوا ايوا ايوا
[02:45.02]'Luminate-luminate-luminate-nate, ايوا (Illuminate)
[02:48.53]'Luminate-luminate-luminate, ايوا ايوا ايوا
[02:52.54]'Luminate-luminate-luminate-nate, ايوا (Illuminate)
$lrc$
) ON CONFLICT (song_id, language) DO UPDATE SET
    lyrics = EXCLUDED.lyrics,
    updated_at = NOW();

UPDATE chants SET
    song_id = 'a5000000-0000-4000-8000-000000000005',
    title = 'Illuminate — Jessie Reyez & Elyanna (World Cup 2026)',
    duration_seconds = 183,
    points = 150,
    flash_duration_ms = 900,
    vibration_duration_ms = 900,
    scheduled_at = NOW() + INTERVAL '110 seconds',
    is_active = true,
    updated_at = NOW()
WHERE id = '386a7878-1d05-449e-9153-8474cb2e15a0';

SELECT c.id, c.title, c.scheduled_at, c.flash_duration_ms, c.vibration_duration_ms, s.audio_url
FROM chants c
JOIN songs s ON s.id = c.song_id
WHERE c.id = '386a7878-1d05-449e-9153-8474cb2e15a0';
