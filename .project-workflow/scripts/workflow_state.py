"""项目工作流状态文件路径解析。"""

from __future__ import annotations

from pathlib import Path


STATE_RELATIVE_PATH = Path(".project-workflow") / "PROJECT.md"


def read_field(content: str, label: str, section: str | None = None) -> str | None:
    prefix = f"- {label}："
    in_section = section is None
    for line in content.splitlines():
        if section is not None and line.startswith("## "):
            in_section = line == f"## {section}"
            continue
        if not in_section:
            continue
        if line.startswith(prefix):
            return line.removeprefix(prefix).strip()
    return None


def find_state(target: Path) -> Path | None:
    resolved = target.resolve()
    for directory in (resolved, *resolved.parents):
        candidate = directory / STATE_RELATIVE_PATH
        if candidate.is_file():
            return candidate
    return None
