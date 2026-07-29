DROP TRIGGER IF EXISTS trg_match_runtime_states_updated_at ON match_runtime_states;
DROP FUNCTION IF EXISTS update_match_runtime_states_updated_at();
DROP INDEX IF EXISTS idx_match_runtime_states_status;
DROP INDEX IF EXISTS idx_match_runtime_states_match_id;
DROP TABLE IF EXISTS match_runtime_states;
