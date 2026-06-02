from __future__ import annotations

import shutil
import sys
from dataclasses import replace
from pathlib import Path

import httpx
import typer
from rich.console import Console

from liner import __version__
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
from liner.manifest import build_manifest, read_manifest, write_manifest
from liner.playwright_env import configure_frozen_playwright_cache
from liner.project import ProjectFolder, init_project
from liner.share import ShareOptions, pack, unpack
from liner.tape import TapeValidationError, load_tape
from liner.types import CompileResult, Tape

app = typer.Typer(
    help="Liner — build mixtape project folders from curated source recipes.",
    no_args_is_help=True,
    add_completion=False,
)
cache_app = typer.Typer(help="Cache management commands.", no_args_is_help=True)
app.add_typer(cache_app, name="cache")

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


@app.command(help="Scaffold a new mixtape project folder.")
def init(
    path: Path = typer.Argument(
        ...,
        help="Folder to create (or populate). Pass a slug to create ./<slug>/.",
    ),
    force: bool = typer.Option(False, "--force", help="Overwrite existing files in the folder."),
) -> None:
    try:
        project = init_project(path, force=force)
    except FileExistsError as e:
        err_console.print(f"[red]{e}[/]")
        raise typer.Exit(code=1) from e

    err_console.print(f"[green]Created project folder[/] {project.path}")
    err_console.print("Next steps:")
    err_console.print(f"  1. Edit [bold]{project.tape_path}[/] with your sources")
    err_console.print(f"  2. Write your synthesis in [bold]{project.synthesis_path}[/]")
    err_console.print(f"  3. Run [bold]liner compile {project.path}[/]")


@app.command(
    help=(
        "Clone a project's JTBD into a fresh folder so the same input can be "
        "run through a new pipeline. Used to A/B-test methodology changes."
    ),
)
def replay(
    source: Path = typer.Argument(
        ...,
        exists=True,
        file_okay=False,
        dir_okay=True,
        readable=True,
        help="Existing project folder to clone the JTBD + clarifications from.",
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
    """Selectively clone tape.yaml (JTBD + clarifications + meta) into a new
    project folder so the curator can run the same input through a fresh
    pipeline. Working artifacts, synthesis, and sources are NOT copied —
    they regenerate from scratch. The new tape records `parent: <source>`
    so the two compiled outputs can be compared later.
    """
    from liner.tape import load_tape

    try:
        src_tape = load_tape(source / "tape.yaml")
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
    err_console.print(f"  parent: [dim]{source.resolve()}[/]")
    err_console.print("Next steps:")
    err_console.print(f"  1. Open [bold]{project.path}[/] in the TUI (or run phases manually)")
    err_console.print("  2. Run all phases; the cloned JTBD + clarifications are pre-filled")
    err_console.print(f"  3. After compile, compare the result against {source.name}")


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


@app.command(help="Compile a project folder: fetch sources, write MIXTAPE.md and sources/.")
def compile(  # noqa: A001 - command name matches the v1 surface
    folder: Path = typer.Argument(
        ...,
        exists=True,
        file_okay=False,
        dir_okay=True,
        readable=True,
        help="Path to the mixtape project folder.",
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
        err_console.print(f"[red]No tape.yaml in {folder}.[/] Run `liner init {folder}` first.")
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
        except MissingSynthesisError as e:
            err_console.print(f"[red]{e}[/]")
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


@app.command(help="Pack a mixtape project folder into a .mixtape zip for sharing.")
def share(
    folder: Path = typer.Argument(
        ...,
        exists=True,
        file_okay=False,
        dir_okay=True,
        readable=True,
        help="Path to the mixtape project folder.",
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
        help="Exclude personal/ from the archive before sharing private local files.",
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

    # Public-sharing soft warning.
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
            f"local_file source{'s' if local_file_count != 1 else ''}. Use "
            "[bold]--no-personal[/] before sharing publicly if those files are private."
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


@app.command(name="list", help="List mixtape project folders in the current directory.")
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
        if (child / "tape.yaml").exists():
            candidates.append(child)
        elif recursive:
            for grand in sorted(p for p in child.iterdir() if p.is_dir()):
                if (grand / "tape.yaml").exists():
                    candidates.append(grand)

    records: list[dict[str, object]] = []
    for project_path in candidates:
        try:
            tape = load_tape(project_path / "tape.yaml")
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
                "modified_iso": _dt.fromtimestamp(project_path.stat().st_mtime).isoformat(),
            }
        )

    if json_output:
        typer.echo(_json.dumps(records, ensure_ascii=False))
        return

    if not records:
        err_console.print("[yellow]No mixtape project folders found.[/]")
        return
    for rec in records:
        modified = _dt.fromisoformat(str(rec["modified_iso"])).strftime("%Y-%m-%d %H:%M")
        mode = rec["mode"] or "—"
        typer.echo(
            f"{rec['name']}\t{rec['title']} "
            f"({rec['source_count']} sources, mode={mode}, modified {modified})"
        )


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
    folder: Path = typer.Argument(..., help="Mixtape project folder."),
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
    manifest = build_manifest(folder)
    payload = manifest.to_dict()
    if not no_write:
        target = write_manifest(folder, manifest)
        err_console.print(f"[green]Wrote[/] {target}")
    if json_output:
        typer.echo(_json.dumps(payload, ensure_ascii=False, indent=2))


@app.command(
    name="status",
    help="Show a per-mixtape summary: tokens, tool calls, fetches, cost, runs.",
)
def status_cmd(
    folder: Path = typer.Argument(..., help="Mixtape project folder."),
    refresh: bool = typer.Option(
        True,
        "--refresh/--no-refresh",
        help="Rebuild process.json from .liner-runs/ before rendering.",
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit JSON instead of a table."),
) -> None:
    """Pretty-print the manifest. Regenerates by default so totals are fresh."""
    import json as _json

    if not folder.is_dir():
        err_console.print(f"[red]Not a directory:[/] {folder}")
        raise typer.Exit(code=1)

    if refresh:
        manifest = build_manifest(folder)
        write_manifest(folder, manifest)
        payload = manifest.to_dict()
    else:
        payload = read_manifest(folder)
        if payload is None:
            err_console.print(
                f"[yellow]No process.json in[/] {folder} — run `liner manifest {folder}` first, "
                "or rerun this command without --no-refresh."
            )
            raise typer.Exit(code=1)

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
