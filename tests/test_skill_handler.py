from __future__ import annotations

from pathlib import Path

from liner.handlers.skill import SkillHandler, find_skill
from liner.project import init_project
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
