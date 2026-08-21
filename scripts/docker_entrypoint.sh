#!/bin/sh
# Run SQL migrations (and role seed) then start the API.
# Used as the container CMD on Render and any other Docker host.
set -eu

export MIGRATIONS_DIR="${MIGRATIONS_DIR:-/root/migrations}"

echo "Running database migrations..."
sh /root/migrate_and_seed.sh

echo "Starting API..."
exec /root/main
