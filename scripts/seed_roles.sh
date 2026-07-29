#!/bin/bash

# Seed default roles into database

set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-clap}"

echo "Seeding default roles..."

PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF
INSERT INTO roles (name, description) VALUES
    ('admin', 'System administrator with full access'),
    ('club_admin', 'Club administrator with limited administrative access'),
    ('moderator', 'Content moderator with limited administrative access'),
    ('user', 'Regular user with standard access')
ON CONFLICT (name) DO NOTHING;
EOF

echo "Default roles seeded successfully"
