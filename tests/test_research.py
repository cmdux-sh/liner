from __future__ import annotations

import json
from datetime import UTC, datetime
from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

from liner.cache import SourceCache
from liner.cli import _compile_research_corpus, app
from liner.compile import compile_project
from liner.config import Config
from liner.handlers.youtube import is_video_url
from liner.library import build_catalog
from liner.maintenance import inspect_project
from liner.operating_layer import render_operating_layer
from liner.project import ProjectFolder, read_liner_metadata
from liner.research import (
    ResearchError,
    approve_step,
    next_step,
    record_clarifications,
    research_plan,
    submit_step,
)
from liner.types import CompileResult, CompileWarning, SourceContent, SourceSpec
from tests.test_research_checks import KEPT, QUALITY, SOURCE, URL

JOB = "Help a future agent pick a dough hydration."
ANSWERS = [{"question": "Which flour?", "answer": "Bread flour."}]
SYNTHESIS = (
    "Follow the flour.[^guide]\n\n## Generative rules\n\n- Start low.\n\n"
    f"## Stances this corpus takes\n\n[^guide]: {URL}\n"
)


class StubHandler:
    def fetch(self, spec: SourceSpec) -> SourceContent:
        return SourceContent(
            title="Guide",
            url=spec.url or "",
            body="Guide body",
            fetched_at=datetime.now(UTC).isoformat(),
        )


def stub_compile(project: ProjectFolder) -> None:
    handlers = {"web": StubHandler(), "youtube": StubHandler()}
    compile_project(project, cache=None, handlers=handlers, config=Config())


@pytest.fixture
def project(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> ProjectFolder:
    monkeypatch.setenv("LINER_DIR", str(tmp_path))
    result = CliRunner().invoke(app, ["research", "start", JOB, "--title", "Hydration", "--json"])
    assert result.exit_code == 0, result.output
    return ProjectFolder(tmp_path / "hydration")


def write_artifact(project: ProjectFolder, step: str) -> None:
    corpus = project.corpus_path
    bodies: dict[str, tuple[str, str]] = {
        "framing": ("working/01-jtbd-and-knowledge-map.md", "# Brief\n\nA capability brief.\n"),
        "candidates": (
            "working/02-candidate-longlist.md",
            f"## Foundations\n- {URL}\n\n## Search log\n\n| Query | Results | Shortlisted |\n|---|---|---|\n| hydration basics | 10 | 1 |\n",
        ),
        "evaluation": ("working/03-evaluation.yaml", yaml.safe_dump({"candidates": [KEPT]})),
        "quality": ("working/04-quality-checks.md", QUALITY),
        "synthesis": ("synthesis.md", SYNTHESIS),
        "assembly": ("working/07-tape-draft.yaml", yaml.safe_dump({"sources": [SOURCE]})),
    }
    relative, body = bodies[step]
    (corpus / relative).write_text(body, encoding="utf-8")


def advance_to(project: ProjectFolder, target: str) -> dict[str, object]:
    step = next_step(project)
    while step["step"] != target:
        if step["kind"] == "person":
            step = submit_step(project, answers=ANSWERS)
        elif step["kind"] == "agent":
            write_artifact(project, str(step["step"]))
            step = submit_step(project)
        elif step["kind"] == "approval":
            step = approve_step(project, str(step["approval"]), str(step["token"]))
        else:
            step = submit_step(project, compile_corpus=stub_compile)
    return step


def test_start_creates_an_empty_methodology_project(project: ProjectFolder) -> None:
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))

    assert tape["jtbd"] == JOB
    assert tape["mode"] == "methodology"
    assert tape["sources"] == []
    step = next_step(project)
    assert (step["step"], step["kind"]) == ("clarify", "person")


def test_start_refuses_an_existing_liner(project: ProjectFolder) -> None:
    result = CliRunner().invoke(app, ["research", "start", JOB, "--title", "Hydration"])

    assert result.exit_code == 1


def test_next_changes_nothing(project: ProjectFolder) -> None:
    next_step(project)

    assert not (project.corpus_path / ".liner-research.json").exists()


