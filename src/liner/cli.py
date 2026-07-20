from __future__ import annotations

import shutil
import sys
from dataclasses import replace
from pathlib import Path

import httpx
import typer
from rich.console import Console

from liner import __version__
from liner.agent_adapters import (
    AdapterError,
    apply_agent_adapter_plan,
    inspect_agent_adapter,
    plan_agent_adapter,
)
from liner.cache import SourceCache
from liner.cli_progress import RichProgressReporter
from liner.compile import MissingSynthesisError, compile_project
from liner.config import load_config
from liner.handlers.base import SourceHandler
from liner.handlers.web import WebHandler
from liner.handlers.youtube import YouTubeHandler
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
)
from liner.share import ShareOptions, pack, unpack
from liner.status import build_status_payload
from liner.tape import TapeValidationError, load_tape
from liner.types import CompileResult, Tape

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
    import json as _json

    try:
        inspection = inspect_agent_adapter(environment, home)
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
) -> None:
    import json as _json

    try:
        plan = plan_agent_adapter(action, environment, home)
        if not approved and plan.approval_required:
            typer.echo(_json.dumps(plan.to_dict(), ensure_ascii=False, indent=2))
            raise typer.Exit(code=2)
        receipt = apply_agent_adapter_plan(plan, approved=approved)
    except AdapterError as error:
        err_console.print(f"[red]{error}[/]")
        raise typer.Exit(code=1) from error
    if json_output:
        typer.echo(_json.dumps(receipt.to_dict(), ensure_ascii=False, indent=2))
    else:
        typer.echo(
            f"{receipt.action} {receipt.environment} adapter at {receipt.target}; "
            f"receipt: {receipt.receipt_path}"
        )


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
        help="Set the project mode: quick or methodology.",
    ),
    jtbd: str | None = typer.Option(None, "--jtbd", help="Set the job-to-be-done."),
    title: str | None = typer.Option(None, "--title", help="Set the mixtape title."),
    description: str | None = typer.Option(
        None, "--description", help="Set the mixtape description."
    ),
    curator: str | None = typer.Option(None, "--curator", help="Set the curator name."),
    tui_construction: bool = typer.Option(
        False,
        "--tui-construction",
        hidden=True,
        help="Prepare an empty one-shot first-run assembly boundary for the Go TUI.",
    ),
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
    if tui_construction:
        _prepare_tui_construction(project)

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


