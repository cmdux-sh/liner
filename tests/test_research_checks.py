from __future__ import annotations

import json
from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

from liner.cli import app
from liner.research_checks import (
    CHECKED_PHASES,
    asking_conflicts,
    check_phase,
    read_candidate_longlist,
    read_search_log,
    research_funnel,
)

URL = "https://example.org/hydration"
KEPT = {
    "url": URL,
    "decision": "kept",
    "rating": 4,
    "jtbd_fit": "direct",
    "section": "Foundations",
    "note": "Role, value, limits.",
    "fetch_status": "readable",
    "content_quality": "high",
    "evidence": ["Gives a baker's percentage table.", "Explains flour absorption."],
}
SOURCE = {
    "type": "web",
    "url": URL,
    "priority": "required",
    "kind": "reference",
    "section": "Foundations",
    "note": "Role, value, limits.",
}
QUALITY = "\n".join(
    [f"## Test {n}" for n in range(9)]
    + ["### Perspectives audit", "Distribution:", "Source-role fit", "Capability-pattern fit"]
)


def write(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    body = value if isinstance(value, str) else yaml.safe_dump(value)
    path.write_text(body, encoding="utf-8")


CITED_SYNTHESIS = (
    f"Claim.[^guide]\n\n## Generative rules\n\n## Stances this corpus takes\n\n[^guide]: {URL}\n"
)


@pytest.fixture
def corpus(tmp_path: Path) -> Path:
    root = tmp_path / "corpus"
    write(root / "tape.yaml", {"title": "Hydration", "sources": []})
    write(
        root / "working/02-candidate-longlist.md",
        f"## 1. Foundations\n\n- Guide {URL}\n\n## Search log\n\n| Query | Results | Shortlisted |\n|---|---|---|\n| hydration basics | 10 | 1 |\n",
    )
    write(root / "working/03-evaluation.yaml", {"candidates": [KEPT]})
    write(root / "working/04-quality-checks.md", QUALITY)
    write(root / "synthesis.md", CITED_SYNTHESIS)
    write(root / "working/07-tape-draft.yaml", {"sources": [SOURCE]})
    return root


def message(corpus: Path, phase: str) -> str:
    result = check_phase(corpus, phase)
    assert result["ok"] is False
    return str(result["message"])


@pytest.mark.parametrize("phase", CHECKED_PHASES)
def test_complete_artifacts_pass(corpus: Path, phase: str) -> None:
    result = check_phase(corpus, phase)
    assert result == {
        "contract": "liner.research_check",
        "version": 1,
        "phase": phase,
        "ok": True,
        "message": None,
    }


def test_longlist_parser_keeps_sections_titles_and_first_mention(tmp_path: Path) -> None:
    path = tmp_path / "longlist.md"
    write(path, f"## 2. Flour\n\n### Named guide\n- {URL}.\n- [Again]({URL})\n")
    [candidate] = read_candidate_longlist(path)
    assert (candidate.url, candidate.title, candidate.section) == (URL, "Named guide", "Flour")


@pytest.mark.parametrize(
    "body", ["- TODO: find sources\n", "- https://example.com/...\n", "None\n"]
)
def test_longlist_rejects_placeholders_and_empty_lists(corpus: Path, body: str) -> None:
    write(corpus / "working/02-candidate-longlist.md", body)
    assert "working/02-candidate-longlist.md" in message(corpus, "candidates")


def test_evaluation_must_cover_every_candidate(corpus: Path) -> None:
    write(
        corpus / "working/02-candidate-longlist.md",
        f"## A\n- {URL}\n- https://example.org/second\n",
    )
    assert "has 1 candidates" in message(corpus, "evaluation")


@pytest.mark.parametrize(
    ("change", "expected"),
    [
        ({"decision": "maybe"}, "decision is invalid: maybe"),
        ({"rating": 6}, "rating must be 1-5"),
        ({"rating": True}, "rating must be 1-5"),
        ({"jtbd_fit": ""}, "jtbd_fit is invalid: (empty)"),
        ({"fetch_status": "unavailable"}, "must be dropped"),
        ({"content_quality": "low"}, "must be dropped"),
        ({"evidence": ["One bullet."]}, "at least two"),
    ],
)
def test_evaluation_rejects_weak_kept_sources(
    corpus: Path, change: dict[str, object], expected: str
) -> None:
    write(corpus / "working/03-evaluation.yaml", {"candidates": [{**KEPT, **change}]})
    assert expected in message(corpus, "evaluation")


def test_dropped_candidates_need_no_evidence(corpus: Path) -> None:
    write(
        corpus / "working/03-evaluation.yaml",
        {"candidates": [{"url": URL, "decision": "dropped"}]},
    )
    assert check_phase(corpus, "evaluation")["ok"] is True


def test_repeated_yaml_keys_are_refused(corpus: Path) -> None:
    write(
        corpus / "working/03-evaluation.yaml",
        f"candidates:\n  - url: {URL}\n    decision: dropped\n    decision: kept\n",
    )
    assert "evaluation YAML could not be parsed" in message(corpus, "evaluation")


def test_yaml_words_and_dates_stay_text(corpus: Path) -> None:
    body = yaml.safe_dump({"candidates": [{**KEPT, "evidence": ["PLACEHOLDER"]}]})
    write(
        corpus / "working/03-evaluation.yaml",
        body.replace("- PLACEHOLDER", "- on\n  - 2026-10-30"),
    )
    assert check_phase(corpus, "evaluation")["ok"] is True


def test_quality_needs_every_test_and_a_valid_evaluation(corpus: Path) -> None:
    write(corpus / "working/04-quality-checks.md", QUALITY.replace("## Test 8", ""))
    assert "is missing ## Test 8" in message(corpus, "quality")
    write(corpus / "working/04-quality-checks.md", QUALITY)
    write(corpus / "working/03-evaluation.yaml", {"candidates": [{**KEPT, "rating": 0}]})
    assert "is invalid after quality checks" in message(corpus, "quality")


def test_quality_allows_prose_about_todo_lists(corpus: Path) -> None:
    write(corpus / "working/04-quality-checks.md", QUALITY + "\nTodo apps are a common example.\n")
    assert check_phase(corpus, "quality")["ok"] is True


def test_synthesis_rejects_placeholder_text(corpus: Path) -> None:
    write(corpus / "synthesis.md", "Replace this placeholder")
    assert "placeholder text" in message(corpus, "synthesis")


def test_assembly_needs_evaluation_evidence(corpus: Path) -> None:
    other = {**SOURCE, "url": "https://example.org/unreviewed"}
    write(corpus / "working/07-tape-draft.yaml", {"sources": [other]})
    assert "has no evaluation evidence" in message(corpus, "assembly")


def test_assembly_keeps_existing_custom_sources(corpus: Path) -> None:
    custom = {"type": "skill", "path": "skills/bread"}
    write(corpus / "tape.yaml", {"sources": [custom]})
    assert "dropped existing custom source: skill skills/bread" in message(corpus, "assembly")


def test_assembly_keeps_active_manifest_sources_without_evaluation(corpus: Path) -> None:
    manual = {**SOURCE, "url": "https://example.org/manual"}
    manifest = corpus / "local-sources/sources-manifest.yaml"
    write(manifest, {"sources": [{"source": manual}]})
    assert "dropped active custom source" in message(corpus, "assembly")
    write(corpus / "working/07-tape-draft.yaml", {"sources": [SOURCE, manual]})
    assert check_phase(corpus, "assembly")["ok"] is True


def test_unknown_phase_is_a_usage_error(corpus: Path) -> None:
    with pytest.raises(ValueError, match="No check for phase framing"):
        check_phase(corpus, "framing")


def test_cli_checks_a_corpus_folder_directly(tmp_path: Path) -> None:
    workspace = tmp_path / "workspace"
    write(workspace / "synthesis.md", "## Generative rules\n\n## Stances this corpus takes\n")
    result = CliRunner().invoke(
        app, ["research", "check", str(workspace), "synthesis", "--corpus", "--json"]
    )
    assert result.exit_code == 0, result.output


def test_cli_reports_a_failed_check(tmp_path: Path) -> None:
    project = tmp_path / "hydration"
    runner = CliRunner()
    assert runner.invoke(app, ["init", str(project), "--mode", "methodology"]).exit_code == 0
    result = runner.invoke(app, ["research", "check", str(project), "candidates", "--json"])
    assert result.exit_code == 1
    payload = json.loads(result.stdout)
    assert payload["ok"] is False
    assert "TODO" in payload["message"] or "example URL" in payload["message"]


def test_search_log_rows_are_read_and_never_become_candidates(corpus: Path) -> None:
    path = corpus / "working/02-candidate-longlist.md"
    write(
        path,
        f"## Foundations\n- {URL}\n\n## Search log\n\n"
        "| Query | Results | Shortlisted |\n|---|---|---|\n"
        "| flour absorption https://example.org/not-a-candidate | ~20 | 2 |\n"
        "| site:youtube.com hydration | 1,200 | 0 |\n"
        "| note without counts | n/a | n/a |\n",
    )

    assert [c.url for c in read_candidate_longlist(path)] == [URL]
    assert [(row["results"], row["shortlisted"]) for row in read_search_log(path)] == [
        (20, 2),
        (1200, 0),
    ]


def test_funnel_counts_each_stage(corpus: Path) -> None:
    write(
        corpus / "working/03-evaluation.yaml",
        {"candidates": [KEPT, {**KEPT, "decision": "dropped"}]},
    )
    write(corpus / "tape.yaml", {"title": "Hydration", "sources": [SOURCE]})
    fetched = corpus / ".liner-runs/fetched"
    write(fetched / "a.md", "page")
    write(fetched / "b.md", "page")

    funnel = research_funnel(corpus)

    assert funnel["searches"] == 1
    assert funnel["search_results"] == 10
    assert funnel["searches_reported_by"] == "agent"
    assert (funnel["candidates"], funnel["pages_read"]) == (1, 2)
    assert (funnel["evaluated"], funnel["kept"], funnel["final"]) == (2, 1, 1)


def test_funnel_has_no_search_counts_without_a_log(tmp_path: Path) -> None:
    funnel = research_funnel(tmp_path)

    assert funnel["searches"] is None
    assert funnel["candidates"] == 0


def test_search_log_columns_are_found_by_header(corpus: Path) -> None:
    path = corpus / "working/02-candidate-longlist.md"
    write(
        path,
        f"## Foundations\n- {URL}\n\n### Search log (web and YouTube)\n\n"
        "| # | Search | Results seen | Became candidates | Notes |\n|---|---|---|---|---|\n"
        "| 1 | flour absorption | 20+ | 2 | good |\n"
        "| 2 | broken row | lots | 1 | skipped |\n",
    )

    assert read_search_log(path) == [{"query": "flour absorption", "results": 20, "shortlisted": 2}]


def test_candidates_check_still_passes_an_older_list_without_a_log(corpus: Path) -> None:
    write(corpus / "working/02-candidate-longlist.md", f"## Foundations\n- {URL}\n")

    assert check_phase(corpus, "candidates")["ok"] is True


def test_asking_conflicts_find_rules_that_hold_the_draft_back() -> None:
    files = {
        "LINER.md": (
            "- Always ask first.\n"
            "- Never hold the draft back to ask questions.\n"
            "- Blocking gaps (ask, at most two questions, before drafting).\n"
            "- If the change list is clear, ask none and deliver the draft.\n"
            "- Ask only the questions that block the answer.\n"
            "- Never ask first; draft, then ask.\n"
            "- Do not ask before drafting.\n"
            "- Do not produce a draft until the person answers.\n"
            "- Before you draft, interview the person.\n"
        ),
        "synthesis.md": "1. **Interview before drafting.** Ask what the designer decided.\n",
    }

    found = [(item["file"], item["line"]) for item in asking_conflicts(files)]

    assert found == [
        ("LINER.md", 1),
        ("LINER.md", 3),
        ("LINER.md", 4),
        ("LINER.md", 8),
        ("LINER.md", 9),
        ("synthesis.md", 1),
    ]


def test_assembly_needs_evaluation_evidence_for_a_reused_source(corpus: Path) -> None:
    reused = {
        **SOURCE,
        "type": "liner",
        "url": "liner://typography/30982dba-4c75-519b-abf4-918662b8a798",
    }
    write(corpus / "working/07-tape-draft.yaml", {"sources": [SOURCE, reused]})
    assert "has no evaluation evidence for liner://typography/" in message(corpus, "assembly")

    whole = {**SOURCE, "type": "liner", "url": "liner://typography"}
    write(corpus / "working/07-tape-draft.yaml", {"sources": [SOURCE, whole]})
    assert check_phase(corpus, "assembly")["ok"] is True
