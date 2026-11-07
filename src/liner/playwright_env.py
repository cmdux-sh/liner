"""Runtime environment helpers for Playwright-backed source extraction."""

from __future__ import annotations

import os
import sys
from importlib.util import find_spec
from pathlib import Path


def js_rendering_ready() -> bool:
    """Whether `render: js` sources can run here: Playwright and a Chromium download.

    `liner setup-js` installs both. When the browser folder can't be found, this
    says yes and leaves the compile to report a missing browser.
    """
    if find_spec("playwright") is None:
        return False
    browsers = _browsers_path()
    if browsers is None or not browsers.is_dir():
        return browsers is None
    return any(child.name.startswith("chromium") for child in browsers.iterdir())


def _browsers_path() -> Path | None:
    configured = os.environ.get("PLAYWRIGHT_BROWSERS_PATH")
    if configured == "0":
        return None  # Browsers live inside the Playwright package.
    if configured:
        return Path(configured)
    if sys.platform == "darwin":
        return Path.home() / "Library" / "Caches" / "ms-playwright"
    if sys.platform.startswith("linux"):
        return Path(os.environ.get("XDG_CACHE_HOME", Path.home() / ".cache")) / "ms-playwright"
    if sys.platform == "win32":
        local = os.environ.get("LOCALAPPDATA", str(Path.home() / "AppData" / "Local"))
        return Path(local) / "ms-playwright"
    return None


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
