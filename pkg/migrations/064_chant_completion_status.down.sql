-- Cancellations only exist as rows carrying this status, so they have to go
-- with the column — leaving them behind would read as zero-point completions
-- and permanently block those chants.
DELETE FROM chant_completions WHERE status = 'cancelled';

ALTER TABLE chant_completions
    DROP CONSTRAINT IF EXISTS chant_completions_status_check;

ALTER TABLE chant_completions
    DROP COLUMN IF EXISTS status;
