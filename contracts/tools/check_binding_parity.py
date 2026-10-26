#!/usr/bin/env python3
"""Check Go/TypeScript/Python binding type parity for contracts v2.

This is a structural conformance gate: every Go contract struct must have an
equivalent TypeScript declaration and Python TypedDict. It does not replace the
runtime fixture conformance tests; it prevents silent cross-language drift.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
GO = ROOT / "contracts" / "go" / "v2"
TS = ROOT / "contracts" / "typescript" / "v2" / "index.ts"
PY = ROOT / "contracts" / "python" / "aa_contracts" / "v2.py"


def go_structs() -> set[str]:
    names: set[str] = set()
    for path in sorted(GO.glob("*.go")):
        if path.name.endswith("_test.go"):
            continue
        text = path.read_text(encoding="utf-8")
        names.update(re.findall(r"^type\s+(\w+)\s+struct", text, re.M))
    return names


def ts_types() -> set[str]:
    text = TS.read_text(encoding="utf-8")
    return set(re.findall(r"^export\s+(?:interface|type)\s+(\w+)", text, re.M))


def py_types() -> set[str]:
    text = PY.read_text(encoding="utf-8")
    return set(re.findall(r"^class\s+(\w+)\(", text, re.M))


def main() -> int:
    go = go_structs()
    ts = ts_types()
    py = py_types()
    missing_ts = sorted(go - ts)
    missing_py = sorted(go - py)

    print(f"Go structs: {len(go)}")
    print(f"TypeScript declarations: {len(ts)}")
    print(f"Python TypedDicts: {len(py)}")

    if missing_ts:
        print("Missing TypeScript declarations:")
        for name in missing_ts:
            print(f"  - {name}")
    if missing_py:
        print("Missing Python TypedDicts:")
        for name in missing_py:
            print(f"  - {name}")

    if missing_ts or missing_py:
        return 1
    print("binding parity ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())
