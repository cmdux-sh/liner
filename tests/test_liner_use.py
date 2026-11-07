from __future__ import annotations

import json
from pathlib import Path

import pytest
from typer.testing import CliRunner

from liner.cli import app
from liner.liner_use import UseError, apply_use, plan_use
from tests.test_library import make_liner

SLUG = "touch-targets"


@pytest.fixture
def library(tmp_path: Path) -> Path:
    root = tmp_path / "library"
    root.mkdir()
    make_liner(root, SLUG, title="Touch targets")
    make_liner(root, "unfinished", title="Unfinished", complete=False)
    return root


@pytest.fixture
def repo(tmp_path: Path) -> Path:
    path = tmp_path / "repo"
    path.mkdir()
    return path


def use(library: Path, repo: Path, **options: object) -> dict[str, object]:
    plan = plan_use(library, SLUG, repo, **options)
    apply_use(plan)
    return plan


def test_a_new_repo_gets_agents_md(library: Path, repo: Path) -> None:
    use(library, repo)

    body = (repo / "AGENTS.md").read_text(encoding="utf-8")
    assert body.startswith(f"<!-- liner-use:start {SLUG} -->\n## Liner: Touch targets\n")
    assert "liner search" in body
    assert body.endswith(f"<!-- liner-use:end {SLUG} -->\n")


def test_the_pointer_holds_no_machine_paths(library: Path, repo: Path) -> None:
    use(library, repo)

    assert str(library) not in (repo / "AGENTS.md").read_text(encoding="utf-8")
    assert "/Users/" not in (repo / "AGENTS.md").read_text(encoding="utf-8")


def test_an_existing_file_is_chosen_and_kept(library: Path, repo: Path) -> None:
    (repo / "CLAUDE.md").write_text("# Rules\n\nBe kind.\n", encoding="utf-8")

    use(library, repo)

    body = (repo / "CLAUDE.md").read_text(encoding="utf-8")
    assert body.startswith("# Rules\n\nBe kind.\n\n<!-- liner-use:start")
    assert not (repo / "AGENTS.md").exists()


def test_two_agent_files_need_a_choice(library: Path, repo: Path) -> None:
    (repo / "CLAUDE.md").write_text("a\n", encoding="utf-8")
    (repo / "AGENTS.md").write_text("b\n", encoding="utf-8")

    with pytest.raises(UseError, match="--agent"):
        plan_use(library, SLUG, repo)

    use(library, repo, agent="claude")
    assert (repo / "AGENTS.md").read_text(encoding="utf-8") == "b\n"


def test_running_twice_changes_nothing(library: Path, repo: Path) -> None:
    use(library, repo)
    before = (repo / "AGENTS.md").read_text(encoding="utf-8")

    plan = use(library, repo)

    assert plan["changed"] is False
    assert (repo / "AGENTS.md").read_text(encoding="utf-8") == before


def test_a_pointer_updates_in_place(library: Path, repo: Path) -> None:
    use(library, repo)
    path = repo / "AGENTS.md"
    path.write_text(
        "Top\n\n" + path.read_text(encoding="utf-8").replace("Touch targets", "Old") + "\nEnd\n",
        encoding="utf-8",
    )

    use(library, repo)

    body = path.read_text(encoding="utf-8")
    assert body.startswith("Top\n\n<!-- liner-use:start")
    assert "Old" not in body
    assert body.endswith("\nEnd\n")


def test_remove_keeps_everything_else(library: Path, repo: Path) -> None:
    path = repo / "CLAUDE.md"
    path.write_text("# Rules\n", encoding="utf-8")
    use(library, repo)

    use(library, repo, remove=True)

    assert path.read_text(encoding="utf-8") == "# Rules\n"


def test_removing_the_only_content_removes_the_file(library: Path, repo: Path) -> None:
    use(library, repo)

    use(library, repo, remove=True)

    assert not (repo / "AGENTS.md").exists()


def test_removing_a_pointer_that_is_not_there_changes_nothing(library: Path, repo: Path) -> None:
    assert use(library, repo, remove=True)["changed"] is False


def test_another_liners_pointer_is_untouched(library: Path, repo: Path) -> None:
    make_liner(library, "second", title="Second")
    apply_use(plan_use(library, "second", repo))
    use(library, repo)

    use(library, repo, remove=True)

    assert "liner-use:start second" in (repo / "AGENTS.md").read_text(encoding="utf-8")


@pytest.mark.parametrize("slug", ["unfinished", "missing"])
def test_only_usable_liners_can_be_pointed_at(library: Path, repo: Path, slug: str) -> None:
    with pytest.raises(UseError):
        plan_use(library, slug, repo)


