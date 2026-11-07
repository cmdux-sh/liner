"""Make Claude Code ask the person before it runs a command that needs their approval.

Two parts: permission `ask` rules for the plain command, and a PreToolUse hook that
reads the whole command, so `bash -c '...'`, `env`, or a full path to `liner` also
asks. This is a guardrail, not a lock: an agent set on approving can still find a
way, and Codex has no equivalent yet.
"""

from __future__ import annotations

import json
import os
import re
import shlex
import shutil
import sys
from pathlib import Path
from typing import Any

GUARD_CONTRACT = "liner.approval_guard"
CONTRACT_VERSION = 1

_COMMANDS = (
    "research approve *",
    "findings accept *",
    "findings reject *",
    "project apply *--approve*",  # Routine changes apply without it, and shouldn't prompt.
    "test add *",
    "test judge *",
)
RULES = tuple(
    f"Bash({launcher} {command})"
    for launcher in ("liner", "npx linersh", "npx -y linersh")
    for command in _COMMANDS
)


# What each approval command starts with, after `liner`.
_APPROVALS = (
    ("research", "approve"),
    ("findings", "accept"),
    ("findings", "reject"),
    ("test", "add"),
    ("test", "judge"),
)
_LINER_NAMES = {"liner", "linersh"}
_SHELLS = {"bash", "sh", "zsh", "dash", "ksh", "fish", "pwsh", "powershell"}
_PUNCTUATION = set(";&|()<>")
_HOOK_MARKER = "liner.approval_guard"
_REASON = "Liner: this records the person's approval. Run it only after they said yes in this chat."


def needs_approval(command: str) -> bool:
    """Whether a shell command would record an approval, however it's wrapped.

    It looks for `liner` anywhere in each command, not only as the program, so
    wrappers such as `sudo -u me`, `find -exec`, or `uv run` don't hide it.
    """
    command = command.replace("\\\n", " ")  # A line continuation is not a new command.
    if "\\" in command and _text_needs_approval(command.replace("\\", "/")):
        return True  # Windows paths, which a POSIX parse would mangle.
    try:
        return _parsed_needs_approval(command, depth=0)
    except ValueError:
        # Unbalanced quotes and the like: fall back to reading the text.
        return _text_needs_approval(command)


def _parsed_needs_approval(command: str, *, depth: int) -> bool:
    if depth > 4:
        return _text_needs_approval(command)
    if ("$(" in command or "`" in command or "<(" in command) and _text_needs_approval(command):
        return True
    lexer = shlex.shlex(command.replace("\n", " ; "), posix=True, punctuation_chars=";&|()<>")
    lexer.whitespace_split = True
    words = list(lexer)
    piped_into_shell = False
    simple: list[str] = []
    for word in [*words, ";"]:
        if set(word) <= _PUNCTUATION:
            if word in {"|", "|&", "<<<", "<<", "<"} and simple:
                piped_into_shell = piped_into_shell or _program(simple[-1]) in _SHELLS
            if simple and _simple_needs_approval(simple, depth=depth):
                return True
            simple = []
        else:
            simple.append(word)
            if _program(word) in _SHELLS and len(simple) == 1 and word is not words[0]:
                piped_into_shell = True
    # A script fed to a shell on stdin can't be parsed apart, so read its text.
    return piped_into_shell and _text_needs_approval(command)


def _program(word: str) -> str:
    name = Path(word.replace("\\", "/")).name.lower()
    for suffix in (".exe", ".cmd", ".bat", ".ps1"):
        name = name.removesuffix(suffix)
    return name


