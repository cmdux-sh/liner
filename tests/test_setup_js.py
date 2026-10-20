"""`liner setup-js` downloads only the browser Liner launches."""

from __future__ import annotations

import subprocess
import sys
import types

import pytest
from typer.testing import CliRunner

from liner import cli


def test_install_args_never_fetch_full_chromium() -> None:
    assert cli._playwright_install_args() == ["install", "--only-shell", "chromium"]


def test_setup_js_states_measured_size_and_installs_headless_shell(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls: list[list[str]] = []

    def fake_run(args: list[str], check: bool) -> subprocess.CompletedProcess[bytes]:
        calls.append(args)
        return subprocess.CompletedProcess(args, 0)

    monkeypatch.setitem(sys.modules, "playwright", types.ModuleType("playwright"))
    monkeypatch.setattr(cli, "_installed_playwright_version", lambda: (1, 64))
    monkeypatch.setattr(cli, "_is_frozen_binary", lambda: False)
    monkeypatch.setattr(subprocess, "run", fake_run)

    result = CliRunner().invoke(cli.app, ["setup-js", "--yes"])

    assert result.exit_code == 0, result.output
    assert calls == [[sys.executable, "-m", "playwright", "install", "--only-shell", "chromium"]]
    output = " ".join(result.output.split())
    assert "about 100 MB to download, about 220 MB on disk" in output
    assert "150" not in output


def test_setup_js_upgrades_playwright_that_cannot_fetch_the_shell_alone(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls: list[list[str]] = []

    def fake_run(args: list[str], check: bool) -> subprocess.CompletedProcess[bytes]:
        calls.append(args)
        return subprocess.CompletedProcess(args, 0)

    monkeypatch.setitem(sys.modules, "playwright", types.ModuleType("playwright"))
    monkeypatch.setattr(cli, "_installed_playwright_version", lambda: (1, 48))
    monkeypatch.setattr(cli, "_is_frozen_binary", lambda: False)
    monkeypatch.setattr(cli, "_is_pipx_venv", lambda: False)
    monkeypatch.setattr(subprocess, "run", fake_run)

    result = CliRunner().invoke(cli.app, ["setup-js", "--yes"])

    assert result.exit_code == 0, result.output
    assert calls == [
        [sys.executable, "-m", "pip", "install", "--upgrade", "playwright>=1.49"],
        [sys.executable, "-m", "playwright", "install", "--only-shell", "chromium"],
    ]
    output = " ".join(result.output.split())
    assert "Upgrade the `playwright` Python package to 1.49 or later" in output


def test_setup_js_stops_a_frozen_binary_with_old_playwright_before_downloading(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls: list[list[str]] = []
    monkeypatch.setitem(sys.modules, "playwright", types.ModuleType("playwright"))
    monkeypatch.setattr(cli, "_installed_playwright_version", lambda: (1, 40))
    monkeypatch.setattr(cli, "_is_frozen_binary", lambda: True)
    monkeypatch.setattr(subprocess, "run", lambda args, check: calls.append(args))

    result = CliRunner().invoke(cli.app, ["setup-js", "--yes"])

    assert result.exit_code == 1
    assert calls == []
    assert "older than 1.49" in " ".join(result.output.split())
