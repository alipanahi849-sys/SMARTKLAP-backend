-- Migration: 068_song_cue_duration
-- Purpose: Each vibrate/light cue stores its own pulse length (duration_ms),
-- not just the second offset. Legacy number arrays become {at, duration_ms}.

UPDATE songs
SET vibration_cues = COALESCE((
    SELECT jsonb_agg(
        jsonb_build_object('at', elem::int, 'duration_ms', 200)
        ORDER BY ord
    )
    FROM jsonb_array_elements_text(vibration_cues) WITH ORDINALITY AS t(elem, ord)
), '[]'::jsonb)
WHERE jsonb_typeof(vibration_cues) = 'array'
  AND (
    jsonb_array_length(vibration_cues) = 0
    OR jsonb_typeof(vibration_cues -> 0) = 'number'
  );

UPDATE songs
SET light_cues = COALESCE((
    SELECT jsonb_agg(
        jsonb_build_object('at', elem::int, 'duration_ms', 200)
        ORDER BY ord
    )
    FROM jsonb_array_elements_text(light_cues) WITH ORDINALITY AS t(elem, ord)
), '[]'::jsonb)
WHERE jsonb_typeof(light_cues) = 'array'
  AND (
    jsonb_array_length(light_cues) = 0
    OR jsonb_typeof(light_cues -> 0) = 'number'
  );
