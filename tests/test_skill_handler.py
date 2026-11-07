from __future__ import annotations

from pathlib import Path

import pytest

from liner.handlers.skill import SkillHandler, find_skill
from liner.project import ProjectFolder, init_project
from liner.types import SourceSpec


def test_local_skill_folder_extraction(tmp_path: Path) -> None:
    project = init_project(tmp_path / "p")
    skill = project.local_sources_dir / "skills" / "writing"
    (skill / "examples").mkdir(parents=True)
    (skill / "SKILL.md").write_text(
        """---
name: writing
description: Writes in the user's voice.
---

# Writing Skill

Use short sentences.
""",
        encoding="utf-8",
    )
    (skill / "examples" / "voice.md").write_text("Example voice sample.", encoding="utf-8")

    content = SkillHandler(project).fetch(
        SourceSpec(type="skill", path="local-sources/skills/writing")
    )

    assert content.title == "writing"
    assert content.metadata["extraction"] == "skill"
    assert "reference material" in content.body
    assert "## SKILL.md" in content.body
    assert "## examples/voice.md" in content.body
    assert "Example voice sample" in content.body


def test_skill_name_discovery_via_liner_skill_paths(tmp_path: Path, monkeypatch) -> None:
    root = tmp_path / "skills"
    skill = root / "local-test-skill"
    skill.mkdir(parents=True)
    (skill / "SKILL.md").write_text(
        """---
name: local-test-skill
description: Terminal UI guidance.
---

Use stable keybindings.
""",
        encoding="utf-8",
    )
    monkeypatch.setenv("LINER_SKILL_PATHS", str(root))

    record = find_skill("local-test-skill")
    assert record is not None
    assert record.path == skill

    content = SkillHandler().fetch(SourceSpec(type="skill", path="local-test-skill"))
    assert content.title == "local-test-skill"
    assert "Use stable keybindings" in content.body


def _local_skill(folder: Path) -> Path:
    folder.mkdir(parents=True)
    (folder / "SKILL.md").write_text("# Secret skill\n", encoding="utf-8")
    return folder


def test_a_tape_cannot_reach_a_folder_outside_the_allowed_roots(
    tmp_path: Path, monkeypatch
) -> None:
    from liner.handlers.base import HandlerHardFailure

    monkeypatch.setenv("HOME", str(tmp_path / "home"))
    monkeypatch.delenv("LINER_SKILL_PATHS", raising=False)
    outside = _local_skill(tmp_path / "elsewhere" / "skill")
    project = tmp_path / "project"
    init_project(project)
    handler = SkillHandler(ProjectFolder(project))
    with pytest.raises(HandlerHardFailure, match="LINER_SKILL_PATHS"):
        handler.fetch(SourceSpec(type="skill", path=str(outside)))

    monkeypatch.setenv("LINER_SKILL_PATHS", str(outside.parent))
    assert "Secret skill" in handler.fetch(SourceSpec(type="skill", path=str(outside))).body


def test_claude_skills_and_the_liner_itself_are_readable(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("HOME", str(tmp_path / "home"))
    monkeypatch.delenv("LINER_SKILL_PATHS", raising=False)
    installed = _local_skill(tmp_path / "home" / ".claude" / "skills" / "mine")
    project = tmp_path / "project"
    init_project(project)
    inside = _local_skill(project / "mixtape" / "local-sources" / "skills" / "kept")
    handler = SkillHandler(ProjectFolder(project))
    assert handler.fetch(SourceSpec(type="skill", path=str(installed))).body
    assert handler.fetch(SourceSpec(type="skill", path=str(inside))).body


def test_a_single_file_must_look_like_a_skill(tmp_path: Path, monkeypatch) -> None:
    from liner.handlers.base import HandlerHardFailure

    monkeypatch.setenv("HOME", str(tmp_path / "home"))
    project = tmp_path / "project"
    init_project(project)
    key = project / "id_rsa"
    key.write_text("secret", encoding="utf-8")
    with pytest.raises(HandlerHardFailure, match="not a readable skill file"):
        SkillHandler(ProjectFolder(project)).fetch(SourceSpec(type="skill", path=str(key)))


def test_a_symlinked_installed_skill_stays_readable(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("HOME", str(tmp_path / "home"))
    monkeypatch.delenv("LINER_SKILL_PATHS", raising=False)
    library = _local_skill(tmp_path / "library" / "adopted")
    skills = tmp_path / "home" / ".claude" / "skills"
    skills.mkdir(parents=True)
    (skills / "adopted").symlink_to(library)
    project = tmp_path / "project"
    init_project(project)
    handler = SkillHandler(ProjectFolder(project))
    assert "Secret skill" in handler.fetch(SourceSpec(type="skill", path=str(skills / "adopted"))).body