def _simple_needs_approval(words: list[str], *, depth: int) -> bool:
    for index, word in enumerate(words):
        program = _program(word)
        rest = words[index + 1 :]
        if program in _LINER_NAMES or program.split("@")[0] in _LINER_NAMES:
            if _is_approval(rest):
                return True
        elif word == "-m" or (
            word.startswith("-") and not word.startswith("--") and word.endswith("m")
        ):
            module = rest[0] if rest else ""
            if (module == "liner" or module.startswith("liner.")) and _is_approval(rest[1:]):
                return True
        elif program == "env" and "-S" in rest:
            split = rest[rest.index("-S") + 1] if rest.index("-S") + 1 < len(rest) else ""
            if _parsed_needs_approval(split, depth=depth + 1):
                return True
        elif program in _SHELLS or program == "eval":
            script = " ".join(rest) if program == "eval" else _shell_script(rest)
            if script is not None and _parsed_needs_approval(script, depth=depth + 1):
                return True
        elif program.startswith("python") or program in {"node", "ruby", "perl", "osascript"}:
            code = _inline_code(rest)
            if code is not None and _text_needs_approval(re.sub(r"[\[\](),]", " ", code)):
                return True
    return False


def _shell_script(args: list[str]) -> str | None:
    for index, arg in enumerate(args):
        if arg.startswith("-") and not arg.startswith("--") and "c" in arg[1:]:
            following = [word for word in args[index + 1 :] if not word.startswith("-")]
            return following[0] if following else None
        if arg.lower() in {"-command", "-c"}:
            return args[index + 1] if index + 1 < len(args) else None
    return None


def _inline_code(args: list[str]) -> str | None:
    for index, arg in enumerate(args):
        if arg in {"-c", "-e", "--eval"} and index + 1 < len(args):
            return args[index + 1]
    return None


def _is_approval(args: list[str]) -> bool:
    words = [word for word in args if not word.startswith("-")]
    if tuple(words[:2]) in _APPROVALS:
        return True
    return words[:2] == ["project", "apply"] and any(
        word == "--approve" or word.startswith("--approve=") for word in args
    )


def _text_needs_approval(command: str) -> bool:
    text = " ".join(re.sub(r"[\"'\\]", " ", command).split())
    pairs = "|".join(rf"{first}\s+{second}" for first, second in _APPROVALS)
    name = r"\b(?:liner|linersh)(?:\.(?:exe|cmd|bat|ps1))?\b"
    approval = rf"{name}(?:\s+-\S+)*\s+(?:{pairs})\b"
    applied = rf"{name}.*\bproject\s+apply\b.*--approve\b"
    return bool(re.search(approval, text) or re.search(applied, text))


def hook_command() -> str | None:
    """The command Claude Code runs for the hook, using this Liner's own Python."""
    if getattr(sys, "frozen", False):
        return f"{shlex.quote(sys.executable)} research guard hook"
    if not sys.executable:
        return None
    return f"{shlex.quote(sys.executable)} -I -m {_HOOK_MARKER}"


def run_hook(stdin: str) -> str:
    """Claude Code's PreToolUse hook: ask the person before an approval runs."""
    try:
        event = json.loads(stdin)
        command = event.get("tool_input", {}).get("command", "")
    except (json.JSONDecodeError, AttributeError):
        return ""
    if not isinstance(command, str) or not needs_approval(command):
        return ""
    return json.dumps(
        {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "ask",
                "permissionDecisionReason": _REASON,
            }
        }
    )


class GuardError(RuntimeError):
    """Claude Code's settings could not be changed safely."""


def inspect_guard(home: Path) -> dict[str, Any]:
    path = _settings_path(home)
    try:
        settings = _read(path)
        present = set(_ask_rules(settings)) & set(RULES)
        command = hook_command()
        hooked = any(
            hook.get("command") == command
            for entry in _our_hooks(settings)
            for hook in entry["hooks"]
        )
        stale_hook = bool(_our_hooks(settings)) and not hooked
        if len(present) == len(RULES) and (hooked or command is None):
            status = "current"
        elif present or hooked or stale_hook:
            status = "update_required"
        else:
            status = "missing"
    except GuardError:
        status = "foreign"
    return {
        "contract": GUARD_CONTRACT,
        "version": CONTRACT_VERSION,
        "path": str(path),
        "status": status,
        "rules": list(RULES),
        "hook": hook_command(),
    }


