from __future__ import annotations

import hashlib
from pathlib import Path

from testmd.hashing import hash_file, hash_files, make_id


class TestHashFile:
    def test_consistent(self, tmp_path: Path):
        (tmp_path / "f.txt").write_text("hello")
        h1 = hash_file(tmp_path, "f.txt")
        h2 = hash_file(tmp_path, "f.txt")
        assert h1 == h2

    def test_includes_path_in_hash(self, tmp_path: Path):
        """Two files with same content but different paths produce different hashes."""
        (tmp_path / "a.txt").write_text("same")
        (tmp_path / "b.txt").write_text("same")
        assert hash_file(tmp_path, "a.txt") != hash_file(tmp_path, "b.txt")

    def test_known_value(self, tmp_path: Path):
        (tmp_path / "x.txt").write_text("data")
        expected = hashlib.sha256(b"x.txt\0data").hexdigest()
        assert hash_file(tmp_path, "x.txt") == expected


class TestHashFiles:
    def test_empty_list(self, tmp_path: Path):
        content_hash, file_hashes = hash_files(tmp_path, [])
        assert file_hashes == {}
        # Hash of empty string
        assert content_hash == hashlib.sha256(b"").hexdigest()

    def test_multiple_files(self, tmp_path: Path):
        (tmp_path / "a.txt").write_text("aaa")
        (tmp_path / "b.txt").write_text("bbb")
        content_hash, file_hashes = hash_files(tmp_path, ["a.txt", "b.txt"])
        assert "a.txt" in file_hashes
        assert "b.txt" in file_hashes
        assert len(content_hash) == 64  # sha256 hex

    def test_content_hash_changes_with_file_content(self, tmp_path: Path):
        (tmp_path / "f.txt").write_text("v1")
        h1, _ = hash_files(tmp_path, ["f.txt"])
        (tmp_path / "f.txt").write_text("v2")
        h2, _ = hash_files(tmp_path, ["f.txt"])
        assert h1 != h2


class TestMakeId:
    def test_with_title_only(self):
        tid = make_id("My Test", None, {})
        assert len(tid) == 13  # 6 + '-' + 6
        assert "-" in tid

    def test_explicit_id_overrides_title(self):
        id1 = make_id("Title A", "custom", {})
        id2 = make_id("Title B", "custom", {})
        # Same explicit_id -> same first part
        assert id1.split("-")[0] == id2.split("-")[0]

    def test_labels_affect_second_part(self):
        id1 = make_id("T", None, {})
        id2 = make_id("T", None, {"svc": "web"})
        # Same title -> same first part
        assert id1.split("-")[0] == id2.split("-")[0]
        # Different labels -> different second part
        assert id1.split("-")[1] != id2.split("-")[1]

    def test_different_titles_different_first_part(self):
        id1 = make_id("Alpha", None, {})
        id2 = make_id("Beta", None, {})
        assert id1.split("-")[0] != id2.split("-")[0]

    def test_deterministic(self):
        assert make_id("X", None, {"a": "1"}) == make_id("X", None, {"a": "1"})
