CREATE TABLE IF NOT EXISTS leagues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    country VARCHAR(100),
    provider VARCHAR(50),
    provider_league_id VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by UUID,
    updated_by UUID,
    CONSTRAINT leagues_provider_unique UNIQUE (provider, provider_league_id)
);

CREATE INDEX idx_leagues_provider ON leagues(provider, provider_league_id);
CREATE INDEX idx_leagues_country ON leagues(country);
CREATE INDEX idx_leagues_is_active ON leagues(is_active);
CREATE INDEX idx_leagues_deleted_at ON leagues(deleted_at);

CREATE TRIGGER update_leagues_updated_at BEFORE UPDATE ON leagues
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
