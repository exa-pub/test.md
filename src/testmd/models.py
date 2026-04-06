from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path


@dataclass
class TestDefinition:
    """One test from TEST.md (before label expansion)."""

    title: str
    explicit_id: str | None
    on_change: list[str]
    matrix: list[dict] | None
    description: str
    source_file: Path
    source_line: int


@dataclass
class TestInstance:
    """Concrete test instance (after label expansion + file hashing)."""

    id: str
    definition: TestDefinition
    labels: dict[str, str]
    resolved_patterns: list[str]
    matched_files: list[str]
    content_hash: str
    file_hashes: dict[str, str]
