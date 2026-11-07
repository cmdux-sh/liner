"""Record whether a liner improved an answer. Agents run the test; the person judges it."""

from __future__ import annotations

import re
import secrets
from datetime import UTC, datetime
from pathlib import Path
from typing import Any
from uuid import uuid4

from liner.maintenance import ProjectInspectionError, inspect_project
from liner.project import ProjectFolder
from liner.working_log import WorkingLogError, locked, read_yaml, write_yaml

TESTS_CONTRACT = "liner.empirical_tests"
CONTRACT_VERSION = 1
TESTS_PATH = "working/empirical-tests.yaml"
PENDING_PATH = "working/empirical-tests-pending.yaml"
BLIND_FOLDER = "working/blind-tests"
VERDICTS = ("better", "same", "worse")
CHOICES = ("a", "b", "same")

_MAX_QUESTION_CHARS = 1000
_MAX_ANSWER_CHARS = 20000
_MAX_NOTE_CHARS = 1000


class EmpiricalTestError(WorkingLogError):
    """A test could not be recorded or read."""


# Marks that would tell the person which answer used the liner. The answer written
# with it cites its sources by habit; the one written without it can't.
_TELLS = (
    ("a link", re.compile(r"https?://", re.IGNORECASE)),
    ("a footnote", re.compile(r"\[\^[^\]]+\]")),
    (
        "a source list",
        re.compile(
            r"^[#*_\s]*(sources?|references)[*_\s]*:|^#+\s*(sources?|references)\s*$",
            re.IGNORECASE | re.MULTILINE,
        ),
    ),
    ("the word liner or mixtape", re.compile(r"\b(liner|mixtape)s?\b", re.IGNORECASE)),
)


def add_test(
    project: ProjectFolder,
    question: str,
    *,
    without_answer: str,
    with_answer: str,
    verdict: str,
    note: str = "",
) -> dict[str, Any]:
    """Keep one comparison, tied to the liner's content at the time it was judged."""
    if verdict not in VERDICTS:
        raise EmpiricalTestError(f"The verdict must be one of {', '.join(VERDICTS)}.")
    asked, without_text, with_text = _checked(project, question, without_answer, with_answer)
    if len(note) > _MAX_NOTE_CHARS:
        raise EmpiricalTestError("Keep the question and note under 1000 characters.")
    return _record(project, asked, without_text, with_text, verdict, note, blind=False)


def start_blind_test(
    project: ProjectFolder,
    question: str,
    *,
    without_answer: str,
    with_answer: str,
) -> dict[str, Any]:
    """Save both answers as A and B in random order, for the person to judge unlabeled."""
    asked, without_text, with_text = _checked(project, question, without_answer, with_answer)
    for label, text in (("without", without_text), ("with", with_text)):
        tells = [name for name, pattern in _TELLS if pattern.search(text)]
        if tells:
            raise EmpiricalTestError(
                f"The answer {label} the liner has {', '.join(tells)}, which would show which "
                "answer is which. Remove links, footnotes, source lists, and mentions of the "
                "liner from both answers, then start the test again."
            )
    test_id = uuid4().hex[:8]
    with_is = secrets.choice(("a", "b"))
    first, second = (with_text, without_text) if with_is == "a" else (without_text, with_text)
    sheet = project.corpus_path / BLIND_FOLDER / f"{test_id}.md"
    sheet.parent.mkdir(parents=True, exist_ok=True)
    sheet.write_text(
        f"# {asked}\n\n## Answer A\n\n{first}\n\n## Answer B\n\n{second}\n", encoding="utf-8"
    )
    pending = _pending_path(project.corpus_path)
    with locked(pending):
        write_yaml(
            pending,
            {
                "version": 1,
                "pending": [
                    *_read_pending(pending),
                    {
                        "id": test_id,
                        "question": asked,
                        "without_answer": without_text,
                        "with_answer": with_text,
                        "with_is": with_is,
                    },
                ],
            },
        )
    return {"id": test_id, "question": asked, "path": str(sheet)}