def _prepare_tui_construction(project: ProjectFolder) -> None:
    import yaml

    raw = yaml.safe_load(project.tape_path.read_text(encoding="utf-8")) or {}
    if not isinstance(raw, dict):
        raw = {}
    raw["sources"] = []
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
    err_console.print(f"  1. Open [bold]{project.path}[/] in the TUI")
    err_console.print("  2. Review the pre-filled Job to Be Done and Clarify Job answers")
    err_console.print(f"  3. Build the Mixtape, then compare its outputs with {source.name}")


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
        project = unpack(archive, destination)
    except (FileNotFoundError, ValueError) as e:
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
    }

    refetched = 0
    skipped = 0
    failed = 0
    with SourceCache(config.cache_path) as cache:
        for spec in tape.sources:
            if cache.get(spec.url) is not None:
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
            response = httpx.get(source, follow_redirects=True, timeout=20.0)
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
    it isn't already there) and downloads the headless Chromium binary
    Playwright drives (~150MB total). Idempotent — safe to re-run.
    """
    configure_frozen_playwright_cache()

    # 1. Decide whether we need to install the playwright Python package.
    try:
        import playwright  # noqa: F401

        need_install = False
    except ImportError:
        need_install = True

    actions: list[str] = []
    if need_install:
        actions.append("Install the `playwright` Python package (~50MB)")
    actions.append("Download the Chromium binary (~150MB on first run; subsequent runs are no-ops)")

    err_console.print("[bold]liner setup-js[/] will:")
    for i, action in enumerate(actions, start=1):
        err_console.print(f"  {i}. {action}")

    if not yes:
        ok = typer.confirm("Continue?", default=True)
        if not ok:
            err_console.print("[yellow]Cancelled.[/]")
            raise typer.Exit(code=1)

    if need_install:
        if _is_frozen_binary():
            err_console.print(
                "[red]This bundled Liner binary was built without the Playwright Python "
                "package.[/]\n"
                "Publish the platform package with Playwright included, then re-run:\n"
                "  liner setup-js"
            )
            raise typer.Exit(code=1)
        _install_playwright_package()

    # 2. Download Chromium. Normal Python installs use sys.executable -m so
    #    we hit the same venv — after pipx inject the freshly-installed
    #    playwright is available in a subprocess, even though the running
    #    process can't import it. Frozen platform binaries can't use
    #    `sys.executable -m`, so they run Playwright's entrypoint in-process.
    err_console.print("[bold]Downloading Chromium…[/]")
    returncode = _run_playwright_install_chromium()
    if returncode != 0:
        err_console.print(f"[red]playwright install chromium exited with code {returncode}.[/]")
        raise typer.Exit(code=returncode)
    err_console.print("[green]Chromium is ready.[/] `render: js` sources will now work.")

    # 3. Friendly nudge if we just installed in a non-pipx environment so the
    #    user knows the package is in *their* python and not contained.
    if need_install and not _is_pipx_venv():
        err_console.print(
            "[dim]Note: playwright was installed into your active Python "
            "environment. If you uninstall liner later, run `pip uninstall playwright` "
            "to remove the package; the Chromium binary lives in Playwright's own cache.[/]"
        )


def _run_playwright_install_chromium() -> int:
    import subprocess
    import sys

    if not _is_frozen_binary():
        proc = subprocess.run(
            [sys.executable, "-m", "playwright", "install", "chromium"],
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
        sys.argv = ["playwright", "install", "chromium"]
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


def _install_playwright_package() -> None:
    """Install the `playwright` package into the environment liner runs from.

    Prefers `pipx inject` (the right tool for pipx-managed installs, and what
    every recent pipx tutorial recommends). Falls back to `python -m pip` for
    non-pipx installs.
    """
    import shutil
    import subprocess
    import sys

    if _is_pipx_venv():
        pipx = shutil.which("pipx")
        if pipx is None:
            err_console.print(
                "[red]Detected a pipx-managed linersh install, but `pipx` is not on PATH.[/]\n"
                "Install pipx (e.g. `brew install pipx`) and re-run, or install playwright manually:\n"
                "  pipx inject linersh playwright"
            )
            raise typer.Exit(code=1)
        err_console.print("[bold]Installing playwright via `pipx inject linersh playwright`…[/]")
        proc = subprocess.run([pipx, "inject", "linersh", "playwright"], check=False)
        if proc.returncode != 0:
            err_console.print(
                f"[red]pipx inject failed with code {proc.returncode}.[/] "
                "Try running it manually for a clearer error."
            )
            raise typer.Exit(code=proc.returncode)
        return

    # Non-pipx install — fall back to pip in the current python env.
    err_console.print("[bold]Installing playwright via `pip install playwright`…[/]")
    proc = subprocess.run(
        [sys.executable, "-m", "pip", "install", "playwright"],
        check=False,
    )
    if proc.returncode != 0:
        err_console.print(
            f"[red]pip install playwright failed with code {proc.returncode}.[/]\n"
            "Install it manually:\n"
            f"  {sys.executable} -m pip install playwright"
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
    tokens = totals.get("tokens", {})
    title = mixtape.get("title") or mixtape.get("name") or "(untitled)"
    console.print()
    console.print(f"[bold cyan]{title}[/]  [dim]({mixtape.get('path', '')})[/]")
    if mixtape.get("jtbd"):
        console.print(f"[dim]JTBD:[/] {mixtape['jtbd']}")
    console.print()

    summary = Table(box=box.SIMPLE, show_header=False, pad_edge=False)
    summary.add_column(style="dim", no_wrap=True)
    summary.add_column()
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
    console.print(summary)

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