def install_guard(home: Path) -> dict[str, Any]:
    path = _settings_path(home)
    settings = _read(path)
    rules = _ask_rules(settings)
    settings.setdefault("permissions", {})["ask"] = [
        *rules,
        *(rule for rule in RULES if rule not in rules),
    ]
    command = hook_command()
    if command is not None:
        _remove_our_hooks(settings)
        hooks = settings.setdefault("hooks", {})
        if not isinstance(hooks, dict) or not isinstance(hooks.setdefault("PreToolUse", []), list):
            raise GuardError("Claude Code's hook settings have an unexpected shape.")
        hooks["PreToolUse"].append(
            {"matcher": "Bash", "hooks": [{"type": "command", "command": command, "timeout": 10}]}
        )
    _write(path, settings)
    return inspect_guard(home)


def remove_guard(home: Path) -> dict[str, Any]:
    path = _settings_path(home)
    settings = _read(path)
    current = _ask_rules(settings)
    rules = [rule for rule in current if rule not in RULES]
    hooked = _remove_our_hooks(settings)
    if rules != current:
        if rules:
            settings["permissions"]["ask"] = rules
        else:
            del settings["permissions"]["ask"]
            if not settings["permissions"]:
                del settings["permissions"]
    if rules != current or hooked:
        _write(path, settings)
    return inspect_guard(home)


def _our_hooks(settings: dict[str, Any]) -> list[dict[str, Any]]:
    hooks = settings.get("hooks")
    entries = hooks.get("PreToolUse") if isinstance(hooks, dict) else None
    if not isinstance(entries, list):
        return []
    return [entry for entry in entries if _is_our_hook(entry)]


def _is_our_hook(entry: Any) -> bool:
    if not isinstance(entry, dict) or not isinstance(entry.get("hooks"), list):
        return False
    return any(
        isinstance(hook, dict)
        and isinstance(hook.get("command"), str)
        and (_HOOK_MARKER in hook["command"] or hook["command"].endswith(" research guard hook"))
        for hook in entry["hooks"]
    )


def _remove_our_hooks(settings: dict[str, Any]) -> bool:
    """Take out Liner's hook entries, and the folders they leave empty."""
    if not _our_hooks(settings):
        return False
    hooks = settings["hooks"]
    remaining = []
    for entry in hooks["PreToolUse"]:
        if _is_our_hook(entry):
            # Keep anything else someone added to the same entry.
            others = [hook for hook in entry["hooks"] if not _is_our_hook({"hooks": [hook]})]
            if others:
                remaining.append({**entry, "hooks": others})
        else:
            remaining.append(entry)
    if remaining:
        hooks["PreToolUse"] = remaining
    else:
        del hooks["PreToolUse"]
        if not hooks:
            del settings["hooks"]
    return True


def _settings_path(home: Path) -> Path:
    configured = os.environ.get("CLAUDE_CONFIG_DIR")
    if configured and home == Path.home():
        return Path(configured).expanduser() / "settings.json"
    return home / ".claude" / "settings.json"


def _read(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    try:
        settings = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as error:
        raise GuardError(f"{path} could not be read, so it was left alone: {error}") from error
    if not isinstance(settings, dict):
        raise GuardError(f"{path} is not a settings file, so it was left alone.")
    return settings


def _ask_rules(settings: dict[str, Any]) -> list[str]:
    permissions = settings.get("permissions", {})
    rules = permissions.get("ask", []) if isinstance(permissions, dict) else None
    if not isinstance(rules, list):
        raise GuardError("Claude Code's permission settings have an unexpected shape.")
    return rules


def _write(path: Path, settings: dict[str, Any]) -> None:
    target = path.resolve()  # Settings are often a symlink into a dotfiles folder.
    target.parent.mkdir(parents=True, exist_ok=True)
    temporary = target.with_name(f"{target.name}.{os.getpid()}.tmp")
    backup = target.with_name(f"{target.name}.liner-backup")
    try:
        temporary.write_text(
            json.dumps(settings, indent=2, ensure_ascii=False) + "\n", encoding="utf-8"
        )
        if target.exists():
            os.chmod(temporary, target.stat().st_mode & 0o777)
            if not backup.exists():
                shutil.copy2(target, backup)
        os.replace(temporary, target)
    finally:
        temporary.unlink(missing_ok=True)


if __name__ == "__main__":
    sys.stdout.write(run_hook(sys.stdin.read()))
