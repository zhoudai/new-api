from __future__ import annotations

import stat
import subprocess
import tempfile
import unittest
from pathlib import Path


INSTALLER = Path(__file__).resolve().parents[1] / "scripts" / "install_git_gate.sh"
MARKER = "# software-delivery-workflow gate"
GATE_COMMAND = 'root=$(git rev-parse --show-toplevel); python3 "$root/.project-workflow/scripts/verify_workflow_gate.py" "$root" --mode commit'


class InstallGitGateTest(unittest.TestCase):
    def test_makes_existing_hook_executable(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            project = Path(temp_dir)
            subprocess.run(["git", "init", "--quiet", str(project)], check=True)
            hook = project / ".git" / "hooks" / "pre-commit"
            hook.write_text("#!/bin/sh\ntrue\n", encoding="utf-8")
            hook.chmod(0o644)

            result = subprocess.run(
                ["sh", str(INSTALLER), str(project)],
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue(hook.stat().st_mode & stat.S_IXUSR)
            self.assertIn(MARKER, hook.read_text(encoding="utf-8"))

    def test_repairs_permissions_when_gate_already_exists(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            project = Path(temp_dir)
            subprocess.run(["git", "init", "--quiet", str(project)], check=True)
            hook = project / ".git" / "hooks" / "pre-commit"
            hook.write_text(f"#!/bin/sh\n{MARKER}\n{GATE_COMMAND}\n", encoding="utf-8")
            hook.chmod(0o644)

            result = subprocess.run(
                ["sh", str(INSTALLER), str(project)],
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue(hook.stat().st_mode & stat.S_IXUSR)

    def test_rejects_marker_without_gate_command(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            project = Path(temp_dir)
            subprocess.run(["git", "init", "--quiet", str(project)], check=True)
            hook = project / ".git" / "hooks" / "pre-commit"
            original = f"#!/bin/sh\n{MARKER}\ntrue\n"
            hook.write_text(original, encoding="utf-8")

            result = subprocess.run(
                ["sh", str(INSTALLER), str(project)],
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("标记存在但命令缺失", result.stderr)
            self.assertEqual(hook.read_text(encoding="utf-8"), original)

    def test_rejects_conditional_inline_exit(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            project = Path(temp_dir)
            subprocess.run(["git", "init", "--quiet", str(project)], check=True)
            hook = project / ".git" / "hooks" / "pre-commit"
            hook.write_text('#!/bin/sh\nif [ -f stop ]; then exit 1; fi\n', encoding="utf-8")

            result = subprocess.run(
                ["sh", str(INSTALLER), str(project)],
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("无法安全追加门禁", result.stderr)
            self.assertNotIn(MARKER, hook.read_text(encoding="utf-8"))

    def test_rejects_exec_command(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            project = Path(temp_dir)
            subprocess.run(["git", "init", "--quiet", str(project)], check=True)
            hook = project / ".git" / "hooks" / "pre-commit"
            hook.write_text("#!/bin/sh\nexec ./existing-check\n", encoding="utf-8")

            result = subprocess.run(
                ["sh", str(INSTALLER), str(project)],
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("exec 标记", result.stderr)
            self.assertNotIn(MARKER, hook.read_text(encoding="utf-8"))

    def test_uses_configured_hook_path_and_resolves_repository_at_runtime(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_root = Path(temp_dir)
            project = temp_root / "original"
            project.mkdir()
            subprocess.run(["git", "init", "--quiet", str(project)], check=True)
            subprocess.run(["git", "-C", str(project), "config", "core.hooksPath", ".hooks"], check=True)
            gate_script = project / ".project-workflow" / "scripts" / "verify_workflow_gate.py"
            gate_script.parent.mkdir(parents=True)
            gate_script.write_text(
                "import pathlib, sys\n"
                "raise SystemExit(0 if pathlib.Path(sys.argv[1]) == pathlib.Path(__file__).parents[2] else 1)\n",
                encoding="utf-8",
            )

            result = subprocess.run(
                ["sh", str(INSTALLER), str(project)],
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            hook = project / ".hooks" / "pre-commit"
            self.assertTrue(hook.exists())

            moved_project = temp_root / "moved"
            project.rename(moved_project)
            run_result = subprocess.run(
                ["sh", str(moved_project / ".hooks" / "pre-commit")],
                cwd=moved_project,
                capture_output=True,
                text=True,
                check=False,
            )

            self.assertEqual(run_result.returncode, 0, run_result.stderr)


if __name__ == "__main__":
    unittest.main()
