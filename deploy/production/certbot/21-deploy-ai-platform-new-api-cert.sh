#!/bin/sh
set -eu

[ "${RENEWED_LINEAGE:-}" = "/etc/letsencrypt/live/new-api.al-bei.cn" ] || exit 0

target=/data/ai-platform/new-api-tls
install -d -o root -g root -m 0750 "$target"
install -o root -g root -m 0440 "$RENEWED_LINEAGE/fullchain.pem" "$target/fullchain.pem"
install -o root -g root -m 0440 "$RENEWED_LINEAGE/privkey.pem" "$target/privkey.pem"

if docker ps --filter label=com.docker.compose.project=ai-platform --filter label=com.docker.compose.service=new-api --filter status=running -q | grep -q .; then
	docker compose --project-name ai-platform --env-file /data/ai-platform/production.env --file /data/ai-platform/bridge-compose.yml restart new-api
fi
