#!/bin/sh
# Run SQL migrations (and role seed) then start the API.
# Used as the container CMD on Render and any other Docker host.
set -eu

export MIGRATIONS_DIR="${MIGRATIONS_DIR:-/root/migrations}"

# Sidecar MailHog (Render): SMTP on localhost so OTP is captured without a
# second service. Free Render web services cannot receive private-network SMTP.
if [ "${SMTP_HOST:-}" = "127.0.0.1" ] || [ "${SMTP_HOST:-}" = "localhost" ] || [ "${MAILHOG_UI:-}" = "1" ]; then
	echo "Starting MailHog (SMTP 127.0.0.1:1025, UI 127.0.0.1:8025)..."
	MailHog \
		-hostname clap-mailhog \
		-smtp-bind-addr 127.0.0.1:1025 \
		-ui-bind-addr 127.0.0.1:8025 \
		-api-bind-addr 127.0.0.1:8025 \
		-ui-web-path mailhog \
		>>/tmp/mailhog.log 2>&1 &
fi

echo "Running database migrations..."
sh /root/migrate_and_seed.sh

echo "Starting API..."
exec /root/main
