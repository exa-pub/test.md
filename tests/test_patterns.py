from __future__ import annotations

from pathlib import Path

import pytest

from testmd.patterns import (
    enumerate_labels,
    expand_matrix,
    find_label_vars,
    resolve_files,
)


class TestFindLabelVars:
    def test_single_var(self):
        assert find_label_vars("$svc/src/**") == ["svc"]

    def test_multiple_vars(self):
        assert find_label_vars("$org/$repo/file") == ["org", "repo"]

    def test_no_vars(self):
        assert find_label_vars("src/**/*") == []

    def test_underscore_var(self):
        assert find_label_vars("$my_var/x") == ["my_var"]


class TestEnumerateLabels:
    def test_no_vars_returns_empty_dict(self):
        result = enumerate_labels(Path("/tmp"), ["src/**/*"])
        assert result == [{}]

    def test_discovers_dirs(self, tmp_path: Path):
        (tmp_path / "alpha" / "src").mkdir(parents=True)
        (tmp_path / "beta" / "src").mkdir(parents=True)
        (tmp_path / "alpha" / "src" / "a.py").write_text("a")
        (tmp_path / "beta" / "src" / "b.py").write_text("b")

        result = enumerate_labels(tmp_path, ["$svc/src"])
        assert {"svc": "alpha"} in result
        assert {"svc": "beta"} in result
        assert len(result) == 2

    def test_hidden_dirs_excluded(self, tmp_path: Path):
        (tmp_path / ".hidden" / "src").mkdir(parents=True)
        (tmp_path / "visible" / "src").mkdir(parents=True)
        result = enumerate_labels(tmp_path, ["$svc/src"])
        assert len(result) == 1
        assert result[0]["svc"] == "visible"

    def test_deduplicates(self, tmp_path: Path):
        (tmp_path / "foo" / "a").mkdir(parents=True)
        (tmp_path / "foo" / "b").mkdir(parents=True)
        # Two patterns that both discover svc=foo
        result = enumerate_labels(tmp_path, ["$svc/a", "$svc/b"])
        assert result.count({"svc": "foo"}) == 1


class TestExpandMatrix:
    def test_const_only(self):
        matrix = [{"const": {"env": ["dev", "prod"], "region": ["us", "eu"]}}]
        result = expand_matrix(Path("/tmp"), matrix)
        assert len(result) == 4
        assert {"env": "dev", "region": "us"} in result
        assert {"env": "prod", "region": "eu"} in result

    def test_match_only(self, tmp_path: Path):
        (tmp_path / "svcA").mkdir()
        (tmp_path / "svcB").mkdir()
        matrix = [{"match": "$svc/"}]
        result = expand_matrix(tmp_path, matrix)
        assert {"svc": "svcA"} in result
        assert {"svc": "svcB"} in result

    def test_match_plus_const(self, tmp_path: Path):
        (tmp_path / "web").mkdir()
        matrix = [{"match": "$svc/", "const": {"env": ["dev", "prod"]}}]
        result = expand_matrix(tmp_path, matrix)
        assert len(result) == 2
        assert {"svc": "web", "env": "dev"} in result
        assert {"svc": "web", "env": "prod"} in result

    def test_union_of_entries(self):
        matrix = [
            {"const": {"x": ["1"]}},
            {"const": {"x": ["2"]}},
        ]
        result = expand_matrix(Path("/tmp"), matrix)
        assert {"x": "1"} in result
        assert {"x": "2"} in result
        assert len(result) == 2

    def test_empty_matrix_returns_empty_dict(self):
        # No entries that produce combos -> [{}]
        result = expand_matrix(Path("/tmp"), [])
        assert result == [{}]

    def test_const_scalar_value(self):
        """A const value that is not a list should be treated as [value]."""
        matrix = [{"const": {"env": "prod"}}]
        result = expand_matrix(Path("/tmp"), matrix)
        assert result == [{"env": "prod"}]


class TestResolveFiles:
    def test_substitutes_labels_and_globs(self, tmp_path: Path):
        (tmp_path / "web" / "src").mkdir(parents=True)
        (tmp_path / "web" / "src" / "main.py").write_text("x")
        result = resolve_files(tmp_path, "$svc/src/*.py", {"svc": "web"})
        assert result == ["web/src/main.py"]

    def test_double_star_normalization(self, tmp_path: Path):
        """Patterns ending with /** should be normalized to /**/* to match files."""
        (tmp_path / "lib").mkdir()
        (tmp_path / "lib" / "a.py").write_text("a")
        result = resolve_files(tmp_path, "lib/**", {})
        assert "lib/a.py" in result

    def test_dot_slash_stripped(self, tmp_path: Path):
        (tmp_path / "foo.txt").write_text("hello")
        result = resolve_files(tmp_path, "./foo.txt", {})
        assert result == ["foo.txt"]

    def test_no_matches(self, tmp_path: Path):
        result = resolve_files(tmp_path, "nonexistent/*.py", {})
        assert result == []
