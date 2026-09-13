"""Reference model identifiers for the shipped default configuration.

This is the single place where the reference architecture is named:
Ollama local + DeepSeek direct API expert. Routing code, recommendation code
and the profile system must not hard-code these values.
"""

from __future__ import annotations

# Local tier (Ollama)
DEFAULT_LOCAL_PROVIDER = "ollama"
DEFAULT_LOCAL_MODEL = "qwen3.5:9b"
DEFAULT_LOCAL_CONTEXT = 64_000
DEFAULT_LOCAL_MAX_OUTPUT = 4096

# Expert tier (DeepSeek direct API)
DEFAULT_EXPERT_PROVIDER = "deepseek"
DEFAULT_EXPERT_MODEL = "deepseek-chat"
DEFAULT_EXPERT_ENDPOINT = "https://api.deepseek.com/v1"
DEFAULT_EXPERT_API_KEY_ENV = "DEEPSEEK_API_KEY"
DEFAULT_EXPERT_CONTEXT = 64_000
DEFAULT_EXPERT_MAX_OUTPUT = 8192
DEFAULT_EXPERT_INPUT_COST = 0.14
DEFAULT_EXPERT_OUTPUT_COST = 0.28

DEFAULT_TEMPERATURE = 0.2
DEFAULT_ROUTING_POLICY = "adaptive"
