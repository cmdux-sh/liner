from __future__ import annotations

import json
from pathlib import Path

import pytest
from typer.testing import CliRunner

from liner.approval_guard import RULES, GuardError, inspect_guard, install_guard, remove_guard
from liner.cli import app


def settings_at(home: Path) -> Path:
    path = home / ".claude" / "settings.json"
    path.parent.mkdir(parents=True, exist_ok=True)
    return path


def test_install_keeps_everything_else_and_can_be_undone(tmp_path: Path) -> None:
    path = settings_at(tmp_path)
    original = {
        "model": "opus",
        "permissions": {"allow": ["Bash(ls *)"], "ask": ["Bash(git push *)"]},
    }
    path.write_text(json.dumps(original), encoding="utf-8")
    path.chmod(0o600)

    assert inspect_guard(tmp_path)["status"] == "missing"
    assert install_guard(tmp_path)["status"] == "current"
    written = json.loads(path.read_text(encoding="utf-8"))
    assert written["model"] == "opus"
    assert written["permissions"]["allow"] == ["Bash(ls *)"]
    assert written["permissions"]["ask"] == ["Bash(git push *)", *RULES]
    assert path.stat().st_mode & 0o777 == 0o600

    assert install_guard(tmp_path)["status"] == "current"
    assert len(json.loads(path.read_text(encoding="utf-8"))["permissions"]["ask"]) == len(RULES) + 1

    assert remove_guard(tmp_path)["status"] == "missing"
    assert json.loads(path.read_text(encoding="utf-8")) == original


def test_install_creates_the_file_and_remove_clears_the_list(tmp_path: Path) -> None:
    assert install_guard(tmp_path)["status"] == "current"
    remove_guard(tmp_path)
    assert json.loads(settings_at(tmp_path).read_text(encoding="utf-8")) == {}


def test_rules_cover_the_person_only_commands() -> None:
    assert "Bash(liner research approve *)" in RULES
    assert "Bash(npx linersh findings accept *)" in RULES
    assert "Bash(npx -y linersh test add *)" in RULES
    assert "Bash(liner project apply *--approve*)" in RULES
    assert "Bash(liner project apply *)" not in RULES


def test_a_partial_install_needs_an_update(tmp_path: Path) -> None:
    settings_at(tmp_path).write_text(
        json.dumps({"permissions": {"ask": [RULES[0]]}}), encoding="utf-8"
    )
    assert inspect_guard(tmp_path)["status"] == "update_required"


@pytest.mark.parametrize(
    "text", ["{not json", "[]", '{"permissions": []}', '{"permissions": {"ask": "x"}}']
)
def test_unreadable_settings_are_left_alone(tmp_path: Path, text: str) -> None:
    path = settings_at(tmp_path)
    path.write_text(text, encoding="utf-8")
    assert inspect_guard(tmp_path)["status"] == "foreign"
    with pytest.raises(GuardError):
        install_guard(tmp_path)
    with pytest.raises(GuardError):
        remove_guard(tmp_path)
    assert path.read_text(encoding="utf-8") == text


def test_a_symlinked_settings_file_stays_a_symlink(tmp_path: Path) -> None:
    real = tmp_path / "dotfiles" / "settings.json"
    real.parent.mkdir()
    real.write_text("{}", encoding="utf-8")
    path = settings_at(tmp_path)
    path.symlink_to(real)
    install_guard(tmp_path)
    assert path.is_symlink()
    assert "permissions" in json.loads(real.read_text(encoding="utf-8"))


def test_cli_previews_until_yes(tmp_path: Path) -> None:
    runner = CliRunner()
    preview = runner.invoke(app, ["research", "guard", "install", "--home", str(tmp_path)])
    assert preview.exit_code == 2
    assert not settings_at(tmp_path).exists()

    done = runner.invoke(
        app, ["research", "guard", "install", "--home", str(tmp_path), "--yes", "--json"]
    )
    assert done.exit_code == 0
    assert json.loads(done.output)["status"] == "current"

    inspected = runner.invoke(app, ["research", "guard", "inspect", "--home", str(tmp_path)])
    assert "current" in inspected.output
    removed = runner.invoke(app, ["research", "guard", "remove", "--home", str(tmp_path), "--yes"])
    assert removed.exit_code == 0


def test_cli_reports_a_broken_settings_file(tmp_path: Path) -> None:
    settings_at(tmp_path).write_text("{not json", encoding="utf-8")
    result = CliRunner().invoke(
        app, ["research", "guard", "install", "--home", str(tmp_path), "--yes"]
    )
    assert result.exit_code == 1
    assert "left alone" in result.output


def test_the_first_change_keeps_a_backup(tmp_path: Path) -> None:
    path = settings_at(tmp_path)
    path.write_text('{"model":   "opus"}', encoding="utf-8")
    install_guard(tmp_path)
    remove_guard(tmp_path)
    backup = path.with_name("settings.json.liner-backup")
    assert backup.read_text(encoding="utf-8") == '{"model":   "opus"}'
    assert not list(path.parent.glob("*.tmp"))


def test_remove_leaves_an_untouched_file_alone(tmp_path: Path) -> None:
    path = settings_at(tmp_path)
    path.write_text('{"permissions":   {"ask": ["Bash(x *)"]}}', encoding="utf-8")
    remove_guard(tmp_path)
    assert path.read_text(encoding="utf-8") == '{"permissions":   {"ask": ["Bash(x *)"]}}'


