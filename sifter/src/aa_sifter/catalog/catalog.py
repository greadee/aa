from __future__ import annotations

import tomllib
from functools import lru_cache
from importlib import resources

from .models import ModelMetadata


@lru_cache(maxsize=1)
def _load_bundled() -> tuple[ModelMetadata, ...]:
    data_file = resources.files("aa_sifter.catalog").joinpath("data/catalog.toml")
    raw = tomllib.loads(data_file.read_text(encoding="utf-8"))
    models: list[ModelMetadata] = []
    for entry in raw.get("models", []):
        models.append(ModelMetadata.model_validate(entry))
    return tuple(models)


class ModelCatalog:
    """Advisory metadata about known models.

    The catalog is never a whitelist: unknown model identifiers are always
    configurable by the user.
    """

    def __init__(self, models: list[ModelMetadata] | tuple[ModelMetadata, ...] | None = None):
        self._models = list(models) if models is not None else list(_load_bundled())
        self._by_key = {model.key: model for model in self._models}

    def all(self) -> list[ModelMetadata]:
        return list(self._models)

    def get(self, provider: str, model_id: str) -> ModelMetadata | None:
        return self._by_key.get(f"{provider}/{model_id}")

    def resolve(self, provider: str, model_id: str) -> ModelMetadata | None:
        return self.get(provider, model_id)

    def local_models(self) -> list[ModelMetadata]:
        return [model for model in self._models if model.local]

    def cloud_models(self) -> list[ModelMetadata]:
        return [model for model in self._models if model.cloud]

    def filter(
        self,
        *,
        local: bool | None = None,
        cloud: bool | None = None,
        providers: set[str] | None = None,
        capabilities: set[str] | None = None,
    ) -> list[ModelMetadata]:
        required = capabilities or set()
        result: list[ModelMetadata] = []
        for model in self._models:
            if local is not None and model.local is not local:
                continue
            if cloud is not None and model.cloud is not cloud:
                continue
            if providers is not None and model.provider not in providers:
                continue
            if required and not model.has_capabilities(required):
                continue
            result.append(model)
        return result
