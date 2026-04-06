from __future__ import annotations

import hashlib
from pathlib import Path


def hash_file(root: Path, rel_path: str) -> str:
    content = (root / rel_path).read_bytes()
    return hashlib.sha256(rel_path.encode() + b"\0" + content).hexdigest()


def hash_files(root: Path, files: list[str]) -> tuple[str, dict[str, str]]:
    """Return (content_hash, {path: hash}) for a list of files."""
    file_hashes = {f: hash_file(root, f) for f in files}
    combined = "".join(file_hashes[f] for f in files)  # files already sorted
    content_hash = hashlib.sha256(combined.encode()).hexdigest()
    return content_hash, file_hashes


def make_id(title: str, explicit_id: str | None, labels: dict[str, str]) -> str:
    source = explicit_id if explicit_id else title
    first = hashlib.sha256(source.encode()).hexdigest()[:6]

    if labels:
        label_str = ",".join(f"{k}={v}" for k, v in sorted(labels.items()))
    else:
        label_str = ""
    second = hashlib.sha256(label_str.encode()).hexdigest()[:6]

    return f"{first}-{second}"
