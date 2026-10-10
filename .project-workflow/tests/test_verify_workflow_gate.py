from __future__ import annotations

import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


GATE_SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "verify_workflow_gate.py"


class VerifyWorkflowGateTest(unittest.TestCase):
    def test_release_gate_rejects_unexecuted_regression_placeholders(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            project = Path(temp_dir)
            state_file = project / ".project-workflow" / "PROJECT.md"
            state_file.parent.mkdir(parents=True)
            state_file.write_text(
                "## 项目\n"
                "- 名称：示例项目\n"
                "- 当前阶段：7. 运行与优化\n"
                "- Flow：发布前检查\n"
                "- Next：完成发布\n"
                "## 当前门禁\n"
                "- 结论：PASS\n"
                "- 发布证据：回滚和监控已准备\n"
                "- 最小验证：定向检查通过\n"
                "- 全量回归：未执行\n",
                encoding="utf-8",
            )

            for placeholder in ("未执行", "未执行；本轮只做定向验证"):
                state_file.write_text(
                    state_file.read_text(encoding="utf-8").replace(
                        "全量回归：未执行", f"全量回归：{placeholder}"
                    ),
                    encoding="utf-8",
                )
                blocked = subprocess.run(
                    [sys.executable, str(GATE_SCRIPT), str(project), "--mode", "release"],
                    capture_output=True,
                    text=True,
                    check=False,
                )
                self.assertNotEqual(blocked.returncode, 0)
                self.assertIn("必须记录全量回归结果", blocked.stderr)

            state_file.write_text(
                state_file.read_text(encoding="utf-8").replace(
                    "全量回归：未执行；本轮只做定向验证", "全量回归：后端与前端回归通过"
                ),
                encoding="utf-8",
            )
            passed = subprocess.run(
                [sys.executable, str(GATE_SCRIPT), str(project), "--mode", "release"],
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(passed.returncode, 0, passed.stderr)

    def test_release_gate_uses_current_gate_section(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            project = Path(temp_dir)
            state_file = project / ".project-workflow" / "PROJECT.md"
            state_file.parent.mkdir(parents=True)
            state_file.write_text(
                "## 项目\n"
                "- 名称：示例项目\n"
                "- 当前阶段：7. 运行与优化\n"
                "- 结论：PASS\n"
                "- Flow：发布前检查\n"
                "- Next：完成发布\n"
                "## 当前门禁\n"
                "- 结论：NEED_VERIFY\n"
                "- 发布证据：回滚和监控已准备\n"
                "- 最小验证：定向检查通过\n"
                "- 全量回归：后端与前端回归通过\n",
                encoding="utf-8",
            )

            blocked = subprocess.run(
                [sys.executable, str(GATE_SCRIPT), str(project), "--mode", "release"],
                capture_output=True,
                text=True,
                check=False,
            )

        self.assertNotEqual(blocked.returncode, 0)
        self.assertIn("当前为 NEED_VERIFY", blocked.stderr)


if __name__ == "__main__":
    unittest.main()
