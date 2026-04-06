from __future__ import annotations

from pathlib import Path

import click

from .parser import parse_testmd
from .report import (
    format_labels,
    print_get,
    print_status,
    write_report_json,
    write_report_md,
)
from .resolver import (
    build_instances,
    compute_statuses,
    fail_test,
    find_instances,
    gc_state,
    resolve_test,
)
from .state import load_state, save_state, strip_state_block


# ---------------------------------------------------------------------------
# Path resolution
# ---------------------------------------------------------------------------


def _find_testmd_upward() -> Path:
    """Search for TEST.md from cwd upward (like git searches for .git)."""
    cwd = Path.cwd().resolve()
    while True:
        candidate = cwd / "TEST.md"
        if candidate.exists():
            return candidate
        parent = cwd.parent
        if parent == cwd:
            raise click.ClickException(
                "No TEST.md found (searched from cwd to filesystem root)"
            )
        cwd = parent


def _resolve_path(testmd: str | None) -> tuple[Path, Path]:
    """Return (test_file, root_dir)."""
    if testmd is None:
        test_file = _find_testmd_upward()
        return test_file, test_file.parent

    p = Path(testmd).resolve()
    if p.is_file():
        return p, p.parent
    if p.is_dir():
        test_file = p / "TEST.md"
        if not test_file.exists():
            raise click.ClickException(f"No TEST.md in {p}")
        return test_file, p
    raise click.ClickException(f"Path not found: {testmd}")


# ---------------------------------------------------------------------------
# Load / save context
# ---------------------------------------------------------------------------


def _load(testmd: str | None):
    test_file, root = _resolve_path(testmd)
    frontmatter, definitions = parse_testmd(
        test_file.read_text(), source_file=test_file
    )

    # Handle includes (one level — nested includes are an error)
    for inc in frontmatter.get("include", []):
        inc_file = (test_file.parent / inc).resolve()
        if not inc_file.exists():
            raise click.ClickException(f"Included file not found: {inc}")
        inc_fm, inc_defs = parse_testmd(inc_file.read_text(), source_file=inc_file)
        if inc_fm.get("include"):
            raise click.ClickException(
                f"Nested includes are not supported: {inc} includes {inc_fm['include']}"
            )
        definitions.extend(inc_defs)

    try:
        instances = build_instances(root, definitions)
    except ValueError as e:
        raise click.ClickException(str(e)) from None

    # Load and merge state from all source files
    source_files = {d.source_file for d in definitions} | {test_file}
    state = {"version": 1, "tests": {}}
    for sf in source_files:
        file_state = load_state(sf)
        state["tests"].update(file_state.get("tests", {}))

    return root, instances, state, source_files


def _save(state: dict, instances, source_files: set[Path]):
    """Save state back to each source file."""
    ids_by_file: dict[Path, set[str]] = {f: set() for f in source_files}
    for inst in instances:
        sf = inst.definition.source_file
        if sf in ids_by_file:
            ids_by_file[sf].add(inst.id)

    for sf, ids in ids_by_file.items():
        file_state = {"version": 1, "tests": {}}
        for tid in ids:
            if tid in state["tests"]:
                file_state["tests"][tid] = state["tests"][tid]
        if file_state["tests"]:
            save_state(sf, file_state)
        else:
            strip_state_block(sf)


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


@click.group()
@click.option("--testmd", default=None, type=click.Path(), help="Path to TEST.md or its directory")
@click.pass_context
def main(ctx, testmd):
    ctx.ensure_object(dict)
    ctx.obj["testmd"] = testmd


@main.command()
@click.option("--report-md", default=None, type=click.Path(), help="Save markdown report")
@click.option("--report-json", default=None, type=click.Path(), help="Save JSON report")
@click.pass_context
def status(ctx, report_md, report_json):
    """Show status of all tests."""
    root, instances, state, _ = _load(ctx.obj["testmd"])
    results = compute_statuses(instances, state)
    print_status(results)
    if report_md:
        write_report_md(results, Path(report_md))
    if report_json:
        write_report_json(results, Path(report_json))


@main.command()
@click.argument("test_id")
@click.pass_context
def resolve(ctx, test_id):
    """Mark test(s) as resolved."""
    _, instances, state, source_files = _load(ctx.obj["testmd"])
    matches = find_instances(instances, test_id)
    if not matches:
        raise click.ClickException(f"No test matching '{test_id}'")
    for inst in matches:
        resolve_test(state, inst)
        label = format_labels(inst.labels)
        suffix = f" ({label})" if label else ""
        click.echo(f"Resolved: {inst.definition.title}{suffix}")
    _save(state, instances, source_files)


@main.command()
@click.argument("test_id")
@click.argument("message")
@click.pass_context
def fail(ctx, test_id, message):
    """Mark test as failed with a message."""
    _, instances, state, source_files = _load(ctx.obj["testmd"])
    matches = find_instances(instances, test_id)
    if not matches:
        raise click.ClickException(f"No test matching '{test_id}'")
    for inst in matches:
        fail_test(state, inst, message)
        label = format_labels(inst.labels)
        suffix = f" ({label})" if label else ""
        click.echo(f"Failed: {inst.definition.title}{suffix}")
        click.echo(f"  Message: {message}")
    _save(state, instances, source_files)


@main.command()
@click.argument("test_id")
@click.pass_context
def get(ctx, test_id):
    """Show test details and description."""
    _, instances, state, _ = _load(ctx.obj["testmd"])
    matches = find_instances(instances, test_id)
    if not matches:
        raise click.ClickException(f"No test matching '{test_id}'")
    for inst in matches:
        rec = state["tests"].get(inst.id)
        if rec is None:
            s = "pending"
        elif rec["content_hash"] != inst.content_hash:
            s = "outdated"
        else:
            s = rec["status"]
        print_get(inst, s, rec)
        if inst != matches[-1]:
            click.echo()


@main.command()
@click.pass_context
def gc(ctx):
    """Remove orphaned test records."""
    _, instances, state, source_files = _load(ctx.obj["testmd"])
    n = gc_state(state, instances)
    _save(state, instances, source_files)
    click.echo(f"Removed {n} orphaned record(s).")


@main.command()
@click.option("--report-md", default=None, type=click.Path(), help="Save markdown report")
@click.option("--report-json", default=None, type=click.Path(), help="Save JSON report")
@click.pass_context
def ci(ctx, report_md, report_json):
    """Check all tests pass (for CI). Exits 1 if any test needs attention."""
    root, instances, state, _ = _load(ctx.obj["testmd"])
    results = compute_statuses(instances, state)

    if report_md:
        write_report_md(results, Path(report_md))
    if report_json:
        write_report_json(results, Path(report_json))

    failing = [(i, s, r) for i, s, r in results if s != "resolved"]
    if not failing:
        click.echo(click.style("OK: all tests resolved", fg="green"))
        return

    click.echo(click.style(f"FAIL: {len(failing)} test(s) require attention", fg="red", bold=True))
    click.echo()
    for inst, s, rec in failing:
        icon, color = {"failed": ("✗", "red"), "outdated": ("⟳", "yellow"), "pending": ("…", "cyan")}[s]
        label = format_labels(inst.labels)
        suffix = f" ({label})" if label else ""
        click.echo(f"  {click.style(icon, fg=color)}  {inst.id}  {inst.definition.title}{suffix}  {click.style(s, fg=color)}")

    raise SystemExit(1)
