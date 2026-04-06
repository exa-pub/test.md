from __future__ import annotations

from pathlib import Path

import pytest

from testmd.hashing import hash_files, make_id
from testmd.models import TestDefinition, TestInstance
from testmd.resolver import (
    build_instances,
    changed_files,
    compute_statuses,
    fail_test,
    find_instances,
    gc_state,
    resolve_test,
    _validate_matrix_vars,
)


def _defn(
    title: str = "Test",
    on_change: list[str] | None = None,
    matrix: list[dict] | None = None,
    explicit_id: str | None = None,
) -> TestDefinition:
    return TestDefinition(
        title=title,
        explicit_id=explicit_id,
        on_change=on_change or ["src/**/*"],
        matrix=matrix,
        description="desc",
        source_file=Path("TEST.md"),
        source_line=1,
    )


def _instance(
    tid: str = "aaa-bbb",
    title: str = "Test",
    labels: dict | None = None,
    matched_files: list[str] | None = None,
    content_hash: str = "hash123",
    file_hashes: dict | None = None,
) -> TestInstance:
    return TestInstance(
        id=tid,
        definition=_defn(title=title),
        labels=labels or {},
        resolved_patterns=["src/**/*"],
        matched_files=matched_files or [],
        content_hash=content_hash,
        file_hashes=file_hashes or {},
    )


def _empty_state() -> dict:
    return {"version": 1, "tests": {}}


class TestBuildInstances:
    def test_without_matrix(self, tmp_path: Path):
        (tmp_path / "src").mkdir()
        (tmp_path / "src" / "main.py").write_text("print('hi')")
        defn = _defn(on_change=["src/*.py"])
        instances = build_instances(tmp_path, [defn])
        assert len(instances) == 1
        assert "src/main.py" in instances[0].matched_files
        assert instances[0].labels == {}

    def test_with_matrix_const(self, tmp_path: Path):
        (tmp_path / "main.py").write_text("x")
        defn = _defn(
            on_change=["main.py"],
            matrix=[{"const": {"env": ["dev", "prod"]}}],
        )
        instances = build_instances(tmp_path, [defn])
        assert len(instances) == 2
        labels_list = [i.labels for i in instances]
        assert {"env": "dev"} in labels_list
        assert {"env": "prod"} in labels_list

    def test_with_matrix_match(self, tmp_path: Path):
        (tmp_path / "svcA" / "src").mkdir(parents=True)
        (tmp_path / "svcA" / "src" / "a.py").write_text("a")
        (tmp_path / "svcB" / "src").mkdir(parents=True)
        (tmp_path / "svcB" / "src" / "b.py").write_text("b")
        defn = _defn(
            on_change=["$svc/src/*.py"],
            matrix=[{"match": "$svc/src"}],
        )
        instances = build_instances(tmp_path, [defn])
        assert len(instances) == 2

    def test_auto_discovery_without_matrix(self, tmp_path: Path):
        (tmp_path / "alpha" / "code").mkdir(parents=True)
        (tmp_path / "alpha" / "code" / "x.py").write_text("x")
        (tmp_path / "beta" / "code").mkdir(parents=True)
        (tmp_path / "beta" / "code" / "y.py").write_text("y")
        defn = _defn(on_change=["$svc/code/*.py"])
        instances = build_instances(tmp_path, [defn])
        assert len(instances) == 2
        svcs = {i.labels["svc"] for i in instances}
        assert svcs == {"alpha", "beta"}


class TestComputeStatuses:
    def test_pending(self):
        inst = _instance()
        state = _empty_state()
        results = compute_statuses([inst], state)
        assert results[0][1] == "pending"
        assert results[0][2] is None

    def test_resolved(self):
        inst = _instance(content_hash="abc")
        state = _empty_state()
        state["tests"]["aaa-bbb"] = {"content_hash": "abc", "status": "resolved"}
        results = compute_statuses([inst], state)
        assert results[0][1] == "resolved"

    def test_outdated(self):
        inst = _instance(content_hash="new_hash")
        state = _empty_state()
        state["tests"]["aaa-bbb"] = {"content_hash": "old_hash", "status": "resolved"}
        results = compute_statuses([inst], state)
        assert results[0][1] == "outdated"

    def test_failed(self):
        inst = _instance(content_hash="h")
        state = _empty_state()
        state["tests"]["aaa-bbb"] = {"content_hash": "h", "status": "failed"}
        results = compute_statuses([inst], state)
        assert results[0][1] == "failed"


