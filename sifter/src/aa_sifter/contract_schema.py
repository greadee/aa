"""Full JSON Schema validation for the aa v1 contract objects.

The JSON Schema files are the source of truth for the cross-module contract
objects. This module loads the bundled v1 schemas and validates instances
against the draft 2020-12 subset the schemas use, so the RPC boundary validates
independently of the generated ``aa_contracts`` Python binding. A schema keyword
outside the supported set is an error, never a silent pass.
"""

from __future__ import annotations

import json
import os
import re
from collections.abc import Mapping
from pathlib import Path
from typing import Any

SCHEMA_DIR_ENV = "SIFTER_SCHEMA_DIR"
_BUNDLED_SCHEMA_DIR = Path(__file__).resolve().parent / "schemas" / "v1"

# kind -> (schema file, pointer into the schema document)
KIND_SCHEMAS: dict[str, tuple[str, str]] = {
    "route_request": ("route.schema.json", "#/$defs/request"),
    "route_response": ("route.schema.json", "#/$defs/response"),
    "memory_record": ("memory-record.schema.json", "#"),
}

_SUPPORTED_KEYWORDS = frozenset(
    {
        "$ref",
        "type",
        "const",
        "enum",
        "required",
        "properties",
        "additionalProperties",
        "items",
        "minItems",
        "maxItems",
        "minLength",
        "maxLength",
        "pattern",
        "format",
        "minimum",
        "maximum",
    }
)
_ANNOTATION_KEYWORDS = frozenset(
    {
        "$schema",
        "$id",
        "title",
        "description",
        "x-contract-version",
        "$defs",
        "$comment",
        "examples",
        "default",
    }
)
_DATETIME_RE = re.compile(r"^\d{4}-\d{2}-\d{2}[Tt]\d{2}:\d{2}:\d{2}(\.\d+)?([Zz]|[+-]\d{2}:\d{2})$")


class SchemaError(ValueError):
    """An instance does not satisfy its JSON Schema."""


def schema_directory() -> Path:
    """The schema directory, honouring the ``SIFTER_SCHEMA_DIR`` override."""
    override = os.environ.get(SCHEMA_DIR_ENV)
    if override:
        return Path(override)
    return _BUNDLED_SCHEMA_DIR