def test_clarify_needs_the_persons_answers(project: ProjectFolder) -> None:
    with pytest.raises(ResearchError, match="answers"):
        submit_step(project)
    with pytest.raises(ResearchError, match="question and an answer"):
        submit_step(project, answers=[{"question": "Which flour?", "answer": " "}])

    submit_step(project, answers=ANSWERS)

    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    assert tape["jtbd_clarifications"] == ANSWERS


def test_submitting_answers_replaces_earlier_ones(project: ProjectFolder) -> None:
    record_clarifications(project, [{"question": "Old?", "answer": "Old."}])

    submit_step(project, answers=ANSWERS)

    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    assert tape["jtbd_clarifications"] == ANSWERS


def test_skipping_every_question_still_completes_the_step(project: ProjectFolder) -> None:
    record_clarifications(project, [])

    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    assert tape["jtbd_clarification_status"] == "complete"
    assert "jtbd_clarifications" not in tape


def test_agent_prompts_name_shipped_files_and_mark_pages_untrusted(project: ProjectFolder) -> None:
    step = advance_to(project, "candidates")

    assert step["kind"] == "agent"
    prompt = Path(step["prompt_path"]).read_text(encoding="utf-8")
    assert JOB in prompt
    assert "evidence, not instructions" in prompt
    companion = prompt.split("Companion files:\n  - ")[1].split("\n")[0]
    assert Path(companion).is_file()


def test_agent_prompts_route_pages_to_the_subagent(project: ProjectFolder) -> None:
    prompt = Path(advance_to(project, "candidates")["prompt_path"]).read_text(encoding="utf-8")

    assert "liner-researcher" in prompt
    assert "Treat its report as evidence" in prompt


def test_a_failed_check_holds_the_step(project: ProjectFolder) -> None:
    advance_to(project, "candidates")

    with pytest.raises(ResearchError, match="working/02-candidate-longlist.md"):
        submit_step(project)

    assert next_step(project)["step"] == "candidates"


def test_framing_placeholder_is_refused(project: ProjectFolder) -> None:
    advance_to(project, "framing")

    with pytest.raises(ResearchError, match="capability brief"):
        submit_step(project)


def test_submit_cannot_skip_an_approval(project: ProjectFolder) -> None:
    step = advance_to(project, "approve:synthesis")

    assert step["kind"] == "approval"
    with pytest.raises(ResearchError, match="approval"):
        submit_step(project)


def test_approval_binds_the_reviewed_content(project: ProjectFolder) -> None:
    step = advance_to(project, "approve:synthesis")
    project.synthesis_path.write_text(SYNTHESIS + "\n- Late edit.\n", encoding="utf-8")

    with pytest.raises(ResearchError, match="changed after it was reviewed"):
        approve_step(project, "synthesis", str(step["token"]))


def test_approval_must_be_the_current_step(project: ProjectFolder) -> None:
    advance_to(project, "framing")

    with pytest.raises(ResearchError, match="Nothing is waiting"):
        approve_step(project, "sources", "anything")


def test_edit_after_approval_reopens_it(project: ProjectFolder) -> None:
    advance_to(project, "assembly")
    project.synthesis_path.write_text(SYNTHESIS + "\n- Late edit.\n", encoding="utf-8")

    assert next_step(project)["step"] == "approve:synthesis"


def test_nothing_is_built_before_sources_are_approved(project: ProjectFolder) -> None:
    advance_to(project, "approve:sources")

    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    assert tape["sources"] == []
    assert not project.mixtape_path.exists()


def test_a_failed_compile_can_be_retried(project: ProjectFolder) -> None:
    advance_to(project, "build")

    def broken(_: ProjectFolder) -> None:
        raise ResearchError("Network down.")

    with pytest.raises(ResearchError, match="Network down"):
        submit_step(project, compile_corpus=broken)
    assert next_step(project)["step"] == "build"

    step = submit_step(project, compile_corpus=stub_compile)
    assert step["step"] == "approve:operating_layer"


