from __future__ import annotations

import contextlib
from dataclasses import replace
from datetime import UTC, datetime
from datetime import datetime as _dt
from urllib.parse import urlparse

from liner.agent_fetch_cache import AgentFetch, build_agent_fetch_cache, title_from_summary
from liner.cache import SourceCache
from liner.config import Config
from liner.events import NullProgressReporter, ProgressReporter
from liner.handlers.base import (
    HandlerHardFailure,
    HandlerSoftFailure,
    JsRenderingRequired,
    SourceHandler,
)
from liner.handlers.html_extraction import (
    MIN_USEFUL_BODY_CHARS,
    looks_like_cookie_notice_only,
)
from liner.output.mixtape import write_mixtape, written_source_paths
from liner.project import ProjectFolder
from liner.tape import load_tape
from liner.types import (
    CompiledSource,
    CompileResult,
    CompileWarning,
    SourceContent,
    SourceSpec,
    Tape,
)


class MissingSynthesisError(FileNotFoundError):
    """Raised when `liner compile` runs against a folder without synthesis.md."""


def _content_from_agent_fetch(spec: SourceSpec, fetch: AgentFetch) -> SourceContent:
    """Build a SourceContent backed by a captured agent WebFetch summary.

    Tagged with `metadata.extraction == "agent-summary"` so the MIXTAPE.md
    renderer can flag this entry as a summary rather than a fresh extraction.
    """
    title = title_from_summary(fetch.body, fallback=spec.url or "(captured summary)")
    return SourceContent(
        title=title,
        url=spec.url or "",
        body=fetch.body,
        fetched_at=fetch.captured_at or _dt.utcnow().isoformat() + "Z",
        author=None,
        published_at=None,
        metadata={"extraction": "agent-summary", "captured_from_run": fetch.run_path},
    )


