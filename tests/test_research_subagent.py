from __future__ import annotations

import json
from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

from liner.cli import app
from liner.research_subagent import (
    SubagentError,
    _bundled,
    inspect_subagent,
    install_subagent,
    remove_subagent,
)


def target(home: Path) -> Path:
    return home / ".claude/agents/liner-researcher.md"


def test_the_subagent_has_no_shell_network_or_write_tools() -> None:
    meta = yaml.safe_load(_bundled().split("---")[1])

    assert meta["name"] == "liner-researcher"
    assert meta["tools"] == "Read, Grep"


def test_install_inspect_and_remove(tmp_path: Path) -> None:
    assert inspect_subagent(tmp_path)["status"] == "missing"

    assert install_subagent(tmp_path)["status"] == "current"
    assert target(tmp_path).read_text(encoding="utf-8") == _bundled()

    assert remove_subagent(tmp_path)["status"] == "missing"
    assert not target(tmp_path).exists()


def test_an_older_copy_can_be_updated(tmp_path: Path) -> None:
    install_subagent(tmp_path)
    target(tmp_path).write_text("<!-- liner-researcher v0 -->\n", encoding="utf-8")

    assert inspect_subagent(tmp_path)["status"] == "update_required"
    assert install_subagent(tmp_path)["status"] == "current"


def test_someone_elses_file_is_left_alone(tmp_path: Path) -> None:
    target(tmp_path).parent.mkdir(parents=True)
    target(tmp_path).write_text("# Mine\n", encoding="utf-8")

    assert inspect_subagent(tmp_path)["status"] == "foreign"
    with pytest.raises(SubagentError):
        install_subagent(tmp_path)
    with pytest.raises(SubagentError):
        remove_subagent(tmp_path)
    assert target(tmp_path).read_text(encoding="utf-8") == "# Mine\n"


def test_a_symlink_is_left_alone(tmp_path: Path) -> None:
    other = tmp_path / "other.md"
    other.write_text("<!-- liner-researcher v1 -->\n", encoding="utf-8")
    target(tmp_path).parent.mkdir(parents=True)
    target(tmp_path).symlink_to(other)

    assert inspect_subagent(tmp_path)["status"] == "foreign"


def test_a_file_that_only_quotes_the_marker_is_left_alone(tmp_path: Path) -> None:
    target(tmp_path).parent.mkdir(parents=True)
    target(tmp_path).write_text("Notes on `<!-- liner-researcher v1 -->`\n", encoding="utf-8")

    assert inspect_subagent(tmp_path)["status"] == "foreign"


def test_a_linked_agents_folder_is_left_alone(tmp_path: Path) -> None:
    (tmp_path / "shared").mkdir()
    (tmp_path / ".claude").mkdir()
    (tmp_path / ".claude/agents").symlink_to(tmp_path / "shared")

    with pytest.raises(SubagentError):
        install_subagent(tmp_path)
    assert not list((tmp_path / "shared").iterdir())


def test_an_edited_copy_is_not_reported_as_current(tmp_path: Path) -> None:
    install_subagent(tmp_path)
    edited = _bundled().replace("tools: Read, Grep", "tools: Read, Grep, Bash")
    target(tmp_path).write_text(edited, encoding="utf-8")

    assert inspect_subagent(tmp_path)["status"] == "update_required"


def test_cli_says_when_it_takes_effect(tmp_path: Path) -> None:
    result = CliRunner().invoke(
        app, ["research", "subagent", "install", "--yes", "--home", str(tmp_path)]
    )

    assert "the next one will" in result.stdout


def test_cli_previews_before_writing(tmp_path: Path) -> None:
    runner = CliRunner()
    args = ["research", "subagent"]
    home = ["--home", str(tmp_path), "--json"]

    preview = runner.invoke(app, [*args, "install", *home])
    assert preview.exit_code == 2
    assert json.loads(preview.stdout)["approval_required"] is True
    assert not target(tmp_path).exists()

    assert runner.invoke(app, [*args, "install", "--yes", *home]).exit_code == 0
    assert target(tmp_path).is_file()
    assert runner.invoke(app, [*args, "remove", *home]).exit_code == 2
    assert target(tmp_path).is_file()
    assert runner.invoke(app, [*args, "remove", "--yes", *home]).exit_code == 0
    assert not target(tmp_path).exists()
