from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[2]
WEB = ROOT / "src" / "aa_sifter" / "desktop" / "web"


def _cli(*args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [sys.executable, "-m", "aa_sifter.cli.main", *args],
        cwd=ROOT,
        capture_output=True,
        text=True,
        timeout=60,
    )


@pytest.mark.smoke
def test_desktop_package_imports():
    import aa_sifter.desktop

    assert aa_sifter.desktop.ComputeSifterService


@pytest.mark.smoke
def test_launch_help():
    result = _cli("launch", "--help")
    assert result.returncode == 0
    assert "query" in result.stdout or "launch" in result.stdout


@pytest.mark.smoke
def test_serve_help():
    assert _cli("serve", "--help").returncode == 0


@pytest.mark.smoke
def test_web_assets_present():
    assert (WEB / "index.html").exists()
    assert (WEB / "app.js").exists()
    assert (WEB / "styles.css").exists()
    assert "__SIFTER_TOKEN__" in (WEB / "index.html").read_text(encoding="utf-8")
    # Offline: the UI must not reference remote CDNs.
    combined = "".join(
        (WEB / name).read_text(encoding="utf-8") for name in ("index.html", "app.js")
    )
    assert "http://" not in combined.replace("http://127.0.0.1", "")
    assert "https://" not in combined


@pytest.mark.smoke
def test_javascript_syntax_when_node_available():
    node = shutil.which("node")
    if node is None:
        pytest.skip("node is not installed")
    result = subprocess.run(
        [node, "--check", str(WEB / "app.js")],
        capture_output=True,
        text=True,
        timeout=60,
    )
    assert result.returncode == 0, result.stderr
