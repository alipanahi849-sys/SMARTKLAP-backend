#!/bin/sh
# Idempotent DB migrate + role seed for Docker and local use.
# Usage:
#   MIGRATIONS_DIR=pkg/migrations ./scripts/migrate_and_seed.sh
#   SKIP_SEED=1 ./scripts/migrate_and_seed.sh   # migrations only

set -eu

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-clap}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-pkg/migrations}"
SKIP_SEED="${SKIP_SEED:-0}"

export PGPASSWORD="$DB_PASSWORD"

psql_cmd() {
	psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 "$@"
}

echo "Waiting for PostgreSQL at ${DB_HOST}:${DB_PORT}..."
i=0
until psql_cmd -c '\q' >/dev/null 2>&1; do
	i=$((i + 1))
	if [ "$i" -ge 60 ]; then
		echo "PostgreSQL did not become ready in time" >&2
		exit 1
	fi
	sleep 1
done
echo "PostgreSQL is ready"

if [ ! -d "$MIGRATIONS_DIR" ]; then
	echo "Migrations directory not found: $MIGRATIONS_DIR" >&2
	exit 1
fi

psql_cmd <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
    filename   TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL

echo "Running database migrations from ${MIGRATIONS_DIR}..."
applied=0
skipped=0

# Portable sorted glob (BusyBox / Alpine friendly)
for f in $(ls -1 "$MIGRATIONS_DIR"/*.up.sql 2>/dev/null | sort); do
	name=$(basename "$f")
	already=$(psql_cmd -tAc "SELECT 1 FROM schema_migrations WHERE filename = '${name}'" | tr -d '[:space:]')
	if [ "$already" = "1" ]; then
		echo "  skip  $name"
		skipped=$((skipped + 1))
		continue
	fi

	echo "  apply $name"
	psql_cmd -f "$f" >/dev/null
	psql_cmd -c "INSERT INTO schema_migrations (filename) VALUES ('${name}')" >/dev/null
	applied=$((applied + 1))
done

echo "Migrations done (applied=${applied}, skipped=${skipped})"

if [ "$SKIP_SEED" = "1" ]; then
	echo "Skipping role seed (SKIP_SEED=1)"
	exit 0
fi

echo "Seeding default roles..."
psql_cmd <<'SQL'
INSERT INTO roles (name, description) VALUES
    ('admin', 'System administrator with full access'),
    ('club_admin', 'Club administrator with limited administrative access'),
    ('moderator', 'Content moderator with limited administrative access'),
    ('user', 'Regular user with standard access')
ON CONFLICT (name) DO NOTHING;
SQL
echo "Default roles seeded successfully"
