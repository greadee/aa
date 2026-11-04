#!/usr/bin/env python3
"""Check local Markdown links and anchors.

External URLs, mail links, and data URLs are skipped. The current lychee job runs
with --offline, so external link rot is not checked by default. This script is
deterministic and offline so it can run locally and on every pull request.
"""

from __future__ import annotations

import re
import sys
import urllib.parse
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SKIP_DIRS = {".git", "node_modules", "vendor", "dist", "build", "__pycache__"}
LINK_RE = re.compile(r"(?<!!)\[[^\]]*\]\(([^)]+)\)")
IMAGE_RE = re.compile(r"!\[[^\]]*\]\(([^)]+)\)")
HEADING_RE = re.compile(r"^(#{1,6})\s+(.*)$", re.M)


def markdown_files() -> list[Path]:
    files: list[Path] = []
    for path in ROOT.rglob("*.md"):
        if any(part in SKIP_DIRS for part in path.parts):
            continue
        files.append(path)
    return sorted(files)


def without_fences(text: str) -> str:
    lines: list[str] = []
    fence = False
    for line in text.splitlines():
        if line.lstrip().startswith("```"):
            fence = not fence
            lines.append("")
            continue
        lines.append("" if fence else line)
    return "\n".join(lines)


def slug(heading: str) -> str:
    heading = heading.strip().lower()
    heading = re.sub(r"`([^`]*)`", r"\1", heading)
    heading = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", heading)
    heading = re.sub(r"[^\w\- ]", "", heading)
    return heading.replace(" ", "-")


def anchors(text: str) -> set[str]:
    found: set[str] = set()
    counts: dict[str, int] = {}
    for match in HEADING_RE.finditer(without_fences(text)):
        base = slug(match.group(2))
        count = counts.get(base, 0)
        counts[base] = count + 1
        found.add(base if count == 0 else f"{base}-{count}")
    return found


def is_external(target: str) -> bool:
    return target.startswith(("http://", "https://", "mailto:", "tel:", "data:"))


def check() -> tuple[int, list[str]]:
    problems: list[str] = []
    checked = 0
    files = markdown_files()
    anchor_cache = {path.resolve(): anchors(path.read_text(encoding="utf-8")) for path in files}

    for path in files:
        text = without_fences(path.read_text(encoding="utf-8"))
        for regex in (LINK_RE, IMAGE_RE):
            for match in regex.finditer(text):
                target = match.group(1).strip()
                if not target or is_external(target):
                    continue
                target = target.strip("<>")
                target = target.split(" ", 1)[0]
                file_part, _, fragment = target.partition("#")
                file_part = urllib.parse.unquote(file_part)
                fragment = urllib.parse.unquote(fragment)
                destination = (path.parent / file_part).resolve() if file_part else path.resolve()
                checked += 1
                try:
                    destination.relative_to(ROOT)
                except ValueError:
                    problems.append(f"{path}: link escapes repository root: {target}")
                    continue
                if file_part and not destination.exists():
                    problems.append(f"{path}: missing target: {target}")
                    continue
                if fragment and destination.suffix.lower() == ".md":
                    if destination not in anchor_cache:
                        anchor_cache[destination] = anchors(destination.read_text(encoding="utf-8"))
                    if fragment not in anchor_cache[destination]:
                        problems.append(f"{path}: missing anchor: {target}")
    return checked, problems


def main() -> int:
    checked, problems = check()
    if problems:
        print("local link failures:")
        for problem in problems:
            print(f"  - {problem}")
        return 1
    print(f"local links and anchors ok ({checked} checked)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
