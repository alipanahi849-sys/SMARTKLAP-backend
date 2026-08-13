#!/usr/bin/env bash
# Ramp WebSocket connections against the Clap realtime stack.
#
# Examples:
#   export CLAP_WS_TOKEN="$(curl -s ...)"   # your JWT
#   ./scripts/run_ws_loadtest.sh -clients 2000 -ramp 30s -duration 60s
#   ./scripts/run_ws_loadtest.sh -addr ws://localhost:8080/api/v1/realtime/ws -clients 5000
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ADDR="${CLAP_WS_ADDR:-ws://localhost:8081/api/v1/realtime/ws}"
TOKEN="${CLAP_WS_TOKEN:-}"

usage() {
  cat <<'EOF'
Usage: run_ws_loadtest.sh [ws_loadtest flags]

Environment:
  CLAP_WS_TOKEN   JWT access token (required unless -token passed)
  CLAP_WS_ADDR    default ws://localhost:8081/api/v1/realtime/ws

Common flags (forwarded to ws_loadtest):
  -clients N      target connections (default 100)
  -ramp DURATION  ramp-up, e.g. 60s
  -duration DURATION  hold time after ramp
  -match UUID     subscribe to match channel
  -addr URL       WebSocket URL

EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

EXTRA_ARGS=("$@")
HAS_TOKEN_FLAG=false
for arg in "${EXTRA_ARGS[@]}"; do
  if [[ "$arg" == "-token" ]]; then
    HAS_TOKEN_FLAG=true
    break
  fi
done

if [[ -z "$TOKEN" && "$HAS_TOKEN_FLAG" == false ]]; then
  echo "error: set CLAP_WS_TOKEN or pass -token" >&2
  usage
  exit 2
fi

echo "Checking API health..."
if curl -sf "http://localhost:8081/health" >/dev/null 2>&1; then
  echo "nginx /health OK"
elif curl -sf "http://localhost:8080/health" >/dev/null 2>&1; then
  echo "api /health OK (direct)"
else
  echo "warning: health check failed — is docker compose up?" >&2
fi

ARGS=(-addr "$ADDR")
if [[ -n "$TOKEN" ]]; then
  ARGS+=(-token "$TOKEN")
fi
ARGS+=("${EXTRA_ARGS[@]}")

echo "Running ws_loadtest ${ARGS[*]}"
exec go run ./scripts/ws_loadtest "${ARGS[@]}"