def test_full_run_produces_a_usable_liner(project: ProjectFolder) -> None:
    review = advance_to(project, "approve:operating_layer")
    assert Path(review["review"]["files"]["LINER.md"]).is_file()
    assert not project.liner_path.exists()

    final = advance_to(project, "done")

    assert final["kind"] == "done"
    assert project.liner_path.is_file()
    assert (project.path / "SKILL.md").is_file()
    assert inspect_project(project.path).lifecycle["milestone"] == "project_complete"
    assert read_liner_metadata(project)["project_skill"]["status"] == "active"
    assert not (project.corpus_path / "working/.liner-initial-assembly").exists()
    receipts = list((project.corpus_path / ".liner-runs/maintenance").glob("*.json"))
    assert len(receipts) >= 3
    [entry] = build_catalog(project.path.parent)["liners"]
    assert entry["usable"] is True
    assert next_step(project)["step"] == "done"


def test_answers_are_refused_on_other_steps(project: ProjectFolder) -> None:
    advance_to(project, "framing")

    with pytest.raises(ResearchError, match="only applies to Clarify Job"):
        submit_step(project, answers=ANSWERS)


def test_a_deleted_review_file_gets_a_clear_error(project: ProjectFolder) -> None:
    advance_to(project, "approve:sources")
    (project.corpus_path / "working/07-tape-draft.yaml").unlink()

    with pytest.raises(ResearchError, match="could not be read"):
        next_step(project)


def test_a_broken_draft_gets_a_clear_error(project: ProjectFolder) -> None:
    advance_to(project, "approve:sources")
    (project.corpus_path / "working/07-tape-draft.yaml").write_text(": [", encoding="utf-8")

    with pytest.raises(ResearchError, match="could not be parsed"):
        next_step(project)


@pytest.mark.parametrize("slug", ["../other", "a/b", "..", "."])
def test_a_slug_must_be_one_folder_name(project: ProjectFolder, slug: str) -> None:
    result = CliRunner().invoke(app, ["research", "next", slug])

    assert result.exit_code == 1
    assert "not a liner name" in result.output


