from __future__ import annotations

import json
from pathlib import Path
from uuid import UUID

import pytest
import yaml
from typer.testing import CliRunner

from liner.cli import app
from liner.maintenance import (
    ProjectInspectionError,
    inspect_project,
    load_project_documents,
)
from liner.project import init_project, initial_liner_metadata, read_liner_metadata

runner = CliRunner()


def test_new_project_has_durable_project_and_source_ids(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")

    metadata = read_liner_metadata(project)
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))

    UUID(metadata["id"])
    assert len(tape["sources"]) == 2
    assert all(UUID(source["id"]) for source in tape["sources"])


def test_identity_is_created_only_for_new_projects(tmp_path: Path) -> None:
    assert "id" not in initial_liner_metadata()

    project = init_project(tmp_path / "demo")
    project_id = read_liner_metadata(project)["id"]

    init_project(project.path, force=True)

    assert read_liner_metadata(project)["id"] == project_id


def test_inspect_returns_versioned_snapshot_without_writing(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    metadata_before = project.liner_metadata_path.read_bytes()
    tape_before = project.tape_path.read_bytes()

    snapshot = inspect_project(project.path)
    payload = snapshot.to_dict()

    assert payload["contract"] == "liner.project_snapshot"
    assert payload["version"] == 1
    assert payload["project_id"] == read_liner_metadata(project)["id"]
    assert payload["root"] == str(project.path.resolve())
    assert payload["format"] == {"artifact": "liner", "version": 2, "layout": "v2"}
    assert payload["revision"].startswith("sha256:")
    assert payload["content_hash"] == payload["revision"].removeprefix("sha256:")
    assert payload["compatibility"]["state"] == "current"
    assert payload["lifecycle"]["milestone"] == "started"
    assert payload["capabilities"] == {
        "inspect": True,
        "plan": True,
        "apply": True,
        "identity_migration_required": False,
    }
    assert [source["source_id"] for source in payload["sources"]]
    assert project.liner_metadata_path.read_bytes() == metadata_before
    assert project.tape_path.read_bytes() == tape_before


def test_inspect_uses_explicit_path_and_verifies_project_id(tmp_path: Path) -> None:
    first = init_project(tmp_path / "first")
    second = init_project(tmp_path / "second")
    nested = second.path / "working" / "notes"
    nested.mkdir(parents=True)

    snapshot = inspect_project(
        nested,
        expected_project_id=read_liner_metadata(second)["id"],
    )

    assert snapshot.root == second.path.resolve()
    assert snapshot.project_id != read_liner_metadata(first)["id"]


def test_inspect_discovers_nearest_project_root(tmp_path: Path) -> None:
    outer = init_project(tmp_path / "outer")
    inner = init_project(outer.path / "children" / "inner")
    nested = inner.path / "working" / "notes"
    nested.mkdir(parents=True)

    snapshot = inspect_project(nested)

    assert snapshot.root == inner.path.resolve()


def test_project_id_resolves_an_ancestor_instead_of_the_nearest_project(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    outer = init_project(tmp_path / "outer")
    inner = init_project(outer.path / "children" / "inner")
    nested = inner.path / "working" / "notes"
    nested.mkdir(parents=True)
    monkeypatch.chdir(nested)

    snapshot = inspect_project(
        expected_project_id=read_liner_metadata(outer)["id"],
    )

    assert snapshot.root == outer.path.resolve()


def test_inspect_rejects_duplicate_project_id_in_ancestor_chain(tmp_path: Path) -> None:
    outer = init_project(tmp_path / "outer")
    inner = init_project(outer.path / "children" / "inner")
    outer_id = read_liner_metadata(outer)["id"]
    inner_metadata = read_liner_metadata(inner)
    inner_metadata["id"] = outer_id
    inner.liner_metadata_path.write_text(yaml.safe_dump(inner_metadata), encoding="utf-8")

    with pytest.raises(ProjectInspectionError, match="Duplicate Project ID"):
        inspect_project(inner.path)


def test_inspect_reads_legacy_project_without_migrating(tmp_path: Path) -> None:
    root = tmp_path / "legacy"
    root.mkdir()
    tape_path = root / "tape.yaml"
    tape_path.write_text(
        "title: Legacy\ndescription: Old layout\nversion: 1\ncurator: Arturo\n"
        "unknown_top: preserved\nsources:\n  - type: web\n    url: https://example.com\n"
        "    unknown_source: preserved\n",
        encoding="utf-8",
    )
    before = tape_path.read_bytes()

    payload = inspect_project(root).to_dict()

    assert payload["project_id"] is None
    assert payload["format"] == {"artifact": "mixtape", "version": 1, "layout": "legacy"}
    assert payload["compatibility"]["state"] == "legacy_missing_identity"
    assert payload["sources"][0]["source_id"] is None
    assert tape_path.read_bytes() == before
    assert not (root / "liner.yaml").exists()


def test_project_documents_round_trip_unknown_fields_semantically(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    metadata = yaml.safe_load(project.liner_metadata_path.read_text(encoding="utf-8"))
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    metadata["extension"] = {"enabled": True}
    tape["extension"] = {"owner": "Arturo"}
    tape["sources"][0]["extension"] = {"confidence": 0.8}
    project.liner_metadata_path.write_text(yaml.safe_dump(metadata), encoding="utf-8")
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")

    documents = load_project_documents(project.path)

    assert yaml.safe_load(documents.serialize_metadata()) == metadata
    assert yaml.safe_load(documents.serialize_tape()) == tape


def test_revision_changes_when_local_source_content_changes(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    local_source = project.local_sources_dir / "note.md"
    local_source.parent.mkdir(parents=True)
    local_source.write_text("first version\n", encoding="utf-8")
    before = inspect_project(project.path).revision

    local_source.write_text("second version\n", encoding="utf-8")
    after = inspect_project(project.path).revision

    assert after != before


def test_inspect_rejects_project_id_mismatch(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")

    with pytest.raises(ProjectInspectionError, match="Project ID mismatch"):
        inspect_project(project.path, expected_project_id=str(UUID(int=0)))


def test_inspect_rejects_duplicate_source_ids(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    tape["sources"][1]["id"] = tape["sources"][0]["id"]
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")

    with pytest.raises(ProjectInspectionError, match="Duplicate Source ID"):
        inspect_project(project.path)


def test_inspect_rejects_ambiguous_layout(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    (project.path / "tape.yaml").write_text(
        "title: Other\ndescription: Other\nversion: 1\ncurator: A\nsources: []\n",
        encoding="utf-8",
    )

    with pytest.raises(ProjectInspectionError, match="Ambiguous Liner Project layout"):
        inspect_project(project.path)


def test_inspect_rejects_unsupported_project_version(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    metadata = read_liner_metadata(project)
    metadata["version"] = 99
    project.liner_metadata_path.write_text(yaml.safe_dump(metadata), encoding="utf-8")

    with pytest.raises(ProjectInspectionError, match="supports format version 2"):
        inspect_project(project.path)


def test_inspect_rejects_unsupported_tape_version_with_upgrade_guidance(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    tape["version"] = 99
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")

    with pytest.raises(ProjectInspectionError, match="Upgrade Liner or migrate"):
        inspect_project(project.path)


def test_inspect_rejects_invalid_root(tmp_path: Path) -> None:
    empty = tmp_path / "empty"
    empty.mkdir()

    with pytest.raises(ProjectInspectionError, match="No Liner Project found"):
        inspect_project(empty)


def test_project_inspect_cli_emits_json_and_readable_output(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")

    json_result = runner.invoke(app, ["project", "inspect", str(project.path), "--json"])
    readable_result = runner.invoke(app, ["project", "inspect", str(project.path)])

    assert json_result.exit_code == 0, json_result.output
    payload = json.loads(json_result.stdout)
    assert payload["contract"] == "liner.project_snapshot"
    assert payload["project_id"] == read_liner_metadata(project)["id"]
    assert readable_result.exit_code == 0, readable_result.output
    assert "Project ID" in readable_result.stdout
    assert "Revision" in readable_result.stdout
    assert "Compatibility" in readable_result.stdout
    assert "Milestone: started" in readable_result.stdout
    assert "Inspect: available" in readable_result.stdout
    assert "Plan: available" in readable_result.stdout
    assert "Project and Source identities are present." in readable_result.stdout


def test_project_inspect_cli_verifies_explicit_project_id(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")

    result = runner.invoke(
        app,
        [
            "project",
            "inspect",
            str(project.path),
            "--project-id",
            str(UUID(int=0)),
            "--json",
        ],
    )

    assert result.exit_code == 1
    assert "Project ID mismatch" in result.output


@pytest.mark.parametrize(
    ("fixture", "message"),
    [
        ("ambiguous", "Ambiguous Liner Project layout"),
        ("unsupported", "Upgrade Liner or migrate"),
        ("invalid", "No Liner Project found"),
    ],
)
def test_project_inspect_cli_fails_closed(
    tmp_path: Path,
    fixture: str,
    message: str,
) -> None:
    target = tmp_path / fixture
    if fixture == "invalid":
        target.mkdir()
    else:
        project = init_project(target)
        if fixture == "ambiguous":
            (project.path / "tape.yaml").write_text(
                "title: Other\ndescription: Other\nversion: 1\ncurator: A\nsources: []\n",
                encoding="utf-8",
            )
        else:
            tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
            tape["version"] = 99
            project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")

    result = runner.invoke(app, ["project", "inspect", str(target), "--json"])

    assert result.exit_code == 1
    assert message in result.output
