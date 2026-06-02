from __future__ import annotations

from typing import Protocol

from liner.types import CompiledSource, CompileWarning, SourceSpec


class ProgressReporter(Protocol):
    def on_start(self, total: int) -> None: ...
    def on_source_start(self, spec: SourceSpec) -> None: ...
    def on_source_done(self, item: CompiledSource) -> None: ...
    def on_source_failed(self, spec: SourceSpec, warning: CompileWarning) -> None: ...
    def on_finish(self) -> None: ...


class NullProgressReporter:
    def on_start(self, total: int) -> None:
        return None

    def on_source_start(self, spec: SourceSpec) -> None:
        return None

    def on_source_done(self, item: CompiledSource) -> None:
        return None

    def on_source_failed(self, spec: SourceSpec, warning: CompileWarning) -> None:
        return None

    def on_finish(self) -> None:
        return None
