from __future__ import annotations

from rich.console import Console

from liner.types import CompiledSource, CompileWarning, SourceSpec


class RichProgressReporter:
    def __init__(self, console: Console | None = None) -> None:
        self._console = console or Console(stderr=True)
        self._index = 0
        self._total = 0

    def on_start(self, total: int) -> None:
        self._total = total
        self._console.print(f"[bold]Compiling {total} source{'s' if total != 1 else ''}…[/]")

    def on_source_start(self, spec: SourceSpec) -> None:
        self._index += 1
        self._console.print(
            f"  [{self._index}/{self._total}] [cyan]{spec.type}[/] {spec.url}"
        )

    def on_source_done(self, item: CompiledSource) -> None:
        marker = "[green]cached[/]" if item.cached else "[green]ok[/]"
        self._console.print(f"    {marker}")

    def on_source_failed(self, spec: SourceSpec, warning: CompileWarning) -> None:
        self._console.print(f"    [yellow]warn[/] {warning.message}")

    def on_finish(self) -> None:
        self._console.print("[bold]Done.[/]")