def test_a_linked_file_is_refused(library: Path, repo: Path, tmp_path: Path) -> None:
    other = tmp_path / "shared.md"
    other.write_text("shared\n", encoding="utf-8")
    (repo / "AGENTS.md").symlink_to(other)

    with pytest.raises(UseError, match="link"):
        plan_use(library, SLUG, repo)


@pytest.mark.parametrize(
    "body",
    [
        f"<!-- liner-use:start {SLUG} -->\nMine\n",
        f"Mine\n<!-- liner-use:end {SLUG} -->\n",
        f"<!-- liner-use:end {SLUG} -->\n<!-- liner-use:start {SLUG} -->\n",
        f"<!-- liner-use:start {SLUG} -->\n<!-- liner-use:start other -->\n"
        f"<!-- liner-use:end {SLUG} -->\n",
        f"<!-- liner-use:start {SLUG} -->\n<!-- liner-use:end {SLUG} -->\n" * 2,
    ],
)
@pytest.mark.parametrize("remove", [False, True])
def test_edited_markers_are_refused(library: Path, repo: Path, body: str, remove: bool) -> None:
    (repo / "AGENTS.md").write_text(body, encoding="utf-8")

    with pytest.raises(UseError, match="looks edited"):
        plan_use(library, SLUG, repo, remove=remove)


def test_remove_leaves_the_rest_byte_identical(library: Path, repo: Path) -> None:
    original = "\n\n# Rules\n\n```\na\n\n\n\nb\n```\n\n\nEnd\n"
    path = repo / "AGENTS.md"
    path.write_text(original, encoding="utf-8")
    use(library, repo)

    use(library, repo, remove=True)

    assert path.read_text(encoding="utf-8") == original


def test_a_pointer_in_the_middle_comes_out_cleanly(library: Path, repo: Path) -> None:
    path = repo / "AGENTS.md"
    path.write_text("Top\n", encoding="utf-8")
    use(library, repo)
    path.write_text(path.read_text(encoding="utf-8") + "\nBottom\n", encoding="utf-8")

    use(library, repo, remove=True)

    assert path.read_text(encoding="utf-8") == "Top\n\nBottom\n"


def test_windows_line_endings_are_kept(library: Path, repo: Path) -> None:
    path = repo / "AGENTS.md"
    path.write_bytes(b"# Rules\r\n\r\nBe kind.\r\n")

    use(library, repo)

    body = path.read_bytes()
    assert body.startswith(b"# Rules\r\n\r\nBe kind.\r\n\r\n<!-- liner-use:start")
    assert b"\n" not in body.replace(b"\r\n", b"")

    use(library, repo, remove=True)
    assert path.read_bytes() == b"# Rules\r\n\r\nBe kind.\r\n"


def test_file_permissions_are_kept(library: Path, repo: Path) -> None:
    path = repo / "AGENTS.md"
    path.write_text("# Rules\n", encoding="utf-8")
    path.chmod(0o640)

    use(library, repo)

    assert path.stat().st_mode & 0o777 == 0o640


def test_a_failed_write_leaves_no_temporary_file(
    library: Path, repo: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    def fail(*_: object) -> None:
        raise OSError("disk full")

    monkeypatch.setattr("liner.liner_use.os.replace", fail)
    with pytest.raises(OSError, match="disk full"):
        use(library, repo)

    assert list(repo.iterdir()) == []


def test_cli_says_when_a_file_will_be_deleted(library: Path, repo: Path) -> None:
    use(library, repo)
    args = ["use", SLUG, "--repo", str(repo), "--root", str(library), "--remove"]
    runner = CliRunner()

    assert "Would delete" in runner.invoke(app, args).stdout
    assert (repo / "AGENTS.md").exists()
    assert "Deleted" in runner.invoke(app, [*args, "--yes"]).stdout


def test_cli_previews_then_writes(library: Path, repo: Path) -> None:
    runner = CliRunner()
    args = ["use", SLUG, "--repo", str(repo), "--root", str(library), "--json"]

    preview = runner.invoke(app, args)
    assert preview.exit_code == 2
    assert json.loads(preview.stdout)["approval_required"] is True
    assert not (repo / "AGENTS.md").exists()

    applied = runner.invoke(app, [*args, "--yes"])
    assert applied.exit_code == 0
    assert (repo / "AGENTS.md").is_file()
    assert runner.invoke(app, args).exit_code == 0
    assert (
        runner.invoke(app, ["use", "nope", "--repo", str(repo), "--root", str(library)]).exit_code
        == 1
    )
