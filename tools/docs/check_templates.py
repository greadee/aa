#!/usr/bin/env python3
"""Validate the GitHub issue and pull-request templates.

The forms are YAML with a small schema. This check catches malformed YAML,
missing required fields, duplicate body ids, and a PR template that has lost its
required sections before reviewers see it on GitHub.
"""

from __future__ import annotations

import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[2]
ISSUE_TEMPLATES = ROOT / ".github" / "ISSUE_TEMPLATE"
PR_TEMPLATE = ROOT / ".github" / "PULL_REQUEST_TEMPLATE.md"
PR_SECTIONS = ["## Summary", "## Related issues", "## Changes", "## Validation", "## Checklist"]


def check_issue_templates() -> list[str]:
    problems: list[str] = []
    for path in sorted(ISSUE_TEMPLATES.glob("*.yml")):
        try:
            data = yaml.safe_load(path.read_text(encoding="utf-8"))
        except yaml.YAMLError as exc:
            problems.append(f"{path}: invalid YAML: {exc}")
            continue
        if not isinstance(data, dict):
            problems.append(f"{path}: expected a YAML mapping")
            continue
        if path.name == "config.yml":
            if "blank_issues_enabled" not in data:
                problems.append(f"{path}: missing blank_issues_enabled")
            continue
        for key in ("name", "description", "body"):
            if key not in data:
                problems.append(f"{path}: missing {key}")
        body = data.get("body")
        if not isinstance(body, list) or not body:
            problems.append(f"{path}: body must be a non-empty list")
            continue
        ids = [item.get("id") for item in body if isinstance(item, dict) and item.get("id")]
        if len(ids) != len(set(ids)):
            problems.append(f"{path}: duplicate body ids")
        if not any(isinstance(item, dict) and item.get("type") == "textarea" for item in body):
            problems.append(f"{path}: expected at least one textarea")
    return problems


def check_pr_template() -> list[str]:
    if not PR_TEMPLATE.exists():
        return [f"{PR_TEMPLATE}: missing"]
    text = PR_TEMPLATE.read_text(encoding="utf-8")
    return [f"{PR_TEMPLATE}: missing section {section}" for section in PR_SECTIONS if section not in text]


def main() -> int:
    problems = check_issue_templates() + check_pr_template()
    if problems:
        print("template validation failures:")
        for problem in problems:
            print(f"  - {problem}")
        return 1
    print("issue and PR templates ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())
