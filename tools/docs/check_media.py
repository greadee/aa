#!/usr/bin/env python3
"""Validate repository media references.

Checks every Markdown image reference in README.md, CONTRIBUTING.md, and docs/
for: a non-empty alt text, an existing local target, a size under 2 MiB, and
well-formed SVG markup. External image URLs are ignored; the docs link check
owns those.
"""

from __future__ import annotations

import re
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
MAX_BYTES = 2 * 1024 * 1024
IMAGE_RE = re.compile(r"!\[([^\]]*)\]\(([^)]+)\)")


def markdown_files() -> list[Path]:
    files = [ROOT / "README.md", ROOT / "CONTRIBUTING.md"]
    files.extend(sorted((ROOT / "docs").rglob("*.md")))
    return [path for path in files if path.exists()]


def references(path: Path) -> list[tuple[str, str]]:
    text = path.read_text(encoding="utf-8")
    refs: list[tuple[str, str]] = []
    fence = False
    for line in text.splitlines():
        if line.strip().startswith("```"):
            fence = not fence
            continue
        if fence:
            continue
        for match in IMAGE_RE.finditer(line):
            refs.append((match.group(1).strip(), match.group(2).strip()))
    return refs


def check() -> list[str]:
    problems: list[str] = []
    checked = 0
    for path in markdown_files():
        for alt, target in references(path):
            if target.startswith(("http://", "https://", "mailto:")):
                continue
            if not alt:
                problems.append(f"{path}: image has empty alt text: {target}")
            asset = (path.parent / target.split(" ", 1)[0]).resolve()
            if not asset.exists():
                problems.append(f"{path}: image target is missing: {target}")
                continue
            checked += 1
            if asset.stat().st_size > MAX_BYTES:
                problems.append(f"{asset}: exceeds {MAX_BYTES} bytes")
            if asset.suffix.lower() == ".svg":
                try:
                    ET.parse(asset)
                except ET.ParseError as exc:
                    problems.append(f"{asset}: invalid SVG: {exc}")
    print(f"checked {checked} local media reference(s)")
    return problems


def main() -> int:
    problems = check()
    if problems:
        print("media validation failures:")
        for problem in problems:
            print(f"  - {problem}")
        return 1
    print("media validation ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())
