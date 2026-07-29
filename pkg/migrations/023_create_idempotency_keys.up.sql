-- Migration: 023_create_idempotency_keys
-- Purpose: Generic idempotency store for mutating endpoints.
-- TTL: rows are soft-expired via expires_at; a cleanup job removes old rows.

CREATE TABLE IF NOT EXISTS idempotency_keys (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    key              VARCHAR(255) NOT NULL,
    endpoint         VARCHAR(255) NOT NULL,
    request_hash     VARCHAR(64)  NOT NULL,
    response_payload TEXT         NOT NULL,
    status_code      INT          NOT NULL DEFAULT 200,
    created_at       TIMESTAMP    NOT NULL DEFAULT NOW(),
    expires_at       TIMESTAMP    NOT NULL,

    CONSTRAINT uidx_idempotency_key_endpoint UNIQUE (key, endpoint)
);

-- Fast expiry scan — used by the cleanup endpoint.
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires_at
    ON idempotency_keys (expires_at);
