#!/usr/bin/env python3
"""Check Go/TypeScript/Python binding parity for contracts v2.

The gate compares the structural shape of every Go contract struct against the
TypeScript and Python bindings in both directions:

* top-level type coverage;
* field presence and extras;
* field JSON names (including embedded Go structs, TS ``extends``, and Python
  ``TypedDict`` bases);
* required/optional parity (Go pointer or ``omitempty`` vs TS ``?`` vs Python
  ``NotRequired``/``total=False``);
* type categories (string, number, boolean, array, object, any).

Go shapes are read with ``go/ast`` by ``tools/goastdump`` so grouped
declarations and struct tags are parsed correctly. This is a structural gate,
not a full type-system equivalence proof; runtime fixture conformance covers
value validation.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[2]
CONTRACTS = ROOT / "contracts"
GO_DIR = Path("go/v2")
TS = CONTRACTS / "typescript" / "v2" / "index.ts"
PY = CONTRACTS / "python" / "aa_contracts" / "v2.py"


# ---------------------------------------------------------------------------
# Go shapes
# ---------------------------------------------------------------------------


def load_go_dir(directory: Path) -> dict[str, Any]:
    """Read Go struct shapes through the go/ast helper."""
    completed = subprocess.run(
        ["go", "run", "./tools/goastdump", "-dir", str(directory)],
        cwd=CONTRACTS,
        check=True,
        capture_output=True,
        text=True,
    )
    return json.loads(completed.stdout)


def expand_go(types: dict[str, Any], name: str, seen: tuple[str, ...] = ()) -> dict[str, dict[str, Any]]:
    if name in seen:
        return {}
    shape = types.get(name)
    if not shape or shape.get("kind") != "struct":
        return {}
    fields = {field["jsonName"]: field for field in shape.get("fields", [])}
    for base in shape.get("embedded", []):
        fields = {**expand_go(types, base, seen + (name,)), **fields}
    return fields


# ---------------------------------------------------------------------------
# TypeScript shapes
# ---------------------------------------------------------------------------


def _matching_brace(text: str, start: int) -> int:
    depth = 0
    for i in range(start, len(text)):
        if text[i] == "{":
            depth += 1
        elif text[i] == "}":
            depth -= 1
            if depth == 0:
                return i
    return -1


def parse_ts(text: str) -> dict[str, Any]:
    aliases: dict[str, str] = {}
    for match in re.finditer(r"(?ms)^export\s+type\s+(\w+)\s*=\s*([^;]+);", text):
        aliases[match.group(1)] = " ".join(match.group(2).split())

    interfaces: dict[str, dict[str, Any]] = {}
    pattern = re.compile(r"export\s+interface\s+(\w+)(?:\s+extends\s+([^\{]+))?\s*\{")
    for match in pattern.finditer(text):
        name, bases = match.group(1), (match.group(2) or "").strip()
        start = match.end() - 1
        end = _matching_brace(text, start)
        if end < 0:
            continue
        body = text[start + 1 : end]
        fields: dict[str, dict[str, Any]] = {}
        for field_match in re.finditer(r"(?m)^\s*(\w+)(\??):\s*([^;]+);", body):
            field_type = " ".join(field_match.group(3).split())
            fields[field_match.group(1)] = {
                "type": field_type,
                "optional": bool(field_match.group(2)) or _ts_has_undefined(field_type),
            }
        interfaces[name] = {
            "fields": fields,
            "bases": [base.strip().split(".")[-1] for base in bases.split(",") if base.strip()],
        }
    return {"interfaces": interfaces, "aliases": aliases}


def expand_ts(model: dict[str, Any], name: str, seen: tuple[str, ...] = ()) -> dict[str, dict[str, Any]]:
    if name in seen:
        return {}
    shape = model["interfaces"].get(name)
    if not shape:
        return {}
    fields: dict[str, dict[str, Any]] = {}
    for base in shape["bases"]:
        fields.update(expand_ts(model, base, seen + (name,)))
    fields.update(shape["fields"])
    return fields


def _ts_has_undefined(field_type: str) -> bool:
    return any(part.strip() == "undefined" for part in _split_top(field_type, "|"))


# ---------------------------------------------------------------------------
# Python shapes
# ---------------------------------------------------------------------------


def _split_top(value: str, separator: str = ",") -> list[str]:
    parts: list[str] = []
    depth = 0
    current = ""
    for char in value:
        if char in "[<(":
            depth += 1
        elif char in "]>)":
            depth -= 1
        if char == separator and depth == 0:
            parts.append(current.strip())
            current = ""
        else:
            current += char
    if current.strip():
        parts.append(current.strip())
    return parts


def parse_py(text: str) -> dict[str, Any]:
    classes: dict[str, dict[str, Any]] = {}
    pattern = re.compile(r"(?m)^class\s+(\w+)\(([^)]*)\):\n((?:[ \t]+.*\n|\n)*)")
    for match in pattern.finditer(text):
        name, bases, body = match.group(1), match.group(2), match.group(3)
        base_parts = [base.strip() for base in bases.split(",") if base.strip()]
        total_false = any(part.replace(" ", "") == "total=False" for part in base_parts)
        bases_out = [
            base.split(".")[-1]
            for base in base_parts
            if not base.startswith("total=") and "." not in base
        ]
        fields: dict[str, dict[str, Any]] = {}
        for field_match in re.finditer(r"(?m)^\s{4}(\w+):\s*(.+?)\s*$", body):
            field_name = field_match.group(1)
            field_type = field_match.group(2).strip()
            json_name = "class" if field_name == "class_" else field_name
            optional = total_false or _py_has_not_required(field_type)
            fields[json_name] = {"type": field_type, "optional": optional}
        classes[name] = {"fields": fields, "bases": bases_out, "total_false": total_false}
    return {"classes": classes}


def expand_py(model: dict[str, Any], name: str, seen: tuple[str, ...] = ()) -> dict[str, dict[str, Any]]:
    if name in seen:
        return {}
    shape = model["classes"].get(name)
    if not shape:
        return {}
    fields: dict[str, dict[str, Any]] = {}
    for base in shape["bases"]:
        fields.update(expand_py(model, base, seen + (name,)))
    fields.update(shape["fields"])
    return fields


def _py_has_not_required(field_type: str) -> bool:
    return bool(re.match(r"^(NotRequired|Required)\[", field_type.strip())) and field_type.strip().startswith("NotRequired[")


# ---------------------------------------------------------------------------
# Type categories
# ---------------------------------------------------------------------------


def go_category(expr: str, types: dict[str, Any]) -> tuple[str, ...]:
    expr = expr.strip()
    while expr.startswith("*"):
        expr = expr[1:].strip()
    shape = types.get(expr)
    if shape and shape.get("kind") == "alias":
        return go_category(shape.get("alias", "any"), types)
    if expr.startswith("[]"):
        return ("array",) + go_category(expr[2:], types)
    if expr.startswith("map["):
        return ("object",)
    if expr in {"string"}:
        return ("string",)
    if expr in {"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64"}:
        return ("number",)
    if expr == "bool":
        return ("boolean",)
    if expr in {"any", "interface{}"}:
        return ("any",)
    return ("object",)


def ts_category(expr: str, model: dict[str, Any], seen: tuple[str, ...] = ()) -> tuple[str, ...]:
    expr = " ".join(expr.split())
    parts = [
        part
        for part in _split_top(expr.strip().strip("|"), "|")
        if part and part not in {"undefined", "null"}
    ]
    if not parts:
        return ("any",)
    if len(parts) > 1:
        categories = [ts_category(part, model, seen) for part in parts]
        if all(category[0] == "string" for category in categories):
            return ("string",)
        if all(category[0] == "number" for category in categories):
            return ("number",)
        return categories[0]
    value = parts[0]
    if value.startswith(('"', "'")):
        return ("string",)
    if value.endswith("[]"):
        return ("array",) + ts_category(value[:-2], model, seen)
    array_match = re.match(r"Array<(.+)>$", value)
    if array_match:
        return ("array",) + ts_category(array_match.group(1), model, seen)
    if value.startswith("Record<"):
        return ("object",)
    if value.startswith("{") and value.endswith("}"):
        return ("object",)
    if value in {"string"}:
        return ("string",)
    if value in {"number"}:
        return ("number",)
    if value in {"boolean", "bool"}:
        return ("boolean",)
    if value in {"unknown", "any"}:
        return ("any",)
    alias = model["aliases"].get(value)
    if alias is not None and value not in seen:
        return ts_category(alias, model, seen + (value,))
    return ("object",)


def py_category(expr: str, model: dict[str, Any], seen: tuple[str, ...] = ()) -> tuple[str, ...]:
    expr = " ".join(expr.split())
    for wrapper in ("NotRequired", "Required", "Optional"):
        if expr.startswith(wrapper + "[") and expr.endswith("]"):
            expr = expr[len(wrapper) + 1 : -1].strip()
    if expr.startswith(("List[", "list[")):
        return ("array",) + py_category(expr[expr.index("[") + 1 : -1], model, seen)
    if expr.startswith(("Dict[", "dict[")):
        return ("object",)
    if expr.startswith("Union["):
        parts = _split_top(expr[6:-1], ",")
        categories = [py_category(part, model, seen) for part in parts]
        if all(category[0] == "string" for category in categories):
            return ("string",)
        return categories[0]
    if expr.startswith("Literal["):
        return ("string",)
    if expr == "str":
        return ("string",)
    if expr in {"int", "float"}:
        return ("number",)
    if expr == "bool":
        return ("boolean",)
    if expr in {"Any", "any", "object"}:
        return ("any",)
    if expr in {"None", "NoneType"}:
        return ("null",)
    return ("object",)


# ---------------------------------------------------------------------------
# Diff
# ---------------------------------------------------------------------------


def _category_name(category: tuple[str, ...]) -> str:
    return "[" + " of ".join(category) + "]"


def diff_models(go: dict[str, Any], ts: dict[str, Any], py: dict[str, Any]) -> list[str]:
    problems: list[str] = []
    go_types = go["types"]
    for name in sorted(go_types):
        if go_types[name].get("kind") != "struct":
            continue
        go_fields = expand_go(go_types, name)
        ts_fields = expand_ts(ts, name)
        py_fields = expand_py(py, name)

        if name not in ts["interfaces"]:
            problems.append(f"missing TypeScript declaration: {name}")
        if name not in py["classes"]:
            problems.append(f"missing Python TypedDict: {name}")

        for field_name, go_field in sorted(go_fields.items()):
            go_cat = go_category(go_field["goType"], go_types)
            if field_name not in ts_fields:
                problems.append(f"TS {name}: missing field {field_name}")
            else:
                ts_field = ts_fields[field_name]
                if go_field["optional"] != ts_field["optional"]:
                    problems.append(
                        f"TS {name}.{field_name}: optionality mismatch "
                        f"(go optional={go_field['optional']}, ts optional={ts_field['optional']})"
                    )
                ts_cat = ts_category(ts_field["type"], ts)
                if go_cat != ts_cat:
                    problems.append(
                        f"TS {name}.{field_name}: type mismatch "
                        f"(go {_category_name(go_cat)}, ts {ts_field['type']} {_category_name(ts_cat)})"
                    )
            if field_name not in py_fields:
                problems.append(f"PY {name}: missing field {field_name}")
            else:
                py_field = py_fields[field_name]
                if go_field["optional"] != py_field["optional"]:
                    problems.append(
                        f"PY {name}.{field_name}: optionality mismatch "
                        f"(go optional={go_field['optional']}, py optional={py_field['optional']})"
                    )
                py_cat = py_category(py_field["type"], py)
                if go_cat != py_cat:
                    problems.append(
                        f"PY {name}.{field_name}: type mismatch "
                        f"(go {_category_name(go_cat)}, py {py_field['type']} {_category_name(py_cat)})"
                    )

        for field_name in sorted(set(ts_fields) - set(go_fields)):
            problems.append(f"TS {name}: undocumented field {field_name}")
        for field_name in sorted(set(py_fields) - set(go_fields)):
            problems.append(f"PY {name}: undocumented field {field_name}")
    return problems


def compare() -> list[str]:
    go = load_go_dir(GO_DIR)
    ts = parse_ts(TS.read_text(encoding="utf-8"))
    py = parse_py(PY.read_text(encoding="utf-8"))
    return diff_models(go, ts, py)


def main() -> int:
    problems = compare()
    if problems:
        print("binding parity failures:")
        for problem in problems:
            print(f"  - {problem}")
        return 1
    print("binding parity ok (types, fields, optionality, and direction)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
