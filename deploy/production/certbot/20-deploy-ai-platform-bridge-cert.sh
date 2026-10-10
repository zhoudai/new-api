#!/bin/sh
set -eu

[ "${RENEWED_LINEAGE:-}" = "/etc/letsencrypt/live/bridge.al-bei.cn" ] || exit 0

target=/data/ai-platform/bridge/tls
install -d -o root -g 65534 -m 0750 "$target"
install -o root -g 65534 -m 0440 "$RENEWED_LINEAGE/fullchain.pem" "$target/fullchain.pem"
install -o root -g 65534 -m 0440 "$RENEWED_LINEAGE/privkey.pem" "$target/privkey.pem"

if systemctl is-active --quiet ai-platform-bridge.service; then
	systemctl restart ai-platform-bridge.service
fi
