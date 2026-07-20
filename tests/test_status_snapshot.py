from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path

from liner.compile import compile_project
from liner.config import Config
from liner.project import (
    ProjectFolder,
    init_project,
    mark_corpus_ready,
    read_liner_metadata,
    record_project_skill_status,
    refresh_status_snapshot,
    status_snapshot,
    write_liner_metadata,
)
from liner.types import SourceContent, SourceSpec


class StubHandler:
    def fetch(self, spec: SourceSpec) -> SourceContent:
        return SourceContent(
            title="Example source",
            url=spec.url,
            body="source body",
            fetched_at=datetime.now(UTC).isoformat(),
        )


def test_status_snapshot_read_preserves_saved_milestone_and_marks_stale(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    metadata = read_liner_metadata(project)
    metadata["status"]["updated"] = "2000-01-01T00:00:00Z"
    write_liner_metadata(project, metadata)
    project.mixtape_path.write_text("# MIXTAPE\n\nReady corpus.\n", encoding="utf-8")

    before = read_liner_metadata(project)
    snapshot = status_snapshot(project)
    after = read_liner_metadata(project)

    assert snapshot["milestone"] == "started"
    assert snapshot["stale"] is True
    assert after == before


def test_refresh_status_updates_only_status_fields(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    project.mixtape_path.write_text("# MIXTAPE\n\nReady corpus.\n", encoding="utf-8")
    metadata = read_liner_metadata(project)
    metadata["owner"] = "Arturo"
    metadata["project_skill"] = {"status": "declined"}
    metadata["status"]["updated"] = "2000-01-01T00:00:00Z"
    write_liner_metadata(project, metadata)
    before = read_liner_metadata(project)
    mixtape_text = project.mixtape_path.read_text(encoding="utf-8")

    snapshot = refresh_status_snapshot(project)
    after = read_liner_metadata(project)

    assert snapshot["milestone"] == "corpus_ready"
    assert snapshot["stale"] is False
    assert after["owner"] == before["owner"]
    assert after["project_skill"] == before["project_skill"]
    assert project.mixtape_path.read_text(encoding="utf-8") == mixtape_text


def test_refresh_status_marks_project_complete_after_liner_and_skill_decision(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    project.mixtape_path.write_text("# MIXTAPE\n\nReady corpus.\n", encoding="utf-8")
    project.liner_path.write_text("# LINER\n\nUse the corpus.\n", encoding="utf-8")
    record_project_skill_status(
        project,
        status="active",
        name="UI Design",
        path="skills/ui-design.md",
    )

    snapshot = refresh_status_snapshot(project)

    assert snapshot["milestone"] == "project_complete"
    assert snapshot["operating_layer"]["state"] == "ready"
    assert read_liner_metadata(project)["project_skill"]["name"] == "UI Design"


def test_compile_marks_corpus_ready_even_when_operating_layer_exists(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    project.synthesis_path.write_text("real synthesis", encoding="utf-8")
    project.liner_path.write_text("# LINER\n\nOld operating layer.\n", encoding="utf-8")
    record_project_skill_status(
        project,
        status="active",
        name="UI Design",
        path="skills/ui-design.md",
    )
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

    compile_project(
        project,
        cache=None,
        handlers={"web": StubHandler()},
        config=Config(),
    )
    metadata = read_liner_metadata(project)

    assert metadata["status"]["milestone"] == "corpus_ready"
    assert metadata["status"]["corpus"]["state"] == "ready"
    assert metadata["status"]["operating_layer"]["state"] == "pending"
    assert metadata["project_skill"]["status"] == "active"


def test_mark_corpus_ready_does_not_create_metadata_for_legacy_root_layout(
    tmp_path: Path,
) -> None:
    root = tmp_path / "legacy"
    root.mkdir()
    (root / "tape.yaml").write_text(
        "title: Legacy\nversion: 1\ncurator: A\ndescription: D\nsources: []\n",
        encoding="utf-8",
    )
    (root / "synthesis.md").write_text("Legacy synthesis.\n", encoding="utf-8")
    (root / "MIXTAPE.md").write_text("# Legacy\n", encoding="utf-8")
    project = ProjectFolder(root)

    snapshot = mark_corpus_ready(project)

    assert snapshot["milestone"] == "corpus_ready"
    assert not project.liner_metadata_path.exists()
