from __future__ import annotations

import os
from pathlib import Path


def app_data_dir() -> Path:
    override = os.environ.get("SIFTER_DATA_DIR")
    if override:
        return Path(os.path.expanduser(override))
    return Path.home() / ".aa_sifter"


def logs_dir() -> Path:
    return app_data_dir() / "logs"


def state_dir() -> Path:
    return app_data_dir() / "state"


def desktop_state_file() -> Path:
    return state_dir() / "desktop.json"


def log_file(name: str = "application.log") -> Path:
    return logs_dir() / name


def ensure_dirs() -> None:
    for directory in (app_data_dir(), logs_dir(), state_dir()):
        directory.mkdir(parents=True, exist_ok=True)
