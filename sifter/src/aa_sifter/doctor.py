from __future__ import annotations

import os
from collections.abc import Callable, Mapping
from pathlib import Path

import httpx

from .config.schema import ApplicationConfig, ModelTierConfig
from .config.validation import validate_profile


def ollama_installed(host: str = "http://localhost:11434") -> set[str] | None:
    """Return installed Ollama model names, or None if Ollama is unreachable."""
    try:
        response = httpx.get(f"{host.rstrip('/')}/api/tags", timeout=2.0)
        if response.status_code != 200:
            return None
        data = response.json()
        return {item.get("name", "") for item in data.get("models", [])}
    except Exception:  # noqa: BLE001 - availability check is best-effort
        return None


def live_expert_check(
    expert: ModelTierConfig, *, environ: Mapping[str, str] | None = None
) -> tuple[bool, str]:
    """One minimal paid request. Only used with an explicit --live flag."""
    env = environ if environ is not None else os.environ

    if expert.provider == "ollama":
        try:
            response = httpx.get(
                f"{(expert.endpoint or 'http://localhost:11434').rstrip('/')}/api/tags",
                timeout=5.0,
            )
        except Exception as exc:  # noqa: BLE001
            return False, str(exc)
        return response.status_code == 200, f"Ollama ({response.status_code})"

    endpoint = expert.endpoint or "https://api.deepseek.com/v1"
    api_key = env.get(expert.api_key_env or "") or None
    headers = {"Authorization": f"Bearer {api_key}"} if api_key else {}
    try:
        response = httpx.post(
            f"{endpoint.rstrip('/')}/chat/completions",
            headers=headers,
            json={
                "model": expert.model,
                "messages": [{"role": "user", "content": "ping"}],
                "max_tokens": 4,
            },
            timeout=30.0,
        )
    except Exception as exc:  # noqa: BLE001
        return False, str(exc)
    if response.status_code >= 400:
        return False, f"API error {response.status_code}"
    return True, "reachable"


def build_doctor_report(
    app: ApplicationConfig,
    *,
    config_path: Path | None = None,
    live: bool = False,
    ollama_installed_fn: Callable[[], set[str] | None] | None = None,
    live_check_fn: Callable[[ModelTierConfig], tuple[bool, str]] | None = None,
    environ: Mapping[str, str] | None = None,
) -> dict:
    env = environ if environ is not None else os.environ
    installed_fn = ollama_installed_fn or ollama_installed
    live_fn = live_check_fn or (lambda expert: live_expert_check(expert, environ=env))
    checks: list[dict] = []

    def record(name: str, ok: bool, detail: str = "", *, optional: bool = False) -> None:
        checks.append({"name": name, "ok": ok, "detail": detail, "optional": optional})

    record("profile", True, f"active={app.active_profile} path={config_path or '-'}")
    profile = app.profiles.get(app.active_profile)
    if profile is None:
        record("configuration", False, "active profile is missing")
        return _finish(checks)

    hardware = profile.hardware or (
        app.hardware.get(app.active_hardware) if app.active_hardware else None
    )
    validation = validate_profile(
        profile, hardware=hardware, environ=env, require_credentials=False
    )
    record("configuration", validation.ok, "; ".join(validation.errors) or "valid")
    for warning in validation.warnings:
        record("warning", True, warning, optional=True)

    local = profile.local
    if local and local.provider == "ollama":
        installed = installed_fn()
        if installed is None:
            record("ollama", False, "not reachable", optional=True)
            record("local model", False, "unknown (Ollama unreachable)", optional=True)
        else:
            present = local.model in installed
            record(
                "local model",
                present,
                f"{local.model}{'' if present else ' NOT installed; run: ollama pull ' + local.model}",
                optional=not present,
            )

    if hardware is not None:
        record(
            "gpu",
            hardware.has_gpu,
            f"{hardware.gpu_name or 'none'} {hardware.gpu_vram_gb or '?'} GB",
            optional=hardware.has_gpu,
        )

    expert = profile.expert
    if expert and expert.enabled and profile.preferences.cloud_allowed:
        env_name = expert.api_key_env
        has_key = bool(env_name and env.get(env_name))
        record(
            "expert credentials",
            has_key,
            f"{env_name or 'api_key_env'} "
            f"{'configured' if has_key else 'not configured (local use still available)'}",
            optional=not has_key,
        )
        if live:
            ok, detail = live_fn(expert)
            record("expert connectivity", ok, detail)
        else:
            record("expert connectivity", True, "skipped (use --live to test)", optional=True)

    record("routing policy", True, profile.routing_policy)
    record(
        "human approval",
        profile.approval.major and profile.approval.critical,
        f"major={'enabled' if profile.approval.major else 'DISABLED'} "
        f"critical={'enabled' if profile.approval.critical else 'DISABLED'}",
    )
    return _finish(checks)


def _finish(checks: list[dict]) -> dict:
    required_ok = all(bool(check["ok"]) for check in checks if not check["optional"])
    return {"ok": required_ok, "checks": checks}
