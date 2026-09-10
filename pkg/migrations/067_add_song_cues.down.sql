ALTER TABLE songs
  DROP COLUMN IF EXISTS vibration_cues,
  DROP COLUMN IF EXISTS light_cues;
