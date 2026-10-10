#!/usr/bin/env bash
set -euo pipefail

readonly docker_user_chain="DOCKER-USER"
readonly bridge_rules_chain="AI_PLATFORM_BRIDGE_IPPORT"
readonly bridge_container_ip="172.30.250.3"
readonly bridge_port="3001"
readonly production_env_file="/data/ai-platform/production.env"
readonly bridge_compose_file="/data/ai-platform/bridge-compose.yml"

if (( EUID != 0 )); then
  echo "此操作需要 root 权限" >&2
  exit 1
fi

if [[ "${1:-}" == "apply" ]]; then
if ! configured_bridge_network="$(docker compose --project-name ai-platform --env-file "$production_env_file" \
  --file "$bridge_compose_file" --profile hdzc config --format json 2>/dev/null | python3 -c '
import json
import sys

try:
    compose = json.load(sys.stdin)
    service = compose["services"]["bridge"]
    address = service["networks"]["hdzc-internal"]["ipv4_address"]
    network_name = compose["networks"]["hdzc-internal"]["name"]
    listen_addr = service["environment"]["AI_BRIDGE_LISTEN_ADDR"]
    if not network_name or listen_addr != f"{address}:{sys.argv[1]}":
        raise ValueError
except (KeyError, TypeError, ValueError, json.JSONDecodeError):
    sys.exit(1)
print(f"{address}\t{network_name}")
' "$bridge_port")"; then
  echo "无法验证 Bridge Compose 地址，未应用防火墙规则" >&2
  exit 1
fi
IFS=$'\t' read -r configured_bridge_ip configured_network_name <<< "$configured_bridge_network"
if [[ "$configured_bridge_ip" != "$bridge_container_ip" ]]; then
  echo "防火墙地址与 Bridge Compose 不一致，未应用防火墙规则" >&2
  exit 1
fi
if ! hdzc_subnet="$(docker network inspect --format '{{json .IPAM.Config}}' "$configured_network_name" 2>/dev/null | python3 -c '
import ipaddress
import json
import sys

try:
    configs = json.load(sys.stdin)
    subnets = [ipaddress.ip_network(item["Subnet"], strict=False) for item in configs if item.get("Subnet")]
    ipv4_subnets = [subnet for subnet in subnets if subnet.version == 4]
    bridge_ip = ipaddress.ip_address(sys.argv[1])
    if len(ipv4_subnets) != 1 or bridge_ip not in ipv4_subnets[0]:
        raise ValueError
except (AttributeError, KeyError, TypeError, ValueError, json.JSONDecodeError):
    sys.exit(1)
print(ipv4_subnets[0])
' "$bridge_container_ip")"; then
  echo "无法验证 Bridge 地址属于外部 Docker 网络的唯一 IPv4 子网，未应用防火墙规则" >&2
  exit 1
fi
fi

apply_rules() {
  iptables -w -S "$docker_user_chain" >/dev/null
  iptables -w -N "$bridge_rules_chain" 2>/dev/null || true

  # DOCKER-USER 中匹配 DNAT 后的容器地址；只放行 HZC Docker 网段访问 Bridge。
  while iptables -w -C "$docker_user_chain" -j "$bridge_rules_chain" 2>/dev/null; do
    iptables -w -D "$docker_user_chain" -j "$bridge_rules_chain"
  done
  iptables -w -F "$bridge_rules_chain"
  iptables -w -A "$bridge_rules_chain" -p tcp -s "$hdzc_subnet" -d "$bridge_container_ip" --dport "$bridge_port" -j ACCEPT
  iptables -w -A "$bridge_rules_chain" -p tcp -d "$bridge_container_ip" --dport "$bridge_port" -j DROP
  iptables -w -A "$bridge_rules_chain" -j RETURN
  iptables -w -I "$docker_user_chain" 1 -j "$bridge_rules_chain"
}

remove_rules() {
  while iptables -w -C "$docker_user_chain" -j "$bridge_rules_chain" 2>/dev/null; do
    iptables -w -D "$docker_user_chain" -j "$bridge_rules_chain"
  done
  if iptables -w -S "$bridge_rules_chain" >/dev/null 2>&1; then
    iptables -w -F "$bridge_rules_chain"
    iptables -w -X "$bridge_rules_chain"
  fi
}

case "${1:-}" in
  apply)
    apply_rules
    ;;
  remove)
    remove_rules
    ;;
  *)
    echo "用法：$0 apply|remove" >&2
    exit 2
    ;;
esac
