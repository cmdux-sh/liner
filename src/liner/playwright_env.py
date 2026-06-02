"""Runtime environment helpers for Playwright-backed source extraction."""

from __future__ import annotations

import os
import sys
from pathlib import Path


def configure_frozen_playwright_cache() -> None:
    """Pin Playwright's browser cache for PyInstaller-built binaries."""

    if not bool(getattr(sys, "frozen", False)):
        return
    if os.environ.get("PLAYWRIGHT_BROWSERS_PATH"):
        return

    if sys.platform == "darwin":
        cache_root = Path.home() / "Library" / "Caches"
    elif sys.platform.startswith("linux"):
        cache_root = Path(os.environ.get("XDG_CACHE_HOME", Path.home() / ".cache"))
    elif sys.platform == "win32":
        cache_root = Path(os.environ.get("LOCALAPPDATA", Path.home() / "AppData" / "Local"))
    else:
        return

    os.environ["PLAYWRIGHT_BROWSERS_PATH"] = str(cache_root / "ms-playwright")
