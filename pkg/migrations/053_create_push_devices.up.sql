-- Migration: 053_create_push_devices
-- Stores one FCM token per physical device. The same token can move between
-- users when a device signs out and another account signs in.

CREATE TABLE IF NOT EXISTS push_devices (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fcm_token  TEXT NOT NULL,
    platform   VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT push_devices_platform_check CHECK (platform IN ('ios', 'android')),
    CONSTRAINT push_devices_fcm_token_unique UNIQUE (fcm_token)
);

CREATE INDEX IF NOT EXISTS idx_push_devices_user_id ON push_devices (user_id);
