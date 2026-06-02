from __future__ import annotations

import json
import sys
from typing import IO, Any

from liner.types import CompiledSource, CompileWarning, SourceSpec

PREVIEW_CHARS = 240


class JsonEventReporter:
    """Emits NDJSON progress events to a stream (default: stdout)."""

    def __init__(self, stream: IO[str] | None = None) -> None:
        self._stream = stream or sys.stdout

    def _emit(self, event: dict[str, Any]) -> None:
        self._stream.write(json.dumps(event, ensure_ascii=False) + "\n")
        self._stream.flush()

    def on_start(self, total: int) -> None:
        self._emit({"type": "start", "total": total})

    def on_source_start(self, spec: SourceSpec) -> None:
        self._emit(
            {
                "type": "source_start",
                "spec": {
                    "type": spec.type,
                    "url": spec.url,
                    "note": spec.note,
                    "section": spec.section,
                    "priority": spec.priority,
                },
            }
        )

    def on_source_done(self, item: CompiledSource) -> None:
        content = item.content
        body_preview = ""
        if content is not None:
            body_preview = content.body[:PREVIEW_CHARS]

        self._emit(
            {
                "type": "source_cached" if item.cached else "source_done",
                "url": item.spec.url,
                "title": content.title if content else None,
                "author": content.author if content else None,
                "published_at": content.published_at if content else None,
                "duration_seconds": content.duration_seconds if content else None,
                "body_chars": len(content.body) if content else 0,
                "body_preview": body_preview,
                "metadata": dict(content.metadata) if content else {},
            }
        )

    def on_source_failed(self, spec: SourceSpec, warning: CompileWarning) -> None:
        self._emit(
            {
                "type": "source_failed",
                "url": spec.url,
                "message": warning.message,
                "severity": warning.severity,
            }
        )

    def on_finish(self) -> None:
        self._emit({"type": "finish"})


def emit_result(result_payload: dict[str, Any], stream: IO[str] | None = None) -> None:
    """Emit the final `result` event carrying the full compiled output."""
    out = stream or sys.stdout
    out.write(json.dumps({"type": "result", "payload": result_payload}, ensure_ascii=False) + "\n")
    out.flush()


def compile_result_to_payload(result: Any, project: Any = None) -> dict[str, Any]:
    """Build the `result` event payload for the TUI to consume.

    Reports the project folder, the compiled MIXTAPE.md path, the per-source
    files written, and warnings. `project` is a ProjectFolder; passed in
    explicitly so this module stays free of project-IO dependencies.
    """
    from liner.output.mixtape import written_source_paths
    from liner.project import ProjectFolder
    from liner.types import CompileResult

    assert isinstance(result, CompileResult)
    tape = result.tape
    tape_meta = {
        "title": tape.title,
        "description": tape.description,
        "version": tape.version,
        "curator": tape.curator,
        "tags": list(tape.tags),
        "created": tape.created,
        "updated": tape.updated,
        "license": tape.license,
        "homepage": tape.homepage,
        "mode": tape.mode,
        "jtbd": tape.jtbd,
        "methodology_version": tape.methodology_version,
    }

    payload: dict[str, Any] = {
        "tape": tape_meta,
        "compiled_at": result.compiled_at.isoformat(),
        "warnings": [
            {"url": w.url, "message": w.message, "severity": w.severity}
            for w in result.warnings
        ],
        "summary": {
            "total": result.total_attempted,
            "succeeded": result.total_succeeded,
            "failed": result.total_attempted - result.total_succeeded,
        },
    }

    if project is not None:
        assert isinstance(project, ProjectFolder)
        payload["folder"] = str(project.path)
        payload["mixtape_path"] = str(project.mixtape_path)
        payload["sources"] = written_source_paths(project, result)
    else:
        payload["sources"] = [
            {
                "index": index,
                "url": item.spec.url,
                "type": item.spec.type,
                "section": item.spec.section,
                "title": item.content.title if item.content else None,
                "succeeded": item.content is not None,
            }
            for index, item in enumerate(result.sources, start=1)
        ]

    return payload


