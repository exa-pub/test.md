from __future__ import annotations

import json
from pathlib import Path

from testmd.state import STATE_VERSION, load_state, save_state, strip_state_block


class TestLoadState:
    def test_no_state_block(self, tmp_path: Path):
        f = tmp_path / "TEST.md"
        f.write_text("# Test\n\nSome content.\n")
        state = load_state(f)
        assert state == {"version": STATE_VERSION, "tests": {}}

    def test_with_state_block(self, tmp_path: Path):
        f = tmp_path / "TEST.md"
        data = {"version": 1, "tests": {"abc-def": {"status": "resolved"}}}
        block = f'<!-- State\n```testmd\n{json.dumps(data)}\n```\n-->\n'
        f.write_text("# Test\n\n" + block)
        state = load_state(f)
        assert state == data
        assert state["tests"]["abc-def"]["status"] == "resolved"


class TestSaveState:
    def test_appends_to_file_without_block(self, tmp_path: Path):
        f = tmp_path / "TEST.md"
        f.write_text("# Test\n\nContent here.")
        state = {"version": 1, "tests": {}}
        save_state(f, state)
        text = f.read_text()
        assert "<!-- State" in text
        assert "```testmd" in text
        assert "```\n-->" in text

    def test_replaces_existing_block(self, tmp_path: Path):
        f = tmp_path / "TEST.md"
        old_state = {"version": 1, "tests": {"old": {}}}
        block = f'<!-- State\n```testmd\n{json.dumps(old_state)}\n```\n-->\n'
        f.write_text("# Test\n\n" + block)

        new_state = {"version": 1, "tests": {"new": {}}}
        save_state(f, new_state)
        text = f.read_text()
        assert '"new"' in text
        assert '"old"' not in text
        # Only one state block
        assert text.count("<!-- State") == 1

    def test_json_indent(self, tmp_path: Path):
        f = tmp_path / "TEST.md"
        f.write_text("# Test\n")
        save_state(f, {"version": 1, "tests": {}})
        text = f.read_text()
        # indent=2 means the "version" key is indented
        assert '  "version": 1' in text

    def test_block_format(self, tmp_path: Path):
        f = tmp_path / "TEST.md"
        f.write_text("# Test\n")
        save_state(f, {"version": 1, "tests": {}})
        text = f.read_text()
        assert "<!-- State\n```testmd\n" in text
        assert "\n```\n-->\n" in text


class TestStripStateBlock:
    def test_removes_block(self, tmp_path: Path):
        f = tmp_path / "TEST.md"
        block = '<!-- State\n```testmd\n{"version":1}\n```\n-->\n'
        f.write_text("# Test\n\n" + block)
        strip_state_block(f)
        text = f.read_text()
        assert "<!-- State" not in text
        assert "# Test" in text

    def test_no_block_is_noop(self, tmp_path: Path):
        f = tmp_path / "TEST.md"
        original = "# Test\n\nNo state here.\n"
        f.write_text(original)
        strip_state_block(f)
        # Content preserved (trailing whitespace may be normalized)
        text = f.read_text()
        assert "# Test" in text
        assert "No state here." in text
