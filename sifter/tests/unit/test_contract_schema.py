"""Full JSON Schema validation tests for the v1 contract boundary."""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from aa_sifter.contract_schema import (
    KIND_SCHEMAS,
    SchemaError,
    SchemaStore,
    schema_directory,
    validate_contract,
)
from aa_sifter.contracts import ContractError, make_memory_record, make_route_response, validate
from aa_sifter.memory import task_candidate

_SCHEMA_NAMES = ("common.schema.json", "route.schema.json", "memory-record.schema.json")


def _route_request() -> dict:
    return {
        "contractVersion": "1.0",
        "kind": "route_request",
        "id": "req_1",
        "prompt": "Write unit tests for a small utility function.",
    }


def _route_response() -> dict:
    return {
        "contractVersion": "1.0",
        "kind": "route_response",
        "id": "resp_1",
        "requestId": "req_1",
        "route": "local",
        "decisionLevel": "routine",
        "requiresHumanApproval": False,
        "reasons": ["small change"],
    }


# -- schema validation -----------------------------------------------------


def test_valid_route_request_passes() -> None:
    assert validate_contract(_route_request()) == _route_request()


def test_valid_route_response_passes() -> None:
    assert validate_contract(_route_response()) == _route_response()


def test_missing_required_property_fails() -> None:
    request = _route_request()
    del request["prompt"]

    with pytest.raises(SchemaError, match="prompt"):
        validate_contract(request)


def test_wrong_type_fails() -> None:
    request = {**_route_request(), "prompt": 42}

    with pytest.raises(SchemaError, match="expected type"):
        validate_contract(request)


def test_identifier_pattern_fails() -> None:
    request = {**_route_request(), "id": "not valid!"}

    with pytest.raises(SchemaError, match="pattern"):
        validate_contract(request)


def test_enum_violation_fails() -> None:
    response = {**_route_response(), "route": "nowhere"}

    with pytest.raises(SchemaError, match="not one of"):
        validate_contract(response)


def test_max_items_violation_fails() -> None:
    response = {**_route_response(), "reasons": ["x"] * 65}

    with pytest.raises(SchemaError, match="maxItems"):
        validate_contract(response)


def test_memory_record_passes() -> None:
    record = make_memory_record(summary="aa-sifter routed a task locally.")
    record["provenance"] = {
        "source": "aa-sifter",
        "producedAt": "2026-09-21T00:00:00+00:00",
    }

    assert validate_contract(record) == record


def test_memory_record_missing_content_fails() -> None:
    record = make_memory_record(summary="x")
    del record["content"]

    with pytest.raises(SchemaError, match="content"):
        validate_contract(record)


def test_unknown_kind_fails() -> None:
    with pytest.raises(SchemaError, match="no schema registered"):
        validate_contract({"kind": "unknown_thing"})


def test_non_object_fails() -> None:
    with pytest.raises(SchemaError, match="JSON object"):
        validate_contract(["not", "an", "object"])


def test_bad_timestamp_format_fails() -> None:
    record = make_memory_record(summary="x")
    record["provenance"] = {"source": "aa-sifter", "producedAt": "yesterday"}

    with pytest.raises(SchemaError, match="date-time"):
        validate_contract(record)


def test_unsupported_keyword_fails_closed(tmp_path: Path) -> None:
    (tmp_path / "bad.schema.json").write_text(
        json.dumps({"$defs": {"x": {"type": "object", "oneOf": [{"type": "string"}]}}}),
        encoding="utf-8",
    )
    store = SchemaStore(tmp_path)
    schema, name = store.resolve("bad.schema.json#/$defs/x", "bad.schema.json")

    with pytest.raises(SchemaError, match="unsupported schema keyword"):
        store.validate({}, schema, name=name)


def test_schema_directory_honours_env_override(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("SIFTER_SCHEMA_DIR", str(tmp_path))

    assert schema_directory() == tmp_path


def test_kind_schemas_are_registered() -> None:
    assert set(KIND_SCHEMAS) == {"route_request", "route_response", "memory_record"}


# -- bundled schema drift --------------------------------------------------


@pytest.mark.parametrize("name", _SCHEMA_NAMES)
def test_bundled_schema_matches_source_of_truth(name: str) -> None:
    repo_dir = Path(__file__).resolve().parents[3] / "contracts" / "schemas" / "v1"
    if not repo_dir.is_dir():
        pytest.skip("contracts schemas are not present in this checkout")

    bundled = (schema_directory() / name).read_text(encoding="utf-8")
    source = (repo_dir / name).read_text(encoding="utf-8")

    assert bundled == source


# -- contract boundary integration -----------------------------------------


def test_contracts_validate_applies_schema() -> None:
    request = _route_request()
    del request["prompt"]

    with pytest.raises(ContractError):
        validate(request)


def test_contracts_validate_accepts_valid_object() -> None:
    request = _route_request()

    assert validate(request) == request


def test_make_route_response_validates() -> None:
    response = make_route_response(
        _route_request(),
        route="local",
        decision_level="routine",
        requires_human_approval=False,
    )
    validate(response)


def test_task_candidate_record_is_schema_valid() -> None:
    record = task_candidate(
        trace_id="trace_1",
        status="completed",
        route="local",
        decision_level="routine",
        requires_human_approval=False,
        prompt="hello",
        cloud_calls=0,
        cloud_cost=0.0,
    )

    validate(record)
