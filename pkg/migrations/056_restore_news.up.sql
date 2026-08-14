-- Restore club news after 038 dropped the table.

CREATE TABLE IF NOT EXISTS news (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    club_id      UUID REFERENCES clubs(id) ON DELETE SET NULL,
    title        VARCHAR(500) NOT NULL,
    body         TEXT,
    image_url    VARCHAR(500),
    published_at TIMESTAMP NOT NULL DEFAULT NOW(),
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMP,
    created_by   UUID,
    updated_by   UUID
);

CREATE INDEX IF NOT EXISTS idx_news_published_at ON news (published_at DESC);
CREATE INDEX IF NOT EXISTS idx_news_deleted_at   ON news (deleted_at);

INSERT INTO news (id, title, body, image_url, published_at, is_active)
SELECT
    'b1000000-0000-4000-8000-000000000001',
    'Flick: the squad is ready for the next stretch',
    '<p>Training this week focused on high pressing and quick transitions ahead of the next La Liga fixture.</p><p>The coach said the group is physically sharp and mentally locked in for the run-in.</p>',
    'https://images.unsplash.com/photo-1574629810360-7efbbe195018?auto=format&fit=crop&w=1200&q=80',
    NOW() - INTERVAL '2 hours',
    TRUE
WHERE NOT EXISTS (
    SELECT 1 FROM news WHERE id = 'b1000000-0000-4000-8000-000000000001'
);

INSERT INTO news (id, title, body, image_url, published_at, is_active)
SELECT
    'b1000000-0000-4000-8000-000000000002',
    'Camp Nou update: matchday operations',
    '<p>Turnstiles open 90 minutes before kickoff. Food and merch stands will stay open through half-time.</p><p>Fans are asked to arrive early and follow steward instructions in the concourse.</p>',
    'https://images.unsplash.com/photo-1522778119026-d647f0596c23?auto=format&fit=crop&w=1200&q=80',
    NOW() - INTERVAL '1 day',
    TRUE
WHERE NOT EXISTS (
    SELECT 1 FROM news WHERE id = 'b1000000-0000-4000-8000-000000000002'
);

INSERT INTO news (id, title, body, image_url, published_at, is_active)
SELECT
    'b1000000-0000-4000-8000-000000000003',
    'Academy night: next generation on the pitch',
    '<p>The under-19s host a midweek showcase at the training ground. Admission is free for season-ticket holders.</p>',
    'https://images.unsplash.com/photo-1431324155629-1a6deb1dec8d?auto=format&fit=crop&w=1200&q=80',
    NOW() - INTERVAL '3 days',
    TRUE
WHERE NOT EXISTS (
    SELECT 1 FROM news WHERE id = 'b1000000-0000-4000-8000-000000000003'
);
