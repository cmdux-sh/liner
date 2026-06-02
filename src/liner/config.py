from __future__ import annotations

import tomllib
from dataclasses import dataclass
from pathlib import Path

DEFAULT_CONFIG_DIR = Path.home() / ".liner"
DEFAULT_CONFIG_PATH = DEFAULT_CONFIG_DIR / "config.toml"
DEFAULT_CACHE_PATH = DEFAULT_CONFIG_DIR / "cache.db"


@dataclass(frozen=True, slots=True)
class CacheConfig:
    youtube_ttl_days: int = 30
    web_ttl_days: int = 7


@dataclass(frozen=True, slots=True)
class FetchConfig:
    timeout_seconds: float = 10.0
    user_agent: str = "liner/0.1 (+https://github.com/cmdux-sh/liner)"
    cookies_file: Path | None = None


@dataclass(frozen=True, slots=True)
class OutputConfig:
    include_preamble: bool = True


@dataclass(frozen=True, slots=True)
class Config:
    cache: CacheConfig = CacheConfig()
    fetch: FetchConfig = FetchConfig()
    output: OutputConfig = OutputConfig()
    config_dir: Path = DEFAULT_CONFIG_DIR
    cache_path: Path = DEFAULT_CACHE_PATH


def load_config(path: Path | None = None) -> Config:
    config_path = path or DEFAULT_CONFIG_PATH
    if not config_path.exists():
        return Config()

    with config_path.open("rb") as f:
        raw = tomllib.load(f)

    cache_raw = raw.get("cache", {})
    fetch_raw = raw.get("fetch", {})
    output_raw = raw.get("output", {})

    cookies_file = fetch_raw.get("cookies_file")
    cookies_path = Path(cookies_file).expanduser() if cookies_file else None

    return Config(
        cache=CacheConfig(
            youtube_ttl_days=int(cache_raw.get("youtube_ttl_days", 30)),
            web_ttl_days=int(cache_raw.get("web_ttl_days", 7)),
        ),
        fetch=FetchConfig(
            timeout_seconds=float(fetch_raw.get("timeout_seconds", 10.0)),
            user_agent=str(
                fetch_raw.get(
                    "user_agent",
                    "liner/0.1 (+https://github.com/cmdux-sh/liner)",
                )
            ),
            cookies_file=cookies_path,
        ),
        output=OutputConfig(
            include_preamble=bool(output_raw.get("include_preamble", True)),
        ),
        config_dir=config_path.parent,
        cache_path=config_path.parent / "cache.db",
    )
