from __future__ import annotations

import pytest

from aa_inference.models.config import InferenceConfig
from aa_inference.models.provider import ProviderError
from aa_inference.tasks.decomposition import Decomposer
from tests.fixtures.providers import FakeProvider, MalformedProvider, ScriptedProvider


@pytest.mark.asyncio
async def test_decompose_returns_structured_subtasks(config: InferenceConfig):
    plan = {
        "tasks": [
            {"id": "task-001", "task": "inspect", "constraints": ["use sqlite"]},
            {"id": "task-002", "task": "implement", "dependencies": ["task-001"]},
        ]
    }
    provider = FakeProvider(structured=plan)
    decomposer = Decomposer()
    specs = await decomposer.decompose(
        "Add feature", provider, config.local_model_config(), constraints=["no postgres"]
    )
    assert len(specs) == 2
    assert specs[1].dependencies == ["task-001"]
    assert "no postgres" in specs[1].constraints


@pytest.mark.asyncio
async def test_decompose_falls_back_on_malformed_output(config: InferenceConfig):
    provider = MalformedProvider()
    specs = await Decomposer().decompose("Add feature", provider, config.local_model_config())
    assert len(specs) == 1
    assert specs[0].task == "Add feature"


@pytest.mark.asyncio
async def test_decompose_falls_back_on_provider_error(config: InferenceConfig):
    provider = ScriptedProvider(local_outcomes=[ProviderError("boom")])
    specs = await Decomposer().decompose("Add feature", provider, config.local_model_config())
    assert len(specs) == 1
