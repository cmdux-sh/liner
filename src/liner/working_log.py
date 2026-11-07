"""Small logs an agent appends to inside a liner's working folder."""

from __future__ import annotations

import os
import time
from collections.abc import Iterator
from contextlib import contextmanager
from pathlib import Path
from typing import Any
from uuid import uuid4

import yaml

LOCK_WAIT_SECONDS = 5.0
LOCK_EXPIRY_SECONDS = 30.0


class WorkingLogError(RuntimeError):
    """A working log could not be read or changed."""


@contextmanager
def locked(path: Path) -> Iterator[None]:
    """Hold a log for one change. A lock left by a crash expires."""
    lock = path.with_name(f"{path.name}.lock")
    lock.parent.mkdir(parents=True, exist_ok=True)
    deadline = time.monotonic() + LOCK_WAIT_SECONDS
    while True:
        try:
            os.close(os.open(lock, os.O_CREAT | os.O_EXCL | os.O_WRONLY))
            break
        except FileExistsError:
            if _lock_is_abandoned(lock):
                lock.unlink(missing_ok=True)
            elif time.monotonic() > deadline:
                raise WorkingLogError(
                    f"Another process is changing {path.name}. Try again."
                ) from None
            else:
                time.sleep(0.05)
    try:
        yield
    finally:
        lock.unlink(missing_ok=True)


def read_yaml(path: Path) -> Any:
    try:
        return yaml.safe_load(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, yaml.YAMLError) as error:
        raise WorkingLogError(f"{path} could not be read: {error}") from error


def write_yaml(path: Path, value: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f"{path.name}.{uuid4().hex}.tmp")
    body = yaml.safe_dump(value, sort_keys=False, allow_unicode=True)
    try:
        temporary.write_text(body, encoding="utf-8")
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def _lock_is_abandoned(lock: Path) -> bool:
    try:
        return time.time() - lock.stat().st_mtime > LOCK_EXPIRY_SECONDS
    except OSError:
        return False
