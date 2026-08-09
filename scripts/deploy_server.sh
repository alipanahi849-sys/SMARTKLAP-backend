#!/bin/sh
# Deploy SMARTKLAP backend on the production server.
# Run on the server as root:
#   cd /root/SMARTKLAP-backend && sh scripts/deploy_server.sh

set -eu

REPO_DIR="${REPO_DIR:-$(cd "$(dirname "$0")/.." && pwd)}"
cd "$REPO_DIR"

echo "==> Pull latest code"
git fetch origin main
git checkout main
git pull --ff-only origin main

echo "==> Rebuild and restart containers"
docker compose pull || true
docker compose up -d --build

echo "==> Wait for API health"
i=0
until wget -qO- http://127.0.0.1:8080/health >/dev/null 2>&1 || wget -qO- http://127.0.0.1:8081/health >/dev/null 2>&1; do
	i=$((i + 1))
	if [ "$i" -ge 60 ]; then
		echo "Health check timed out" >&2
		docker compose ps
		docker compose logs --tail 50 api migrate nginx
		exit 1
	fi
	sleep 2
done

echo "==> Deploy complete"
docker compose ps
wget -qO- http://127.0.0.1:8081/health || wget -qO- http://127.0.0.1/health
