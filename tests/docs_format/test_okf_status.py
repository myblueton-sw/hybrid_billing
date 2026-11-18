import re
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
ALLOWED = {"draft", "stable", "deprecated"}


def tracked_markdown():
    listed = subprocess.run(
        ["git", "ls-files", "*.md"], cwd=ROOT, capture_output=True, text=True, check=True
    ).stdout.split()
    return [ROOT / name for name in listed]


def front_matter_status(text):
    if not text.startswith("---\n"):
        return None
    end = text.find("\n---", 4)
    if end < 0:
        return None
    found = re.search(r"^status:\s*(.*?)\s*$", text[:end], re.M)
    return found.group(1) if found else None


class OkfStatusTests(unittest.TestCase):
    def test_front_matter_status_uses_the_okf_values(self):
        wrong = []
        for path in tracked_markdown():
            status = front_matter_status(path.read_text(encoding="utf-8"))
            if status is not None and status not in ALLOWED:
                wrong.append(f"{path.relative_to(ROOT)}: {status}")
        self.assertEqual(wrong, [])

    def test_the_reader_finds_status_only_inside_front_matter(self):
        self.assertEqual(front_matter_status("---\ntype: Plan\nstatus: final\n---\nbody\n"), "final")
        self.assertIsNone(front_matter_status("# Title\n\nstatus: final\n"))
        self.assertIsNone(front_matter_status("---\ntype: Plan\n---\nstatus: final\n"))


if __name__ == "__main__":
    unittest.main()
