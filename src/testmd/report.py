from __future__ import annotations

import json
from datetime import datetime, timezone
from pathlib import Path

import click

from .models import TestInstance
from .resolver import StatusResult, changed_files

STATUS_ICONS = {
    "resolved": ("✓", "green"),
    "failed": ("✗", "red"),
    "outdated": ("⟳", "yellow"),
    "pending": ("…", "cyan"),
}


def print_status(results: list[StatusResult]) -> None:
    by_title: dict[str, list[StatusResult]] = {}
    for r in results:
        by_title.setdefault(r[0].definition.title, []).append(r)

    for title, group in by_title.items():
        click.echo(click.style(title, bold=True))
        for inst, status, rec in group:
            _print_instance_line(inst, status, rec)
        click.echo()

    _print_summary(results)


def print_get(inst: TestInstance, status: str, record: dict | None) -> None:
    title = _substitute_labels(inst.definition.title, inst.labels)
    click.echo(click.style(f"# {title}", bold=True))

    if inst.labels:
        click.echo(f"Labels: {format_labels(inst.labels)}")
    click.echo(f"Status: {_styled_status(status)}")

    if record:
        if record.get("resolved_at"):
            click.echo(f"Resolved at: {record['resolved_at']}")
        if record.get("failed_at"):
            click.echo(f"Failed at: {record['failed_at']}")
        if record.get("message"):
            click.echo(f"Message: {record['message']}")

    click.echo(f"Patterns: {', '.join(inst.resolved_patterns)}")
    click.echo(f"Files: {len(inst.matched_files)}")

    if status == "outdated":
        diff = changed_files(inst, record)
        if diff:
            click.echo(click.style("Changed:", fg="yellow"))
            for f in diff:
                click.echo(f"  {f}")

    click.echo("---")
    click.echo(_substitute_labels(inst.definition.description, inst.labels))


def write_report_md(results: list[StatusResult], path: Path) -> None:
    lines = ["# Test Report", ""]

    by_title: dict[str, list[StatusResult]] = {}
    for r in results:
        by_title.setdefault(r[0].definition.title, []).append(r)

    for title, group in by_title.items():
        lines.append(f"## {title}")
        lines.append("")
        lines.append("| ID | Labels | Status | Message |")
        lines.append("|----|--------|--------|---------|")
        for inst, status, rec in group:
            icon = STATUS_ICONS[status][0]
            labels = format_labels(inst.labels) or "—"
            msg = (rec or {}).get("message") or ""
            lines.append(f"| `{inst.id}` | {labels} | {icon} {status} | {msg} |")
        lines.append("")

    counts = _count(results)
    lines.append("## Summary")
    lines.append("")
    for s in ("resolved", "failed", "outdated", "pending"):
        lines.append(f"- {s}: {counts.get(s, 0)}")

    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("\n".join(lines) + "\n")
    click.echo(f"Report saved to {path}")


def write_report_json(results: list[StatusResult], path: Path) -> None:
    data = {
        "tests": [
            {
                "id": inst.id,
                "title": inst.definition.title,
                "labels": inst.labels,
                "status": status,
                "message": (rec or {}).get("message"),
                "files": inst.matched_files,
            }
            for inst, status, rec in results
        ],
        "summary": _count(results),
    }
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n")
    click.echo(f"Report saved to {path}")


def format_labels(labels: dict[str, str]) -> str:
    if not labels:
        return ""
    return " ".join(f"{k}={v}" for k, v in sorted(labels.items()))


def _substitute_labels(text: str, labels: dict[str, str]) -> str:
    for var, val in labels.items():
        text = text.replace(f"${var}", val)
    return text


def _print_instance_line(
    inst: TestInstance, status: str, rec: dict | None
) -> None:
    icon, color = STATUS_ICONS[status]
    parts = [
        "  ",
        click.style(icon, fg=color),
        " ",
        click.style(inst.id, dim=True),
    ]

    if inst.labels:
        parts.append("  " + format_labels(inst.labels))

    parts.append("  " + _styled_status(status))

    if status == "failed" and rec and rec.get("message"):
        parts.append(f'  "{rec["message"]}"')
    elif status in ("resolved", "failed") and rec:
        ts = rec.get("resolved_at") or rec.get("failed_at")
        ago = _time_ago(ts)
        if ago:
            parts.append(f"  ({ago})")

    click.echo("".join(parts))


def _styled_status(status: str) -> str:
    _, color = STATUS_ICONS[status]
    return click.style(status, fg=color)


def _time_ago(iso_str: str | None) -> str:
    if not iso_str:
        return ""
    dt = datetime.fromisoformat(iso_str)
    secs = int((datetime.now(timezone.utc) - dt).total_seconds())
    if secs < 0:
        return ""
    if secs < 60:
        return f"{secs}s ago"
    if secs < 3600:
        return f"{secs // 60}m ago"
    if secs < 86400:
        return f"{secs // 3600}h ago"
    return f"{secs // 86400}d ago"


def _print_summary(results: list[StatusResult]) -> None:
    counts = _count(results)
    parts = []
    for s, color in [("resolved", "green"), ("failed", "red"), ("outdated", "yellow"), ("pending", "cyan")]:
        n = counts.get(s, 0)
        parts.append(click.style(f"{n} {s}", fg=color))
    click.echo("Summary: " + ", ".join(parts))


def _count(results: list[StatusResult]) -> dict[str, int]:
    counts: dict[str, int] = {}
    for _, status, _ in results:
        counts[status] = counts.get(status, 0) + 1
    return counts
