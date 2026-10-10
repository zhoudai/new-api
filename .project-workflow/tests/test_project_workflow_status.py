from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


STATUS_SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "project_workflow_status.py"
GATE_SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "verify_workflow_gate.py"
BEGIN_SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "begin_workflow_task.py"


class ProjectWorkflowStatusTest(unittest.TestCase):
    def test_ocr_review_rounds_are_distinct_in_the_handoff(self) -> None:
        root = Path(__file__).resolve().parents[2]
        project = (root / ".project-workflow/PROJECT.md").read_text(encoding="utf-8")
        report = (root / "docs/reports/2026-10-09-ocr-review.md").read_text(encoding="utf-8")

        self.assertIn("- R8 复审：", project)
        self.assertNotIn("- 修复后最终 OCR 复审：", project)
        self.assertIn("## R8 复审（恢复会话，2026-10-10）", report)
        self.assertIn("## R9 复审结果（2026-10-10）", report)
        self.assertIn("R10-M-02 | 中 | bridge/.dockerignore", report)
        self.assertIn("R10-M-02 定向验证：Bridge 构建上下文忽略规则测试 14/14 通过", project)
        self.assertIn("R10-M-03 | 中 | bridge/Dockerfile.production", report)
        self.assertIn("R10-M-03 定向验证：Bridge 部署配置测试 15/15 通过", project)
        self.assertIn("R11-M-03 | 中 | web/src/features/system-settings/models/model-ratio-visual-editor.tsx", report)
        self.assertIn("R11-M-03 定向验证：管理员模型比率视觉编辑器测试 5/5", project)
        self.assertIn("R11-M-04 | 中 | deploy/production/生产部署执行清单.md:12", report)
        self.assertIn("R11-M-04 定向验证：生产清单回归 18/18 通过", project)
        r8_r9_history = next(
            line for line in project.splitlines() if "完成 R8/R9 中优先级问题修复" in line
        )
        self.assertIn("均已修复并逐项验证", r8_r9_history)
        self.assertNotIn("处理中", r8_r9_history)
        self.assertIn("R10 最终复审发现补充问题", project)

    def test_project_ledger_does_not_claim_unavailable_systemd_validation(self) -> None:
        root = Path(__file__).resolve().parents[2]
        project = (root / ".project-workflow/PROJECT.md").read_text(encoding="utf-8")
        validation = next(
            line for line in project.splitlines() if line.startswith("- 本次配置命名验证：")
        )

        self.assertIn("未执行 `systemd-analyze verify`", validation)
        self.assertNotIn("通过 `systemd-analyze verify`", validation)

    def test_status_and_gate_find_parent_project_state(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            project = Path(temp_dir)
            nested = project / "src" / "feature"
            nested.mkdir(parents=True)
            state_file = project / ".project-workflow" / "PROJECT.md"
            state_file.parent.mkdir(parents=True)
            state_file.write_text(
                "## 项目\n"
                "- 名称：示例项目\n"
                "- 当前阶段：7. 运行与优化\n"
                "- Flow：本地验证\n"
                "- Next：继续核验\n"
                "## 当前门禁\n"
                "- 结论：NEED_VERIFY\n",
                encoding="utf-8",
            )

            status_result = subprocess.run(
                [sys.executable, str(STATUS_SCRIPT), str(nested)],
                capture_output=True,
                text=True,
                check=True,
            )
            self.assertEqual(Path(json.loads(status_result.stdout)["path"]), state_file.resolve())
            self.assertEqual(json.loads(status_result.stdout)["gate"], "NEED_VERIFY")

            gate_result = subprocess.run(
                [sys.executable, str(GATE_SCRIPT), str(nested), "--mode", "commit"],
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(gate_result.returncode, 0, gate_result.stderr)

            begin_result = subprocess.run(
                [sys.executable, str(BEGIN_SCRIPT), str(nested)],
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(begin_result.returncode, 0, begin_result.stderr)
            self.assertTrue((state_file.parent / ".task-checkpoint.json").is_file())

    def test_initializes_target_and_preserves_existing_state(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            target = Path(temp_dir)
            command = [
                sys.executable,
                str(STATUS_SCRIPT),
                str(target),
                "--initialize",
                "--project",
                "示例项目",
            ]
            result = subprocess.run(command, capture_output=True, text=True, check=True)
            state_path = Path(json.loads(result.stdout)["path"])
            content = state_path.read_text(encoding="utf-8")
            self.assertIn("- 名称：示例项目", content)
            self.assertIn("## 当前门禁\n\n- 结论：NEED_CONFIRM", content)
            self.assertIn("- Next：", content)
            self.assertNotIn("<项目名称>", content)

            state_path.write_text(content + "\n保留现有状态\n", encoding="utf-8")
            subprocess.run(command, capture_output=True, text=True, check=True)
            self.assertIn("保留现有状态", state_path.read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