def test_a_moved_claude_config_is_honored(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("HOME", str(tmp_path))
    monkeypatch.setenv("CLAUDE_CONFIG_DIR", str(tmp_path / "work-profile"))
    assert install_guard(Path.home())["status"] == "current"
    assert (tmp_path / "work-profile" / "settings.json").is_file()


@pytest.mark.parametrize(
    "command",
    [
        "liner research approve demo synthesis --token abc",
        "cd ~/liner && liner research approve demo sources --token abc",
        "bash -c 'liner research approve demo sources --token abc'",
        'sh -lc "cd x; liner findings accept demo 1 --yes"',
        "env FOO=1 liner research approve demo synthesis --token abc",
        "timeout 30 /Users/me/.local/bin/liner test judge demo 1 --preferred a",
        "npx -y linersh research approve demo synthesis --token abc",
        "python -m liner project apply . --change-set-json x --approve",
        'echo "$(liner research approve demo synthesis --token abc)"',
        "LINER_DIR=/tmp liner findings reject demo 2 --yes",
        "eval 'liner test add demo q --verdict better'",
        "{ liner research approve x; }",
        "if true; then liner research approve x; fi",
        "find . -exec sh -c 'liner research approve x' \\;",
        "sudo -u me liner research approve x",
        "timeout -k 3 5 liner research approve x",
        "echo x | xargs -I {} liner findings accept {}",
        "uv run -- liner research approve x",
        "python -Im liner.cli research approve x",
        "python -c \"import subprocess; subprocess.run(['liner', 'research', 'approve', 'x'])\"",
        "liner \\\nresearch approve x",
        "echo 'liner research approve x' | bash",
        "bash <<< 'liner research approve x'",
        "bash -c -e 'liner research approve x'",
        "env -S 'liner research approve x'",
        "C:\\bin\\liner.cmd research approve x",
    ],
)
def test_hook_asks_for_approvals_however_they_are_written(command: str) -> None:
    from liner.approval_guard import needs_approval, run_hook

    assert needs_approval(command)
    output = json.loads(run_hook(json.dumps({"tool_input": {"command": command}})))
    assert output["hookSpecificOutput"]["permissionDecision"] == "ask"


@pytest.mark.parametrize(
    "command",
    [
        "liner research next demo --json",
        "liner research submit demo --json",
        "liner project apply . --change-set-json x --json",
        "grep -n 'research approve' docs/README.md",
        "git commit -m 'Explain liner research approve'",
        "ls -la",
        "rg 'liner research approve'",
        "cat notes.txt | sh -c 'wc -l'",
        "liner findings list --status accept",
    ],
)
def test_hook_lets_other_commands_through(command: str) -> None:
    from liner.approval_guard import needs_approval, run_hook

    assert not needs_approval(command)
    assert run_hook(json.dumps({"tool_input": {"command": command}})) == ""


def test_install_adds_the_hook_and_remove_takes_it_out(tmp_path: Path) -> None:
    path = settings_at(tmp_path)
    original = {"hooks": {"PreToolUse": [{"matcher": "Edit", "hooks": []}]}}
    path.write_text(json.dumps(original), encoding="utf-8")

    install_guard(tmp_path)
    install_guard(tmp_path)
    entries = json.loads(path.read_text(encoding="utf-8"))["hooks"]["PreToolUse"]
    assert len(entries) == 2
    assert entries[1]["matcher"] == "Bash"
    assert "liner.approval_guard" in entries[1]["hooks"][0]["command"]

    remove_guard(tmp_path)
    assert json.loads(path.read_text(encoding="utf-8")) == original


def test_rules_without_the_hook_need_an_update(tmp_path: Path) -> None:
    path = settings_at(tmp_path)
    path.write_text(json.dumps({"permissions": {"ask": list(RULES)}}), encoding="utf-8")

    assert inspect_guard(tmp_path)["status"] == "update_required"


def test_the_installed_hook_command_runs(tmp_path: Path) -> None:
    import subprocess

    from liner.approval_guard import hook_command

    command = hook_command()
    assert command is not None
    event = json.dumps({"tool_input": {"command": "bash -c 'liner research approve d s'"}})
    result = subprocess.run(
        command, shell=True, input=event, capture_output=True, text=True, check=True
    )
    assert json.loads(result.stdout)["hookSpecificOutput"]["permissionDecision"] == "ask"


def test_a_hook_from_an_old_install_needs_an_update(tmp_path: Path) -> None:
    path = settings_at(tmp_path)
    stale = {"type": "command", "command": "/gone/python -I -m liner.approval_guard"}
    path.write_text(
        json.dumps(
            {
                "permissions": {"ask": list(RULES)},
                "hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [stale]}]},
            }
        ),
        encoding="utf-8",
    )

    assert inspect_guard(tmp_path)["status"] == "update_required"
    assert install_guard(tmp_path)["status"] == "current"
    entries = json.loads(path.read_text(encoding="utf-8"))["hooks"]["PreToolUse"]
    assert [hook["command"] for entry in entries for hook in entry["hooks"]] == [
        entries[-1]["hooks"][0]["command"]
    ]


def test_remove_keeps_other_hooks_in_a_shared_entry(tmp_path: Path) -> None:
    path = settings_at(tmp_path)
    mine = {"type": "command", "command": "~/bin/check-bash"}
    install_guard(tmp_path)
    settings = json.loads(path.read_text(encoding="utf-8"))
    settings["hooks"]["PreToolUse"][0]["hooks"].append(mine)
    path.write_text(json.dumps(settings), encoding="utf-8")

    remove_guard(tmp_path)

    assert json.loads(path.read_text(encoding="utf-8"))["hooks"]["PreToolUse"] == [
        {"matcher": "Bash", "hooks": [mine]}
    ]
