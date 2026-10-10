from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


CHECKER = Path(__file__).resolve().parents[1] / "scripts" / "verify_workflow_completion.py"
STATE = """## 当前门禁

- 结论：进行中

- 名称：临时项目
- 当前阶段：6. 交付
- db-mcp 状态：未连接
- Flow：继续修复
- Next：验证改动
- 延后事项：无
- 待办清单：无

## 历史记录
"""


class VerifyWorkflowCompletionTest(unittest.TestCase):
    def run_checker(self, checkpoint: str) -> subprocess.CompletedProcess[str]:
        with tempfile.TemporaryDirectory() as temp_dir:
            target = Path(temp_dir)
            state_dir = target / ".project-workflow"
            state_dir.mkdir()
            (state_dir / "PROJECT.md").write_text(STATE, encoding="utf-8")
            snapshot = state_dir / ".task-checkpoint.json"
            snapshot.write_text(checkpoint, encoding="utf-8")
            result = subprocess.run(
                [sys.executable, str(CHECKER), str(target), "--require-change"],
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertTrue(snapshot.exists())
            return result

    def test_invalid_json_shapes_report_snapshot_error_without_traceback(self) -> None:
        for checkpoint in ("[]", "null", '{"fingerprint": null}'):
            with self.subTest(checkpoint=checkpoint):
                result = self.run_checker(checkpoint)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("本轮任务快照无效", result.stderr)
                self.assertNotIn("Traceback", result.stderr)

    def test_success_keeps_checkpoint_file(self) -> None:
        result = self.run_checker(json.dumps({"fingerprint": "old"}))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("工作流结束校验通过", result.stdout)


if __name__ == "__main__":
    unittest.main()