def judge_blind_test(
    project: ProjectFolder, test_id: str, preferred: str, *, note: str = ""
) -> dict[str, Any]:
    """Turn the person's choice into a verdict, then say which answer was which."""
    if preferred not in CHOICES:
        raise EmpiricalTestError(f"The choice must be one of {', '.join(CHOICES)}.")
    if len(note) > _MAX_NOTE_CHARS:
        raise EmpiricalTestError("Keep the note under 1000 characters.")
    pending = _pending_path(project.corpus_path)
    with locked(pending):
        waiting = _read_pending(pending)
        test = next((item for item in waiting if item.get("id") == test_id), None)
        if test is None:
            raise EmpiricalTestError(f"No blind test is waiting with the id {test_id}.")
        verdict = (
            "same" if preferred == "same" else "better" if preferred == test["with_is"] else "worse"
        )
        # Record first, so a failure leaves the test waiting instead of losing it.
        result = _record(
            project,
            test["question"],
            test["without_answer"],
            test["with_answer"],
            verdict,
            note,
            blind=True,
            test_id=test_id,
        )
        write_yaml(
            pending,
            {"version": 1, "pending": [item for item in waiting if item is not test]},
        )
    (project.corpus_path / BLIND_FOLDER / f"{test_id}.md").unlink(missing_ok=True)
    return {**result, "with_liner_was": str(test["with_is"]).upper()}


def _checked(
    project: ProjectFolder, question: str, without_answer: str, with_answer: str
) -> tuple[str, str, str]:
    asked = " ".join(question.split())
    without_text, with_text = without_answer.strip(), with_answer.strip()
    if not asked or not without_text or not with_text:
        raise EmpiricalTestError("A test needs a question and both answers.")
    if len(asked) > _MAX_QUESTION_CHARS:
        raise EmpiricalTestError("Keep the question and note under 1000 characters.")
    if len(without_text) > _MAX_ANSWER_CHARS or len(with_text) > _MAX_ANSWER_CHARS:
        raise EmpiricalTestError(f"Keep each answer under {_MAX_ANSWER_CHARS} characters.")
    if not project.mixtape_path.is_file():
        raise EmpiricalTestError(f"{project.path.name} has no compiled liner to test yet.")
    return asked, without_text, with_text


def _record(
    project: ProjectFolder,
    asked: str,
    without_text: str,
    with_text: str,
    verdict: str,
    note: str,
    *,
    blind: bool,
    test_id: str | None = None,
) -> dict[str, Any]:
    test = {
        "id": test_id or uuid4().hex[:8],
        "question": asked,
        "without_answer": without_text,
        "with_answer": with_text,
        "verdict": verdict,
        "note": note.strip(),
        "blind": blind,
        "revision": _revision(project),
        "tested_at": datetime.now(UTC).isoformat(),
    }
    path = _path(project.corpus_path)
    with locked(path):
        write_yaml(path, {"version": 1, "tests": [*_read(path), test]})
    return _public(test)


def list_tests(project: ProjectFolder) -> dict[str, Any]:
    path = _path(project.corpus_path)
    return {
        "contract": TESTS_CONTRACT,
        "version": CONTRACT_VERSION,
        "slug": project.path.name,
        "path": str(path),
        "summary": summarize_tests(project),
        "tests": [_public(test) for test in _read(path)],
    }


def summarize_tests(project: ProjectFolder) -> dict[str, Any]:
    """Count verdicts. `current` means the latest test judged the liner as it is now."""
    tests = _read(_path(project.corpus_path))
    latest = tests[-1] if tests else None
    return {
        "runs": len(tests),
        "blind": sum(test.get("blind") is True for test in tests),
        **{verdict: sum(test["verdict"] == verdict for test in tests) for verdict in VERDICTS},
        "last_tested": latest["tested_at"] if latest else None,
        "current": latest is not None
        and latest["revision"] is not None
        and latest["revision"] == _revision(project),
    }


def _revision(project: ProjectFolder) -> str | None:
    try:
        return inspect_project(project.path).revision
    except (ProjectInspectionError, OSError):
        return None


def _public(test: dict[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in test.items() if not key.endswith("_answer")}


def _path(corpus: Path) -> Path:
    return corpus / TESTS_PATH


def _pending_path(corpus: Path) -> Path:
    return corpus / PENDING_PATH


def _read_pending(path: Path) -> list[dict[str, Any]]:
    if not path.exists():
        return []
    raw = read_yaml(path)
    pending = raw.get("pending") if isinstance(raw, dict) else None
    if not isinstance(pending, list) or not all(isinstance(item, dict) for item in pending):
        raise EmpiricalTestError(f"{path} is not a test log. Fix it by hand; nothing was changed.")
    return pending


def _read(path: Path) -> list[dict[str, Any]]:
    if not path.exists():
        return []
    raw = read_yaml(path)
    tests = raw.get("tests") if isinstance(raw, dict) else None
    if not isinstance(tests, list) or not all(_valid(test) for test in tests):
        raise EmpiricalTestError(f"{path} is not a test log. Fix it by hand; nothing was changed.")
    return tests


def _valid(test: Any) -> bool:
    return (
        isinstance(test, dict)
        and test.get("verdict") in VERDICTS
        and isinstance(test.get("question"), str)
        and isinstance(test.get("tested_at"), str)
        and "revision" in test
    )
