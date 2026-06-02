from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path

import pytest

from liner.cache import SourceCache
from liner.compile import MissingSynthesisError, compile_project, compile_tape
from liner.config import Config
from liner.handlers.base import HandlerHardFailure, HandlerSoftFailure, JsRenderingRequired
from liner.project import init_project
from liner.types import SourceContent, SourceSpec, Tape


class StubHandler:
    def __init__(self, body: str = "content body") -> None:
        self._body = body
        self.calls: list[str] = []

    def fetch(self, spec: SourceSpec) -> SourceContent:
        url = spec.url
        self.calls.append(url)
        return SourceContent(
            title=f"title for {url}",
            url=url,
            body=self._body,
            fetched_at=datetime.now(UTC).isoformat(),
        )


class FailingHandler:
    def fetch(self, spec: SourceSpec) -> SourceContent:
        raise HandlerHardFailure(f"boom for {spec.url}", spec.url)


def _tape(sources: list[SourceSpec]) -> Tape:
    return Tape(
        title="T",
        description="d",
        version=1,
        curator="c",
        sources=tuple(sources),
    )


def test_compile_succeeds_with_all_sources(tmp_path: Path) -> None:
    tape = _tape(
        [
            SourceSpec(type="web", url="https://a", section="x"),
            SourceSpec(type="web", url="https://b"),
        ]
    )
    handler = StubHandler()
    cache = SourceCache(tmp_path / "c.db")
    result = compile_tape(
        tape,
        cache=cache,
        handlers={"web": handler},
        config=Config(),
    )
    assert result.total_attempted == 2
    assert result.total_succeeded == 2
    assert result.warnings == ()
    assert handler.calls == ["https://a", "https://b"]


def test_partial_failure_isolates(tmp_path: Path) -> None:
    tape = _tape(
        [
            SourceSpec(type="web", url="https://good"),
            SourceSpec(type="web", url="https://bad"),
        ]
    )

    class MixedHandler:
        def fetch(self, spec: SourceSpec) -> SourceContent:
            url = spec.url
            if "bad" in url:
                raise HandlerHardFailure("nope", url)
            return SourceContent(
                title="t", url=url, body="body", fetched_at="2026-05-16T00:00:00+00:00"
            )

    result = compile_tape(
        tape,
        cache=None,
        handlers={"web": MixedHandler()},
        config=Config(),
    )
    assert result.total_attempted == 2
    assert result.total_succeeded == 1
    assert len(result.warnings) == 1
    assert result.warnings[0].url == "https://bad"


def test_cache_hit_skips_handler(tmp_path: Path) -> None:
    tape = _tape([SourceSpec(type="web", url="https://a")])
    handler = StubHandler()
    cache = SourceCache(tmp_path / "c.db")

    compile_tape(tape, cache=cache, handlers={"web": handler}, config=Config())
    assert handler.calls == ["https://a"]

    compile_tape(tape, cache=cache, handlers={"web": handler}, config=Config())
    assert handler.calls == ["https://a"], "second call should hit cache"


def test_bad_cached_web_pdf_is_refetched(tmp_path: Path) -> None:
    tape = _tape([SourceSpec(type="web", url="https://example.com/report.pdf")])
    cache = SourceCache(tmp_path / "c.db")
    stale = SourceContent(
        title="bad pdf",
        url="https://example.com/report.pdf",
        body="%PDF-1.7 raw binary pretending to be text",
        fetched_at="2026-05-19T00:00:00+00:00",
        metadata={"extraction": "soft-fallback"},
    )
    cache.put(stale, source_type="web", ttl_days=30)

    handler = StubHandler(body="fresh extracted PDF text")
    result = compile_tape(tape, cache=cache, handlers={"web": handler}, config=Config())

    assert handler.calls == ["https://example.com/report.pdf"]
    assert result.sources[0].content is not None
    assert result.sources[0].content.body == "fresh extracted PDF text"


