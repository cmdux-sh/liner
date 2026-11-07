from __future__ import annotations

import shutil
import sqlite3
import sys
from dataclasses import replace
from pathlib import Path
from typing import Any

import httpx
import typer
from rich.console import Console

from liner import __version__
from liner.agent_adapters import (
    ROUTER_SKILL_NAME,
    SKILL_NAME,
    SUPPORTED_ENVIRONMENTS,
    AdapterError,
    apply_agent_adapter_plan,
    inspect_agent_adapter,
    plan_agent_adapter,
)
from liner.cache import CacheSchemaMismatchError, SourceCache
from liner.cli_progress import RichProgressReporter
from liner.compile import IncompleteCompileError, MissingSynthesisError, compile_project
from liner.config import Config, load_config
from liner.handlers.base import HandlerHardFailure, SourceHandler
from liner.handlers.fetch_guard import get_bounded, guarded_client
from liner.handlers.repository import RepositoryHandler
from liner.handlers.web import WebHandler
from liner.handlers.youtube import YouTubeHandler, is_video_url
from liner.json_events import (
    JsonEventReporter,
    compile_result_to_payload,
    emit_result,
)
from liner.maintenance import (
    MAINTENANCE_REQUEST_CONTRACT,
    PROJECT_CHANGE_SET_VERSION,
    FailureReport,
    ProjectApplyError,
    ProjectChangeSet,
    ProjectInspectionError,
    ProjectSnapshot,
    apply_change_set,
    inspect_project,
    maintenance_guidance,
    plan_operating_layer_create,
    plan_operating_layer_review,
    plan_pointer_adapter,
    plan_project_guidance_upgrade,
    plan_project_move,
    plan_project_rename,
    plan_source_add,
    plan_source_add_batch,
    plan_source_purge,
    plan_source_remove,
    plan_source_replace,
    plan_source_update,
    plan_synthesis_review,
)
from liner.manifest import build_manifest, read_manifest, write_manifest
from liner.playwright_env import configure_frozen_playwright_cache
from liner.project import (
    ProjectFolder,
    SynthesisReviewRequiredError,
    init_project,
    refresh_status_snapshot,
    slugify,
)
from liner.share import ShareOptions, pack, unpack
from liner.status import build_status_payload
from liner.tape import TapeValidationError, load_tape
from liner.types import CompileResult, SourceContent, SourceSpec, Tape

app = typer.Typer(
    help="Liner — build Liner project folders from curated source recipes.",
    no_args_is_help=True,
    add_completion=False,
)
cache_app = typer.Typer(help="Cache management commands.", no_args_is_help=True)
app.add_typer(cache_app, name="cache")
skills_app = typer.Typer(
    help="Find installed skills that can be used as sources.", no_args_is_help=True
)
app.add_typer(skills_app, name="skills")
project_app = typer.Typer(help="Inspect Liner Project state safely.", no_args_is_help=True)
app.add_typer(project_app, name="project")
sources_app = typer.Typer(help="Plan and apply Source changes safely.", no_args_is_help=True)
app.add_typer(sources_app, name="sources")
adapters_app = typer.Typer(
    help="Inspect and explicitly manage optional agent maintenance adapters.",
    no_args_is_help=True,
)
app.add_typer(adapters_app, name="adapters")
agent_skill_app = typer.Typer(
    help="Inspect and explicitly manage the optional `liner` skill that finds your liners.",
    no_args_is_help=True,
)
app.add_typer(agent_skill_app, name="agent-skill")
research_app = typer.Typer(
    help="Build a liner step by step, with your own agent doing the research.",
    no_args_is_help=True,
)
app.add_typer(research_app, name="research")
gaps_app = typer.Typer(
    help="Track what a liner could not answer.",
    no_args_is_help=True,
)
app.add_typer(gaps_app, name="gaps")
findings_app = typer.Typer(
    help="Review what real work learned before it reaches a liner.",
    no_args_is_help=True,
)
app.add_typer(findings_app, name="findings")
tests_app = typer.Typer(
    add_completion=False,
    no_args_is_help=True,
    help="Record whether a liner improved an answer.",
)
app.add_typer(tests_app, name="test")
research_subagent_app = typer.Typer(
    help="Manage the optional Claude Code subagent that reads fetched pages.",
    no_args_is_help=True,
)
research_app.add_typer(research_subagent_app, name="subagent")
research_guard_app = typer.Typer(
    help="Make Claude Code ask you before it runs a command that needs your approval.",
    no_args_is_help=True,
)
research_app.add_typer(research_guard_app, name="guard")

err_console = Console(stderr=True)


def _version_callback(value: bool) -> None:
    if value:
        typer.echo(f"liner {__version__}")
        raise typer.Exit()


@app.callback()
def _root(
    version: bool = typer.Option(
        False, "--version", callback=_version_callback, is_eager=True, help="Show version and exit."
    ),
) -> None:
    if sys.platform == "win32":
        try:
            sys.stdout.reconfigure(encoding="utf-8")  # type: ignore[attr-defined]
        except Exception:
            pass


@adapters_app.command("inspect", help="Inspect an optional adapter without changing it.")
def adapter_inspect(
    environment: str = typer.Argument(..., help="Supported agent environment: codex or claude."),
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to inspect."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    _run_adapter_inspect(environment, home, json_output=json_output)


def _run_adapter_inspect(
    environment: str, home: Path, *, json_output: bool, skill: str = SKILL_NAME
) -> None:
    import json as _json

    try:
        inspection = inspect_agent_adapter(environment, home, skill=skill)
    except AdapterError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(inspection, ensure_ascii=False, indent=2))
    else:
        typer.echo(
            f"{inspection['environment']}: {inspection['status']} "
            f"({inspection['compatibility']}) at {inspection['target']}"
        )


def _run_adapter_action(
    action: str,
    environment: str,
    home: Path,
    *,
    approved: bool,
    json_output: bool,
    skill: str = SKILL_NAME,
    safeguards: tuple[str, ...] = (),
) -> None:
    import json as _json

    try:
        plan = plan_agent_adapter(action, environment, home, skill=skill)
        if not approved and (plan.approval_required or safeguards):
            preview = plan.to_dict()
            if safeguards:
                preview["safeguards"] = [_SAFEGUARD_EFFECTS[name] for name in safeguards]
            typer.echo(_json.dumps(preview, ensure_ascii=False, indent=2))
            raise typer.Exit(code=2)
        receipt = apply_agent_adapter_plan(plan, approved=approved)
    except AdapterError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    results = _apply_safeguards(home, safeguards)
    if json_output:
        payload = receipt.to_dict()
        if safeguards:
            payload["safeguards"] = results
        typer.echo(_json.dumps(payload, ensure_ascii=False, indent=2))
    else:
        typer.echo(
            f"{receipt.action} {receipt.environment} adapter at {receipt.target}; "
            f"receipt: {receipt.receipt_path}"
        )
        for result in results:
            detail = result.get("reason") or result.get("path")
            typer.echo(f"{result['name']}: {result['status']}: {detail}")


_SAFEGUARD_EFFECTS = {
    "subagent": "install the liner-researcher subagent in ~/.claude/agents",
    "guard": "install the approval prompt rules in ~/.claude/settings.json",
}


def _safeguards(environment: str, *, subagent: bool = True, guard: bool = True) -> tuple[str, ...]:
    """Name the Claude Code safeguards that travel with the `liner` skill."""
    if environment != "claude":
        return ()
    return tuple(name for name, wanted in (("subagent", subagent), ("guard", guard)) if wanted)


def _apply_safeguards(home: Path, safeguards: tuple[str, ...]) -> list[dict[str, object]]:
    """Report a safeguard that can't change instead of failing the finished skill step."""
    from liner.approval_guard import GuardError, install_guard
    from liner.research_subagent import SubagentError, install_subagent

    steps = {"subagent": install_subagent, "guard": install_guard}
    results: list[dict[str, object]] = []
    for name in safeguards:
        try:
            results.append({"name": name, **steps[name](home)})
        except (SubagentError, GuardError, OSError) as error:
            results.append({"name": name, "status": "skipped", "reason": str(error)})
    return results


