from __future__ import annotations

import re
from pathlib import Path

import yaml

from .models import TestDefinition

_STATE_BLOCK_RE = re.compile(r"<!-- State\n```testmd\n.*?```\n-->\n?", re.DOTALL)


def parse_testmd(
    text: str, source_file: Path
) -> tuple[dict, list[TestDefinition]]:
    """Parse TEST.md into (frontmatter, tests).

    Strips the frontmatter and ```testmd state block before parsing tests.
    """
    # 1. Extract frontmatter
    frontmatter: dict = {}
    line_offset = 0
    if text.startswith("---\n"):
        end = text.find("\n---\n", 4)
        if end != -1:
            frontmatter = yaml.safe_load(text[4:end]) or {}
            line_offset = text[: end + 5].count("\n")
            text = text[end + 5 :]

    # 2. Strip state block
    text = _STATE_BLOCK_RE.sub("", text)

    # 3. Parse tests
    tests: list[TestDefinition] = []
    lines = text.split("\n")
    i = 0

    while i < len(lines):
        if not lines[i].startswith("# "):
            i += 1
            continue

        title = lines[i][2:].strip()
        source_line = i + 1 + line_offset
        i += 1

        body_lines: list[str] = []
        while i < len(lines) and not lines[i].startswith("# "):
            body_lines.append(lines[i])
            i += 1

        body = "\n".join(body_lines)

        m = re.search(r"```ya?ml\n(.*?)```", body, re.DOTALL)
        if not m:
            raise ValueError(
                f"Test '{title}' (line {source_line}): missing yaml config block"
            )

        config = yaml.safe_load(m.group(1)) or {}
        on_change = config.get("on_change")
        if not on_change:
            raise ValueError(
                f"Test '{title}' (line {source_line}): missing on_change"
            )
        if isinstance(on_change, str):
            on_change = [on_change]

        description = (body[: m.start()] + body[m.end() :]).strip()

        tests.append(
            TestDefinition(
                title=title,
                explicit_id=config.get("id"),
                on_change=on_change,
                matrix=config.get("matrix"),
                description=description,
                source_file=source_file,
                source_line=source_line,
            )
        )

    return frontmatter, tests
