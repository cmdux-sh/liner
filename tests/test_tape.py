from __future__ import annotations

from pathlib import Path

import pytest

from liner.tape import TapeValidationError, load_tape, write_starter_tape

FIXTURES = Path(__file__).parent / "fixtures"


def test_load_valid_tape() -> None:
    tape = load_tape(FIXTURES / "valid_tape.yaml")
    assert tape.title == "Test Tape"
    assert tape.version == 1
    assert tape.curator == "tester"
    assert len(tape.sources) == 3
    assert tape.sources[0].type == "youtube"
    assert tape.sources[0].section == "intro"
    assert tape.sources[2].priority == "optional"


def test_missing_required_field() -> None:
    with pytest.raises(TapeValidationError) as exc_info:
        load_tape(FIXTURES / "invalid_missing_field.yaml")
    assert exc_info.value.field_path == "curator"


def test_bad_source_type() -> None:
    with pytest.raises(TapeValidationError) as exc_info:
        load_tape(FIXTURES / "invalid_bad_type.yaml")
    assert "type" in exc_info.value.field_path


def test_bad_version() -> None:
    with pytest.raises(TapeValidationError) as exc_info:
        load_tape(FIXTURES / "invalid_bad_version.yaml")
    assert exc_info.value.field_path == "version"


def test_write_starter_tape_round_trip(tmp_path: Path) -> None:
    target = tmp_path / "tape.yaml"
    write_starter_tape(target)
    tape = load_tape(target)
    assert tape.version == 1
    assert len(tape.sources) >= 2
    assert tape.mode == "quick"
    assert tape.jtbd is not None


def test_write_starter_tape_refuses_overwrite(tmp_path: Path) -> None:
    target = tmp_path / "tape.yaml"
    write_starter_tape(target)
    with pytest.raises(FileExistsError):
        write_starter_tape(target)
    write_starter_tape(target, force=True)


def test_mode_field_accepted(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c
mode: methodology
jtbd: Test the JTBD field roundtrip.
methodology_version: "2.0"

sources:
  - type: web
    url: https://example.com
""",
        encoding="utf-8",
    )
    tape = load_tape(p)
    assert tape.mode == "methodology"
    assert tape.jtbd == "Test the JTBD field roundtrip."
    assert tape.methodology_version == "2.0"


def test_local_file_source_valid(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c

sources:
  - type: local_file
    path: personal/example.pdf
    citation: "Example, 2024"
    note: "Local PDF"
""",
        encoding="utf-8",
    )
    tape = load_tape(p)
    assert tape.sources[0].type == "local_file"
    assert tape.sources[0].path == "personal/example.pdf"
    assert tape.sources[0].citation == "Example, 2024"


def test_local_file_missing_path_rejected(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c

sources:
  - type: local_file
    citation: "x"
""",
        encoding="utf-8",
    )
    with pytest.raises(TapeValidationError) as exc:
        load_tape(p)
    assert exc.value.field_path == "sources[0].path"


def test_local_file_missing_citation_rejected(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c

sources:
  - type: local_file
    path: personal/x.md
""",
        encoding="utf-8",
    )
    with pytest.raises(TapeValidationError) as exc:
        load_tape(p)
    assert exc.value.field_path == "sources[0].citation"


def test_local_file_path_must_start_with_personal(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c

sources:
  - type: local_file
    path: docs/x.md
    citation: "y"
""",
        encoding="utf-8",
    )
    with pytest.raises(TapeValidationError) as exc:
        load_tape(p)
    assert exc.value.field_path == "sources[0].path"


def test_local_file_absolute_path_rejected(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c

sources:
  - type: local_file
    path: /etc/passwd
    citation: "y"
""",
        encoding="utf-8",
    )
    with pytest.raises(TapeValidationError) as exc:
        load_tape(p)
    assert exc.value.field_path == "sources[0].path"


def test_local_file_dotdot_rejected(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c

sources:
  - type: local_file
    path: personal/../secret.md
    citation: "y"
""",
        encoding="utf-8",
    )
    with pytest.raises(TapeValidationError) as exc:
        load_tape(p)
    assert exc.value.field_path == "sources[0].path"


def test_local_file_bad_extension_rejected(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c

sources:
  - type: local_file
    path: personal/payload.exe
    citation: "y"
""",
        encoding="utf-8",
    )
    with pytest.raises(TapeValidationError) as exc:
        load_tape(p)
    assert exc.value.field_path == "sources[0].path"


def test_web_render_js_accepted(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c

sources:
  - type: web
    url: https://example.com
    render: js
""",
        encoding="utf-8",
    )
    tape = load_tape(p)
    assert tape.sources[0].render == "js"


def test_web_render_invalid_rejected(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c

sources:
  - type: web
    url: https://example.com
    render: turbo
""",
        encoding="utf-8",
    )
    with pytest.raises(TapeValidationError) as exc:
        load_tape(p)
    assert exc.value.field_path == "sources[0].render"


def test_url_source_with_path_rejected(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c

sources:
  - type: web
    url: https://example.com
    path: personal/x.md
""",
        encoding="utf-8",
    )
    with pytest.raises(TapeValidationError) as exc:
        load_tape(p)
    assert exc.value.field_path == "sources[0].path"


def test_mode_rejected_when_invalid(tmp_path: Path) -> None:
    p = tmp_path / "t.yaml"
    p.write_text(
        """title: T
description: d
version: 1
curator: c
mode: turbo

sources:
  - type: web
    url: https://example.com
""",
        encoding="utf-8",
    )
    with pytest.raises(TapeValidationError) as exc_info:
        load_tape(p)
    assert exc_info.value.field_path == "mode"


def test_empty_sources_list_is_accepted(tmp_path: Path) -> None:
    """An empty `sources:` list is a valid in-progress state — a freshly-
    replayed folder, a just-init-scaffolded project before Phase 7 runs.
    `load_tape` must accept it so `liner list` doesn't silently drop the
    folder, which would hide it from the TUI mixtape browser."""
    p = tmp_path / "tape.yaml"
    p.write_text(
        """title: Work in progress
description: empty list of sources
version: 1
curator: tester
mode: quick
sources: []
""",
        encoding="utf-8",
    )
    tape = load_tape(p)
    assert tape.title == "Work in progress"
    assert len(tape.sources) == 0


def test_sources_field_must_be_a_list(tmp_path: Path) -> None:
    """Empty is fine; non-list still gets rejected — the schema invariant
    is the shape, not the population."""
    p = tmp_path / "tape.yaml"
    p.write_text(
        """title: Bad shape
description: sources is a string
version: 1
curator: tester
sources: "not a list"
""",
        encoding="utf-8",
    )
    with pytest.raises(TapeValidationError) as exc_info:
        load_tape(p)
    assert exc_info.value.field_path == "sources"