def test_cookie_notice_cached_web_content_is_refetched(tmp_path: Path) -> None:
    tape = _tape([SourceSpec(type="web", url="https://pair.withgoogle.com/guidebook/")])
    cache = SourceCache(tmp_path / "c.db")
    stale = SourceContent(
        title="PAIR Guidebook",
        url="https://pair.withgoogle.com/guidebook/",
        body=(
            "This site uses cookies from Google to deliver and enhance the quality "
            "of its services and to analyze traffic. Learn more OK, got it"
        ),
        fetched_at="2026-05-19T00:00:00+00:00",
        metadata={"extraction": "playwright"},
    )
    cache.put(stale, source_type="web", ttl_days=30)

    handler = StubHandler(body="fresh rendered guidebook content")
    result = compile_tape(tape, cache=cache, handlers={"web": handler}, config=Config())

    assert handler.calls == ["https://pair.withgoogle.com/guidebook/"]
    assert result.sources[0].content is not None
    assert result.sources[0].content.body == "fresh rendered guidebook content"


def test_no_cache_bypasses_lookup(tmp_path: Path) -> None:
    tape = _tape([SourceSpec(type="web", url="https://a")])
    handler = StubHandler()
    cache = SourceCache(tmp_path / "c.db")

    compile_tape(tape, cache=cache, handlers={"web": handler}, config=Config())
    compile_tape(
        tape,
        cache=cache,
        handlers={"web": handler},
        config=Config(),
        no_cache=True,
    )
    assert handler.calls == ["https://a", "https://a"]


def test_filters_apply_correctly() -> None:
    tape = _tape(
        [
            SourceSpec(type="web", url="https://a", section="alpha"),
            SourceSpec(type="web", url="https://b", section="beta"),
            SourceSpec(type="web", url="https://c", priority="optional"),
        ]
    )

    handler = StubHandler()
    r1 = compile_tape(
        tape,
        cache=None,
        handlers={"web": handler},
        config=Config(),
        include_sections={"alpha"},
    )
    assert [s.spec.url for s in r1.sources] == ["https://a"]

    handler2 = StubHandler()
    r2 = compile_tape(
        tape,
        cache=None,
        handlers={"web": handler2},
        config=Config(),
        exclude_sections={"alpha"},
    )
    assert [s.spec.url for s in r2.sources] == ["https://b", "https://c"]

    handler3 = StubHandler()
    r3 = compile_tape(
        tape,
        cache=None,
        handlers={"web": handler3},
        config=Config(),
        skip_optional=True,
    )
    assert [s.spec.url for s in r3.sources] == ["https://a", "https://b"]


def test_max_transcript_length_truncates() -> None:
    tape = _tape([SourceSpec(type="web", url="https://a")])
    handler = StubHandler(body="x" * 1000)
    result = compile_tape(
        tape,
        cache=None,
        handlers={"web": handler},
        config=Config(),
        max_transcript_length=100,
    )
    body = result.sources[0].content.body  # type: ignore[union-attr]
    assert "truncated" in body
    assert len(body) < 200


def test_missing_handler_fails_gracefully() -> None:
    tape = _tape([SourceSpec(type="youtube", url="https://yt")])
    result = compile_tape(
        tape,
        cache=None,
        handlers={"web": StubHandler()},
        config=Config(),
    )
    assert result.total_succeeded == 0
    assert len(result.warnings) == 1


class JsStubHandler:
    """Server-side handler that always reports the page needs JS rendering."""

    def __init__(self) -> None:
        self.calls: list[str] = []

    def fetch(self, spec: SourceSpec) -> SourceContent:
        self.calls.append(spec.url)
        raise JsRenderingRequired(f"{spec.url} is JavaScript-rendered.", spec.url)


