#!/bin/sh
# 为已纳入工作流的 Git 项目追加提交门禁，不覆盖既有检查。

set -eu

target=${1:-.}
root=$(git -C "$target" rev-parse --show-toplevel 2>/dev/null) || {
  echo "无法安装工作流提交门禁：目标不是 Git 仓库。" >&2
  exit 1
}
if hooks_path=$(git -C "$root" config --path core.hooksPath 2>/dev/null) && [ -n "$hooks_path" ]; then
	case "$hooks_path" in
		/*) ;;
		*) hooks_path="$root/$hooks_path" ;;
	esac
else
	hooks_path=$(git -C "$root" rev-parse --git-path hooks)
	case "$hooks_path" in
		/*) ;;
		*) hooks_path="$root/$hooks_path" ;;
	esac
fi
mkdir -p "$hooks_path"
hook="$hooks_path/pre-commit"
marker="# software-delivery-workflow gate"
gate_command='root=$(git rev-parse --show-toplevel); python3 "$root/.project-workflow/scripts/verify_workflow_gate.py" "$root" --mode commit'

if [ -f "$hook" ] && grep -Fq "$marker" "$hook"; then
  if ! grep -Fxq "$gate_command" "$hook"; then
    echo "工作流提交门禁标记存在但命令缺失：请人工修复 ${hook}。" >&2
    exit 1
  fi
  chmod +x "$hook"
  echo "工作流提交门禁已启用：$hook"
  exit 0
fi

# 不完整解析 shell 控制流时，发现 exit 或 exec 标记就交由人工合并。
if [ -f "$hook" ] && grep -Eq '(^|[^[:alnum:]_])(exit|exec)([^[:alnum:]_]|$)' "$hook"; then
  echo "无法安全追加门禁：现有 pre-commit 含 exit 或 exec 标记，请人工合并后再执行。" >&2
  exit 1
fi

if [ ! -f "$hook" ]; then
  printf '%s\n' '#!/bin/sh' > "$hook"
fi

{
  printf '\n%s\n' "$marker"
  printf '%s\n' "$gate_command"
} >> "$hook"
chmod +x "$hook"

echo "工作流提交门禁已启用：$hook"
