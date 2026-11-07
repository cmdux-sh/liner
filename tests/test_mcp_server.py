from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

import pytest

from liner.mcp_server import PROTOCOL_VERSIONS, READ_LINES, TOOLS, handle_line, tool_list
from tests.test_library import make_liner

SLUG = "touch-targets"


def rpc(root: Path, method: str, params: dict | None = None) -> dict:
    line = json.dumps({"jsonrpc": "2.0", "id": 1, "method": method, "params": params or {}})
    reply = handle_line(line.encode(), root)
    assert reply is not None
    return reply


def call(root: Path, name: str, **arguments: object) -> tuple[str, bool]:
    result = rpc(root, "tools/call", {"name": name, "arguments": arguments})["result"]
    return result["content"][0]["text"], result["isError"]


@pytest.fixture
def root(tmp_path: Path) -> Path:
    make_liner(tmp_path, SLUG, sources={"01-a.md": "Line one.\n" * (READ_LINES + 5)})
    return tmp_path


def test_initialize_agrees_a_version_and_names_the_server(root: Path) -> None:
    result = rpc(root, "initialize", {"protocolVersion": "2025-06-18"})["result"]
    assert result["protocolVersion"] == "2025-06-18"
    assert result["serverInfo"]["name"] == "liner"
    assert "tools" in result["capabilities"]

    newer = rpc(root, "initialize", {"protocolVersion": "2099-01-01"})["result"]
    assert newer["protocolVersion"] == PROTOCOL_VERSIONS[0]


def test_nothing_that_records_the_persons_approval_is_exposed() -> None:
    names = " ".join(tool["name"] for tool in tool_list())
    for word in ("approve", "accept", "reject", "apply", "test_add", "tests_add", "remove"):
        assert word not in names
    arguments = {"slug": "s", "query": "q", "question": "q", "finding": "f", "url": "u", "job": "j"}
    for tool in TOOLS.values():
        argv = tool.argv(arguments)
        assert not {"approve", "accept", "reject", "apply", "--yes"} & set(argv)


def test_notifications_get_no_reply(root: Path) -> None:
    line = b'{"jsonrpc": "2.0", "method": "notifications/initialized"}'
    assert handle_line(line, root) is None


def test_bad_requests_get_protocol_errors(root: Path) -> None:
    assert handle_line(b"nonsense", root)["error"]["code"] == -32700
    assert rpc(root, "nope")["error"]["code"] == -32601
    assert rpc(root, "tools/call", {"name": "nope"})["error"]["code"] == -32602
    assert rpc(root, "tools/call", {"name": "liner_gaps_add"})["error"]["code"] == -32602


def test_catalog_runs_the_real_command(root: Path) -> None:
    text, failed = call(root, "liner_catalog")

    assert not failed
    assert json.loads(text)["liners"][0]["slug"] == SLUG


def test_a_write_goes_through_the_real_command(root: Path) -> None:
    call(root, "liner_gaps_add", slug=SLUG, question="How big on a watch?")

    text, failed = call(root, "liner_gaps_list", slug=SLUG)

    assert not failed
    assert "How big on a watch?" in text


def test_a_value_cannot_become_a_flag(root: Path) -> None:
    call(root, "liner_gaps_add", slug=SLUG, question="--all")

    text, _ = call(root, "liner_gaps_list", slug=SLUG)

    assert '"question": "--all"' in text


def test_a_failed_command_comes_back_as_a_tool_error(root: Path) -> None:
    text, failed = call(root, "liner_gaps_list", slug="nowhere")

    assert failed
    assert "nowhere" in text


def test_clarify_answers_reach_the_command_as_a_list(
    root: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    seen: list[object] = []

    def record(argv: list[str]) -> str:
        path = Path(argv[argv.index("--answers") + 1])
        seen.append(json.loads(path.read_text(encoding="utf-8")))
        return "{}"

    monkeypatch.setattr("liner.mcp_server._execute", record)
    answers = [{"question": "Platform?", "answer": "iOS"}]

    _, failed = call(root, "liner_research_submit", slug=SLUG, answers=answers)

    assert not failed
    assert seen == [answers]
    schema = TOOLS["liner_research_submit"].schema()["properties"]["answers"]
    assert schema["type"] == "array"


def test_read_pages_through_long_files(root: Path) -> None:
    text, failed = call(root, "liner_read", slug=SLUG, path="mixtape/sources/01-a.md")
    assert not failed
    assert f"[lines 1-{READ_LINES} of {READ_LINES + 5}]" in text

    rest, _ = call(root, "liner_read", slug=SLUG, path="mixtape/sources/01-a.md", offset=401)
    assert rest.count("Line one.") == 5


@pytest.mark.parametrize(
    ("slug", "path"),
    [(SLUG, "../outside.md"), ("..", "outside.md"), ("", "touch-targets/LINER.md")],
)
def test_read_stays_inside_the_liner(root: Path, slug: str, path: str) -> None:
    (root.parent / "outside.md").write_text("secret", encoding="utf-8")

    text, failed = call(root, "liner_read", slug=slug, path=path)

    assert failed
    assert "secret" not in text


def test_read_refuses_a_link_that_leaves_the_liner(root: Path, tmp_path_factory) -> None:
    outside = tmp_path_factory.mktemp("elsewhere") / "secret.md"
    outside.write_text("secret", encoding="utf-8")
    (root / SLUG / "link.md").symlink_to(outside)

    text, failed = call(root, "liner_read", slug=SLUG, path="link.md")

    assert failed
    assert "secret" not in text.replace("link.md", "")


def test_the_real_process_speaks_line_delimited_json(root: Path) -> None:
    requests = [
        {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}},
        {"jsonrpc": "2.0", "method": "notifications/initialized"},
        {"jsonrpc": "2.0", "id": 2, "method": "tools/list"},
    ]
    done = subprocess.run(
        [sys.executable, "-m", "liner", "mcp", "--root", str(root)],
        input="".join(json.dumps(item) + "\n" for item in requests),
        capture_output=True,
        text=True,
        timeout=30,
        check=True,
    )
    replies = [json.loads(line) for line in done.stdout.splitlines()]
    assert [reply["id"] for reply in replies] == [1, 2]
    assert len(replies[1]["result"]["tools"]) == len(TOOLS) + 1


def test_research_steps_ask_for_inline_prompts() -> None:
    for name in ("liner_research_next", "liner_research_submit"):
        assert "--with-prompt" in TOOLS[name].argv({"slug": "demo"})
