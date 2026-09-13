from __future__ import annotations

from typing import Any


def dumps(data: dict[str, Any]) -> str:
    """Serialize a nested dict to TOML.

    Supports the scalar/list/dict shapes used by Compute Sifter profiles.
    ``None`` values are omitted by the caller.
    """
    lines: list[str] = []
    _dump_table(data, [], lines)
    return "\n".join(lines).rstrip() + "\n"


def _dump_table(data: dict[str, Any], path: list[str], lines: list[str]) -> None:
    scalars: dict[str, Any] = {}
    tables: dict[str, dict[str, Any]] = {}
    arrays: dict[str, list[dict[str, Any]]] = {}
    for key, value in data.items():
        if isinstance(value, dict):
            tables[key] = value
        elif isinstance(value, list) and value and all(isinstance(item, dict) for item in value):
            arrays[key] = value
        else:
            scalars[key] = value

    for key, value in scalars.items():
        lines.append(f"{key} = {_scalar(value)}")

    for key, value in tables.items():
        full = [*path, key]
        lines.append("")
        lines.append(f"[{'.'.join(full)}]")
        _dump_table(value, full, lines)

    for key, items in arrays.items():
        full = [*path, key]
        for item in items:
            lines.append("")
            lines.append(f"[[{'.'.join(full)}]]")
            _dump_table(item, full, lines)


def _scalar(value: Any) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, str):
        escaped = value.replace("\\", "\\\\").replace('"', '\\"')
        return f'"{escaped}"'
    if isinstance(value, (int, float)):
        return repr(value)
    if isinstance(value, list):
        return "[" + ", ".join(_scalar(item) for item in value) + "]"
    if isinstance(value, dict):
        return _inline_table(value)
    return f'"{value}"'


def _inline_table(value: dict[str, Any]) -> str:
    parts = [f"{key} = {_scalar(item)}" for key, item in value.items()]
    return "{ " + ", ".join(parts) + " }"