def test_js_required_auto_falls_back_to_web_js(tmp_path: Path) -> None:
    """When server fetch hits a JS stub and web_js is registered, retry."""
    tape = _tape([SourceSpec(type="web", url="https://spa.example.com")])
    server = JsStubHandler()
    js = StubHandler(body="rendered via playwright")

    result = compile_tape(
        tape,
        cache=None,
        handlers={"web": server, "web_js": js},
        config=Config(),
    )
    assert result.total_succeeded == 1
    assert "rendered via playwright" in (result.sources[0].content.body or "")
    # The fallback emits a notice as a warning so the curator sees what happened.
    assert any("auto-fell back to render: js" in w.message for w in result.warnings)


def test_js_required_without_web_js_handler_surfaces_setup_hint(tmp_path: Path) -> None:
    """When [js] isn't installed, the message points at `liner setup-js`."""
    tape = _tape([SourceSpec(type="web", url="https://spa.example.com")])
    server = JsStubHandler()

    result = compile_tape(
        tape,
        cache=None,
        handlers={"web": server},
        config=Config(),
    )
    assert result.total_succeeded == 0
    assert len(result.warnings) == 1
    assert "liner setup-js" in result.warnings[0].message


def test_render_server_opts_out_of_auto_fallback(tmp_path: Path) -> None:
    """`render: server` makes the JS-stub failure fatal even if web_js is registered."""
    tape = _tape([SourceSpec(type="web", url="https://spa.example.com", render="server")])
    server = JsStubHandler()
    js = StubHandler(body="should not be used")

    result = compile_tape(
        tape,
        cache=None,
        handlers={"web": server, "web_js": js},
        config=Config(),
    )
    assert result.total_succeeded == 0
    assert js.calls == []  # web_js was never invoked
    # The error message should not push the curator toward setup-js since
    # they explicitly opted out.
    assert "liner setup-js" not in result.warnings[0].message


def test_soft_failure_content_is_not_cached(tmp_path: Path) -> None:
    """Best-effort soft-failure content should not be baked into the cache."""

    class SoftFailHandler:
        def __init__(self) -> None:
            self.calls: list[str] = []

        def fetch(self, spec: SourceSpec) -> SourceContent:
            url = spec.url
            self.calls.append(url)
            content = SourceContent(
                title="t",
                url=url,
                body="short",
                fetched_at="2026-05-19T00:00:00+00:00",
            )
            raise HandlerSoftFailure(content, "extraction was very short")

    tape = _tape([SourceSpec(type="web", url="https://soft")])
    handler = SoftFailHandler()
    cache = SourceCache(tmp_path / "c.db")

    result = compile_tape(tape, cache=cache, handlers={"web": handler}, config=Config())
    assert result.total_succeeded == 1
    assert len(result.warnings) == 1

    # Compile again: cache must not be holding the soft-failure body, so the
    # handler should be called a second time.
    compile_tape(tape, cache=cache, handlers={"web": handler}, config=Config())
    assert handler.calls == ["https://soft", "https://soft"]


