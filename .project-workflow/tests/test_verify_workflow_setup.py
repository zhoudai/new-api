from __future__ import annotations

import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


CHECKER = Path(__file__).resolve().parents[1] / "scripts" / "verify_workflow_setup.py"
REQUIRED_SCRIPTS = (
    "project_workflow_status.py",
    "begin_workflow_task.py",
    "verify_workflow_completion.py",
    "verify_workflow_gate.py",
)
REQUIRED_AGENT_MARKERS = (
    ".project-workflow/PROJECT.md",
    "project_workflow_status.py",
    "begin_workflow_task.py",
    "verify_workflow_completion.py",
    "verify_workflow_gate.py",
)


class VerifyWorkflowSetupTest(unittest.TestCase):
    def test_checks_scripts_in_target_project(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            target = Path(temp_dir)
            scripts_dir = target / ".project-workflow" / "scripts"
            scripts_dir.mkdir(parents=True)
            (target / ".project-workflow" / "PROJECT.md").write_text("state", encoding="utf-8")
            (target / "AGENTS.md").write_text("\n".join(REQUIRED_AGENT_MARKERS), encoding="utf-8")
            for name in REQUIRED_SCRIPTS:
                script_body = "raise SystemExit('target script failure')\n" if name == REQUIRED_SCRIPTS[0] else "pass\n"
                (scripts_dir / name).write_text(script_body, encoding="utf-8")

            result = subprocess.run(
                [sys.executable, str(CHECKER), str(target)],
                capture_output=True,
                text=True,
                check=False,
            )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("脚本不可运行：project_workflow_status.py", result.stderr)

    def test_requires_release_gate_script(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            target = Path(temp_dir)
            scripts_dir = target / ".project-workflow" / "scripts"
            scripts_dir.mkdir(parents=True)
            (target / ".project-workflow" / "PROJECT.md").write_text("state", encoding="utf-8")
            (target / "AGENTS.md").write_text("\n".join(REQUIRED_AGENT_MARKERS), encoding="utf-8")
            for name in REQUIRED_SCRIPTS:
                if name != "verify_workflow_gate.py":
                    (scripts_dir / name).write_text("pass\n", encoding="utf-8")

            result = subprocess.run(
                [sys.executable, str(CHECKER), str(target)],
                capture_output=True,
                text=True,
                check=False,
            )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("缺少工作流脚本：", result.stderr)
        self.assertIn("verify_workflow_gate.py", result.stderr)


if __name__ == "__main__":
    unittest.main()
