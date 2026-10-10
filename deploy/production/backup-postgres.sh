#!/usr/bin/env bash
set -euo pipefail
umask 077

backup_dir=/data/backups/ai-platform/postgres
env_file=/data/ai-platform/production.env
if [[ ! -s "$env_file" ]]; then
    echo "生产环境文件缺失或为空：$env_file" >&2
    exit 1
fi

release=$(awk -F= '$1 == "AI_PLATFORM_RELEASE" {print $2}' "$env_file")
if [[ ! "$release" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$ ]]; then
    echo "AI_PLATFORM_RELEASE 格式无效" >&2
    exit 1
fi
compose_file="/data/ai-platform/bridge-compose.yml"
if [[ ! -f "$compose_file" ]]; then
    echo "生产 Compose 文件不存在：$compose_file" >&2
    exit 1
fi
install -d -o root -g root -m 700 "$backup_dir"
# shortcut: PID 防止并发导出共用文件；若并发负载或临时超出保留数，再用 flock 串行化。
timestamp="$(date -u +%Y%m%dT%H%M%SZ)-$$"
backup_file="$backup_dir/ai-platform-$timestamp.dump"
partial_file="$backup_file.partial"
partial_checksum="$partial_file.sha256"
if [[ -e "$backup_file" || -e "$partial_file" || -e "$backup_file.sha256" ]]; then
    echo "备份文件已存在，拒绝覆盖" >&2
    exit 1
fi
trap 'rm -f -- "$partial_file" "$partial_checksum"' EXIT
trap 'exit 1' HUP INT TERM

docker compose --project-name ai-platform --env-file "$env_file" -f "$compose_file" \
    exec -T postgres sh -ec 'pg_dump --format=custom --no-owner --no-acl --username "$POSTGRES_USER" "$POSTGRES_DB"' \
    > "$partial_file"
test -s "$partial_file"

checksum=$(sha256sum "$partial_file" | cut -d' ' -f1)
printf '%s  %s\n' "$checksum" "$partial_file" | sha256sum --check --status -
printf '%s  %s\n' "$checksum" "$(basename "$backup_file")" > "$partial_checksum"
mv "$partial_file" "$backup_file"
mv "$partial_checksum" "$backup_file.sha256"
(
    cd "$backup_dir"
    sha256sum --check "$(basename "$backup_file").sha256"
)

mapfile -t backups < <(find "$backup_dir" -maxdepth 1 -type f -name 'ai-platform-*.dump' -printf '%f\n' | sort -r)
for old_backup in "${backups[@]:14}"; do
    rm -f -- "$backup_dir/$old_backup" "$backup_dir/$old_backup.sha256"
done

printf '备份校验通过：%s（%s 字节）\n' "$(basename "$backup_file")" "$(stat -c %s "$backup_file")"
