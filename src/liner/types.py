from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from typing import Any, Literal

SourceType = Literal["youtube", "web", "local_file"]
Priority = Literal["required", "optional"]
Mode = Literal["quick", "methodology"]
Render = Literal["server", "js"]
WarningSeverity = Literal["warning", "error"]
# Role this source plays in the corpus — see the docstring on SourceSpec.kind
# in the TUI's types.ts for the role definitions. The set of values must stay
# in sync between Python and TS.
SourceKind = Literal["reference", "principle", "prescription", "example"]


@dataclass(frozen=True, slots=True)
class SourceSpec:
    type: SourceType
    # For youtube/web: the source URL. For local_file: empty string (use `path` instead).
    url: str = ""
    note: str | None = None
    section: str | None = None
    priority: Priority = "required"
    # For web sources only: server (default) or js. None means "use default".
    render: Render | None = None
    # For local_file sources only:
    path: str | None = None
    citation: str | None = None
    # Role this source plays in the corpus. Optional — None means "unspecified"
    # and renders without a kind label.
    kind: SourceKind | None = None


@dataclass(frozen=True, slots=True)
class JtbdClarification:
    """A single Q&A pair captured by the wizard's JTBD-clarify step.

    The wizard generates 3-4 targeted questions after the user types the raw
    JTBD; the answers sharpen the framing for Phase 1.
    """

    question: str
    answer: str


@dataclass(frozen=True, slots=True)
class SourceContent:
    title: str
    url: str
    body: str
    fetched_at: str
    author: str | None = None
    published_at: str | None = None
    duration_seconds: int | None = None
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass(frozen=True, slots=True)
class Tape:
    title: str
    description: str
    version: int
    curator: str
    sources: tuple[SourceSpec, ...]
    tags: tuple[str, ...] = ()
    created: str | None = None
    updated: str | None = None
    license: str | None = None
    homepage: str | None = None
    mode: Mode | None = None
    jtbd: str | None = None
    # Sharpening Q&A captured by the wizard's JTBD-clarify step. Each pair
    # records one question the agent asked and the curator's answer; Phase 1
    # reads these alongside jtbd when drafting the knowledge map.
    jtbd_clarifications: tuple[JtbdClarification, ...] = ()
    methodology_version: str | None = None
    # When this tape was created by `liner replay <other>`, the absolute path
    # of the source folder it was cloned from. This lets later comparison
    # tooling or manual review relate a replayed run to its parent.
    parent: str | None = None


@dataclass(frozen=True, slots=True)
class CompileWarning:
    url: str
    message: str
    severity: WarningSeverity = "warning"


@dataclass(frozen=True, slots=True)
class CompiledSource:
    spec: SourceSpec
    content: SourceContent | None
    cached: bool = False


@dataclass(frozen=True, slots=True)
class CompileResult:
    tape: Tape
    compiled_at: datetime
    sources: tuple[CompiledSource, ...]
    warnings: tuple[CompileWarning, ...] = ()

    @property
    def total_attempted(self) -> int:
        return len(self.sources)

    @property
    def total_succeeded(self) -> int:
        return sum(1 for s in self.sources if s.content is not None)