class TestResolveTest:
    def test_updates_state(self):
        inst = _instance(
            content_hash="ch",
            file_hashes={"f.txt": "fh"},
            matched_files=["f.txt"],
        )
        state = _empty_state()
        resolve_test(state, inst)
        rec = state["tests"]["aaa-bbb"]
        assert rec["status"] == "resolved"
        assert rec["content_hash"] == "ch"
        assert rec["files"] == {"f.txt": "fh"}
        assert rec["resolved_at"] is not None
        assert rec["failed_at"] is None
        assert rec["message"] is None


class TestFailTest:
    def test_updates_state_with_message(self):
        inst = _instance()
        state = _empty_state()
        fail_test(state, inst, "broken")
        rec = state["tests"]["aaa-bbb"]
        assert rec["status"] == "failed"
        assert rec["message"] == "broken"
        assert rec["failed_at"] is not None


class TestGcState:
    def test_removes_orphans(self):
        inst = _instance(tid="aaa-bbb")
        state = _empty_state()
        state["tests"]["aaa-bbb"] = {"status": "resolved"}
        state["tests"]["old-one"] = {"status": "resolved"}
        removed = gc_state(state, [inst])
        assert removed == 1
        assert "old-one" not in state["tests"]
        assert "aaa-bbb" in state["tests"]

    def test_nothing_to_gc(self):
        inst = _instance(tid="aaa-bbb")
        state = _empty_state()
        state["tests"]["aaa-bbb"] = {"status": "resolved"}
        removed = gc_state(state, [inst])
        assert removed == 0


class TestFindInstances:
    def _instances(self):
        return [
            _instance(tid="abc123-def456"),
            _instance(tid="abc123-aaa111"),
            _instance(tid="xyz789-bbb222"),
        ]

    def test_exact_match(self):
        insts = self._instances()
        result = find_instances(insts, "abc123-def456")
        assert len(result) == 1
        assert result[0].id == "abc123-def456"

    def test_first_part_match(self):
        insts = self._instances()
        result = find_instances(insts, "abc123")
        assert len(result) == 2

    def test_prefix_match(self):
        insts = self._instances()
        result = find_instances(insts, "xyz")
        assert len(result) == 1
        assert result[0].id == "xyz789-bbb222"

    def test_no_match(self):
        insts = self._instances()
        result = find_instances(insts, "zzzzz")
        assert result == []


class TestChangedFiles:
    def test_no_record(self):
        inst = _instance(matched_files=["a.txt", "b.txt"])
        result = changed_files(inst, None)
        assert result == ["a.txt", "b.txt"]

    def test_modified_file(self):
        inst = _instance(
            file_hashes={"a.txt": "new_hash", "b.txt": "same"},
        )
        record = {"files": {"a.txt": "old_hash", "b.txt": "same"}}
        result = changed_files(inst, record)
        assert result == ["a.txt"]

    def test_added_file(self):
        inst = _instance(
            file_hashes={"a.txt": "h1", "new.txt": "h2"},
        )
        record = {"files": {"a.txt": "h1"}}
        result = changed_files(inst, record)
        assert result == ["new.txt"]

    def test_deleted_file(self):
        inst = _instance(
            file_hashes={"a.txt": "h1"},
        )
        record = {"files": {"a.txt": "h1", "gone.txt": "h2"}}
        result = changed_files(inst, record)
        assert result == ["gone.txt"]

    def test_no_changes(self):
        inst = _instance(file_hashes={"a.txt": "h"})
        record = {"files": {"a.txt": "h"}}
        result = changed_files(inst, record)
        assert result == []


class TestValidateMatrixVars:
    def test_undefined_var_raises(self):
        defn = _defn(
            on_change=["$svc/src/$module/*.py"],
            matrix=[{"match": "$svc/src"}],
        )
        with pytest.raises(ValueError, match="not defined in matrix"):
            _validate_matrix_vars(defn)

    def test_unused_var_warns(self, capsys):
        defn = _defn(
            on_change=["src/*.py"],
            matrix=[{"const": {"env": ["dev"]}}],
        )
        _validate_matrix_vars(defn)
        captured = capsys.readouterr()
        assert "not used in on_change" in captured.err

    def test_valid_matrix_no_error(self):
        defn = _defn(
            on_change=["$svc/src/*.py"],
            matrix=[{"match": "$svc/src"}],
        )
        _validate_matrix_vars(defn)  # should not raise
