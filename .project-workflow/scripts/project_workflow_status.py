#!/usr/bin/env python3
"""读取或初始化目标项目的工作流状态。"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from workflow_state import STATE_RELATIVE_PATH, find_state, read_field


def state_path(target: Path) -> Path:
    return target.resolve() / STATE_RELATIVE_PATH


def initialize(target: Path, project_name: str) -> Path:
    destination = state_path(target)
    if destination.exists():
        return destination

    template = Path(__file__).resolve().parent.parent / "references" / "project-workflow-state-template.md"
    content = template.read_text(encoding="utf-8").replace("<项目名称>", project_name)
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text(content, encoding="utf-8")
    return destination


def main() -> None:
    parser = argparse.ArgumentParser(description="读取或初始化项目工作流状态")
    parser.add_argument("target", nargs="?", default=".", help="目标项目目录")
    parser.add_argument("--initialize", action="store_true", help="状态不存在时按模板初始化")
    parser.add_argument("--project", help="初始化时写入的项目名称")
    args = parser.parse_args()

    target = Path(args.target)
    if not target.is_dir():
        parser.error(f"目标目录不存在：{target}")
    if args.initialize and not args.project:
        parser.error("--initialize 必须同时提供 --project")

    destination = find_state(target)
    if destination is None and args.initialize:
        destination = initialize(target, args.project)

    if destination is None:
        print(json.dumps({"state": "missing", "target": str(target.resolve())}, ensure_ascii=False))
        return

    content = destination.read_text(encoding="utf-8")
    print(
        json.dumps(
            {
                "state": "found",
                "path": str(destination),
                "project": read_field(content, "名称"),
                "stage": read_field(content, "当前阶段"),
                "gate": read_field(content, "结论", "当前门禁"),
                "db_mcp": read_field(content, "db-mcp 状态"),
                "next": read_field(content, "Next"),
            },
            ensure_ascii=False,
        )
    )


if __name__ == "__main__":
    main()