class SchemaStore:
    """Loads schema documents and validates instances against them."""

    def __init__(self, directory: Path | str | None = None) -> None:
        self.directory = Path(directory) if directory is not None else schema_directory()
        self._documents: dict[str, dict[str, Any]] = {}

    def document(self, name: str) -> dict[str, Any]:
        if name not in self._documents:
            path = self.directory / name
            try:
                text = path.read_text(encoding="utf-8")
            except FileNotFoundError as exc:
                raise SchemaError(f"schema {name!r} not found in {self.directory}") from exc
            self._documents[name] = json.loads(text)
        return self._documents[name]

    def resolve(self, reference: str, current_name: str) -> tuple[Any, str]:
        """Resolve a ``$ref`` into ``(schema node, document name)``."""
        file_part, _, pointer = reference.partition("#")
        name = file_part or current_name
        node: Any = self.document(name)
        for token in (token for token in pointer.split("/") if token):
            token = token.replace("~1", "/").replace("~0", "~")
            if not isinstance(node, Mapping) or token not in node:
                raise SchemaError(f"unresolvable $ref {reference!r}")
            node = node[token]
        return node, name

    def validate(self, instance: Any, schema: Any, *, name: str, path: str = "$") -> None:
        if not isinstance(schema, Mapping):
            raise SchemaError(f"{path}: schema node must be an object")
        self._check_keywords(schema, path)
        if "$ref" in schema:
            target, target_name = self.resolve(str(schema["$ref"]), name)
            self.validate(instance, target, name=target_name, path=path)
            return
        if "const" in schema and instance != schema["const"]:
            raise SchemaError(f"{path}: expected {schema['const']!r}")
        if "enum" in schema and instance not in schema["enum"]:
            raise SchemaError(f"{path}: {instance!r} is not one of {schema['enum']!r}")
        expected = schema.get("type")
        if expected is not None and not _matches_type(instance, expected):
            raise SchemaError(f"{path}: expected type {expected!r}")
        if isinstance(instance, Mapping):
            self._validate_object(instance, schema, name, path)
        elif isinstance(instance, str):
            self._validate_string(instance, schema, path)
        elif isinstance(instance, (int, float)) and not isinstance(instance, bool):
            self._validate_number(instance, schema, path)
        elif isinstance(instance, list):
            self._validate_array(instance, schema, name, path)

    # -- helpers -----------------------------------------------------------
    def _check_keywords(self, schema: Mapping[str, Any], path: str) -> None:
        unsupported = set(schema) - _SUPPORTED_KEYWORDS - _ANNOTATION_KEYWORDS
        if unsupported:
            raise SchemaError(f"{path}: unsupported schema keyword(s) {sorted(unsupported)}")

    def _validate_object(
        self, instance: Mapping[str, Any], schema: Mapping[str, Any], name: str, path: str
    ) -> None:
        for field in schema.get("required", []):
            if field not in instance:
                raise SchemaError(f"{path}: missing required property {field!r}")
        properties = schema.get("properties", {})
        extra = schema.get("additionalProperties", True)
        for key, value in instance.items():
            child = f"{path}.{key}"
            if key in properties:
                self.validate(value, properties[key], name=name, path=child)
            elif extra is False:
                raise SchemaError(f"{path}: unexpected property {key!r}")
            elif isinstance(extra, Mapping):
                self.validate(value, extra, name=name, path=child)

    def _validate_string(self, instance: str, schema: Mapping[str, Any], path: str) -> None:
        minimum = schema.get("minLength")
        if minimum is not None and len(instance) < minimum:
            raise SchemaError(f"{path}: shorter than minLength {minimum}")
        maximum = schema.get("maxLength")
        if maximum is not None and len(instance) > maximum:
            raise SchemaError(f"{path}: longer than maxLength {maximum}")
        pattern = schema.get("pattern")
        if pattern is not None and re.search(pattern, instance) is None:
            raise SchemaError(f"{path}: does not match pattern {pattern!r}")
        if schema.get("format") == "date-time" and _DATETIME_RE.match(instance) is None:
            raise SchemaError(f"{path}: not an RFC 3339 date-time")

    def _validate_number(self, instance: float, schema: Mapping[str, Any], path: str) -> None:
        minimum = schema.get("minimum")
        if minimum is not None and instance < minimum:
            raise SchemaError(f"{path}: less than minimum {minimum}")
        maximum = schema.get("maximum")
        if maximum is not None and instance > maximum:
            raise SchemaError(f"{path}: greater than maximum {maximum}")

    def _validate_array(
        self, instance: list[Any], schema: Mapping[str, Any], name: str, path: str
    ) -> None:
        minimum = schema.get("minItems")
        if minimum is not None and len(instance) < minimum:
            raise SchemaError(f"{path}: fewer than minItems {minimum}")
        maximum = schema.get("maxItems")
        if maximum is not None and len(instance) > maximum:
            raise SchemaError(f"{path}: more than maxItems {maximum}")
        items = schema.get("items")
        if isinstance(items, Mapping):
            for index, value in enumerate(instance):
                self.validate(value, items, name=name, path=f"{path}[{index}]")


def _matches_type(value: Any, expected: Any) -> bool:
    if isinstance(expected, list):
        return any(_matches_type(value, option) for option in expected)
    if expected == "object":
        return isinstance(value, Mapping)
    if expected == "array":
        return isinstance(value, list)
    if expected == "string":
        return isinstance(value, str)
    if expected == "boolean":
        return isinstance(value, bool)
    if expected == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if expected == "number":
        return isinstance(value, (int, float)) and not isinstance(value, bool)
    if expected == "null":
        return value is None
    raise SchemaError(f"unsupported type {expected!r}")


def validate_contract(instance: Any, *, store: SchemaStore | None = None) -> Any:
    """Validate a contract object against the schema its ``kind`` names."""
    if not isinstance(instance, Mapping):
        raise SchemaError("contract object must be a JSON object")
    kind = instance.get("kind")
    if not isinstance(kind, str):
        raise SchemaError(f"contract object has no kind: {kind!r}")
    target = KIND_SCHEMAS.get(kind)
    if target is None:
        raise SchemaError(f"no schema registered for kind {kind!r}")
    filename, pointer = target
    active = store if store is not None else SchemaStore()
    schema, name = active.resolve(f"{filename}{pointer}", filename)
    active.validate(instance, schema, name=name)
    return instance
