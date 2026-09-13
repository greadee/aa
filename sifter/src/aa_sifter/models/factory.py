from __future__ import annotations

from .config import SifterConfig
from .ollama import OllamaProvider
from .openai_compatible import DeepSeekProvider, OpenAICompatibleProvider
from .provider import ModelProvider, ModelRegistry, ProviderError


def build_provider(name: str, config: SifterConfig) -> ModelProvider:
    if name == "ollama":
        host = config.local_endpoint or config.ollama_host or "http://localhost:11434"
        return OllamaProvider(host, default_timeout=config.request_timeout_seconds)
    if name == "deepseek":
        return DeepSeekProvider(
            base_url=config.expert_endpoint or "https://api.deepseek.com/v1",
            api_key_env=config.expert_api_key_env or "DEEPSEEK_API_KEY",
            default_timeout=config.request_timeout_seconds,
        )
    if name == "openai-compatible":
        endpoint = config.expert_endpoint or config.local_endpoint
        if not endpoint:
            raise ProviderError(
                "provider 'openai-compatible' requires an endpoint "
                "(set expert_endpoint or local_endpoint)"
            )
        return OpenAICompatibleProvider(
            endpoint,
            api_key_env=config.expert_api_key_env or config.local_api_key_env,
            default_timeout=config.request_timeout_seconds,
        )
    raise ProviderError(f"Unknown provider '{name}'")


def build_registry(config: SifterConfig, *, injected: ModelProvider | None = None) -> ModelRegistry:
    if injected is not None:
        return ModelRegistry({injected.name: injected})
    providers: dict[str, ModelProvider] = {}
    for name in {config.local_provider, config.expert_provider}:
        if not name or name in providers:
            continue
        try:
            providers[name] = build_provider(name, config)
        except ProviderError:
            # Unknown providers surface when used; configuration validation warns first.
            continue
    return ModelRegistry(providers)
