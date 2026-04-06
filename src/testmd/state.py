from __future__ import annotations

import json
import re
from pathlib import Path

STATE_VERSION = 1
_STATE_RE = re.compile(
    r"<!-- State\n```testmd\n(.*?)```\n-->\n?", re.DOTALL
)


def load_state(test_file: Path) -> dict:
    """Extract state from the <!-- State ```testmd ... ``` --> block."""
    text = test_file.read_text()
    m = _STATE_RE.search(text)
    if m:
        return json.loads(m.group(1))
    return {"version": STATE_VERSION, "tests": {}}


def save_state(test_file: Path, state: dict) -> None:
    """Write state as formatted JSON wrapped in <!-- State -->."""
    text = test_file.read_text()
    body = json.dumps(state, ensure_ascii=False, indent=2)
    block = f"<!-- State\n```testmd\n{body}\n```\n-->\n"

    m = _STATE_RE.search(text)
    if m:
        text = text[: m.start()] + block
    else:
        text = text.rstrip() + "\n\n" + block

    test_file.write_text(text)


def strip_state_block(test_file: Path) -> None:
    """Remove the state block from a TEST.md file if present."""
    text = test_file.read_text()
    m = _STATE_RE.search(text)
    if m:
        text = text[: m.start()].rstrip() + "\n"
        test_file.write_text(text)