@adapters_app.command("install", help="Plan or explicitly install an optional adapter.")
def adapter_install(
    environment: str = typer.Argument(..., help="Supported agent environment: codex or claude."),
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to update."),
    yes: bool = typer.Option(False, "--yes", help="Approve the exact previewed file effects."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    _run_adapter_action("install", environment, home, approved=yes, json_output=json_output)


@adapters_app.command("update", help="Plan or explicitly update a managed adapter.")
def adapter_update(
    environment: str = typer.Argument(..., help="Supported agent environment: codex or claude."),
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to update."),
    yes: bool = typer.Option(False, "--yes", help="Approve the exact previewed file effects."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    _run_adapter_action("update", environment, home, approved=yes, json_output=json_output)


@adapters_app.command("remove", help="Plan or explicitly remove only Liner-managed content.")
def adapter_remove(
    environment: str = typer.Argument(..., help="Supported agent environment: codex or claude."),
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to update."),
    yes: bool = typer.Option(False, "--yes", help="Approve the exact previewed file effects."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    _run_adapter_action("remove", environment, home, approved=yes, json_output=json_output)


@agent_skill_app.command("inspect", help="Inspect the `liner` skill without changing it.")
def agent_skill_inspect(
    environment: str = typer.Argument(..., help="Supported agent environment: codex or claude."),
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to inspect."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    _run_adapter_inspect(environment, home, json_output=json_output, skill=ROUTER_SKILL_NAME)


@agent_skill_app.command(
    "install",
    help="Plan or explicitly install the `liner` skill, plus Claude Code's safeguards.",
)
def agent_skill_install(
    environment: str = typer.Argument(..., help="Supported agent environment: codex or claude."),
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to update."),
    yes: bool = typer.Option(False, "--yes", help="Approve the exact previewed file effects."),
    subagent: bool = typer.Option(
        True, "--subagent/--no-subagent", help="On Claude Code, add the page-reading subagent."
    ),
    guard: bool = typer.Option(
        True, "--guard/--no-guard", help="On Claude Code, ask you before approval commands."
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    _run_adapter_action(
        "install",
        environment,
        home,
        approved=yes,
        json_output=json_output,
        skill=ROUTER_SKILL_NAME,
        safeguards=_safeguards(environment, subagent=subagent, guard=guard),
    )


@agent_skill_app.command("update", help="Refresh the `liner` skill and its subject list.")
def agent_skill_update(
    environment: str = typer.Argument(..., help="Supported agent environment: codex or claude."),
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to update."),
    yes: bool = typer.Option(False, "--yes", help="Approve the exact previewed file effects."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    _run_adapter_action(
        "update", environment, home, approved=yes, json_output=json_output, skill=ROUTER_SKILL_NAME
    )


@agent_skill_app.command("remove", help="Plan or explicitly remove the `liner` skill.")
def agent_skill_remove(
    environment: str = typer.Argument(..., help="Supported agent environment: codex or claude."),
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to update."),
    yes: bool = typer.Option(False, "--yes", help="Approve the exact previewed file effects."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    _run_adapter_action(
        "remove", environment, home, approved=yes, json_output=json_output, skill=ROUTER_SKILL_NAME
    )
    if environment == "claude" and not json_output:
        typer.echo(
            "The subagent and approval rules stay. Use `liner research subagent remove --yes` "
            "and `liner research guard remove --yes` to take them out."
        )


@app.command(help="Scaffold a new Liner project folder.")
def init(
    path: Path = typer.Argument(
        ...,
        help="Folder to create (or populate). Pass a slug to create ./<slug>/.",
    ),
    force: bool = typer.Option(False, "--force", help="Overwrite existing files in the folder."),
    mode: str | None = typer.Option(
        None,
        "--mode",
        help="A label on the tape. It changes nothing.",
        hidden=True,
    ),
    jtbd: str | None = typer.Option(None, "--jtbd", help="Set the job-to-be-done."),
    title: str | None = typer.Option(None, "--title", help="Set the mixtape title."),
    description: str | None = typer.Option(
        None, "--description", help="Set the mixtape description."
    ),
    curator: str | None = typer.Option(None, "--curator", help="Set the curator name."),
) -> None:
    try:
        mode_value = _normalize_init_mode(mode)
        metadata = _init_metadata_overrides(
            mode=mode_value,
            jtbd=jtbd,
            title=title,
            description=description,
            curator=curator,
        )
    except ValueError as e:
        err_console.print(f"[red]{e}[/]")
        raise typer.Exit(code=1) from e

    try:
        project = init_project(path, force=force)
    except FileExistsError as e:
        err_console.print(f"[red]{e}[/]")
        raise typer.Exit(code=1) from e

    if metadata:
        _apply_init_metadata(project, metadata)
    if jtbd is not None and jtbd.strip():
        _prefill_jtbd(project, jtbd.strip())

    err_console.print(f"[green]Created Liner project[/] {project.path}")
    err_console.print("Next steps:")
    err_console.print(f"  1. Edit [bold]{project.tape_path}[/] with your sources")
    err_console.print(f"  2. Write your synthesis in [bold]{project.synthesis_path}[/]")
    err_console.print(f"  3. Run [bold]liner compile {project.path}[/]")


def _normalize_init_mode(mode: str | None) -> str | None:
    if mode is None:
        return None
    value = mode.strip().lower()
    if value not in {"quick", "methodology"}:
        raise ValueError("--mode must be quick or methodology")
    return value


def _init_metadata_overrides(
    *,
    mode: str | None,
    jtbd: str | None,
    title: str | None,
    description: str | None,
    curator: str | None,
) -> dict[str, str]:
    metadata: dict[str, str] = {}
    if mode is not None:
        metadata["mode"] = mode
    for key, value in {
        "jtbd": jtbd,
        "title": title,
        "description": description,
        "curator": curator,
    }.items():
        if value is None:
            continue
        if key != "description" and not value.strip():
            raise ValueError(f"--{key} cannot be empty")
        metadata[key] = value.strip() if key != "description" else value
    return metadata


def _apply_init_metadata(project: ProjectFolder, metadata: dict[str, str]) -> None:
    import yaml

    raw = yaml.safe_load(project.tape_path.read_text(encoding="utf-8")) or {}
    if not isinstance(raw, dict):
        raw = {}
    raw.update(metadata)
    project.tape_path.write_text(
        yaml.safe_dump(raw, sort_keys=False, allow_unicode=True),
        encoding="utf-8",
    )


def _prepare_research_start(project: ProjectFolder) -> None:
    import yaml

    raw = yaml.safe_load(project.tape_path.read_text(encoding="utf-8")) or {}
    if not isinstance(raw, dict):
        raw = {}
    raw["sources"] = []
    # The starter tape's description and tags describe its sample sources, not this job.
    raw["description"] = f"Guidance for {raw.get('title') or project.path.name}."
    raw["tags"] = []
    project.tape_path.write_text(
        yaml.safe_dump(raw, sort_keys=False, allow_unicode=True),
        encoding="utf-8",
    )
    (project.working_dir / ".liner-initial-assembly").write_text(
        "Created by Liner Core for one first-run Go TUI assembly acceptance.\n",
        encoding="utf-8",
    )


def _prefill_jtbd(project: ProjectFolder, jtbd: str) -> None:
    path = project.working_dir / "01-jtbd-and-knowledge-map.md"
    try:
        text = path.read_text(encoding="utf-8")
    except OSError:
        return
    heading = "## Job-to-be-done"
    next_heading = "\n## Knowledge map"
    if heading not in text or next_heading not in text:
        return
    heading_end = text.index(heading) + len(heading)
    suffix_start = text.index(next_heading)
    replacement = (
        f"\n\n{jtbd}\n\n"
        "_Set via `liner init --jtbd`. Revise here if your understanding sharpens during research._\n"
    )
    path.write_text(text[:heading_end] + replacement + text[suffix_start:], encoding="utf-8")


@app.command(
    help=(
        "Clone a Liner Project's Job to Be Done and Clarify Job answers into a fresh "
        "Liner Project. Retains lineage for later output comparison."
    ),
)
def replay(
    source: Path = typer.Argument(
        ...,
        exists=True,
        file_okay=False,
        dir_okay=True,
        readable=True,
        help="Existing Liner Project to clone the Job to Be Done and Clarify Job answers from.",
    ),
    out: Path | None = typer.Option(
        None,
        "--out",
        help=("Destination folder. Defaults to <source>-replay alongside the source folder."),
    ),
    name: str | None = typer.Option(
        None,
        "--name",
        help=("Override the destination folder's slug. Ignored when --out is explicitly set."),
    ),
    force: bool = typer.Option(
        False, "--force", help="Overwrite the destination folder if it already exists."
    ),
) -> None:
    """Selectively clone tape.yaml inputs into a new Liner Project so the
    Curator can run the same Job to Be Done through fresh Corpus Creation.
    Working artifacts, synthesis, and Sources are not copied. The new tape
    records `parent: <source>` so later comparison can use the original
    Liner Project as a lineage reference.
    """
    from liner.tape import load_tape

    try:
        src_project = ProjectFolder(source)
        src_tape = load_tape(src_project.tape_path)
    except Exception as e:  # noqa: BLE001 - rich passthrough
        err_console.print(f"[red]Could not load source tape:[/] {e}")
        raise typer.Exit(code=1) from e

    # Resolve the output folder. --out wins; otherwise --name; otherwise
    # append -replay to the source folder name.
    if out is not None:
        dest = out
    elif name is not None:
        dest = source.parent / name
    else:
        dest = source.parent / f"{source.name}-replay"

    if dest.exists() and not force:
        err_console.print(f"[red]Destination already exists:[/] {dest}. Use --force to overwrite.")
        raise typer.Exit(code=1)

    try:
        project = init_project(dest, force=force)
    except FileExistsError as e:
        err_console.print(f"[red]{e}[/]")
        raise typer.Exit(code=1) from e

    # Read the just-scaffolded tape, overwrite the curator-facing fields from
    # the source tape, set parent, then write back. Crucially we do NOT copy
    # sources, working/, or synthesis — those have to be regenerated for the
    # replay to mean anything.
    _replay_tape(project, src_tape, source.resolve())

    err_console.print(f"[green]Replay folder ready:[/] {project.path}")
    err_console.print(f"  parent: [dim]{source.resolve()}[/] (lineage source for later comparison)")
    err_console.print("Next steps:")
    err_console.print(f"  1. Review the Job to Be Done and answers in {project.tape_path}")
    err_console.print(
        f"  2. Ask your agent to build it, then compare its outputs with {source.name}"
    )


def _replay_tape(project: ProjectFolder, src_tape: Tape, parent_path: Path) -> None:
    """Copy curator-facing fields from `src_tape` into the freshly-scaffolded
    project's tape.yaml, set `parent`, and write back. Implementation detail:
    we go through a YAML round-trip so any comments in the starter tape are
    preserved.
    """
    import yaml

    tape_path = project.tape_path
    raw_text = tape_path.read_text(encoding="utf-8")
    doc = yaml.safe_load(raw_text) or {}

    # Curator-facing fields that DEFINE the input. Anything that's a result
    # of the previous pipeline (sources, generated working files) stays out.
    doc["title"] = src_tape.title
    doc["description"] = src_tape.description
    doc["curator"] = src_tape.curator
    if src_tape.mode is not None:
        doc["mode"] = src_tape.mode
    if src_tape.jtbd is not None:
        doc["jtbd"] = src_tape.jtbd
    if src_tape.jtbd_clarifications:
        doc["jtbd_clarifications"] = [
            {"question": c.question, "answer": c.answer} for c in src_tape.jtbd_clarifications
        ]
    if src_tape.methodology_version is not None:
        doc["methodology_version"] = src_tape.methodology_version
    doc["parent"] = str(parent_path)
    # Sources reset to empty — the new pipeline will rediscover them.
    doc["sources"] = []

    tape_path.write_text(yaml.safe_dump(doc, sort_keys=False, allow_unicode=True), encoding="utf-8")


@app.command(
    help="Compile a Liner project: fetch sources, write mixtape/MIXTAPE.md and mixtape/sources/."
)
def compile(  # noqa: A001 - command name matches the v1 surface
    folder: Path = typer.Argument(
        ...,
        exists=True,
        file_okay=False,
        dir_okay=True,
        readable=True,
        help="Path to the Liner project folder.",
    ),
    skip_optional: bool = typer.Option(
        False, "--skip-optional", help="Exclude sources marked priority: optional."
    ),
    no_cache: bool = typer.Option(
        False, "--no-cache", help="Skip cache; force re-fetch and do not store results."
    ),
    include_sections: str | None = typer.Option(
        None, "--include-sections", help="Comma-separated section names to include."
    ),
    exclude_sections: str | None = typer.Option(
        None, "--exclude-sections", help="Comma-separated section names to exclude."
    ),
    max_transcript_length: int | None = typer.Option(
        None, "--max-transcript-length", help="Truncate transcripts to N characters."
    ),
    cookies: Path | None = typer.Option(
        None,
        "--cookies",
        exists=True,
        dir_okay=False,
        readable=True,
        help="Path to a Netscape-format cookies file for yt-dlp / youtube-transcript-api.",
    ),
    emit_events: bool = typer.Option(
        False,
        "--emit-events",
        help="Emit newline-delimited JSON progress events to stdout (for TUI / programmatic use). "
        "Final line is a `result` event describing the produced folder.",
    ),
) -> None:
    project = ProjectFolder(folder)

    if not project.tape_path.exists():
        err_console.print(
            f"[red]No tape.yaml in {project.corpus_path}.[/] Run `liner init {folder}` first."
        )
        raise typer.Exit(code=1)

    try:
        load_tape(project.tape_path)
    except TapeValidationError as e:
        err_console.print(f"[red]Tape validation failed:[/] {e}")
        raise typer.Exit(code=1) from e

    config = load_config()
    if cookies is not None:
        config = replace(config, fetch=replace(config.fetch, cookies_file=cookies))

    handlers: dict[str, SourceHandler] = {
        "youtube": YouTubeHandler(config.fetch),
        "web": WebHandler(config.fetch),
    }

    cache: SourceCache | None = None
    if not no_cache:
        cache = SourceCache(config.cache_path)

    include = _split_csv(include_sections)
    exclude = _split_csv(exclude_sections)

    progress = JsonEventReporter() if emit_events else RichProgressReporter()

    try:
        try:
            result = compile_project(
                project,
                cache=cache,
                handlers=handlers,
                config=config,
                include_sections=include,
                exclude_sections=exclude,
                skip_optional=skip_optional,
                max_transcript_length=max_transcript_length,
                no_cache=no_cache,
                progress=progress,
            )
        except (MissingSynthesisError, SynthesisReviewRequiredError) as e:
            err_console.print(f"[red]{e}[/]")
            raise typer.Exit(code=1) from e
        except ProjectApplyError as e:
            err_console.print(f"[red]{e.report.message}[/]")
            for action in e.report.recovery:
                err_console.print(f"  {action}")
            raise typer.Exit(code=1) from e
    finally:
        if cache is not None:
            cache.close()
        web_handler = handlers["web"]
        if isinstance(web_handler, WebHandler):
            web_handler.close()

    if emit_events:
        emit_result(compile_result_to_payload(result, project=project))
    else:
        err_console.print(f"[green]Wrote[/] {project.mixtape_path}")
        source_summary = _source_output_summary(result)
        err_console.print(f"[green]Wrote[/] {project.sources_dir}/ {source_summary}")
    from liner.citations import unlinked_citations

    filtered = bool(include or exclude or skip_optional)
    unlinked = [] if filtered else unlinked_citations(project)
    if unlinked:
        labels = ", ".join(f"[^{label}]" for label in unlinked)
        err_console.print(f"[yellow]No compiled source matches these citations:[/] {labels}")

    total = result.total_attempted
    succeeded = result.total_succeeded
    if total > 0 and succeeded == 0:
        raise typer.Exit(code=3)
    if succeeded < total:
        raise typer.Exit(code=2)


def _source_output_summary(result: CompileResult) -> str:
    total = result.total_attempted
    succeeded = result.total_succeeded
    unavailable = total - succeeded
    if total == 0:
        return "(0 sources)"
    if unavailable == 0:
        return f"({succeeded}/{total} usable sources)"
    return f"({succeeded}/{total} usable sources, {unavailable} unavailable placeholders)"


@app.command(help="Pack a Liner project folder into a .mixtape zip for sharing.")
def share(
    folder: Path = typer.Argument(
        ...,
        exists=True,
        file_okay=False,
        dir_okay=True,
        readable=True,
        help="Path to the Liner project folder.",
    ),
    out: Path | None = typer.Option(
        None, "--out", help="Output archive path. Default: <folder>.mixtape next to the folder."
    ),
    no_working_notes: bool = typer.Option(
        False, "--no-working-notes", help="Exclude working/ from the archive."
    ),
    no_source_content: bool = typer.Option(
        False, "--no-source-content", help="Exclude sources/ from the archive."
    ),
    no_personal: bool = typer.Option(
        False,
        "--no-personal",
        help="Exclude local-sources/ and personal/ from the archive. Required for library submissions.",
    ),
    minimal: bool = typer.Option(
        False, "--minimal", help="Archive only tape.yaml. Recipient compiles from scratch."
    ),
) -> None:
    project = ProjectFolder(folder)
    options = ShareOptions(
        include_working_notes=not no_working_notes,
        include_source_content=not no_source_content,
        include_personal=not no_personal,
        minimal=minimal,
    )

    # Library-eligibility soft warning.
    local_file_count = 0
    if project.tape_path.exists():
        try:
            tape = load_tape(project.tape_path)
            local_file_count = sum(1 for s in tape.sources if s.type == "local_file")
            from liner.handlers.liner import reused_local_sources

            reused_local = reused_local_sources(project, tape.sources)
            if reused_local and options.include_source_content and not minimal:
                err_console.print(
                    "[yellow]Before you send this:[/] these sources were local files in "
                    "other liners, and their saved text goes into the archive: "
                    + ", ".join(reused_local)
                    + ". Add --no-source-content to leave saved text out."
                )
            local_file_count += len(reused_local)
        except TapeValidationError:
            # Validation issues are surfaced by compile; share keeps going.
            pass

    try:
        result = pack(project, out_path=out, options=options)
    except FileNotFoundError as e:
        err_console.print(f"[red]{e}[/]")
        raise typer.Exit(code=1) from e

    err_console.print(f"[green]Wrote[/] {result.archive_path} ({result.entry_count} entries)")
    if local_file_count > 0:
        err_console.print(
            f"[yellow]Note:[/] this mixtape contains {local_file_count} "
            f"local_file source{'s' if local_file_count != 1 else ''} and is not "
            "library-eligible. Library submissions must use only public, fetchable sources."
        )


@app.command(name="import", help="Unpack a .mixtape archive and refetch any uncached sources.")
def import_(
    archive: Path = typer.Argument(
        ...,
        exists=True,
        dir_okay=False,
        readable=True,
        help="Path to the .mixtape archive.",
    ),
    destination: Path = typer.Argument(
        Path("."), help="Folder to extract into. Default: current directory."
    ),
    no_refetch: bool = typer.Option(
        False, "--no-refetch", help="Skip refetching sources after extraction."
    ),
    overwrite: bool = typer.Option(
        False, "--overwrite", help="Replace matching files in a folder that already exists."
    ),
    cookies: Path | None = typer.Option(
        None,
        "--cookies",
        exists=True,
        dir_okay=False,
        readable=True,
        help="Cookies file for refetch (same semantics as compile --cookies).",
    ),
) -> None:
    try:
        project = unpack(archive, destination, overwrite=overwrite)
    except (FileExistsError, FileNotFoundError, ValueError) as e:
        err_console.print(f"[red]{e}[/]")
        raise typer.Exit(code=1) from e

    err_console.print(f"[green]Extracted[/] {project.path}")

    if no_refetch:
        return

    try:
        tape = load_tape(project.tape_path)
    except TapeValidationError as e:
        err_console.print(f"[red]Tape validation failed:[/] {e}")
        raise typer.Exit(code=1) from e

    config = load_config()
    if cookies is not None:
        config = replace(config, fetch=replace(config.fetch, cookies_file=cookies))

    handlers: dict[str, SourceHandler] = {
        "youtube": YouTubeHandler(config.fetch),
        "web": WebHandler(config.fetch),
        "repository": RepositoryHandler(),
    }

    from liner.compile import content_is_usable

    refetched = 0
    skipped = 0
    failed = 0
    with SourceCache(config.cache_path) as cache:
        for spec in tape.sources:
            cached = cache.get(spec.url)
            if cached is not None and content_is_usable(spec, cached):
                skipped += 1
                continue
            handler = handlers.get(spec.type)
            if handler is None:
                failed += 1
                err_console.print(f"[yellow]warn[/] no handler for {spec.type!r}: {spec.url}")
                continue
            try:
                content = handler.fetch(spec)
            except Exception as e:
                failed += 1
                err_console.print(f"[yellow]warn[/] {spec.url}: {e}")
                continue
            ttl = (
                config.cache.youtube_ttl_days
                if spec.type == "youtube"
                else config.cache.web_ttl_days
            )
            cache.put(content, source_type=spec.type, ttl_days=ttl)
            refetched += 1

    web_handler = handlers["web"]
    if isinstance(web_handler, WebHandler):
        web_handler.close()

    err_console.print(f"[green]Refetch:[/] {refetched} new, {skipped} cached, {failed} failed")


@app.command(help="Fetch a remote tape (URL) or copy a local tape to a destination.")
def clone(
    source: str = typer.Argument(..., help="Remote URL or local path to a tape file."),
    destination: Path | None = typer.Argument(None, help="Where to write the cloned tape."),
) -> None:
    if source.startswith(("http://", "https://")):
        try:
            with guarded_client(
                allow_private_network=load_config().fetch.allow_private_network,
                follow_redirects=True,
                timeout=20.0,
            ) as client:
                response = get_bounded(client, source)
            response.raise_for_status()
        except httpx.HTTPError as e:
            err_console.print(f"[red]Failed to fetch[/] {source}: {e}")
            raise typer.Exit(code=1) from e
        filename = source.rsplit("/", 1)[-1] or "tape.yaml"
        dest = destination or Path.cwd() / filename
        dest.write_text(response.text, encoding="utf-8")
    else:
        src_path = Path(source)
        if not src_path.exists():
            err_console.print(f"[red]Source not found:[/] {source}")
            raise typer.Exit(code=1)
        dest = destination or Path.cwd() / src_path.name
        shutil.copy2(src_path, dest)

    err_console.print(f"[green]Wrote[/] {dest}")


@app.command(
    name="setup-js",
    help="Install Playwright and the Chromium binary needed by `render: js` web sources.",
)
def setup_js(
    yes: bool = typer.Option(False, "--yes", "-y", help="Skip the confirmation prompt."),
) -> None:
    """One-shot setup for JavaScript-rendered web sources.

    Installs the `playwright` Python package into the linersh environment (if
    it isn't already there) and downloads only the headless Chromium shell that
    `render: js` launches. Idempotent — safe to re-run.
    """
    configure_frozen_playwright_cache()

    # 1. Decide whether we need to install or upgrade the playwright Python
    #    package. Versions before 1.49 cannot fetch the headless shell alone.
    try:
        import playwright  # noqa: F401

        need_install = False
    except ImportError:
        need_install = True
    installed = None if need_install else _installed_playwright_version()
    need_upgrade = installed is not None and installed < _ONLY_SHELL_MIN_VERSION

    actions: list[str] = []
    if need_install:
        actions.append(f"Install the `playwright` Python package ({PLAYWRIGHT_PACKAGE_SIZE})")
    elif need_upgrade:
        actions.append(
            f"Upgrade the `playwright` Python package to 1.49 or later ({PLAYWRIGHT_PACKAGE_SIZE})"
        )
    actions.append(
        f"Download Playwright's headless Chromium ({HEADLESS_CHROMIUM_SIZE} on first run; "
        "subsequent runs are no-ops)"
    )

    err_console.print("[bold]liner setup-js[/] will:")
    for i, action in enumerate(actions, start=1):
        err_console.print(f"  {i}. {action}")

    if not yes:
        ok = typer.confirm("Continue?", default=True)
        if not ok:
            err_console.print("[yellow]Cancelled.[/]")
            raise typer.Exit(code=1)

    if need_install or need_upgrade:
        if _is_frozen_binary():
            built_with = "with a Playwright older than 1.49"
            if need_install:
                built_with = "without the Playwright Python package"
            err_console.print(
                f"[red]This bundled Liner binary was built {built_with}.[/]\n"
                "Publish the platform package with Playwright 1.49 or later included, "
                "then re-run:\n"
                "  liner setup-js"
            )
            raise typer.Exit(code=1)
        _install_playwright_package(upgrade=need_upgrade)

    # 2. Download Chromium. Normal Python installs use sys.executable -m so
    #    we hit the same venv — after pipx inject the freshly-installed
    #    playwright is available in a subprocess, even though the running
    #    process can't import it. Frozen platform binaries can't use
    #    `sys.executable -m`, so they run Playwright's entrypoint in-process.
    err_console.print("[bold]Downloading Chromium…[/]")
    returncode = _run_playwright_install_chromium()
    if returncode != 0:
        err_console.print(
            f"[red]playwright {' '.join(_playwright_install_args())} exited with code {returncode}.[/]"
        )
        raise typer.Exit(code=returncode)
    err_console.print("[green]Chromium is ready.[/] `render: js` sources will now work.")

    # 3. Friendly nudge if we just installed in a non-pipx environment so the
    #    user knows the package is in *their* python and not contained.
    if (need_install or need_upgrade) and not _is_pipx_venv():
        err_console.print(
            "[dim]Note: playwright was installed into your active Python "
            "environment. If you uninstall liner later, run `pip uninstall playwright` "
            "to remove the package; the Chromium binary lives in Playwright's own cache.[/]"
        )


# Measured with Playwright 1.64 on macOS arm64. `render: js` launches Chromium
# headless, which Playwright serves from its separate headless shell, so setup
# skips the full Chromium build (about 290 MB to download, 575 MB on disk).
HEADLESS_CHROMIUM_SIZE = "about 100 MB to download, about 220 MB on disk"
PLAYWRIGHT_PACKAGE_SIZE = "about 45 MB to download, about 135 MB on disk"

# Playwright added `install --only-shell` in 1.49.
_ONLY_SHELL_MIN_VERSION = (1, 49)
PLAYWRIGHT_REQUIREMENT = "playwright>=1.49"


def _installed_playwright_version() -> tuple[int, ...] | None:
    try:
        from playwright._repo_version import version
    except ImportError:
        try:
            from importlib.metadata import version as dist_version

            version = dist_version("playwright")
        except Exception:
            return None
    parts: list[int] = []
    for part in str(version).split(".")[:2]:
        digits = "".join(ch for ch in part if ch.isdigit())
        if not digits:
            break
        parts.append(int(digits))
    return tuple(parts) or None


def _playwright_install_args() -> list[str]:
    """Arguments for `playwright install` that fetch only what Liner launches.

    `setup-js` upgrades Playwright to 1.49 or later first, so the full Chromium
    build is never downloaded.
    """

    return ["install", "--only-shell", "chromium"]


def _run_playwright_install_chromium() -> int:
    import subprocess
    import sys

    install_args = _playwright_install_args()
    if not _is_frozen_binary():
        proc = subprocess.run(
            [sys.executable, "-m", "playwright", *install_args],
            check=False,
        )
        return proc.returncode

    try:
        from playwright.__main__ import main as playwright_main
    except ImportError:
        err_console.print(
            "[red]This bundled Liner binary cannot find Playwright.[/]\n"
            "Publish the platform package with Playwright included, then re-run:\n"
            "  liner setup-js"
        )
        return 1

    original_argv = sys.argv[:]
    try:
        sys.argv = ["playwright", *install_args]
        try:
            playwright_main()
        except SystemExit as exc:
            if exc.code is None:
                return 0
            if isinstance(exc.code, int):
                return exc.code
            return 1
        return 0
    finally:
        sys.argv = original_argv


def _install_playwright_package(*, upgrade: bool = False) -> None:
    """Install (or upgrade) `playwright` in the environment liner runs from.

    Prefers `pipx inject` (the right tool for pipx-managed installs, and what
    every recent pipx tutorial recommends). Falls back to `python -m pip` for
    non-pipx installs. Either way it asks for Playwright 1.49 or later, which
    can download the headless shell alone.
    """
    import shutil
    import subprocess
    import sys

    if _is_pipx_venv():
        pipx = shutil.which("pipx")
        inject = ["inject", "linersh", PLAYWRIGHT_REQUIREMENT]
        if upgrade:
            inject.append("--force")
        if pipx is None:
            err_console.print(
                "[red]Detected a pipx-managed linersh install, but `pipx` is not on PATH.[/]\n"
                "Install pipx (e.g. `brew install pipx`) and re-run, or install playwright manually:\n"
                f"  pipx {' '.join(inject)}"
            )
            raise typer.Exit(code=1)
        err_console.print(f"[bold]Installing playwright via `pipx {' '.join(inject)}`…[/]")
        proc = subprocess.run([pipx, *inject], check=False)
        if proc.returncode != 0:
            err_console.print(
                f"[red]pipx inject failed with code {proc.returncode}.[/] "
                "Try running it manually for a clearer error."
            )
            raise typer.Exit(code=proc.returncode)
        return

    # Non-pipx install — fall back to pip in the current python env.
    pip_args = ["install", PLAYWRIGHT_REQUIREMENT]
    if upgrade:
        pip_args.insert(1, "--upgrade")
    err_console.print(f"[bold]Installing playwright via `pip {' '.join(pip_args)}`…[/]")
    proc = subprocess.run([sys.executable, "-m", "pip", *pip_args], check=False)
    if proc.returncode != 0:
        err_console.print(
            f"[red]pip install playwright failed with code {proc.returncode}.[/]\n"
            "Install it manually:\n"
            f"  {sys.executable} -m pip {' '.join(pip_args)}"
        )
        raise typer.Exit(code=proc.returncode)


def _is_pipx_venv() -> bool:
    """True if liner is running from a pipx-managed virtual environment."""
    import sys
    from pathlib import Path

    return "pipx" in Path(sys.prefix).parts


def _is_frozen_binary() -> bool:
    import sys

    return bool(getattr(sys, "frozen", False))


@app.command(name="list", help="List Liner project folders in the current directory.")
def list_projects(
    directory: Path = typer.Option(Path("."), "--dir", help="Directory to search."),
    json_output: bool = typer.Option(False, "--json", help="Emit JSON for programmatic use."),
    recursive: bool = typer.Option(
        False, "--recursive", "-r", help="Recurse one level into subdirs."
    ),
) -> None:
    import json as _json
    from datetime import datetime as _dt

    candidates: list[Path] = []
    for child in sorted(p for p in directory.iterdir() if p.is_dir()):
        if _is_liner_project_dir(child):
            candidates.append(child)
        elif recursive:
            for grand in sorted(p for p in child.iterdir() if p.is_dir()):
                if _is_liner_project_dir(grand):
                    candidates.append(grand)

    records: list[dict[str, object]] = []
    for project_path in candidates:
        project = ProjectFolder(project_path)
        try:
            tape = load_tape(project.tape_path)
        except (TapeValidationError, Exception):
            continue
        records.append(
            {
                "path": str(project_path),
                "name": project_path.name,
                "title": tape.title,
                "description": tape.description,
                "curator": tape.curator,
                "mode": tape.mode,
                "jtbd": tape.jtbd,
                "tags": list(tape.tags),
                "source_count": len(tape.sources),
                "modified_iso": _dt.fromtimestamp(
                    max(project_path.stat().st_mtime, project.corpus_path.stat().st_mtime)
                ).isoformat(),
            }
        )

    if json_output:
        typer.echo(_json.dumps(records, ensure_ascii=False))
        return

    if not records:
        err_console.print("[yellow]No Liner project folders found.[/]")
        return
    for rec in records:
        modified = _dt.fromisoformat(str(rec["modified_iso"])).strftime("%Y-%m-%d %H:%M")
        mode = rec["mode"] or "—"
        typer.echo(
            f"{rec['name']}\t{rec['title']} "
            f"({rec['source_count']} sources, mode={mode}, modified {modified})"
        )


def _is_liner_project_dir(path: Path) -> bool:
    project = ProjectFolder(path)
    return project.tape_path.exists()


def _research_project(slug: str, root: Path | None) -> ProjectFolder:
    from liner.library import default_library_root

    if slug in {"", ".", ".."} or Path(slug).name != slug:
        err_console.print(f"[red]{slug} is not a liner name.[/]")
        raise typer.Exit(code=1)
    project = ProjectFolder((root or default_library_root()).expanduser().resolve() / slug)
    if not project.is_valid():
        err_console.print(f"[red]No liner named {slug} in {project.path.parent}.[/]")
        raise typer.Exit(code=1)
    return project


def _print_research_step(step: dict[str, object], json_output: bool) -> None:
    import json as _json

    if json_output:
        typer.echo(_json.dumps(step, ensure_ascii=False, indent=2))
        return
    typer.echo(f"{step['step']}: {step['instructions']}")
    if "prompt_path" in step:
        typer.echo(f"prompt_path: {step['prompt_path']}")


def _refresh_liner_skills() -> list[str]:
    """Update the installed `liner` skill's subject list in each agent that has it.

    Installing the skill was the person's go-ahead, and the update only rewrites
    Liner's own managed files, so a finished build refreshes it without asking.
    """
    updated: list[str] = []
    home = Path.home()
    for environment in sorted(SUPPORTED_ENVIRONMENTS):
        try:
            inspection = inspect_agent_adapter(environment, home, skill=ROUTER_SKILL_NAME)
            if inspection["status"] != "update_required":
                continue
            plan = plan_agent_adapter("update", environment, home, skill=ROUTER_SKILL_NAME)
            apply_agent_adapter_plan(plan, approved=True)
        except (AdapterError, OSError, ValueError):
            continue
        updated.append(environment)
    return updated


def _compile_research_corpus(project: ProjectFolder) -> None:
    from liner.citations import unlinked_citations
    from liner.research import ResearchError

    config = load_config()
    handlers: dict[str, SourceHandler] = {
        "youtube": YouTubeHandler(config.fetch),
        "web": WebHandler(config.fetch),
    }
    cache = SourceCache(config.cache_path)
    try:
        compile_project(
            project, cache=cache, handlers=handlers, config=config, require_all_sources=True
        )
    except (
        IncompleteCompileError,
        MissingSynthesisError,
        SynthesisReviewRequiredError,
        TapeValidationError,
        httpx.HTTPError,
    ) as error:
        raise ResearchError(str(error)) from error
    finally:
        cache.close()
    unlinked = unlinked_citations(project)
    if unlinked:
        labels = ", ".join(f"[^{label}]" for label in unlinked)
        raise ResearchError(
            f"The synthesis cites {labels}, but no compiled source has that address. "
            "Each footnote must be only a source's URL or local path. Point it at one, or "
            "add the address to that source's citation, then compile again."
        )


_ROOT_OPTION = typer.Option(
    None, "--root", help="Library folder. Defaults to LINER_DIR, then ~/liner/projects."
)


@research_app.command("start", help="Start a new liner from a job story.")
def research_start(
    job: str = typer.Argument(..., help="What a future agent should be able to do."),
    title: str | None = typer.Option(None, "--title", help="Name for the liner."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from liner.library import default_library_root
    from liner.research import next_step

    job = job.strip()
    if not job:
        err_console.print("[red]Describe the job for this liner.[/]")
        raise typer.Exit(code=1)
    name = (title or job[:60]).strip()
    path = (root or default_library_root()).expanduser().resolve() / slugify(name)
    try:
        project = init_project(path)
    except FileExistsError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    _apply_init_metadata(project, {"mode": "methodology", "jtbd": job, "title": name})
    _prefill_jtbd(project, job)
    _prepare_research_start(project)
    _print_research_step(next_step(project), json_output)


@research_app.command("improve", help="Show how to strengthen a finished liner.")
def research_improve(
    slug: str = typer.Argument(..., help="Liner folder name."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.research import ResearchError, improve_step

    try:
        step = improve_step(_research_project(slug, root))
    except ResearchError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(step, ensure_ascii=False, indent=2))
        return
    for gap in step["open_gaps"]:
        typer.echo(f"gap {gap['id']}: {gap['question']}")
    typer.echo(step["instructions"])


@research_app.command("next", help="Show the step an agent should do now.")
def research_next(
    slug: str = typer.Argument(..., help="Liner folder name."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
    with_prompt: bool = typer.Option(
        False, "--with-prompt", help="Include the prompt text, for clients that can't read files."
    ),
) -> None:
    from liner.research import ResearchError, next_step

    try:
        step = next_step(_research_project(slug, root), include_prompt=with_prompt)
    except ResearchError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    _print_research_step(step, json_output)


@research_app.command("plan", help="List the agent steps ahead and the work each needs.")
def research_plan_command(
    slug: str = typer.Argument(..., help="Liner folder name."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.research import ResearchError, research_plan

    try:
        plan = research_plan(_research_project(slug, root))
    except ResearchError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(plan, ensure_ascii=False, indent=2))
        return
    for item in plan["steps"]:
        typer.echo(f"{item['step']}: {item['work']}, {plan['work_kinds'][item['work']]}")
    typer.echo(plan["instructions"])


@research_app.command("submit", help="Check the current step and move to the next one.")
def research_submit(
    slug: str = typer.Argument(..., help="Liner folder name."),
    answers: Path | None = typer.Option(
        None, "--answers", help="JSON file of the person's Clarify Job answers."
    ),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
    with_prompt: bool = typer.Option(
        False, "--with-prompt", help="Include the prompt text, for clients that can't read files."
    ),
) -> None:
    import json as _json

    from liner.research import ResearchError, submit_step

    try:
        parsed = _json.loads(answers.read_text(encoding="utf-8")) if answers else None
        step = submit_step(
            _research_project(slug, root),
            answers=parsed,
            compile_corpus=_compile_research_corpus,
            include_prompt=with_prompt,
            refresh_skills=_refresh_liner_skills,
        )
    except (ResearchError, OSError, ValueError) as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    _print_research_step(step, json_output)


@research_app.command("approve", help="Record the person's approval of what they reviewed.")
def research_approve(
    slug: str = typer.Argument(..., help="Liner folder name."),
    kind: str = typer.Argument(..., help="synthesis, sources, or operating_layer."),
    token: str = typer.Option(..., "--token", help="Token from `liner research next`."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from liner.research import ResearchError, approve_step

    try:
        step = approve_step(_research_project(slug, root), kind, token)
    except ResearchError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    _print_research_step(step, json_output)


@app.command(help="Save a web page or a reused source to a file and return its path.")
def fetch(
    slug: str = typer.Argument(..., help="Liner folder name."),
    url: str = typer.Argument(
        ..., help="Web page, YouTube video, or liner://<slug>/<source-id> to save."
    ),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from urllib.parse import urlsplit

    from liner.research import THIN_PAGE_CHARACTERS, save_fetched_page

    project = _research_project(slug, root)
    parts = urlsplit(url)
    if parts.scheme == "liner":
        _print_fetched(save_fetched_page(project, _reused_page(project, url)), json_output)
        return
    if parts.scheme not in {"http", "https"} or not parts.netloc:
        err_console.print(f"[red]{url} is not a web page or liner:// address.[/]")
        raise typer.Exit(code=1)
    config = load_config()
    spec = SourceSpec(type="youtube" if is_video_url(url) else "web", url=url)
    page = _cached_page(config, spec)
    if page is None:
        try:
            if spec.type == "youtube":
                page = YouTubeHandler(config.fetch).fetch(spec)
            else:
                page = WebHandler(config.fetch).fetch(spec)
        except Exception as error:  # Transcript libraries raise many types of their own.
            err_console.print(f"[red]Could not fetch {url}: {error}[/]")
            raise typer.Exit(code=1) from error
        if len(page.body) >= THIN_PAGE_CHARACTERS:
            _keep_page(config, spec, page)
    _print_fetched(save_fetched_page(project, page), json_output)


def _reused_page(project: ProjectFolder, url: str) -> SourceContent:
    """Another liner's saved copy of a source, read the way compile will read it."""
    from liner.handlers.liner import LinerHandler

    try:
        return LinerHandler(project).fetch(SourceSpec(type="liner", url=url))
    except (HandlerHardFailure, ValueError) as error:
        err_console.print(f"[red]Could not read {url}: {error}[/]")
        raise typer.Exit(code=1) from error


def _print_fetched(result: dict[str, Any], json_output: bool) -> None:
    import json as _json

    if "warning" in result and not json_output:
        err_console.print(f"[yellow]{result['warning']}[/]")
    typer.echo(_json.dumps(result, ensure_ascii=False, indent=2) if json_output else result["path"])


def _cached_page(config: Config, spec: SourceSpec) -> SourceContent | None:
    """A broken cache only costs a download, so it never stops a fetch."""
    from liner.compile import content_is_usable

    try:
        with SourceCache(config.cache_path) as cache:
            page = cache.get(spec.url)
    except (CacheSchemaMismatchError, sqlite3.Error, OSError, TypeError, ValueError):
        return None
    return page if page is not None and content_is_usable(spec, page) else None


def _keep_page(config: Config, spec: SourceSpec, page: SourceContent) -> None:
    """Share a page with compile and other liners, under the address they'll ask for."""
    from liner.compile import content_is_usable

    if not content_is_usable(spec, page):
        return
    days = config.cache.web_ttl_days
    if spec.type == "youtube":
        days = config.cache.youtube_ttl_days
    try:
        with SourceCache(config.cache_path) as cache:
            cache.put(replace(page, url=spec.url), spec.type, days)
    except (CacheSchemaMismatchError, sqlite3.Error, OSError):
        pass


def _print_build_result(result: dict[str, object]) -> None:
    import json as _json

    typer.echo(_json.dumps(result, ensure_ascii=False))


def _print_subagent(result: dict[str, object], json_output: bool) -> None:
    import json as _json

    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
        return
    typer.echo(f"{result['status']}: {result['path']}")
    if result["status"] == "current":
        typer.echo("If this session can't use it yet, the next one will.")


@research_subagent_app.command("inspect", help="Show whether the subagent is installed.")
def research_subagent_inspect(
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to inspect."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from liner.research_subagent import inspect_subagent

    _print_subagent(inspect_subagent(home), json_output)


@research_subagent_app.command("install", help="Plan or explicitly install the subagent.")
def research_subagent_install(
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to update."),
    yes: bool = typer.Option(False, "--yes", help="Approve the install."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from liner.research_subagent import SubagentError, inspect_subagent, install_subagent

    if not yes:
        _print_subagent({**inspect_subagent(home), "approval_required": True}, json_output)
        raise typer.Exit(code=2)
    try:
        _print_subagent(install_subagent(home), json_output)
    except SubagentError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error


@research_subagent_app.command("remove", help="Plan or explicitly remove the subagent.")
def research_subagent_remove(
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to update."),
    yes: bool = typer.Option(False, "--yes", help="Approve the removal."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from liner.research_subagent import SubagentError, inspect_subagent, remove_subagent

    if not yes:
        _print_subagent({**inspect_subagent(home), "approval_required": True}, json_output)
        raise typer.Exit(code=2)
    try:
        _print_subagent(remove_subagent(home), json_output)
    except SubagentError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error


def _print_guard(result: dict[str, object], json_output: bool) -> None:
    import json as _json

    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
        return
    typer.echo(f"{result['status']}: {result['path']}")


@research_guard_app.command("inspect", help="Show whether Claude Code asks before approvals.")
def research_guard_inspect(
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to inspect."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from liner.approval_guard import inspect_guard

    _print_guard(inspect_guard(home), json_output)


@research_guard_app.command(
    "hook", help="Claude Code's PreToolUse hook: reads the tool call and asks before approvals."
)
def research_guard_hook() -> None:
    import sys

    from liner.approval_guard import run_hook

    typer.echo(run_hook(sys.stdin.read()), nl=False)


@research_guard_app.command("install", help="Plan or explicitly add the permission rules.")
def research_guard_install(
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to update."),
    yes: bool = typer.Option(False, "--yes", help="Approve the change."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from liner.approval_guard import GuardError, inspect_guard, install_guard

    if not yes:
        _print_guard({**inspect_guard(home), "approval_required": True}, json_output)
        raise typer.Exit(code=2)
    try:
        _print_guard(install_guard(home), json_output)
    except GuardError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error


@research_guard_app.command("remove", help="Plan or explicitly remove the permission rules.")
def research_guard_remove(
    home: Path = typer.Option(Path.home(), "--home", help="Agent home to update."),
    yes: bool = typer.Option(False, "--yes", help="Approve the change."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from liner.approval_guard import GuardError, inspect_guard, remove_guard

    if not yes:
        _print_guard({**inspect_guard(home), "approval_required": True}, json_output)
        raise typer.Exit(code=2)
    try:
        _print_guard(remove_guard(home), json_output)
    except GuardError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error


@research_app.command("check", help="Check the artifact a research phase produced.")
def research_check(
    path: Path = typer.Argument(..., help="Liner project folder."),
    phase: str = typer.Argument(..., help="Phase to check."),
    corpus: bool = typer.Option(
        False, "--corpus", help="Treat the path as the corpus folder itself."
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.research_checks import check_phase

    resolved = path.expanduser().resolve()
    try:
        result = check_phase(resolved if corpus else ProjectFolder(resolved).corpus_path, phase)
    except ValueError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=2) from error
    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
    else:
        typer.echo("passed" if result["ok"] else result["message"])
    if not result["ok"]:
        raise typer.Exit(code=1)


def _print_gap(gap: dict[str, str], json_output: bool) -> None:
    import json as _json

    if json_output:
        typer.echo(_json.dumps(gap, ensure_ascii=False, indent=2))
    else:
        typer.echo(f"{gap['id']}\t{gap['status']}\t{gap['question']}")


@gaps_app.command("add", help="Record a question a liner could not answer.")
def gaps_add(
    slug: str = typer.Argument(..., help="Liner folder name."),
    question: str = typer.Argument(..., help="What it could not answer."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from liner.gaps import add_gap
    from liner.working_log import WorkingLogError

    try:
        _print_gap(add_gap(_research_project(slug, root), question), json_output)
    except WorkingLogError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error


@gaps_app.command("close", help="Mark a gap answered.")
def gaps_close(
    slug: str = typer.Argument(..., help="Liner folder name."),
    gap_id: str = typer.Argument(..., help="Gap id from `liner gaps list`."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from liner.gaps import close_gap
    from liner.working_log import WorkingLogError

    try:
        _print_gap(close_gap(_research_project(slug, root), gap_id), json_output)
    except WorkingLogError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error


@gaps_app.command("list", help="List a liner's open gaps.")
def gaps_list(
    slug: str = typer.Argument(..., help="Liner folder name."),
    include_closed: bool = typer.Option(False, "--all", help="Include closed gaps."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.gaps import list_gaps
    from liner.working_log import WorkingLogError

    try:
        result = list_gaps(_research_project(slug, root), include_closed=include_closed)
    except WorkingLogError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
        return
    for gap in result["gaps"]:
        _print_gap(gap, False)


def _print_finding(finding: dict[str, object], json_output: bool) -> None:
    import json as _json

    if json_output:
        typer.echo(_json.dumps(finding, ensure_ascii=False, indent=2))
        return
    typer.echo(f"{finding['id']}\t{finding['status']}\t{finding['finding']}")
    if finding.get("next"):
        typer.echo(str(finding["next"]))


@findings_app.command("add", help="Put something real work learned in a liner's inbox.")
def findings_add(
    slug: str = typer.Argument(..., help="Liner folder name."),
    finding: str = typer.Argument(..., help="What the work found."),
    evidence: str = typer.Option("", "--evidence", help="Where it was seen."),
    closes: str | None = typer.Option(None, "--closes", help="Open gap this would answer."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    from liner.findings import add_finding
    from liner.working_log import WorkingLogError

    try:
        result = add_finding(
            _research_project(slug, root), finding, evidence=evidence, closes=closes
        )
    except WorkingLogError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    _print_finding(result, json_output)


@findings_app.command("list", help="List findings waiting for review.")
def findings_list(
    slug: str = typer.Argument(..., help="Liner folder name."),
    include_reviewed: bool = typer.Option(False, "--all", help="Include reviewed findings."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.findings import list_findings
    from liner.working_log import WorkingLogError

    try:
        result = list_findings(_research_project(slug, root), include_reviewed=include_reviewed)
    except WorkingLogError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
        return
    for item in result["findings"]:
        _print_finding(item, False)


def _review_finding(
    slug: str, finding_id: str, *, accept: bool, yes: bool, root: Path | None, json_output: bool
) -> None:
    from liner.findings import accept_finding, reject_finding
    from liner.working_log import WorkingLogError

    if not yes:
        verb = "accept" if accept else "reject"
        err_console.print(
            f"Ask the person, then run: liner findings {verb} {slug} {finding_id} --yes"
        )
        raise typer.Exit(code=2)
    project = _research_project(slug, root)
    try:
        result = (accept_finding if accept else reject_finding)(project, finding_id)
    except WorkingLogError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    _print_finding(result, json_output)


@findings_app.command("accept", help="Add a reviewed finding to the liner as a Source.")
def findings_accept(
    slug: str = typer.Argument(..., help="Liner folder name."),
    finding_id: str = typer.Argument(..., help="Finding id."),
    yes: bool = typer.Option(False, "--yes", help="The person approved this finding."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    _review_finding(slug, finding_id, accept=True, yes=yes, root=root, json_output=json_output)


@findings_app.command("reject", help="Decline a finding and keep the record.")
def findings_reject(
    slug: str = typer.Argument(..., help="Liner folder name."),
    finding_id: str = typer.Argument(..., help="Finding id."),
    yes: bool = typer.Option(False, "--yes", help="The person declined this finding."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    _review_finding(slug, finding_id, accept=False, yes=yes, root=root, json_output=json_output)


@app.command(help="Serve your liners to an agent over MCP on standard input and output.")
def mcp(root: Path | None = _ROOT_OPTION) -> None:
    import sys

    from liner.mcp_server import serve

    serve(root.expanduser().resolve() if root else None, sys.stdin.buffer, sys.stdout.buffer)


@app.command(help="Report everything in your library, for the terminal navigator.")
def library(root: Path | None = _ROOT_OPTION) -> None:
    from liner.library import default_library_root
    from liner.library_view import build_library_view, to_json

    typer.echo(to_json(build_library_view((root or default_library_root()).expanduser().resolve())))


@app.command(help="Write prompts for your agent, for the terminal navigator.")
def handoff(
    context: str = typer.Argument(..., help="What is on screen."),
    slug: str = typer.Option("", "--slug", help="Liner folder name."),
    line: int = typer.Option(0, "--line", help="Line in view."),
    subject: str = typer.Option("", "--subject", help="Source or item in view."),
) -> None:
    import json as _json

    from liner.handoff import handoff_prompts

    try:
        result = handoff_prompts(context, slug, line=line, subject=subject)
    except ValueError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))


@app.command(help="List the citations in a liner's synthesis and their source files.")
def citations(
    slug: str = typer.Argument(..., help="Liner folder name."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.citations import list_citations

    try:
        result = list_citations(_research_project(slug, root))
    except OSError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
        return
    for citation in result["citations"]:
        typer.echo(
            f"[^{citation['label']}]\t{citation['source']}\t{citation['file'] or 'not compiled'}"
        )
    if not result["citations"]:
        typer.echo("This synthesis has no citations.")


@tests_app.command("add", help="Record one with-and-without comparison the person judged.")
def tests_add(
    slug: str = typer.Argument(..., help="Liner folder name."),
    question: str = typer.Argument(..., help="The question both answers respond to."),
    without_file: Path = typer.Option(
        ..., "--without-file", help="File with the answer written without the liner."
    ),
    with_file: Path = typer.Option(
        ..., "--with-file", help="File with the answer written with the liner."
    ),
    verdict: str = typer.Option(
        ..., "--verdict", help="The person's judgment: better, same, or worse."
    ),
    note: str = typer.Option("", "--note", help="Why the person judged it that way."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.empirical_tests import add_test
    from liner.working_log import WorkingLogError

    project = _research_project(slug, root)
    try:
        result = add_test(
            project,
            question,
            without_answer=without_file.read_text(encoding="utf-8", errors="replace"),
            with_answer=with_file.read_text(encoding="utf-8", errors="replace"),
            verdict=verdict,
            note=note,
        )
    except (WorkingLogError, OSError) as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
    else:
        typer.echo(f"{result['id']}\t{result['verdict']}\t{result['question']}")


@tests_app.command("start", help="Set up a blind comparison for the person to judge.")
def tests_start(
    slug: str = typer.Argument(..., help="Liner folder name."),
    question: str = typer.Argument(..., help="The question both answers respond to."),
    without_file: Path = typer.Option(
        ..., "--without-file", help="File with the answer written without the liner."
    ),
    with_file: Path = typer.Option(
        ..., "--with-file", help="File with the answer written with the liner."
    ),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.empirical_tests import start_blind_test
    from liner.working_log import WorkingLogError

    try:
        result = start_blind_test(
            _research_project(slug, root),
            question,
            without_answer=without_file.read_text(encoding="utf-8", errors="replace"),
            with_answer=with_file.read_text(encoding="utf-8", errors="replace"),
        )
    except (WorkingLogError, OSError) as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
    else:
        typer.echo(f"{result['id']}\t{result['path']}")


@tests_app.command("judge", help="Record which blind answer the person preferred.")
def tests_judge(
    slug: str = typer.Argument(..., help="Liner folder name."),
    test_id: str = typer.Argument(..., help="The id from `liner test start`."),
    preferred: str = typer.Option(..., "--preferred", help="The person's choice: a, b, or same."),
    note: str = typer.Option("", "--note", help="Why the person judged it that way."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.empirical_tests import judge_blind_test
    from liner.working_log import WorkingLogError

    try:
        result = judge_blind_test(_research_project(slug, root), test_id, preferred, note=note)
    except (WorkingLogError, OSError) as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
    else:
        typer.echo(f"{result['id']}\t{result['verdict']}\t{result['question']}")
        typer.echo(f"The answer that used the liner was {result['with_liner_was']}.")


@tests_app.command("list", help="Show a liner's with-and-without tests.")
def tests_list(
    slug: str = typer.Argument(..., help="Liner folder name."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.empirical_tests import list_tests
    from liner.working_log import WorkingLogError

    try:
        result = list_tests(_research_project(slug, root))
    except WorkingLogError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
        return
    for test in result["tests"]:
        typer.echo(f"{test['id']}\t{test['verdict']}\t{test['question']}")
    if result["tests"]:
        typer.echo(f"Answers are in {result['path']}")
    else:
        typer.echo("No tests yet.")


@app.command(help="Refetch a liner's online sources and report what changed.")
def refresh(
    slug: str = typer.Argument(..., help="Liner folder name."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.refresh import RefreshError, check_sources

    project = _research_project(slug, root)
    config = load_config()
    web_handler = WebHandler(config.fetch)
    handlers: dict[str, SourceHandler] = {
        "youtube": YouTubeHandler(config.fetch),
        "web": web_handler,
    }
    try:
        report = check_sources(
            project,
            handlers=handlers,
            config=config,
            progress=None if json_output else RichProgressReporter(),
        )
    except (RefreshError, TapeValidationError) as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    finally:
        web_handler.close()
    if json_output:
        typer.echo(_json.dumps(report, ensure_ascii=False, indent=2))
    else:
        _print_refresh(report)
    counts = report["counts"]
    checked = len(report["sources"])
    if checked and counts["unreachable"] == checked:
        raise typer.Exit(code=3)
    if counts["unreachable"]:
        raise typer.Exit(code=2)


def _print_refresh(report: dict[str, Any]) -> None:
    counts = report["counts"]
    typer.echo(
        f"{counts['changed']} changed, {counts['unchanged']} unchanged, "
        f"{counts['unreachable']} unreachable, {counts['no_baseline']} with nothing to compare, "
        f"{counts['not_checked']} local not checked."
    )
    for source in report["sources"]:
        if source["status"] == "changed":
            typer.echo(
                f"changed: {source['url']} "
                f"(+{source['lines_added']} -{source['lines_removed']} lines)\n"
                f"  diff: {source['diff_path']}"
            )
        elif source["status"] != "unchanged":
            typer.echo(f"{source['status']}: {source['url']} ({source['reason']})")
    for source in report.get("reused", []):
        typer.echo(f"reused: {source['url']} ({source['reason']})")
    if report["next"]:
        typer.echo(report["next"])


@app.command(name="use", help="Point a repository's CLAUDE.md or AGENTS.md at a liner.")
def use_liner(
    slug: str = typer.Argument(..., help="Liner folder name."),
    repo: Path = typer.Option(Path("."), "--repo", help="Repository to point at the liner."),
    agent: str | None = typer.Option(None, "--agent", help="claude or codex."),
    remove: bool = typer.Option(False, "--remove", help="Remove the pointer instead."),
    yes: bool = typer.Option(False, "--yes", help="Approve the previewed change."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.library import LibraryError, default_library_root
    from liner.liner_use import UseError, apply_use, plan_use

    library = (root or default_library_root()).expanduser().resolve()
    try:
        plan = plan_use(library, slug, repo.expanduser().resolve(), agent=agent, remove=remove)
        if yes:
            apply_use(plan)
    except (UseError, LibraryError, OSError) as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    needs_approval = plan["changed"] and not yes
    if json_output:
        result = {**plan, "applied": yes and plan["changed"], "approval_required": needs_approval}
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
    elif not plan["changed"]:
        typer.echo(f"{plan['file']} already matches.")
    elif plan["deletes_file"]:
        verb = "Deleted" if yes else "Would delete"
        typer.echo(f"{verb} {plan['file']}, which holds only this pointer.")
    elif yes:
        typer.echo(f"Updated {plan['file']}.")
    else:
        typer.echo(f"Would update {plan['file']}:\n\n{plan['proposed']}\nRun again with --yes.")
    if needs_approval:
        raise typer.Exit(code=2)


@app.command(help="List every liner in your library with its job and readiness.")
def catalog(
    root: Path | None = typer.Option(
        None, "--root", help="Library folder. Defaults to LINER_DIR, then ~/liner/projects."
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.library import LibraryError, build_catalog, default_library_root

    try:
        result = build_catalog((root or default_library_root()).expanduser().resolve())
    except LibraryError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
        return
    if not result["liners"]:
        err_console.print(f"[yellow]No liners found in {result['root']}.[/]")
    for entry in result["liners"]:
        state = "stale" if entry["stale"] else entry["milestone"]
        typer.echo(
            f"{entry['slug']}\t{entry['description'] or entry['title']} "
            f"({state}, {entry['source_count']} sources{_test_label(entry['test'])})"
        )
    for broken in result["unreadable"]:
        err_console.print(f"[yellow]Skipped {broken['path']}: {broken['reason']}[/]")


def _test_label(summary: dict[str, Any] | None) -> str:
    if not summary or not summary["runs"]:
        return ""
    state = "" if summary["current"] else ", since changed"
    return f", better in {summary['better']} of {summary['runs']} tests{state}"


@app.command(help="Search the syntheses and sources in your liners.")
def search(
    query: str = typer.Argument(..., help="Words to look for."),
    liner: list[str] | None = typer.Option(
        None, "--liner", help="Only search this liner. Repeat for more than one."
    ),
    limit: int = typer.Option(10, "--limit", help="Most results to return."),
    root: Path | None = typer.Option(
        None, "--root", help="Library folder. Defaults to LINER_DIR, then ~/liner/projects."
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.library import LibraryError, default_library_root, search_library

    try:
        result = search_library(
            query,
            root=(root or default_library_root()).expanduser().resolve(),
            liners=liner,
            limit=limit,
        )
    except LibraryError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
        return
    if not result["results"]:
        err_console.print("[yellow]No matching passages.[/]")
    for hit in result["results"]:
        typer.echo(f"{hit['path']}:{hit['line']}\n  {' '.join(hit['snippet'].split())}")


@project_app.command("inspect", help="Inspect a Liner Project without changing it.")
def project_inspect(
    path: Path | None = typer.Argument(
        None,
        help="Project root or a path inside it. Defaults to the current directory.",
    ),
    project_id: str | None = typer.Option(
        None,
        "--project-id",
        help="Expected immutable Project ID. Inspection fails if it does not match.",
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    try:
        snapshot = inspect_project(path, expected_project_id=project_id)
    except ProjectInspectionError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(snapshot.to_dict(), ensure_ascii=False, indent=2))
        return
    _print_project_snapshot(snapshot)


def _print_project_snapshot(snapshot: ProjectSnapshot) -> None:
    typer.echo(f"Project ID: {snapshot.project_id or 'missing'}")
    typer.echo(f"Name: {snapshot.name}")
    typer.echo(f"Root: {snapshot.root}")
    typer.echo(f"Format: {snapshot.artifact} v{snapshot.format_version} ({snapshot.layout})")
    typer.echo(f"Revision: {snapshot.revision}")
    typer.echo(f"Compatibility: {snapshot.compatibility_state}")
    typer.echo(f"  {snapshot.compatibility_message}")
    typer.echo(f"Milestone: {snapshot.lifecycle.get('milestone', 'unknown')}")
    typer.echo(f"Inspect: {'available' if snapshot.capabilities['inspect'] else 'unavailable'}")
    typer.echo(f"Plan: {'available' if snapshot.capabilities['plan'] else 'unavailable'}")
    typer.echo(f"Apply: {'available' if snapshot.capabilities['apply'] else 'unavailable'}")
    typer.echo(f"Sources: {len(snapshot.sources)}")
    for source in snapshot.sources:
        typer.echo(f"  {source.source_id or 'missing'}  {source.type}  {source.locator}")


@project_app.command(
    "guidance",
    help="Publish the running CLI's versioned Project maintenance guidance.",
)
def project_guidance(
    path: Path = typer.Argument(..., help="Project root or a path inside it."),
    output_format: str = typer.Option(
        "markdown",
        "--format",
        help="Output format: json or markdown.",
    ),
) -> None:
    import json as _json

    if output_format not in {"json", "markdown"}:
        err_console.print("[red]Guidance format must be json or markdown.[/]")
        raise typer.Exit(code=1)
    try:
        guidance = maintenance_guidance(path)
    except ProjectInspectionError as error:
        report = FailureReport(
            code="maintenance_unavailable",
            message=str(error),
            recovery=(
                "Install or upgrade a compatible Liner CLI, then restart from "
                "`liner project inspect`; do not edit Project YAML directly.",
            ),
        )
        if output_format == "json":
            typer.echo(_json.dumps(report.to_dict(), ensure_ascii=False, indent=2))
        else:
            err_console.print(f"[red]{report.message}[/]")
            for action in report.recovery:
                err_console.print(f"  {action}")
        raise typer.Exit(code=1) from error
    if output_format == "json":
        typer.echo(_json.dumps(guidance.to_dict(), ensure_ascii=False, indent=2))
    else:
        typer.echo(guidance.to_markdown(), nl=False)


@project_app.command("plan", help="Plan a Project change without writing to it.")
def project_plan(
    path: Path = typer.Argument(..., help="Project root or a path inside it."),
    request_json: str = typer.Option(
        ..., "--request-json", help="Versioned maintenance request JSON."
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    try:
        request = _json.loads(request_json)
        if not isinstance(request, dict):
            raise ProjectInspectionError("Maintenance request must be a JSON object.")
        if request.get("contract") != MAINTENANCE_REQUEST_CONTRACT:
            raise ProjectInspectionError("Invalid maintenance request contract.")
        request_version = request.get("version")
        if type(request_version) is not int or request_version != PROJECT_CHANGE_SET_VERSION:
            raise ProjectInspectionError(
                "Unsupported maintenance request version; expected version 1."
            )
        operation = request.get("operation")
        if not isinstance(operation, dict):
            raise ProjectInspectionError("Maintenance operation must be a JSON object.")
        operation_type = operation.get("type")
        if operation_type == "project.guidance_upgrade":
            change_set = plan_project_guidance_upgrade(path)
        elif operation_type == "synthesis.review":
            change_set = plan_synthesis_review(
                path,
                str(operation.get("disposition", "")),
                content=operation.get("content"),
            )
        elif operation_type == "operating_layer.review":
            change_set = plan_operating_layer_review(
                path,
                str(operation.get("disposition", "")),
                liner_content=operation.get("liner_content"),
                skill_content=operation.get("skill_content"),
            )
        elif operation_type == "operating_layer.create":
            change_set = plan_operating_layer_create(path)
        elif operation_type == "project.rename":
            name = operation.get("name")
            if not isinstance(name, str):
                raise ProjectInspectionError("project.rename requires a string name.")
            change_set = plan_project_rename(path, name)
        elif operation_type == "project.move":
            destination = operation.get("destination")
            if not isinstance(destination, str) or not destination.strip():
                raise ProjectInspectionError("project.move requires a destination path.")
            change_set = plan_project_move(path, Path(destination))
        elif operation_type == "pointer.adapter":
            change_set = plan_pointer_adapter(
                path,
                str(operation.get("environment", "")),
                str(operation.get("action", "")),
            )
        elif operation_type == "source.add":
            source = operation.get("source")
            sources = operation.get("sources")
            if source is not None and sources is not None:
                raise ProjectInspectionError(
                    "source.add accepts either source or sources, not both."
                )
            if sources is not None:
                if not isinstance(sources, list) or not all(
                    isinstance(item, dict) for item in sources
                ):
                    raise ProjectInspectionError(
                        "source.add sources must be a list of Source mappings."
                    )
                change_set = plan_source_add_batch(path, sources)
            else:
                if not isinstance(source, dict):
                    raise ProjectInspectionError("source.add requires a Source mapping.")
                change_set = plan_source_add(path, source)
        elif operation_type == "source.update":
            changes = operation.get("changes")
            if not isinstance(changes, dict):
                raise ProjectInspectionError("source.update requires a changes mapping.")
            change_set = plan_source_update(
                path,
                str(operation.get("source_id", "")),
                changes,
            )
        elif operation_type == "source.replace":
            source = operation.get("source")
            if not isinstance(source, dict):
                raise ProjectInspectionError("source.replace requires a Source mapping.")
            change_set = plan_source_replace(
                path,
                str(operation.get("source_id", "")),
                source,
                provenance_intent=operation.get("provenance_intent"),
                provenance_reason=operation.get("provenance_reason"),
            )
        elif operation_type == "source.remove":
            change_set = plan_source_remove(
                path,
                str(operation.get("source_id", "")),
            )
        elif operation_type == "source.purge":
            change_set = plan_source_purge(
                path,
                str(operation.get("source_id", "")),
            )
        else:
            raise ProjectInspectionError(f"Unsupported maintenance operation {operation_type!r}.")
    except (_json.JSONDecodeError, ProjectInspectionError) as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False, indent=2))
        return
    _print_change_set(change_set)


@project_app.command(
    "pointer",
    help="Compile an opt-in managed AGENTS.md or CLAUDE.md pointer Change Set.",
)
def project_pointer(
    path: Path = typer.Argument(..., help="Project root or a path inside it."),
    environment: str = typer.Option(
        ..., "--environment", help="Supported pointer environment: codex or claude."
    ),
    action: str = typer.Option(
        ..., "--action", help="Explicit pointer action: install, update, or remove."
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    try:
        change_set = plan_pointer_adapter(path, environment, action)
    except ProjectInspectionError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False, indent=2))
    else:
        _print_change_set(change_set)
        typer.echo("Exact Change Set JSON:")
        typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False))


@project_app.command("rename", help="Compile a managed Project display-name Change Set.")
def project_rename(
    path: Path = typer.Argument(..., help="Project root or a path inside it."),
    name: str = typer.Option(..., "--name", help="New Project display name."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    try:
        change_set = plan_project_rename(path, name)
    except ProjectInspectionError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False, indent=2))
    else:
        _print_change_set(change_set)
        typer.echo("Exact Change Set JSON:")
        typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False))


@project_app.command("move", help="Compile an identity-preserving Project root move.")
def project_move(
    path: Path = typer.Argument(..., help="Project root or a path inside it."),
    destination: Path = typer.Option(
        ..., "--destination", help="Absent destination root on the same filesystem."
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    try:
        change_set = plan_project_move(path, destination)
    except ProjectInspectionError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False, indent=2))
    else:
        _print_change_set(change_set)
        typer.echo("Exact Change Set JSON:")
        typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False))
    err_console.print(
        "[yellow]Project move changes the authoritative root.[/] Review the exact Change Set, "
        "then pass it to `liner project apply --change-set-json '<json>' --approve "
        f"--approved-destination '{change_set.operations[0]['new_root']}'`."
    )
    raise typer.Exit(code=2)


@project_app.command("apply", help="Atomically apply a versioned Project Change Set.")
def project_apply(
    path: Path = typer.Argument(..., help="Project root or a path inside it."),
    change_set_json: str = typer.Option(
        ..., "--change-set-json", help="Versioned Change Set JSON."
    ),
    approve: bool = typer.Option(
        False,
        "--approve",
        help="Approve a structural or semantic Change Set after reviewing its preview.",
    ),
    approved_destination: Path | None = typer.Option(
        None,
        "--approved-destination",
        help="Repeat the exact reviewed destination for a Project move.",
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    try:
        raw = _json.loads(change_set_json)
        if not isinstance(raw, dict):
            raise ProjectInspectionError("Change Set must be a JSON object.")
        receipt = apply_change_set(
            path,
            ProjectChangeSet.from_dict(raw),
            approved=approve,
            approved_destination=approved_destination,
        )
    except (_json.JSONDecodeError, ProjectInspectionError) as error:
        report = FailureReport(
            code="invalid_change_set",
            message=f"Change Set validation failed: {error}",
            recovery=("Discard this Change Set and create a fresh plan.",),
        )
        if json_output:
            typer.echo(_json.dumps(report.to_dict(), ensure_ascii=False, indent=2))
        else:
            err_console.print(f"[red]{report.message}[/]")
        raise typer.Exit(code=1) from error
    except ProjectApplyError as error:
        if json_output:
            typer.echo(_json.dumps(error.report.to_dict(), ensure_ascii=False, indent=2))
        else:
            err_console.print(f"[red]{error.report.message}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(receipt.to_dict(), ensure_ascii=False, indent=2))
        return
    _print_change_receipt(receipt)


@app.command(
    name="find-sources", help="Find sources already in your library, with earlier verdicts."
)
def find_library_sources(
    query: str = typer.Argument(..., help="Words to look for."),
    limit: int = typer.Option(20, "--limit", min=1, help="Most rows to return."),
    root: Path | None = _ROOT_OPTION,
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    from liner.library import LibraryError, default_library_root, find_sources

    try:
        result = find_sources(
            query, root=(root or default_library_root()).expanduser().resolve(), limit=limit
        )
    except LibraryError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(result, ensure_ascii=False, indent=2))
        return
    if not result["sources"]:
        err_console.print("[yellow]No matching sources.[/]")
    for row in result["sources"]:
        typer.echo(f"{row['liner']}  {row['decision']}  {row['title']}  {row['url'] or ''}")
    if result["total"] > len(result["sources"]):
        err_console.print(f"Showing {len(result['sources'])} of {result['total']}.")
    err_console.print(str(result["notice"]))


@sources_app.command("add", help="Add one Source through the canonical plan/apply path.")
def sources_add(
    path: Path = typer.Argument(..., help="Project root or a path inside it."),
    source_type: str = typer.Option(..., "--type", help="Source type."),
    url: str | None = typer.Option(None, "--url", help="Remote Source URL."),
    source_path: str | None = typer.Option(None, "--path", help="Local Source path."),
    note: str | None = typer.Option(None, "--note", help="Source use note."),
    section: str | None = typer.Option(None, "--section", help="Knowledge-map section."),
    priority: str | None = typer.Option(None, "--priority", help="required or optional."),
    render: str | None = typer.Option(None, "--render", help="server or js for web Sources."),
    citation: str | None = typer.Option(None, "--citation", help="Local-file citation."),
    kind: str | None = typer.Option(None, "--kind", help="Source role."),
    content_hash: str | None = typer.Option(
        None, "--content-hash", help="Known Source content hash (sha256:<hex>)."
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    source: dict[str, object] = {"type": source_type}
    for key, value in {
        "url": url,
        "path": source_path,
        "note": note,
        "section": section,
        "priority": priority,
        "render": render,
        "citation": citation,
        "kind": kind,
        "content_hash": content_hash,
    }.items():
        if value is not None:
            source[key] = value
    try:
        change_set = plan_source_add(path, source)
    except ProjectInspectionError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if change_set.approval_required:
        if json_output:
            typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False, indent=2))
        else:
            _print_change_set(change_set)
        err_console.print(
            "[yellow]Structural identity migration requires review.[/] "
            "Apply this exact Change Set with `liner project apply --approve`."
        )
        raise typer.Exit(code=2)
    try:
        receipt = apply_change_set(path, change_set)
    except ProjectApplyError as error:
        if json_output:
            typer.echo(_json.dumps(error.report.to_dict(), ensure_ascii=False, indent=2))
        else:
            err_console.print(f"[red]{error.report.message}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(receipt.to_dict(), ensure_ascii=False, indent=2))
        return
    _print_change_receipt(receipt)


@sources_app.command(
    "update", help="Update Source metadata or locator while preserving its immutable ID."
)
def sources_update(
    path: Path = typer.Argument(..., help="Project root or a path inside it."),
    source_id: str = typer.Option(..., "--source-id", help="Immutable Source ID."),
    url: str | None = typer.Option(None, "--url", help="Remote Source URL."),
    source_path: str | None = typer.Option(None, "--path", help="Local Source path."),
    note: str | None = typer.Option(None, "--note", help="Source use note."),
    section: str | None = typer.Option(None, "--section", help="Knowledge-map section."),
    priority: str | None = typer.Option(None, "--priority", help="required or optional."),
    render: str | None = typer.Option(None, "--render", help="server or js for web Sources."),
    citation: str | None = typer.Option(None, "--citation", help="Local-file citation."),
    kind: str | None = typer.Option(None, "--kind", help="Source role."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    changes = {
        key: value
        for key, value in {
            "url": url,
            "path": source_path,
            "note": note,
            "section": section,
            "priority": priority,
            "render": render,
            "citation": citation,
            "kind": kind,
        }.items()
        if value is not None
    }
    try:
        change_set = plan_source_update(path, source_id, changes)
        receipt = apply_change_set(path, change_set)
    except ProjectInspectionError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    except ProjectApplyError as error:
        if json_output:
            typer.echo(_json.dumps(error.report.to_dict(), ensure_ascii=False, indent=2))
        else:
            err_console.print(f"[red]{error.report.message}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(receipt.to_dict(), ensure_ascii=False, indent=2))
        return
    _print_change_receipt(receipt)


@sources_app.command("replace", help="Preview a semantic Source replacement with explicit lineage.")
def sources_replace(
    path: Path = typer.Argument(..., help="Project root or a path inside it."),
    source_id: str = typer.Option(..., "--source-id", help="Predecessor Source ID."),
    source_type: str = typer.Option(..., "--type", help="Replacement Source type."),
    url: str | None = typer.Option(None, "--url", help="Remote Source URL."),
    source_path: str | None = typer.Option(None, "--path", help="Local Source path."),
    note: str | None = typer.Option(None, "--note", help="Source use note."),
    section: str | None = typer.Option(None, "--section", help="Knowledge-map section."),
    priority: str | None = typer.Option(None, "--priority", help="required or optional."),
    render: str | None = typer.Option(None, "--render", help="server or js for web Sources."),
    citation: str | None = typer.Option(None, "--citation", help="Local-file citation."),
    kind: str | None = typer.Option(None, "--kind", help="Source role."),
    content_hash: str | None = typer.Option(
        None, "--content-hash", help="Known Source content hash (sha256:<hex>)."
    ),
    provenance_intent: str | None = typer.Option(
        None, "--provenance-intent", help="Use distinct for same-content provenance."
    ),
    provenance_reason: str | None = typer.Option(
        None, "--provenance-reason", help="Private reason for distinct provenance."
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    source: dict[str, object] = {"type": source_type}
    for key, value in {
        "url": url,
        "path": source_path,
        "note": note,
        "section": section,
        "priority": priority,
        "render": render,
        "citation": citation,
        "kind": kind,
        "content_hash": content_hash,
    }.items():
        if value is not None:
            source[key] = value
    try:
        change_set = plan_source_replace(
            path,
            source_id,
            source,
            provenance_intent=provenance_intent,
            provenance_reason=provenance_reason,
        )
    except ProjectInspectionError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if change_set.approval_required:
        if json_output:
            typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False, indent=2))
        else:
            _print_change_set(change_set)
            typer.echo("Exact Change Set JSON:")
            typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False))
        err_console.print(
            "[yellow]Semantic Source replacement requires review.[/] "
            "Pass the exact JSON above to `liner project apply --change-set-json "
            "'<json>' --approve`."
        )
        raise typer.Exit(code=2)
    try:
        receipt = apply_change_set(path, change_set)
    except ProjectApplyError as error:
        if json_output:
            typer.echo(_json.dumps(error.report.to_dict(), ensure_ascii=False, indent=2))
        else:
            err_console.print(f"[red]{error.report.message}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(receipt.to_dict(), ensure_ascii=False, indent=2))
        return
    _print_change_receipt(receipt)


@sources_app.command(
    "remove", help="Preview retention-first Source detachment from the active corpus."
)
def sources_remove(
    path: Path = typer.Argument(..., help="Project root or a path inside it."),
    source_id: str = typer.Option(..., "--source-id", help="Active Source ID."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    try:
        change_set = plan_source_remove(path, source_id)
    except ProjectInspectionError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False, indent=2))
    else:
        _print_change_set(change_set)
        typer.echo("Exact Change Set JSON:")
        typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False))
    err_console.print(
        "[yellow]Source removal retains captures but changes the active corpus.[/] "
        "Pass the exact JSON to `liner project apply --change-set-json '<json>' --approve`."
    )
    raise typer.Exit(code=2)


@sources_app.command(
    "purge", help="Preview irreversible deletion of one retained Source and its captures."
)
def sources_purge(
    path: Path = typer.Argument(..., help="Project root or a path inside it."),
    source_id: str = typer.Option(..., "--source-id", help="Detached Source ID."),
    json_output: bool = typer.Option(False, "--json", help="Emit versioned JSON."),
) -> None:
    import json as _json

    try:
        change_set = plan_source_purge(path, source_id)
    except ProjectInspectionError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False, indent=2))
    else:
        _print_change_set(change_set)
        typer.echo("Exact Change Set JSON:")
        typer.echo(_json.dumps(change_set.to_dict(), ensure_ascii=False))
    err_console.print(
        "[red]Source purge permanently deletes every listed retained artifact.[/] "
        "Pass the exact JSON to `liner project apply --change-set-json '<json>' --approve`."
    )
    raise typer.Exit(code=2)


def _print_change_set(change_set: ProjectChangeSet) -> None:
    import json as _json

    typer.echo("Project Change Set")
    typer.echo(f"ID: {change_set.change_set_id}")
    typer.echo(f"Project ID: {change_set.project_id}")
    typer.echo(f"Expected revision: {change_set.expected_revision}")
    typer.echo(f"Expected content hash: {change_set.expected_content_hash}")
    typer.echo(f"Risk: {change_set.risk}")
    typer.echo(f"Approval required: {'yes' if change_set.approval_required else 'no'}")
    typer.echo("Operations:")
    for operation in change_set.operations:
        detail = ""
        source = operation.get("source")
        if isinstance(source, dict):
            detail = f" {_json.dumps(source, ensure_ascii=False, sort_keys=True)}"
        elif operation.get("source_id"):
            detail = f" source_id={operation['source_id']}"
        typer.echo(f"  - {operation.get('type', 'unknown')}{detail}")
    typer.echo("File effects:")
    for effect, paths in change_set.file_effects.items():
        for path in paths:
            typer.echo(f"  - {effect}: {path}")
    typer.echo("Validation:")
    for validation in change_set.validation:
        typer.echo(f"  - {validation}")


def _print_change_receipt(receipt: object) -> None:
    payload = receipt.to_dict()  # type: ignore[attr-defined]
    typer.echo("Project Change Receipt")
    typer.echo(f"Receipt ID: {payload['receipt_id']}")
    typer.echo(f"Project ID: {payload['project_id']}")
    typer.echo(f"Revision: {payload['after']['revision']}")
    typer.echo(f"Synthesis: {payload['synthesis_disposition']}")


@skills_app.command("list", help="List locally installed skills Liner can import as sources.")
def skills_list(
    json_output: bool = typer.Option(False, "--json", help="Emit JSON for programmatic use."),
) -> None:
    import json as _json

    from liner.handlers.skill import discover_skills

    records = [
        {"name": s.name, "path": str(s.path), "description": s.description}
        for s in discover_skills()
    ]
    if json_output:
        typer.echo(_json.dumps(records, ensure_ascii=False))
        return
    if not records:
        err_console.print("[yellow]No installed skills found.[/]")
        return
    for rec in records:
        desc = f"\t{rec['description']}" if rec["description"] else ""
        typer.echo(f"{rec['name']}\t{rec['path']}{desc}")


@cache_app.command("info", help="Show cache size and entry count.")
def cache_info() -> None:
    config = load_config()
    with SourceCache(config.cache_path) as cache:
        info = cache.info()
    typer.echo(f"path: {info.path}")
    typer.echo(f"entries: {info.entry_count}")
    typer.echo(f"size: {info.size_bytes} bytes")


@cache_app.command("list", help="List cached source URLs.")
def cache_list(
    limit: int = typer.Option(100, "--limit", help="Maximum number of entries to list."),
    offset: int = typer.Option(0, "--offset", help="Pagination offset."),
) -> None:
    config = load_config()
    with SourceCache(config.cache_path) as cache:
        entries = cache.list(limit=limit, offset=offset)
    if not entries:
        err_console.print("[yellow]Cache is empty.[/]")
        return
    for entry in entries:
        typer.echo(
            f"{entry.source_type}\t{entry.url}\tfetched={entry.fetched_at}\texpires={entry.expires_at}"
        )


@cache_app.command("show", help="Print a cached source's body to stdout.")
def cache_show(url: str) -> None:
    config = load_config()
    with SourceCache(config.cache_path) as cache:
        content = cache.get_raw(url)
    if content is None:
        err_console.print(f"[yellow]No cache entry for[/] {url}")
        raise typer.Exit(code=1)
    typer.echo(f"# {content.title}")
    typer.echo(f"URL: {content.url}")
    if content.author:
        typer.echo(f"Author: {content.author}")
    typer.echo(f"Fetched: {content.fetched_at}")
    typer.echo("")
    typer.echo(content.body)


@cache_app.command("clear", help="Remove all cache entries.")
def cache_clear(
    yes: bool = typer.Option(False, "--yes", help="Skip confirmation prompt."),
) -> None:
    if not yes:
        confirm = typer.confirm("Clear all cached entries?", default=False)
        if not confirm:
            raise typer.Exit(code=1)
    config = load_config()
    with SourceCache(config.cache_path) as cache:
        removed = cache.clear()
    err_console.print(f"[green]Cleared {removed} entries.[/]")


@cache_app.command("purge", help="Remove a single cached URL.")
def cache_purge(url: str) -> None:
    config = load_config()
    with SourceCache(config.cache_path) as cache:
        removed = cache.purge(url)
    if removed:
        err_console.print(f"[green]Purged[/] {url}")
    else:
        err_console.print(f"[yellow]No cache entry for[/] {url}")
        raise typer.Exit(code=1)


@app.command(
    name="manifest",
    help="Aggregate .liner-runs/* into process.json (tokens, tool calls, fetches, cost).",
)
def manifest_cmd(
    folder: Path = typer.Argument(..., help="Liner project folder."),
    json_output: bool = typer.Option(
        False,
        "--json",
        help="Print the manifest to stdout instead of (in addition to) writing the file.",
    ),
    no_write: bool = typer.Option(
        False,
        "--no-write",
        help="Skip writing process.json; useful with --json for piping.",
    ),
) -> None:
    """Walk `.liner-runs/` and write a roll-up to `<folder>/process.json`.

    Always overwrites — the file is purely derived from the run logs. Run
    after every agent execution or whenever the curator wants a fresh tally.
    """
    import json as _json

    if not folder.is_dir():
        err_console.print(f"[red]Not a directory:[/] {folder}")
        raise typer.Exit(code=1)
    project = ProjectFolder(folder)
    manifest = build_manifest(project.corpus_path)
    payload = manifest.to_dict()
    if not no_write:
        target = write_manifest(project.corpus_path, manifest)
        err_console.print(f"[green]Wrote[/] {target}")
    if json_output:
        typer.echo(_json.dumps(payload, ensure_ascii=False, indent=2))


@app.command(
    name="status",
    help="Show a per-project summary: tokens, tool calls, fetches, cost, runs.",
)
def status_cmd(
    folder: Path = typer.Argument(..., help="Liner project folder."),
    refresh: bool = typer.Option(
        True,
        "--refresh/--no-refresh",
        help="Rebuild process.json from .liner-runs/ before rendering.",
    ),
    no_write: bool = typer.Option(
        False,
        "--no-write",
        help="When refreshing, skip writing process.json; useful with --json for TUI/status consumers.",
    ),
    status_only: bool = typer.Option(
        False,
        "--status-only",
        help="Refresh only the Status Snapshot in liner.yaml; do not write process.json.",
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit JSON instead of a table."),
) -> None:
    """Pretty-print the manifest. Regenerates by default so totals are fresh."""
    import json as _json

    if not folder.is_dir():
        err_console.print(f"[red]Not a directory:[/] {folder}")
        raise typer.Exit(code=1)

    project = ProjectFolder(folder)
    corpus = project.corpus_path

    if refresh:
        manifest = build_manifest(corpus)
        if not no_write and not status_only:
            write_manifest(corpus, manifest)
        payload = manifest.to_dict()
    else:
        stored_payload = read_manifest(corpus)
        if stored_payload is None:
            err_console.print(
                f"[yellow]No process.json in[/] {corpus} — run `liner manifest {folder}` first, "
                "or rerun this command without --no-refresh."
            )
            raise typer.Exit(code=1)
        payload = stored_payload

    status_payload = build_status_payload(project, payload)
    from liner.research_checks import research_funnel

    status_payload["funnel"] = research_funnel(corpus)
    if not no_write:
        status_payload["status_snapshot"] = refresh_status_snapshot(project)
    payload.update(status_payload)

    if json_output:
        typer.echo(_json.dumps(payload, ensure_ascii=False, indent=2))
        return
    _print_status(payload)


def _print_status(payload: dict) -> None:  # type: ignore[type-arg]
    """Render the manifest as a Rich terminal report."""
    from rich import box
    from rich.table import Table

    console = Console()
    mixtape = payload.get("mixtape", {})
    totals = payload.get("totals", {})
    title = mixtape.get("title") or mixtape.get("name") or "(untitled)"
    console.print()
    console.print(f"[bold cyan]{title}[/]  [dim]({mixtape.get('path', '')})[/]")
    if mixtape.get("jtbd"):
        console.print(f"[dim]JTBD:[/] {mixtape['jtbd']}")
    console.print()

    summary = Table(box=box.SIMPLE, show_header=False, pad_edge=False)
    summary.add_column(style="dim", no_wrap=True)
    summary.add_column()
    funnel = payload.get("funnel") or {}
    if funnel.get("candidates"):
        # Agent builds leave no app run logs, so this is what shows their work.
        searches = funnel.get("searches")
        summary.add_row(
            "Searches",
            f"{searches} with {funnel.get('search_results') or 0:,} results, reported by the agent"
            if searches
            else "— (no search log)",
        )
        for label, key in (
            ("Candidates", "candidates"),
            ("Pages read", "pages_read"),
            ("Evaluated", "evaluated"),
            ("Kept", "kept"),
            ("Final sources", "final"),
        ):
            summary.add_row(label, str(funnel.get(key, 0)))
    if totals.get("runs") or not funnel.get("candidates"):
        _add_run_totals(summary, payload)
    console.print(summary)
    _print_runs(payload)


def _add_run_totals(summary: Any, payload: dict) -> None:  # type: ignore[type-arg]
    totals = payload.get("totals", {})
    tokens = totals.get("tokens", {})
    summary.add_row("Runs", str(totals.get("runs", 0)))
    summary.add_row("Tool calls", str(totals.get("tool_calls", 0)))
    summary.add_row("Fetches", str(totals.get("fetches", 0)))
    cost = totals.get("cost_usd")
    summary.add_row("Cost (USD)", f"${cost:.4f}" if isinstance(cost, (int, float)) else "—")
    summary.add_row(
        "Tokens",
        f"in {tokens.get('input', 0):,} · out {tokens.get('output', 0):,} · "
        f"cache_read {tokens.get('cache_read', 0):,} · cache_create {tokens.get('cache_create', 0):,}",
    )
    summary.add_row("Agents", ", ".join(payload.get("agents_used") or []) or "—")
    summary.add_row("Models", ", ".join(payload.get("models_used") or []) or "—")


def _print_runs(payload: dict) -> None:  # type: ignore[type-arg]
    from rich import box
    from rich.table import Table

    console = Console()
    runs = payload.get("runs") or []
    if runs:
        rt = Table(title="Runs", box=box.SIMPLE_HEAD, header_style="bold")
        rt.add_column("Task")
        rt.add_column("Agent / Model", overflow="fold")
        rt.add_column("Started", overflow="fold")
        rt.add_column("Duration", justify="right")
        rt.add_column("Turns", justify="right")
        rt.add_column("Tool calls", justify="right")
        rt.add_column("Fetches", justify="right")
        rt.add_column("Tokens (in/out)", justify="right")
        rt.add_column("Cost", justify="right")
        rt.add_column("Exit", justify="right")
        for r in runs:
            t = r.get("tokens") or {}
            d = r.get("duration_s")
            c = r.get("cost_usd")
            rt.add_row(
                r.get("task_label", "?"),
                f"{r.get('agent', '?')} / {r.get('model', '?')}",
                (r.get("started_at") or "")[:19].replace("T", " "),
                f"{d:.1f}s" if isinstance(d, (int, float)) else "—",
                str(r.get("num_turns") or 0),
                str(sum((r.get("tools") or {}).values())),
                str(len(r.get("fetches") or [])),
                f"{t.get('input', 0):,}/{t.get('output', 0):,}",
                f"${c:.3f}" if isinstance(c, (int, float)) else "—",
                str(r.get("exit_code") if r.get("exit_code") is not None else "—"),
            )
        console.print(rt)

    domains = payload.get("domains") or []
    if domains:
        dt = Table(title="Domains fetched", box=box.SIMPLE_HEAD, header_style="bold")
        dt.add_column("Domain")
        dt.add_column("Fetches", justify="right")
        for d in domains:
            dt.add_row(d.get("domain", "(unknown)"), str(d.get("count", 0)))
        console.print(dt)


def _split_csv(value: str | None) -> set[str] | None:
    if value is None:
        return None
    return {part.strip() for part in value.split(",") if part.strip()}


if __name__ == "__main__":  # pragma: no cover
    app()
