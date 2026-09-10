-- Revert cue objects back to plain second offsets (duration is discarded).

UPDATE songs
SET vibration_cues = COALESCE((
    SELECT jsonb_agg(to_jsonb((elem ->> 'at')::int) ORDER BY ord)
    FROM jsonb_array_elements(vibration_cues) WITH ORDINALITY AS t(elem, ord)
), '[]'::jsonb)
WHERE jsonb_typeof(vibration_cues) = 'array'
  AND (
    jsonb_array_length(vibration_cues) = 0
    OR jsonb_typeof(vibration_cues -> 0) = 'object'
  );

UPDATE songs
SET light_cues = COALESCE((
    SELECT jsonb_agg(to_jsonb((elem ->> 'at')::int) ORDER BY ord)
    FROM jsonb_array_elements(light_cues) WITH ORDINALITY AS t(elem, ord)
), '[]'::jsonb)
WHERE jsonb_typeof(light_cues) = 'array'
  AND (
    jsonb_array_length(light_cues) = 0
    OR jsonb_typeof(light_cues -> 0) = 'object'
  );
