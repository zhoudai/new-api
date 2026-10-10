#!/usr/bin/env python3
"""校验项目工作流状态是否具备结束当前任务的最小交接信息。"""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

from workflow_state import find_state, read_field

CHECKPOINT_NAME = ".task-checkpoint.json"
REQUIRED_FIELDS = ("名称", "当前阶段", "结论", "db-mcp 状态", "Flow", "Next")
TEST_ASSET_FIELDS = ("本次变更", "最小验证")
TODO_FIELDS = ("延后事项", "待办清单")


def main() -> None:
    parser = argparse.ArgumentParser(description="校验项目工作流结束交接")
    parser.add_argument("target", nargs="?", default=".", help="项目目录或其子目录")
    parser.add_argument("--require-change", action="store_true", help="要求状态文件已在本轮任务中回写")
    args = parser.parse_args()

    state_file = find_state(Path(args.target))
    if state_file is None:
        raise SystemExit("工作流结束校验未通过：未找到 .project-workflow/PROJECT.md。")

    content = state_file.read_text(encoding="utf-8")
    missing = [
        field
        for field in REQUIRED_FIELDS
        if not read_field(content, field, "当前门禁" if field == "结论" else None)
    ]
    if missing:
        raise SystemExit(f"工作流结束校验未通过：缺少 {'、'.join(missing)}。")

    missing_todo_fields = [field for field in TODO_FIELDS if not read_field(content, field)]
    if missing_todo_fields:
        raise SystemExit(f"工作流结束校验未通过：缺少后续待办“{'、'.join(missing_todo_fields)}”。")

    deferred_items = read_field(content, "延后事项")
    todo_items = read_field(content, "待办清单")
    if deferred_items != "无" and todo_items in (None, "-", "无"):
        raise SystemExit("工作流结束校验未通过：存在延后事项时必须创建待办，才能进入下一步。")

    current_stage = read_field(content, "当前阶段")
    if current_stage and current_stage.startswith(("5. 迭代开发", "7. 运行与优化")):
        missing_test_assets = [
            field
            for field in TEST_ASSET_FIELDS
            if not read_field(content, field) or read_field(content, field) == "-"
        ]
        if missing_test_assets:
            raise SystemExit(
                f"工作流结束校验未通过：{current_stage} 必须回写测试资产“"
                f"{'、'.join(missing_test_assets)}”。"
            )

    if "## 历史记录" not in content:
        raise SystemExit("工作流结束校验未通过：缺少历史记录。")

    if args.require_change:
        checkpoint = state_file.parent / CHECKPOINT_NAME
        if not checkpoint.is_file():
            raise SystemExit("工作流结束校验未通过：未找到本轮任务快照。")
        try:
            snapshot = json.loads(checkpoint.read_text(encoding="utf-8"))
            if not isinstance(snapshot, dict):
                raise TypeError
            baseline = snapshot["fingerprint"]
            if not isinstance(baseline, str):
                raise TypeError
        except (json.JSONDecodeError, KeyError, TypeError):
            raise SystemExit("工作流结束校验未通过：本轮任务快照无效。") from None
        current = hashlib.sha256(state_file.read_bytes()).hexdigest()
        if baseline == current:
            raise SystemExit("工作流结束校验未通过：本轮未回写 PROJECT.md。")

    print(f"工作流结束校验通过：{state_file}")


if __name__ == "__main__":
    main()
