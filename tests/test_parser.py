from __future__ import annotations

from pathlib import Path

import pytest

from testmd.parser import parse_testmd


FAKE_FILE = Path("TEST.md")


def _make_test(title: str = "My Test", on_change: str = "src/**/*") -> str:
    return f"# {title}\n\nSome description.\n\n```yaml\non_change: {on_change}\n```\n"


class TestParseSingleTest:
    def test_basic(self):
        text = _make_test()
        fm, tests = parse_testmd(text, FAKE_FILE)
        assert fm == {}
        assert len(tests) == 1
        t = tests[0]
        assert t.title == "My Test"
        assert t.on_change == ["src/**/*"]
        assert t.explicit_id is None
        assert t.matrix is None
        assert "Some description." in t.description

    def test_source_line_no_frontmatter(self):
        text = _make_test()
        _, tests = parse_testmd(text, FAKE_FILE)
        assert tests[0].source_line == 1

    def test_on_change_as_list(self):
        text = "# T\n\n```yaml\non_change:\n  - a.txt\n  - b.txt\n```\n"
        _, tests = parse_testmd(text, FAKE_FILE)
        assert tests[0].on_change == ["a.txt", "b.txt"]

    def test_on_change_as_string(self):
        text = "# T\n\n```yaml\non_change: foo.py\n```\n"
        _, tests = parse_testmd(text, FAKE_FILE)
        assert tests[0].on_change == ["foo.py"]

    def test_explicit_id(self):
        text = "# T\n\n```yaml\nid: my-id\non_change: x\n```\n"
        _, tests = parse_testmd(text, FAKE_FILE)
        assert tests[0].explicit_id == "my-id"

    def test_matrix_parsed(self):
        text = "# T\n\n```yaml\non_change: $svc/**/*\nmatrix:\n  - match: $svc/\n```\n"
        _, tests = parse_testmd(text, FAKE_FILE)
        assert tests[0].matrix == [{"match": "$svc/"}]


class TestParseMultipleTests:
    def test_two_tests(self):
        text = _make_test("First", "a.txt") + "\n" + _make_test("Second", "b.txt")
        _, tests = parse_testmd(text, FAKE_FILE)
        assert len(tests) == 2
        assert tests[0].title == "First"
        assert tests[1].title == "Second"


class TestFrontmatter:
    def test_frontmatter_extracted(self):
        text = "---\ninclude: other.md\nfoo: bar\n---\n" + _make_test()
        fm, tests = parse_testmd(text, FAKE_FILE)
        assert fm["include"] == "other.md"
        assert fm["foo"] == "bar"
        assert len(tests) == 1

    def test_source_line_with_frontmatter(self):
        frontmatter = "---\ninclude: other.md\n---\n"
        text = frontmatter + _make_test()
        _, tests = parse_testmd(text, FAKE_FILE)
        # Frontmatter is 3 lines (including closing ---\n), so offset is 3
        assert tests[0].source_line == 1 + 3


class TestStateBlockStripped:
    def test_state_block_not_in_description(self):
        state_block = '<!-- State\n```testmd\n{"version":1,"tests":{}}\n```\n-->\n'
        text = _make_test() + "\n" + state_block
        _, tests = parse_testmd(text, FAKE_FILE)
        assert len(tests) == 1
        assert "State" not in tests[0].description
        assert "testmd" not in tests[0].description

    def test_state_block_does_not_create_extra_test(self):
        state_block = '<!-- State\n```testmd\n{"version":1,"tests":{}}\n```\n-->\n'
        text = _make_test() + "\n" + state_block
        _, tests = parse_testmd(text, FAKE_FILE)
        assert len(tests) == 1


class TestErrors:
    def test_missing_yaml_block(self):
        text = "# T\n\nNo yaml here.\n"
        with pytest.raises(ValueError, match="missing yaml config block"):
            parse_testmd(text, FAKE_FILE)

    def test_missing_on_change(self):
        text = "# T\n\n```yaml\nid: foo\n```\n"
        with pytest.raises(ValueError, match="missing on_change"):
            parse_testmd(text, FAKE_FILE)


class TestDescriptionContent:
    def test_yaml_block_excluded_from_description(self):
        text = "# T\n\nBefore yaml.\n\n```yaml\non_change: x\n```\n\nAfter yaml.\n"
        _, tests = parse_testmd(text, FAKE_FILE)
        assert "Before yaml." in tests[0].description
        assert "After yaml." in tests[0].description
        assert "on_change" not in tests[0].description
