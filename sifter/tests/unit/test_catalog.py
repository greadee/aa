from __future__ import annotations

from aa_sifter.catalog import ModelCatalog


def test_bundled_catalog_loads_default_models():
    catalog = ModelCatalog()
    qwen = catalog.get("ollama", "qwen3.5:9b")
    deepseek = catalog.get("deepseek", "deepseek-chat")
    assert qwen is not None and qwen.local is True
    assert deepseek is not None and deepseek.cloud is True
    assert deepseek.cost is not None


def test_unknown_model_returns_none():
    assert ModelCatalog().get("ollama", "does-not-exist:12b") is None


def test_filter_local_by_capabilities():
    catalog = ModelCatalog()
    models = catalog.filter(local=True, capabilities={"coding", "tool_use", "agentic"})
    ids = {model.model_id for model in models}
    assert "qwen3.5:9b" in ids
    assert "qwen2.5-coder:7b" not in ids  # no tool_use / agentic


def test_filter_cloud_providers():
    catalog = ModelCatalog()
    models = catalog.filter(cloud=True, providers={"deepseek"})
    assert models
    assert all(model.provider == "deepseek" for model in models)


def test_local_and_cloud_partitions_are_disjoint():
    catalog = ModelCatalog()
    local_ids = {model.key for model in catalog.local_models()}
    cloud_ids = {model.key for model in catalog.cloud_models()}
    assert local_ids.isdisjoint(cloud_ids)
