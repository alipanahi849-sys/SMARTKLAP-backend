#!/bin/bash

# Migration script for Clap backend
# Usage: ./scripts/migrate.sh [up|down]

set -e

MIGRATIONS_DIR="../pkg/migrations"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-clap}"

ACTION=${1:-up}

if [ "$ACTION" = "up" ]; then
    echo "Running up migrations..."
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $MIGRATIONS_DIR/001_init_schema.up.sql
    echo "Migrations completed successfully"
elif [ "$ACTION" = "down" ]; then
    echo "Running down migrations..."
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $MIGRATIONS_DIR/001_init_schema.down.sql
    echo "Rollback completed successfully"
else
    echo "Usage: ./scripts/migrate.sh [up|down]"
    exit 1
fi
