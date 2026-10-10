#!/usr/bin/env python3
"""校验项目工作流状态，供 Git Hook 或 CI 发布门禁调用。"""

from __future__ import annotations

import argparse
from pathlib import Path

from workflow_state import find_state, read_field

REQUIRED_FIELDS = ("名称", "当前阶段", "结论", "Flow", "Next")


def fail(message: str) -> None:
    raise SystemExit(f"工作流门禁未通过：{message}")


def main() -> None:
    parser = argparse.ArgumentParser(description="校验项目工作流门禁")
    parser.add_argument("target", nargs="?", default=".", help="目标项目目录")
    parser.add_argument(
        "--mode", choices=("commit", "release"), default="commit", help="校验级别"
    )
    args = parser.parse_args()

    state_file = find_state(Path(args.target))
    if state_file is None:
        fail("缺少 .project-workflow/PROJECT.md；请先通过 software-delivery-workflow 初始化项目状态。")

    content = state_file.read_text(encoding="utf-8")
    missing = [
        field
        for field in REQUIRED_FIELDS
        if not read_field(content, field, "当前门禁" if field == "结论" else None)
    ]
    if missing:
        fail(f"状态文件缺少字段：{'、'.join(missing)}。")

    gate = read_field(content, "结论", "当前门禁")
    if args.mode == "commit":
        print(f"工作流提交门禁通过：{read_field(content, '名称')} / {read_field(content, '当前阶段')} / {gate}")
        return

    if gate != "PASS":
        fail(f"发布必须将“当前门禁-结论”更新为 PASS，当前为 {gate}。")
    release_evidence = read_field(content, "发布证据")
    if not release_evidence or release_evidence == "-":
        fail("发布必须记录审查、回滚或监控证据。")
    change_verification = read_field(content, "最小验证")
    if not change_verification or change_verification == "-":
        fail("发布必须记录本次变更的最小验证结果。")
    full_regression = read_field(content, "全量回归")
    if not full_regression or full_regression == "-" or full_regression.startswith("未执行"):
        fail("发布必须记录全量回归结果。")
    print(f"工作流发布门禁通过：{read_field(content, '名称')} / {read_field(content, '当前阶段')}")


if __name__ == "__main__":
    main()
