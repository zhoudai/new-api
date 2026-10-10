#!/usr/bin/env python3
"""记录当前项目状态快照，供任务结束时确认状态已回写。"""

from __future__ import annotations

import argparse
import hashlib
import json
from datetime import datetime, timezone
from pathlib import Path

from workflow_state import find_state

CHECKPOINT_NAME = ".task-checkpoint.json"


def fingerprint(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main() -> None:
    parser = argparse.ArgumentParser(description="开始项目工作流任务")
    parser.add_argument("target", nargs="?", default=".", help="项目目录或其子目录")
    args = parser.parse_args()

    state_file = find_state(Path(args.target))
    if state_file is None:
        raise SystemExit("未找到 .project-workflow/PROJECT.md。")

    checkpoint = state_file.parent / CHECKPOINT_NAME
    checkpoint.write_text(
        json.dumps(
            {
                "state_file": str(state_file),
                "fingerprint": fingerprint(state_file),
                "started_at": datetime.now(timezone.utc).isoformat(),
            },
            ensure_ascii=False,
        )
        + "\n",
        encoding="utf-8",
    )
    print(f"工作流任务已开始：{state_file}")


if __name__ == "__main__":
    main()
