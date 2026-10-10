from __future__ import annotations

import json
import re
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]


class BridgeDeploymentConfigTest(unittest.TestCase):
    def test_listener_firewall_and_compose_use_the_hzc_network_address(self) -> None:
        compose = (ROOT / "deploy/production/docker-compose.yml").read_text(encoding="utf-8")
        firewall = (ROOT / "deploy/production/bridge-firewall.sh").read_text(encoding="utf-8")
        listen_addr = re.search(r"AI_BRIDGE_LISTEN_ADDR:\s*([^\s]+)", compose)
        compose_ip = re.search(r"ipv4_address:\s*([0-9.]+)", compose)
        firewall_ip = re.search(r'bridge_container_ip="([0-9.]+)"', firewall)

        self.assertIsNotNone(listen_addr)
        self.assertIsNotNone(compose_ip)
        self.assertIsNotNone(firewall_ip)
        self.assertEqual(listen_addr.group(1), f"{compose_ip.group(1)}:3001")
        self.assertEqual(firewall_ip.group(1), compose_ip.group(1))

    def test_firewall_uses_the_external_network_subnet(self) -> None:
        firewall = (ROOT / "deploy/production/bridge-firewall.sh").read_text(encoding="utf-8")
        guide = (ROOT / "docs/部署与凭据说明.md").read_text(encoding="utf-8")

        self.assertIn("docker network inspect --format '{{json .IPAM.Config}}'", firewall)
        self.assertIn('compose["networks"]["hdzc-internal"]["name"]', firewall)
        self.assertIn("ipaddress.ip_network(item[\"Subnet\"], strict=False)", firewall)
        self.assertIn("len(ipv4_subnets) != 1", firewall)
        self.assertIn("bridge_ip not in ipv4_subnets[0]", firewall)
        self.assertNotIn('readonly hdzc_subnet="172.', firewall)
        self.assertIn("读取 Compose 所用外部 Docker 网络的实际 IPv4 子网", guide)
        self.assertIn("网络子网不唯一或地址不匹配时，脚本拒绝应用规则", guide)

    def test_systemd_supervises_bridge_after_process_exit(self) -> None:
        compose = (ROOT / "deploy/production/docker-compose.yml").read_text(encoding="utf-8")
        bridge = re.search(r"(?ms)^  bridge:\n(.*?)(?=^  [\w-]+:\n|^networks:\n|\Z)", compose)
        unit = (ROOT / "deploy/production/systemd/ai-platform-bridge.service").read_text(encoding="utf-8")

        self.assertIsNotNone(bridge)
        self.assertRegex(bridge.group(1), r'(?m)^    restart:\s+"no"\s*$')
        self.assertIn("Type=simple", unit)
        self.assertIn("Restart=always", unit)
        self.assertIn("RestartSec=5s", unit)
        self.assertIn("up --no-deps --exit-code-from bridge bridge", unit)
        self.assertNotIn("--detach", unit)

    def test_bridge_requires_tls_without_http_fallback(self) -> None:
        source = (ROOT / "bridge/main.go").read_text(encoding="utf-8")
        compose = (ROOT / "deploy/production/docker-compose.yml").read_text(encoding="utf-8")
        guide = (ROOT / "docs/部署与凭据说明.md").read_text(encoding="utf-8")

        self.assertIn("必须同时配置，禁止 HTTP 回退", source)
        self.assertNotIn("server.ListenAndServe()", source)
        self.assertIn("AI_BRIDGE_TLS_CERT_FILE:", compose)
        self.assertIn("AI_BRIDGE_TLS_KEY_FILE:", compose)
        self.assertIn("不会回退到 HTTP", guide)

    def test_local_bridge_profiles_provide_tls_and_https_readiness(self) -> None:
        compose = (ROOT / "docker-compose.yml").read_text(encoding="utf-8")
        hdzc_env = (ROOT / "bridge/.env.hedezhongcheng.example").read_text(encoding="utf-8")
        edu_env = (ROOT / "bridge/.env.edu-training.example").read_text(encoding="utf-8")
        guide = (ROOT / "docs/部署与凭据说明.md").read_text(encoding="utf-8")
        hdzc = re.search(r"(?ms)^  project-bridge:\n(.*?)(?=^  edu-training-project-bridge:\n)", compose)
        edu = re.search(r"(?ms)^  edu-training-project-bridge:\n(.*?)(?=^  new-api:\n)", compose)

        self.assertIsNotNone(hdzc)
        self.assertIsNotNone(edu)
        for service in (hdzc.group(1), edu.group(1)):
            self.assertIn("AI_BRIDGE_TLS_CERT_FILE=/run/bridge-tls/fullchain.pem", service)
            self.assertIn("AI_BRIDGE_TLS_KEY_FILE=/run/bridge-tls/privkey.pem", service)
            self.assertIn("https://localhost:3001/health/ready", service)
            self.assertIn("--no-check-certificate", service)
        self.assertIn("AI_BRIDGE_TLS_CERT_HOST_DIR=./bridge/certs/hedezhongcheng", hdzc_env)
        self.assertIn("EDU_TRAINING_BRIDGE_TLS_CERT_HOST_DIR=./bridge/certs/edu-training", edu_env)
        self.assertIn("https://localhost:3001/health/ready", guide)
        self.assertIn("https://localhost:3002/health/ready", guide)
        self.assertNotIn("http://127.0.0.1:3001/health/ready", guide)

    def test_bridge_build_context_excludes_local_tls_certificates(self) -> None:
        dockerignore = (ROOT / "bridge/.dockerignore").read_text(encoding="utf-8")

        self.assertIn("certs/", dockerignore.splitlines())
        self.assertIn("*.pem", dockerignore.splitlines())

    def test_gitignore_excludes_local_bridge_tls_certificates(self) -> None:
        result = subprocess.run(
            ["git", "check-ignore", "--quiet", "--no-index", "bridge/certs/hedezhongcheng/privkey.pem"],
            cwd=ROOT,
            check=False,
        )

        self.assertEqual(result.returncode, 0)

    def test_root_dockerignore_excludes_bridge_tls_certificates(self) -> None:
        dockerignore = (ROOT / ".dockerignore").read_text(encoding="utf-8")

        self.assertIn("/bridge/certs", dockerignore.splitlines())

    def test_production_bridge_build_cross_compiles_for_compose_platform(self) -> None:
        dockerfile = (ROOT / "bridge/Dockerfile.production").read_text(encoding="utf-8")
        compose = (ROOT / "deploy/production/docker-compose.yml").read_text(encoding="utf-8")
        bridge = re.search(r"(?ms)^  bridge:\n(.*?)(?=^  [\w-]+:\n|^networks:\n|\Z)", compose)

        self.assertIn("FROM --platform=$BUILDPLATFORM golang:", dockerfile)
        self.assertIn("ARG TARGETOS", dockerfile)
        self.assertIn("ARG TARGETARCH", dockerfile)
        self.assertIn("GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}", dockerfile)
        self.assertIsNotNone(bridge)
        self.assertRegex(bridge.group(1), r"(?m)^    platform:\s*linux/amd64\s*$")

    def test_firewall_unit_is_installed_before_bridge_service(self) -> None:
        unit = (ROOT / "deploy/production/systemd/ai-platform-bridge.service").read_text(encoding="utf-8")
        checklist = (ROOT / "deploy/production/生产部署执行清单.md").read_text(encoding="utf-8")

        self.assertIn("Requires=docker.service ai-platform-bridge-firewall.service", unit)
        self.assertIn("After=docker.service ai-platform-bridge-firewall.service", unit)
        self.assertIn("PartOf=docker.service", unit)
        self.assertIn("先安装并启用 Bridge 防火墙 systemd 单元", checklist)
        self.assertIn("再安装或启用硬依赖该单元的 Bridge systemd 服务", checklist)

    def test_bridge_recovery_docs_use_systemd_restart_and_readiness(self) -> None:
        guide = (ROOT / "docs/部署与凭据说明.md").read_text(encoding="utf-8")

        self.assertIn("systemctl restart ai-platform-bridge.service", guide)
        self.assertIn("前台运行 Compose，并在 Bridge 容器退出时重启", guide)
        self.assertIn("先恢复防火墙规则，再启动 Bridge", guide)
        self.assertIn("`active` 只表示 Compose 进程仍在运行", guide)
        self.assertIn("还要检查容器状态并访问 readiness", guide)

    def test_production_status_lists_open_p08_blockers(self) -> None:
        status = json.loads((ROOT / "docs/生产服务器部署状态.json").read_text(encoding="utf-8"))
        p08 = next(item for item in status["phases"] if item["id"] == "P08")
        blockers = "\n".join(p08["blockedBy"])

        self.assertIn("防火墙 systemd 单元尚未安装", blockers)
        self.assertIn("生产证书迁移、Compose/Certbot 更新及 readiness 验证尚未执行", blockers)
        self.assertIn("官方机器可读价格源与节假日计费规则未确认", blockers)
        self.assertIn("价格编辑器修复仅在本地，尚未部署", blockers)

    def test_p05_checklist_separates_completed_evidence_from_unpublished_backup_hardening(self) -> None:
        checklist = (ROOT / "deploy/production/生产部署执行清单.md").read_text(encoding="utf-8")
        p05 = next(line for line in checklist.splitlines() if line.startswith("| P05 |"))

        self.assertIn("✅ 已完成", p05)
        self.assertNotIn("尚未部署", p05)
        self.assertIn("P05 后续加固（未部署）", checklist)
        self.assertIn("不属于上表 P05 已完成的验收证据", checklist)

    def test_new_api_tls_cert_source_is_outside_writable_data_mount(self) -> None:
        compose = (ROOT / "deploy/production/docker-compose.yml").read_text(encoding="utf-8")
        cert_hook = (ROOT / "deploy/production/certbot/21-deploy-ai-platform-new-api-cert.sh").read_text(encoding="utf-8")
        checklist = (ROOT / "deploy/production/生产部署执行清单.md").read_text(encoding="utf-8")
        data_mount = re.search(r"(?ms)^      - type: bind\n        source: (/data/[^\n]+)\n        target: /data\n", compose)
        tls_mount = re.search(
            r"(?ms)^      - type: bind\n        source: (/data/[^\n]+)\n        target: /run/ai-platform/new-api-tls\n        read_only: true\n",
            compose,
        )

        self.assertIsNotNone(data_mount)
        self.assertIsNotNone(tls_mount)
        self.assertFalse(tls_mount.group(1).startswith(f"{data_mount.group(1)}/"))
        self.assertIn(f"target={tls_mount.group(1)}", cert_hook)
        self.assertIn("先迁移 TLS 证书到独立目录并确认只读挂载与 Certbot 钩子一致", checklist)

    def test_backup_cleanup_runs_after_termination_signal(self) -> None:
        script = (ROOT / "deploy/production/backup-postgres.sh").read_text(encoding="utf-8")
        trap_lines = [line for line in script.splitlines() if line.startswith("trap ")]

        self.assertEqual(len(trap_lines), 2)
        self.assertIn("'rm -f -- \"$partial_file\" \"$partial_checksum\"' EXIT", trap_lines[0])
        self.assertIn("'exit 1' HUP INT TERM", trap_lines[1])

        with tempfile.TemporaryDirectory() as temp_dir:
            partial = Path(temp_dir) / "backup.dump.partial"
            checksum = Path(temp_dir) / "backup.dump.partial.sha256"
            command = (
                "partial_file=$1\n"
                "partial_checksum=$2\n"
                + "\n".join(trap_lines)
                + '\ntouch "$partial_file" "$partial_checksum"\nkill -TERM "$$"\n'
            )
            result = subprocess.run(
                ["bash", "-c", command, "backup-trap-test", str(partial), str(checksum)],
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertEqual(result.returncode, 1)
            self.assertFalse(partial.exists())
            self.assertFalse(checksum.exists())

    def test_backup_uses_the_fixed_production_compose_path(self) -> None:
        script = (ROOT / "deploy/production/backup-postgres.sh").read_text(encoding="utf-8")
        guide = (ROOT / "docs/部署与凭据说明.md").read_text(encoding="utf-8")

        self.assertIn('compose_file="/data/ai-platform/bridge-compose.yml"', script)
        self.assertNotIn('/data/ai-platform/releases/$release/docker-compose.yml', script)
        self.assertIn('AI_PLATFORM_RELEASE 格式无效', script)
        self.assertIn('读取 `/data/ai-platform/production.env` 和 `/data/ai-platform/bridge-compose.yml`', guide)

    def test_nginx_forwards_websocket_upgrade_headers(self) -> None:
        nginx = (ROOT / "deploy/production/nginx/ai-platform.conf").read_text(encoding="utf-8")
        guide = (ROOT / "docs/部署与凭据说明.md").read_text(encoding="utf-8")

        self.assertIn("map $http_upgrade $connection_upgrade", nginx)
        self.assertIn("proxy_set_header Upgrade $http_upgrade;", nginx)
        self.assertIn("proxy_set_header Connection $connection_upgrade;", nginx)
        self.assertNotIn('proxy_set_header Connection "";', nginx)
        self.assertIn("WebSocket 接口使用 HTTPS/WSS", guide)

    def test_backup_service_retries_failures_within_a_bound(self) -> None:
        service = (ROOT / "deploy/production/systemd/ai-platform-backup.service").read_text(encoding="utf-8")
        guide = (ROOT / "docs/部署与凭据说明.md").read_text(encoding="utf-8")

        self.assertIn("StartLimitIntervalSec=6h", service)
        self.assertIn("StartLimitBurst=4", service)
        self.assertIn("Restart=on-failure", service)
        self.assertIn("RestartSec=5min", service)
        self.assertIn("6 小时内最多启动 4 次（包括首次）", guide)
        self.assertIn("当前没有失败通知配置", guide)


if __name__ == "__main__":
    unittest.main()