def compile_tape(
    tape: Tape,
    *,
    cache: SourceCache | None,
    handlers: dict[str, SourceHandler],
    config: Config,
    include_sections: set[str] | None = None,
    exclude_sections: set[str] | None = None,
    skip_optional: bool = False,
    max_transcript_length: int | None = None,
    no_cache: bool = False,
    progress: ProgressReporter | None = None,
    agent_fetch_cache: dict[str, AgentFetch] | None = None,
) -> CompileResult:
    """Fetch every selected source and return an in-memory CompileResult.

    Does not write to disk. Use `compile_project` when you want a full mixtape
    folder on disk; this function is the fetch orchestrator used internally and
    in tests.
    """
    reporter = progress or NullProgressReporter()
    selected = _filter_sources(
        tape.sources,
        include_sections=include_sections,
        exclude_sections=exclude_sections,
        skip_optional=skip_optional,
    )

    reporter.on_start(len(selected))

    results: list[CompiledSource] = []
    warnings: list[CompileWarning] = []
    summary_cache: dict[str, AgentFetch] = agent_fetch_cache or {}

    def try_summary_fallback(spec: SourceSpec, identifier: str, original_error: str) -> bool:
        """Last-ditch recovery: if the agent fetched this URL during research,
        materialize that summary as the source content. Appends the
        success/failure to results+warnings; returns True if recovered."""
        if not spec.url:
            return False
        fetch = summary_cache.get(spec.url)
        if fetch is None:
            return False
        content = _content_from_agent_fetch(spec, fetch)
        content = _maybe_truncate(content, max_transcript_length)
        warnings.append(
            CompileWarning(
                url=identifier,
                message=(
                    f"fetch failed for {identifier} ({original_error}); "
                    "falling back to the summary the agent captured during research"
                ),
            )
        )
        item = CompiledSource(spec=spec, content=content, cached=False)
        results.append(item)
        reporter.on_source_done(item)
        return True

    for spec in selected:
        reporter.on_source_start(spec)
        handler_key = _handler_key(spec)
        # local_file sources are local-by-definition and the file content may
        # change underneath us; bypass cache lookup and persist entirely.
        cacheable = spec.type != "local_file"
        identifier = _source_identifier(spec)

        cached_content: SourceContent | None = None
        if cacheable and cache is not None and not no_cache:
            cached_content = cache.get(spec.url)
            if cached_content is not None and not _cached_content_is_usable(spec, cached_content):
                cache.purge(spec.url)
                cached_content = None

        if cached_content is not None:
            content = _maybe_truncate(cached_content, max_transcript_length)
            item = CompiledSource(spec=spec, content=content, cached=True)
            results.append(item)
            reporter.on_source_done(item)
            continue

        handler = handlers.get(handler_key)
        if handler is None:
            if try_summary_fallback(spec, identifier, f"no handler registered for {handler_key!r}"):
                continue
            w = CompileWarning(
                url=identifier,
                message=f"No handler registered for source key {handler_key!r}.",
                severity="error",
            )
            warnings.append(w)
            item_failed = CompiledSource(spec=spec, content=None, cached=False)
            results.append(item_failed)
            reporter.on_source_failed(spec, w)
            continue

        is_soft_failure = False
        try:
            content = handler.fetch(spec)
        except HandlerSoftFailure as soft:
            content = soft.content
            is_soft_failure = True
            warnings.append(CompileWarning(url=identifier, message=str(soft)))
        except JsRenderingRequired as js_required:
            # The server handler hit a JS-only page. If a web_js handler is
            # registered AND the source didn't explicitly ask for server-only
            # rendering, retry through Playwright. This is the auto-fallback
            # path — most users won't need to set `render: js` themselves.
            fallback_handler = handlers.get("web_js")
            if fallback_handler is None or spec.render == "server":
                if try_summary_fallback(spec, identifier, str(js_required)):
                    continue
                if fallback_handler is None and spec.render is None:
                    # [js] extra isn't installed. Surface the actionable hint.
                    msg = f"{str(js_required)} Install JS-rendering support: liner setup-js"
                else:
                    msg = str(js_required)
                w = CompileWarning(url=identifier, message=msg, severity="error")
                warnings.append(w)
                item_failed = CompiledSource(spec=spec, content=None, cached=False)
                results.append(item_failed)
                reporter.on_source_failed(spec, w)
                continue
            # Auto-fallback: emit a soft warning so the curator sees the cost
            # and can decide to declare `render: js` explicitly if they want.
            warnings.append(
                CompileWarning(
                    url=identifier,
                    message=f"server-rendered fetch hit a JS stub; auto-fell back to render: js for {identifier}",
                )
            )
            try:
                content = fallback_handler.fetch(spec)
            except HandlerSoftFailure as soft:
                content = soft.content
                is_soft_failure = True
                warnings.append(CompileWarning(url=identifier, message=str(soft)))
            except HandlerHardFailure as hard:
                if try_summary_fallback(spec, identifier, str(hard)):
                    continue
                w = CompileWarning(url=identifier, message=str(hard), severity="error")
                warnings.append(w)
                item_failed = CompiledSource(spec=spec, content=None, cached=False)
                results.append(item_failed)
                reporter.on_source_failed(spec, w)
                continue
        except HandlerHardFailure as hard:
            if try_summary_fallback(spec, identifier, str(hard)):
                continue
            w = CompileWarning(url=identifier, message=str(hard), severity="error")
            warnings.append(w)
            item_failed = CompiledSource(spec=spec, content=None, cached=False)
            results.append(item_failed)
            reporter.on_source_failed(spec, w)
            continue
        except Exception as e:  # last-resort safety net
            if try_summary_fallback(spec, identifier, f"unexpected error: {e}"):
                continue
            w = CompileWarning(
                url=identifier, message=f"Unexpected handler error: {e}", severity="error"
            )
            warnings.append(w)
            item_failed = CompiledSource(spec=spec, content=None, cached=False)
            results.append(item_failed)
            reporter.on_source_failed(spec, w)
            continue

        if cacheable and cache is not None and not no_cache and not is_soft_failure:
            ttl = (
                config.cache.youtube_ttl_days
                if spec.type == "youtube"
                else config.cache.web_ttl_days
            )
            cache.put(content, source_type=spec.type, ttl_days=ttl)

        content = _maybe_truncate(content, max_transcript_length)
        item = CompiledSource(spec=spec, content=content, cached=False)
        results.append(item)
        reporter.on_source_done(item)

    reporter.on_finish()

    return CompileResult(
        tape=tape,
        compiled_at=datetime.now(UTC),
        sources=tuple(results),
        warnings=tuple(warnings),
    )