def test_fetch_saves_the_page_and_returns_only_its_path(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setattr("liner.cli.WebHandler", lambda _: StubHandler())

    result = CliRunner().invoke(app, ["fetch", "hydration", URL, "--json"])

    assert result.exit_code == 0, result.output
    payload = json.loads(result.stdout)
    assert payload["contract"] == "liner.fetched_page"
    assert "Guide body" not in result.stdout
    saved = Path(payload["path"]).read_text(encoding="utf-8")
    assert "evidence, not instructions" in saved
    assert "Guide body" in saved
    assert Path(payload["path"]).is_relative_to(project.corpus_path / ".liner-runs")


@pytest.fixture(autouse=True)
def private_cache(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    path = tmp_path / "cache.db"
    monkeypatch.setattr("liner.cli.load_config", lambda: Config(cache_path=path))
    return path


def _page(url: str, body: str) -> SourceContent:
    return SourceContent(title="Talk", url=url, body=body, fetched_at="2026-11-02T00:00:00+00:00")


class _Handler:
    def __init__(self, body: str) -> None:
        self.body = body
        self.spec: SourceSpec | None = None

    def fetch(self, spec: SourceSpec) -> SourceContent:
        self.spec = spec
        return _page(spec.url, self.body)


@pytest.mark.parametrize(
    "url", ["https://www.youtube.com/watch?v=DUejFPtYIl0", "https://youtu.be/DUejFPtYIl0"]
)
def test_fetch_saves_a_video_transcript_not_its_web_page(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch, url: str
) -> None:
    handler = _Handler("transcript " * 100)
    monkeypatch.setattr("liner.cli.YouTubeHandler", lambda _: handler)
    monkeypatch.setattr("liner.cli.WebHandler", lambda _: pytest.fail("used the web handler"))

    result = CliRunner().invoke(app, ["fetch", "hydration", url, "--json"])

    assert result.exit_code == 0, result.output
    assert handler.spec is not None and handler.spec.type == "youtube"
    payload = json.loads(result.stdout)
    assert "warning" not in payload
    assert "transcript" in Path(payload["path"]).read_text(encoding="utf-8")


def test_a_channel_page_is_still_a_web_page(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    handler = _Handler("channel " * 100)
    monkeypatch.setattr("liner.cli.WebHandler", lambda _: handler)

    result = CliRunner().invoke(app, ["fetch", "hydration", "https://www.youtube.com/@someone"])

    assert result.exit_code == 0, result.output
    assert handler.spec is not None and handler.spec.type == "web"


def test_fetch_warns_when_almost_nothing_was_saved(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setattr("liner.cli.WebHandler", lambda _: _Handler("About Press Copyright"))

    result = CliRunner().invoke(app, ["fetch", "hydration", URL, "--json"])

    assert result.exit_code == 0, result.output
    assert "may be missing" in json.loads(result.stdout)["warning"]


def test_a_full_fetch_is_reused_without_downloading_again(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch, private_cache: Path
) -> None:
    url = "https://www.youtube.com/watch?v=DUejFPtYll0"
    handler = _Handler("transcript " * 100)
    monkeypatch.setattr("liner.cli.YouTubeHandler", lambda _: handler)
    runner = CliRunner()

    assert runner.invoke(app, ["fetch", "hydration", url]).exit_code == 0
    handler.spec = None
    assert runner.invoke(app, ["fetch", "hydration", url]).exit_code == 0

    assert handler.spec is None
    with SourceCache(private_cache) as cache:
        cached = cache.get(url)
    assert cached is not None and "transcript" in cached.body


def test_a_thin_fetch_is_not_kept_for_other_liners(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch, private_cache: Path
) -> None:
    monkeypatch.setattr("liner.cli.WebHandler", lambda _: _Handler("About Press"))

    CliRunner().invoke(app, ["fetch", "hydration", URL])

    with SourceCache(private_cache) as cache:
        assert cache.get(URL) is None


def test_a_broken_cache_does_not_stop_a_fetch(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch, private_cache: Path
) -> None:
    private_cache.write_text("not a database", encoding="utf-8")
    monkeypatch.setattr("liner.cli.WebHandler", lambda _: _Handler("guide " * 200))

    result = CliRunner().invoke(app, ["fetch", "hydration", URL, "--json"])

    assert result.exit_code == 0, result.output


def test_a_build_with_a_failed_source_writes_nothing(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    result = CompileResult(
        tape=object(),  # type: ignore[arg-type]
        compiled_at=datetime.now(UTC),
        sources=(),
        warnings=(CompileWarning(url=URL, message="rate-limited", severity="error"),),
    )
    project.synthesis_path.write_text("# Synthesis\n", encoding="utf-8")
    monkeypatch.setattr("liner.compile.ensure_compile_review_approved", lambda _: None)
    monkeypatch.setattr("liner.compile.compile_tape", lambda *_a, **_k: result)
    monkeypatch.setattr("liner.cli.YouTubeHandler", lambda _: None)
    monkeypatch.setattr("liner.cli.WebHandler", lambda _: None)

    with pytest.raises(ResearchError, match="nothing was written"):
        _compile_research_corpus(project)

    assert not project.mixtape_path.exists()


def test_research_prompts_explain_the_fetch_warning(project: ProjectFolder) -> None:
    assert "result has a `warning`" in next_step(project, include_prompt=True)["prompt"]


def test_plain_fetch_keeps_the_path_on_stdout_and_warns_apart(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setattr("liner.cli.WebHandler", lambda _: _Handler("About Press Copyright"))

    result = CliRunner().invoke(app, ["fetch", "hydration", URL])

    assert result.exit_code == 0, result.output
    assert "may be missing" in result.stderr
    assert Path(result.stdout.strip()).exists()


def test_an_unexpected_transcript_error_is_a_clear_failure(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    class Broken:
        def fetch(self, _spec: SourceSpec) -> SourceContent:
            raise ValueError("transcript service changed")

    monkeypatch.setattr("liner.cli.YouTubeHandler", lambda _: Broken())

    result = CliRunner().invoke(
        app, ["fetch", "hydration", "https://www.youtube.com/watch?v=DUejFPtYll0"]
    )

    assert result.exit_code == 1
    assert "Could not fetch" in result.stderr


@pytest.mark.parametrize(
    "url",
    [
        "https://youtube.com.evil.example/watch?v=DUejFPtYll0",
        "https://notyoutube.com/embed/DUejFPtYll0",
        "https://notyoutu.be/DUejFPtYll0",
    ],
)
def test_a_lookalike_host_is_not_a_video(url: str) -> None:
    assert not is_video_url(url)


@pytest.mark.parametrize(
    "url",
    ["https://m.youtube.com/watch?v=DUejFPtYll0", "https://music.youtube.com/watch?v=DUejFPtYll0"],
)
def test_youtube_subdomains_are_videos(url: str) -> None:
    assert is_video_url(url)


@pytest.mark.parametrize("url", ["file:///etc/passwd", "ftp://example.org/a", "not a url"])
def test_fetch_refuses_anything_but_web_pages(project: ProjectFolder, url: str) -> None:
    result = CliRunner().invoke(app, ["fetch", "hydration", url])

    assert result.exit_code == 1


def test_cli_walks_a_step_with_json(project: ProjectFolder, tmp_path: Path) -> None:
    answers = tmp_path / "answers.json"
    answers.write_text(json.dumps(ANSWERS), encoding="utf-8")
    runner = CliRunner()

    result = runner.invoke(
        app, ["research", "submit", "hydration", "--answers", str(answers), "--json"]
    )

    assert result.exit_code == 0, result.output
    assert json.loads(result.stdout)["step"] == "framing"
    failed = runner.invoke(app, ["research", "submit", "hydration"])
    assert failed.exit_code == 1
    assert runner.invoke(app, ["research", "next", "missing"]).exit_code == 1


def test_finish_recovers_when_the_operating_layer_was_already_written(
    project: ProjectFolder,
) -> None:
    review = advance_to(project, "approve:operating_layer")
    approve_step(project, "operating_layer", review["token"])
    # A crash after Core wrote the files, before the step was recorded.
    state = project.corpus_path / ".liner-research.json"
    before = state.read_text(encoding="utf-8")
    submit_step(project)
    state.write_text(before, encoding="utf-8")

    step = next_step(project)
    assert step["step"] == "finish"
    assert submit_step(project)["step"] == "done"


def test_a_hand_made_liner_file_stops_finish_with_a_way_out(project: ProjectFolder) -> None:
    review = advance_to(project, "approve:operating_layer")
    approve_step(project, "operating_layer", review["token"])
    project.liner_path.write_text("# Mine\n", encoding="utf-8")

    assert next_step(project)["step"] == "finish"
    with pytest.raises(ResearchError, match="Move it"):
        submit_step(project)
    assert project.liner_path.read_text(encoding="utf-8") == "# Mine\n"

    project.liner_path.unlink()
    assert submit_step(project)["step"] == "done"


def test_the_operating_layer_token_covers_the_skill_too(project: ProjectFolder) -> None:
    review = advance_to(project, "approve:operating_layer")
    project.liner_path.write_text(render_operating_layer(project).liner_content, encoding="utf-8")
    assert next_step(project)["token"] == review["token"]


def test_steps_return_a_prompt_path_and_leave_liner_content_alone(
    project: ProjectFolder,
) -> None:
    advance_to(project, "framing")
    before = inspect_project(project.path).revision

    step = next_step(project)

    assert "prompt" not in step
    assert step["work"] == "plan"
    prompt_path = Path(str(step["prompt_path"]))
    assert prompt_path.parent == project.corpus_path / ".liner-runs" / "steps"
    assert JOB in prompt_path.read_text(encoding="utf-8")
    assert inspect_project(project.path).revision == before
    assert next_step(project, include_prompt=True)["prompt"] == prompt_path.read_text(
        encoding="utf-8"
    )


def test_plan_lists_the_agent_steps_ahead_with_their_work(project: ProjectFolder) -> None:
    plan = research_plan(project)

    assert [item["step"] for item in plan["steps"]] == [
        "framing",
        "candidates",
        "evaluation",
        "quality",
        "synthesis",
        "assembly",
    ]
    assert plan["steps"][1]["work"] == "find"
    assert set(plan["work_kinds"]) == {"plan", "find", "judge", "check", "synthesize", "assemble"}
    assert "approve:synthesis" in plan["people_steps"]

    advance_to(project, "approve:synthesis")
    assert [item["step"] for item in research_plan(project)["steps"]] == ["assembly"]


def test_cli_plan_prints_json(project: ProjectFolder) -> None:
    result = CliRunner().invoke(app, ["research", "plan", project.path.name, "--json"])

    assert result.exit_code == 0, result.output
    assert json.loads(result.stdout)["contract"] == "liner.research_plan"


def test_approval_reviews_point_to_files(project: ProjectFolder) -> None:
    sources = advance_to(project, "approve:sources")["review"]
    assert sources["count"] >= 1
    assert "sources" not in sources

    review = advance_to(project, "approve:operating_layer")["review"]
    for path in review["files"].values():
        assert Path(path).is_file()


def test_done_reports_the_research_funnel(project: ProjectFolder) -> None:
    final = advance_to(project, "done")

    funnel = final["funnel"]
    assert funnel["searches"] == 1
    assert funnel["candidates"] == 1
    assert "subagents" in str(final["instructions"])


def test_status_shows_the_funnel_for_agent_builds(project: ProjectFolder) -> None:
    advance_to(project, "done")

    result = CliRunner().invoke(app, ["status", str(project.path), "--no-write"])

    assert result.exit_code == 0, result.output
    assert "Searches" in result.output
    assert "reported by the agent" in result.output
    assert "Tool calls" not in result.output


def test_new_builds_need_a_search_log(project: ProjectFolder) -> None:
    advance_to(project, "candidates")
    path = project.corpus_path / "working/02-candidate-longlist.md"
    path.write_text(f"## Foundations\n- {URL}\n", encoding="utf-8")

    with pytest.raises(ResearchError, match="no search log"):
        submit_step(project)


def test_builds_started_before_the_search_log_skip_it(project: ProjectFolder) -> None:
    advance_to(project, "candidates")
    state_path = project.corpus_path / ".liner-research.json"
    state = json.loads(state_path.read_text(encoding="utf-8"))
    state.pop("search_log")
    state_path.write_text(json.dumps(state), encoding="utf-8")
    path = project.corpus_path / "working/02-candidate-longlist.md"
    path.write_text(f"## Foundations\n- {URL}\n", encoding="utf-8")

    assert submit_step(project)["step"] == "evaluation"


def test_inline_reviews_carry_the_content(project: ProjectFolder) -> None:
    advance_to(project, "approve:sources")
    sources = next_step(project, include_prompt=True)["review"]
    assert sources["sources"]

    advance_to(project, "approve:operating_layer")
    review = next_step(project, include_prompt=True)["review"]
    for name, path in review["files"].items():
        assert Path(path).read_text(encoding="utf-8") == review["content"][name]


def test_edited_review_copies_block_the_instructions_approval(project: ProjectFolder) -> None:
    step = advance_to(project, "approve:operating_layer")
    before = inspect_project(project.path).revision
    Path(step["review"]["files"]["LINER.md"]).write_text("# Not what Liner writes\n")

    with pytest.raises(ResearchError, match="review copies"):
        approve_step(project, "operating_layer", str(step["token"]))
    assert inspect_project(project.path).revision == before

    step = next_step(project)
    assert approve_step(project, "operating_layer", str(step["token"]))["step"] == "finish"


def test_plan_after_the_last_agent_step_says_nothing_is_left(project: ProjectFolder) -> None:
    advance_to(project, "approve:operating_layer")
    plan = research_plan(project)

    assert plan["steps"] == []
    assert plan["people_steps"] == ["approve:operating_layer"]
    assert "No agent steps are left" in plan["instructions"]


def test_status_json_carries_the_funnel(project: ProjectFolder) -> None:
    advance_to(project, "done")

    result = CliRunner().invoke(app, ["status", str(project.path), "--no-write", "--json"])

    assert result.exit_code == 0, result.output
    assert json.loads(result.stdout)["funnel"]["searches"] == 1


def test_cli_with_prompt_includes_the_text(project: ProjectFolder) -> None:
    advance_to(project, "framing")
    result = CliRunner().invoke(
        app, ["research", "next", project.path.name, "--json", "--with-prompt"]
    )

    assert result.exit_code == 0, result.output
    assert JOB in json.loads(result.stdout)["prompt"]


def test_synthesis_approval_shows_ask_first_rules(project: ProjectFolder) -> None:
    advance_to(project, "synthesis")
    project.synthesis_path.write_text(
        SYNTHESIS + "\n6. **Ask before guessing.** Ask two questions before drafting.\n",
        encoding="utf-8",
    )
    step = submit_step(project)

    assert step["step"] == "approve:synthesis"
    conflicts = step["review"]["conflicts"]
    assert [conflict["text"] for conflict in conflicts] == [
        "6. **Ask before guessing.** Ask two questions before drafting."
    ]
    assert "draft first, then ask" in str(step["instructions"])


def test_approvals_without_ask_first_rules_have_no_conflicts(project: ProjectFolder) -> None:
    step = advance_to(project, "approve:synthesis")
    assert "conflicts" not in step["review"]
    assert "draft first" not in str(step["instructions"])

    step = advance_to(project, "approve:operating_layer")
    assert "conflicts" not in step["review"]


def test_plan_names_each_approval_in_plain_words(project: ProjectFolder) -> None:
    advance_to(project, "framing")

    approvals = research_plan(project)["approvals"]

    assert [(item["what"], item["after"]) for item in approvals] == [
        ("the synthesis", "synthesis"),
        ("the source list", "assembly"),
        ("the instructions (LINER.md and SKILL.md)", "build"),
    ]


def _with_js_source(project: ProjectFolder) -> None:
    draft = project.corpus_path / "working/07-tape-draft.yaml"
    draft.write_text(yaml.safe_dump({"sources": [{**SOURCE, "render": "js"}]}), encoding="utf-8")


def test_sources_approval_asks_about_playwright_when_it_is_missing(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setattr("liner.research.js_rendering_ready", lambda: False)
    advance_to(project, "assembly")
    _with_js_source(project)
    step = submit_step(project)

    assert step["step"] == "approve:sources"
    assert step["review"]["js_setup"] == {
        "sources": [SOURCE["url"]],
        "command": "liner setup-js --yes",
    }
    assert "As its own question" in str(step["instructions"])

    step = approve_step(project, "sources", str(step["token"]))
    assert step["step"] == "build"
    with pytest.raises(ResearchError, match="liner setup-js --yes"):
        submit_step(project, compile_corpus=stub_compile)


def test_sources_approval_skips_playwright_when_it_is_ready(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setattr("liner.research.js_rendering_ready", lambda: True)
    advance_to(project, "assembly")
    _with_js_source(project)
    step = submit_step(project)

    assert "js_setup" not in step["review"]


def test_finished_build_refreshes_the_liner_skill(project: ProjectFolder) -> None:
    advance_to(project, "finish")
    calls: list[str] = []

    def refresh() -> list[str]:
        calls.append("refresh")
        return ["claude"]

    step = submit_step(project, refresh_skills=refresh)

    assert step["step"] == "done"
    assert step["skills_updated"] == ["claude"]
    assert calls == ["refresh"]


def test_a_failed_skill_refresh_still_finishes_the_build(project: ProjectFolder) -> None:
    advance_to(project, "finish")

    def refresh() -> list[str]:
        raise PermissionError("skills folder is locked")

    step = submit_step(project, refresh_skills=refresh)

    assert step["step"] == "done"
    assert step["skills_updated"] == []


def test_a_new_liner_drops_the_starter_description_and_tags(project: ProjectFolder) -> None:
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))

    assert tape["description"] == "Guidance for Hydration."
    assert tape["tags"] == []


def test_mixtape_leaves_out_a_starter_description(project: ProjectFolder) -> None:
    advance_to(project, "build")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    tape["description"] = (
        "A starter tape with one YouTube video and one article so you can see the output "
        "immediately."
    )
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")

    submit_step(project, compile_corpus=stub_compile)

    mixtape = project.mixtape_path.read_text(encoding="utf-8")
    assert "starter tape" not in mixtape
    assert "Guidance for Hydration." not in mixtape
    assert f"**JTBD:** {JOB}" in mixtape
