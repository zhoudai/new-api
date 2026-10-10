#!/usr/bin/env python3
"""只读检查项目是否已正确接入系统交付工作流。"""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

from workflow_state import STATE_RELATIVE_PATH

REQUIRED_AGENT_MARKERS = (
    ".project-workflow/PROJECT.md",
    "project_workflow_status.py",
    "begin_workflow_task.py",
    "verify_workflow_completion.py",
    "verify_workflow_gate.py",
)
REQUIRED_SCRIPTS = (
    "project_workflow_status.py",
    "begin_workflow_task.py",
    "verify_workflow_completion.py",
    "verify_workflow_gate.py",
)


def check_command(script: Path) -> str | None:
    result = subprocess.run(
        [sys.executable, str(script), "--help"],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode == 0:
        return None
    detail = result.stderr.strip().splitlines()
    return detail[-1] if detail else f"退出码 {result.returncode}"


def main() -> None:
    parser = argparse.ArgumentParser(description="只读检查项目工作流接入状态")
    parser.add_argument("target", nargs="?", default=".", help="目标项目目录")
    args = parser.parse_args()

    target = Path(args.target).resolve()
    if not target.is_dir():
        parser.error(f"目标目录不存在：{target}")

    issues: list[str] = []
    state_file = target / STATE_RELATIVE_PATH
    if not state_file.is_file():
        issues.append(f"缺少 {STATE_RELATIVE_PATH}")

    agents_file = target / "AGENTS.md"
    if not agents_file.is_file():
        issues.append("缺少 AGENTS.md")
    else:
        content = agents_file.read_text(encoding="utf-8")
        missing = [marker for marker in REQUIRED_AGENT_MARKERS if marker not in content]
        if missing:
            issues.append(f"AGENTS.md 缺少工作流规则：{'、'.join(missing)}")

    scripts_dir = target / ".project-workflow" / "scripts"
    for name in REQUIRED_SCRIPTS:
        script = scripts_dir / name
        if not script.is_file():
            issues.append(f"缺少工作流脚本：{script}")
            continue
        error = check_command(script)
        if error:
            issues.append(f"脚本不可运行：{name}（{error}）")

    if issues:
        raise SystemExit("工作流接入自检未通过：" + "；".join(issues))
    print(f"工作流接入自检通过：{target}")


if __name__ == "__main__":
    main()