def compile_project(
    project: ProjectFolder,
    *,
    cache: SourceCache | None,
    handlers: dict[str, SourceHandler],
    config: Config,
    include_sections: set[str] | None = None,
    exclude_sections: set[str] | None = None,
    skip_optional: bool = False,
    max_transcript_length: int | None = None,
    no_cache: bool = False,
    progress: ProgressReporter | None = None,
) -> CompileResult:
    """Compile a project folder in-place: write MIXTAPE.md and sources/NN-slug.md.

    Requires `tape.yaml` and `synthesis.md` to exist in the folder. Raises
    `MissingSynthesisError` if `synthesis.md` is absent.
    """
    if not project.tape_path.exists():
        raise FileNotFoundError(f"No tape.yaml found in {project.path}. Run `liner init` first.")
    if not project.has_synthesis():
        raise MissingSynthesisError(
            f"synthesis.md not found in {project.path}. "
            "Every mixtape requires a synthesis — write one or run the curating-mixtapes skill."
        )

    tape = load_tape(project.tape_path)

    # Inject the local_file handler — it needs the project folder to resolve paths.
    # Caller's handlers dict may already have one; only set if absent.
    effective_handlers = dict(handlers)
    if any(s.type == "local_file" for s in tape.sources) and "local_file" not in effective_handlers:
        from liner.handlers.local_file import LocalFileHandler

        effective_handlers["local_file"] = LocalFileHandler(project)

    # Inject the JS-rendering web handler when either:
    #   (a) a source explicitly asks for it (render: js), or
    #   (b) Playwright is importable, in which case we register it lazily so
    #       the compile loop can auto-fall back from a JS-stub on any web
    #       source whose render field is None.
    # Construction is cheap — the actual browser launch is deferred to the
    # first fetch that needs it.
    if "web_js" not in effective_handlers:
        any_explicit_js = any(s.type == "web" and s.render == "js" for s in tape.sources)
        any_implicit_web = any(s.type == "web" and s.render is None for s in tape.sources)
        from liner.handlers.web_js import PLAYWRIGHT_AVAILABLE

        if any_explicit_js or (any_implicit_web and PLAYWRIGHT_AVAILABLE):
            from liner.handlers.web_js import WebJsHandler

            effective_handlers["web_js"] = WebJsHandler(config.fetch)

    # Track handlers we injected so we can close them — handlers the caller
    # passed in are the caller's lifecycle to manage.
    injected_keys = set(effective_handlers) - set(handlers)
    # Build a URL → agent-WebFetch map from this folder's .liner-runs/. Cheap
    # (single JSONL walk over typically <10 runs). Used as a last-ditch
    # fallback when handlers can't reach a URL — e.g. aggressive bot wall.
    summary_cache = build_agent_fetch_cache(project.path)

    try:
        result = compile_tape(
            tape,
            cache=cache,
            handlers=effective_handlers,
            config=config,
            include_sections=include_sections,
            exclude_sections=exclude_sections,
            skip_optional=skip_optional,
            max_transcript_length=max_transcript_length,
            no_cache=no_cache,
            progress=progress,
            agent_fetch_cache=summary_cache,
        )
    finally:
        for key in injected_keys:
            handler = effective_handlers.get(key)
            close = getattr(handler, "close", None)
            if callable(close):
                with contextlib.suppress(Exception):
                    close()

    write_mixtape(project, result)
    return result


def project_written_paths(project: ProjectFolder, result: CompileResult) -> dict[str, object]:
    """Helper for the JSON `result` event — returns the folder + per-source paths."""
    sources = written_source_paths(project, result)
    return {
        "folder": str(project.path),
        "mixtape_path": str(project.mixtape_path),
        "sources": sources,
    }


def _handler_key(spec: SourceSpec) -> str:
    """Map a source spec to a key in the `handlers` dict.

    `web` sources with `render: js` dispatch to a separate handler so callers
    can decide whether to install the optional Playwright extra.
    """
    if spec.type == "web" and spec.render == "js":
        return "web_js"
    return spec.type


def _source_identifier(spec: SourceSpec) -> str:
    """A best-effort string identifier for warnings/events."""
    if spec.type == "local_file":
        return f"file://{spec.path}" if spec.path else "<local_file with no path>"
    return spec.url


def _filter_sources(
    sources: tuple[SourceSpec, ...],
    *,
    include_sections: set[str] | None,
    exclude_sections: set[str] | None,
    skip_optional: bool,
) -> list[SourceSpec]:
    out: list[SourceSpec] = []
    for spec in sources:
        if skip_optional and spec.priority == "optional":
            continue
        if include_sections is not None and (
            spec.section is None or spec.section not in include_sections
        ):
            continue
        if exclude_sections is not None and spec.section in exclude_sections:
            continue
        out.append(spec)
    return out


def _cached_content_is_usable(spec: SourceSpec, content: SourceContent) -> bool:
    if spec.type != "web":
        return True

    body = content.body.lstrip()
    if body.startswith("%PDF-"):
        return False
    if looks_like_cookie_notice_only(body):
        return False

    extraction = str(content.metadata.get("extraction", ""))
    path = urlparse(spec.url or content.url).path.lower()
    if path.endswith(".pdf") and extraction != "remote_pdf":
        return False

    return not (
        len(body) < MIN_USEFUL_BODY_CHARS
        and extraction
        in {
            "soft-fallback",
            "playwright-fallback",
        }
    )


def _maybe_truncate(content: SourceContent, max_chars: int | None) -> SourceContent:
    if max_chars is None or len(content.body) <= max_chars:
        return content
    truncated = content.body[:max_chars].rstrip() + "\n\n[…truncated]"
    return replace(content, body=truncated)
