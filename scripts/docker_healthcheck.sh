#!/bin/sh
# Quick Docker stack health check for Clap.
# Run on the server from the project root:
#   sh scripts/docker_healthcheck.sh

set -eu

API_DIRECT="${API_DIRECT:-http://127.0.0.1:8080}"
API_NGINX="${API_NGINX:-http://127.0.0.1:8081}"
SWAGGER_PATH="${SWAGGER_PATH:-/swagger/index.html}"

ok=0
fail=0

pass() { echo "  OK  $*"; ok=$((ok + 1)); }
bad()  { echo "  FAIL $*"; fail=$((fail + 1)); }

echo "== Containers =="
if ! command -v docker >/dev/null 2>&1; then
	echo "docker not found"
	exit 1
fi

docker compose ps || docker-compose ps

echo
echo "== Expected services =="
for name in clap_postgres clap_redis clap_api clap_nginx; do
	status=$(docker inspect -f '{{.State.Status}}' "$name" 2>/dev/null || echo missing)
	health=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}n/a{{end}}' "$name" 2>/dev/null || echo missing)
	if [ "$status" = "running" ]; then
		pass "$name status=$status health=$health"
	else
		bad "$name status=$status health=$health"
	fi
done

migrate_status=$(docker inspect -f '{{.State.Status}} exit={{.State.ExitCode}}' clap_migrate 2>/dev/null || echo missing)
case "$migrate_status" in
	*exit=0*) pass "clap_migrate $migrate_status" ;;
	*) bad "clap_migrate $migrate_status (want exited with 0)" ;;
esac

echo
echo "== HTTP checks =="
check_http() {
	url=$1
	label=$2
	code=$(curl -sS -o /tmp/clap_health_body -w '%{http_code}' --max-time 5 "$url" 2>/dev/null || echo 000)
	if [ "$code" = "200" ]; then
		pass "$label -> $code ($url)"
		# show body briefly for health
		if echo "$label" | grep -q health; then
			echo "       body: $(tr -d '\n' </tmp/clap_health_body | cut -c1-120)"
		fi
	else
		bad "$label -> $code ($url)"
	fi
}

check_http "$API_DIRECT/health" "api /health"
check_http "$API_NGINX/health" "nginx /health"
check_http "$API_NGINX$SWAGGER_PATH" "nginx swagger"

echo
echo "== Recent migrate logs =="
docker logs --tail 30 clap_migrate 2>/dev/null || echo "(no migrate logs)"

echo
echo "== Summary: ok=$ok fail=$fail =="
[ "$fail" -eq 0 ]