def test_compile_project_writes_mixtape_and_sources(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    project.synthesis_path.write_text("synth", encoding="utf-8")
    project.tape_path.write_text(
        """title: T
description: d
version: 1
curator: c
mode: quick

sources:
  - type: web
    url: https://example.com/a
    section: intro
""",
        encoding="utf-8",
    )

    handler = StubHandler()
    result = compile_project(
        project,
        cache=None,
        handlers={"web": handler},
        config=Config(),
    )
    assert result.total_succeeded == 1
    assert project.mixtape_path.exists()
    written = sorted(project.sources_dir.iterdir())
    assert len(written) == 1
    assert "## Synthesis" in project.mixtape_path.read_text(encoding="utf-8")


def test_compile_project_fails_without_synthesis(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    project.synthesis_path.unlink()
    with pytest.raises(MissingSynthesisError):
        compile_project(
            project,
            cache=None,
            handlers={"web": StubHandler()},
            config=Config(),
        )


def test_compile_project_with_local_file(tmp_path: Path) -> None:
    """A local_file source should compile end-to-end with the auto-injected handler."""
    project = init_project(tmp_path / "demo")
    project.synthesis_path.write_text("real synthesis", encoding="utf-8")
    project.personal_dir.mkdir(parents=True, exist_ok=True)
    (project.personal_dir / "note.md").write_text(
        "# Local Note\nSome content from disk.\n", encoding="utf-8"
    )
    project.tape_path.write_text(
        """title: T
description: d
version: 1
curator: c
mode: quick

sources:
  - type: local_file
    path: personal/note.md
    citation: "Local Note, 2026-05-19"
""",
        encoding="utf-8",
    )

    # Handlers dict left empty — compile_project should auto-inject LocalFileHandler.
    result = compile_project(
        project,
        cache=None,
        handlers={},
        config=Config(),
    )
    assert result.total_succeeded == 1
    sources_written = list(project.sources_dir.iterdir())
    assert len(sources_written) == 1
    body = sources_written[0].read_text(encoding="utf-8")
    assert "Some content from disk" in body
    assert "Local Note, 2026-05-19" in body  # citation surfaces as title


def test_compile_project_fails_without_tape(tmp_path: Path) -> None:
    from liner.project import ProjectFolder

    project = ProjectFolder(tmp_path / "empty")
    project.path.mkdir()
    with pytest.raises(FileNotFoundError):
        compile_project(
            project,
            cache=None,
            handlers={"web": StubHandler()},
            config=Config(),
        )


# --- Fix B: agent-summary fallback at compile time -------------------------


def test_compile_falls_back_to_agent_summary_on_hard_failure() -> None:
    """When a handler raises HardFailure but the URL has a cached agent
    WebFetch summary, compile uses the summary instead of marking failed."""
    from liner.agent_fetch_cache import AgentFetch

    url = "https://medium.com/blocked-article"
    summary = (
        "# Article Summary\n\n**Title:** A Blocked Article\n\n"
        "Key points about CLI UX, written before the page started gating crawlers."
    )
    cache = {
        url: AgentFetch(url=url, body=summary, captured_at="2026-05-22T10:00:00Z", run_path="x")
    }
    tape = _tape([SourceSpec(type="web", url=url)])

    result = compile_tape(
        tape,
        cache=None,
        handlers={"web": FailingHandler()},
        config=Config(),
        agent_fetch_cache=cache,
    )

    assert len(result.sources) == 1
    item = result.sources[0]
    assert item.content is not None
    assert item.content.body == summary
    assert item.content.title == "A Blocked Article"
    assert item.content.metadata.get("extraction") == "agent-summary"
    assert any("falling back to the summary" in w.message for w in result.warnings)
    assert not any(w.severity == "error" for w in result.warnings)


def test_compile_marks_failed_when_neither_handler_nor_summary_succeed() -> None:
    url = "https://example.com/unreachable"
    tape = _tape([SourceSpec(type="web", url=url)])

    result = compile_tape(
        tape,
        cache=None,
        handlers={"web": FailingHandler()},
        config=Config(),
        agent_fetch_cache={},
    )

    assert result.sources[0].content is None
    assert any(w.severity == "error" for w in result.warnings)


def test_summary_fallback_handles_unregistered_handler() -> None:
    """Even when there's no handler at all for the source type, the summary rescues."""
    from liner.agent_fetch_cache import AgentFetch

    url = "https://example.com/x"
    summary = "# Article Summary\n\n**Title:** Test\n\n" + "Body content. " * 20
    cache = {
        url: AgentFetch(url=url, body=summary, captured_at="2026-05-22T10:00:00Z", run_path="x")
    }
    tape = _tape([SourceSpec(type="web", url=url)])

    result = compile_tape(
        tape,
        cache=None,
        handlers={},
        config=Config(),
        agent_fetch_cache=cache,
    )

    assert result.sources[0].content is not None
    assert result.sources[0].content.metadata.get("extraction") == "agent-summary"


@pytest.fixture(autouse=True)
def _no_real_filesystem(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """Belt and suspenders: keep tests from writing to ~/.liner."""
    monkeypatch.setenv("HOME", str(tmp_path))
