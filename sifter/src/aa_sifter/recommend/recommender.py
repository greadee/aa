from __future__ import annotations

from dataclasses import dataclass

from ..catalog.catalog import ModelCatalog
from ..catalog.models import ModelMetadata
from ..config.schema import (
    ComputeProfile,
    HardwareProfile,
    ModelTierConfig,
    Recommendation,
    UserGoal,
)

IMPLEMENTED_CLOUD_PROVIDERS = {"deepseek", "openai-compatible"}
IMPLEMENTED_LOCAL_PROVIDERS = {"ollama"}

_CONTEXT_STEPS = [8192, 16384, 32768, 65536, 131072]
_WEIGHTS_GB_PER_B = 0.6


@dataclass
class _Scored:
    model: ModelMetadata
    score: float
    comfortable: bool
    compatible: bool
    reasons: list[str]


def recommend_profile(
    *,
    hardware: HardwareProfile | None,
    goal: UserGoal,
    catalog: ModelCatalog | None = None,
    name: str = "recommended",
    available_local_providers: set[str] | None = None,
    available_cloud_providers: set[str] | None = None,
) -> Recommendation:
    catalog = catalog or ModelCatalog()
    local_providers = available_local_providers or IMPLEMENTED_LOCAL_PROVIDERS
    cloud_providers = available_cloud_providers or IMPLEMENTED_CLOUD_PROVIDERS

    required = goal.required_capabilities()
    vram = hardware.gpu_vram_gb if hardware else None
    ram = hardware.system_ram_gb if hardware else None

    local_scored = _score_local(catalog, required, goal, vram, ram, local_providers)
    local_best = next((s for s in local_scored if s.compatible), None)

    expert_scored = _score_cloud(catalog, required, goal, cloud_providers)
    expert_best = expert_scored[0] if expert_scored else None

    cloud_allowed = goal.cloud_allowed
    notes: list[str] = []
    reasons: list[str] = []

    if local_best is None:
        if vram is not None:
            notes.append(
                "No catalog local model fits the available GPU memory comfortably; "
                "cloud-heavy routing is recommended."
            )
        else:
            notes.append("No GPU detected; local inference would be CPU-bound.")
    if not cloud_allowed:
        notes.append("Cloud use is disabled by the user goal; only local models were considered.")
        expert_best = None

    local_tier: ModelTierConfig | None = None
    local_context: int | None = None
    if local_best is not None:
        local_context, context_notes = recommend_context(
            local_best.model, vram=vram, preferred=goal.preferred_context
        )
        notes.extend(context_notes)
        local_tier = ModelTierConfig(
            provider=local_best.model.provider,
            model=local_best.model.model_id,
            context_limit=local_context,
            max_output_tokens=4096,
            temperature=0.2,
        )
        reasons.extend(local_best.reasons)

    expert_tier: ModelTierConfig | None = None
    if expert_best is not None and cloud_allowed:
        expert_meta = expert_best.model
        expert_context = _expert_context(expert_meta, goal)
        expert_tier = ModelTierConfig(
            provider=expert_meta.provider,
            model=expert_meta.model_id,
            endpoint=_default_expert_endpoint(expert_meta.provider),
            api_key_env=_default_expert_key_env(expert_meta.provider),
            context_limit=expert_context,
            max_output_tokens=8192,
            temperature=0.2,
            input_cost_per_mtok=expert_meta.cost.input_per_mtok if expert_meta.cost else None,
            output_cost_per_mtok=expert_meta.cost.output_per_mtok if expert_meta.cost else None,
        )
        reasons.append(
            f"{expert_meta.display_name} selected for the expert tier"
            + (
                f" (${expert_meta.cost.input_per_mtok:g}/${expert_meta.cost.output_per_mtok:g} per Mtok)"
                if expert_meta.cost
                else ""
            )
            + "."
        )

    routing_policy = _recommend_policy(goal, cloud_allowed, local_tier, expert_tier)
    reasons.append(f"Routing strategy: {routing_policy}.")

    profile = ComputeProfile(
        name=name,
        local=local_tier,
        expert=expert_tier,
        routing_policy=routing_policy,
        preferences=_preferences_from_goal(goal),
        hardware=hardware,
        goal=goal,
    )
    # Clear the schedule for downstream callers.
    confidence = _confidence(hardware, local_best, expert_best, required)

    local_alternatives = _alternatives(local_scored, local_best, vram)
    expert_alternatives = [
        ModelTierConfig(
            provider=item.model.provider,
            model=item.model.model_id,
            endpoint=_default_expert_endpoint(item.model.provider),
            api_key_env=_default_expert_key_env(item.model.provider),
            input_cost_per_mtok=item.model.cost.input_per_mtok if item.model.cost else None,
            output_cost_per_mtok=item.model.cost.output_per_mtok if item.model.cost else None,
        )
        for item in expert_scored[1:3]
    ]

    if local_best is None and expert_best is None:
        confidence = "low"
        notes.append(
            "No compatible model was found; configure a model manually with `aa_sifter model set`."
        )

    return Recommendation(
        profile=profile,
        reasons=reasons,
        local_alternatives=local_alternatives,
        expert_alternatives=expert_alternatives,
        confidence=confidence,
        notes=notes,
    )


def recommend_context(
    model: ModelMetadata,
    *,
    vram: float | None,
    preferred: int | None = None,
) -> tuple[int, list[str]]:
    notes: list[str] = []
    theoretical = model.context_window or 32768
    if vram is None:
        chosen = min(preferred or 8192, theoretical)
        if not preferred:
            notes.append(f"Recommended local context: {chosen} tokens (unknown GPU memory).")
        return chosen, notes

    weights = (model.parameter_count_b or 8.0) * _WEIGHTS_GB_PER_B
    headroom = max(0.0, vram - weights)
    capacity = 8192
    for step in _CONTEXT_STEPS:
        if headroom >= _step_headroom(step):
            capacity = step
    capacity = min(capacity, theoretical)
    if preferred is not None:
        if preferred > capacity:
            notes.append(
                f"Requested context {preferred} exceeds the safe local estimate {capacity}; "
                "using the safe estimate."
            )
            chosen = capacity
        else:
            chosen = preferred
    else:
        chosen = capacity
    notes.append(
        f"Recommended local context: {chosen} tokens "
        f"(model maximum {theoretical}; operational estimate leaves room for KV cache and tooling)."
    )
    return chosen, notes


def _step_headroom(step: int) -> float:
    return {8192: 0.5, 16384: 1.5, 32768: 3.0, 65536: 4.5, 131072: 8.0}.get(step, 99.0)


def _expert_context(model: ModelMetadata, goal: UserGoal) -> int:
    window = model.context_window or 64000
    if goal.preferred_context:
        return min(goal.preferred_context, window)
    if "long_context" in goal.required_capabilities():
        return min(131072, window)
    return min(64000, window)


def _score_local(
    catalog: ModelCatalog,
    required: set[str],
    goal: UserGoal,
    vram: float | None,
    ram: float | None,
    providers: set[str],
) -> list[_Scored]:
    scored: list[_Scored] = []
    for model in catalog.filter(local=True, providers=providers, capabilities=required):
        reasons: list[str] = []
        compatible = True
        comfortable = False
        if vram is not None:
            recommended = model.recommended_vram_gb or model.minimum_vram_gb or 0.0
            minimum = model.minimum_vram_gb or recommended
            if minimum > vram:
                compatible = False
            elif recommended <= vram:
                comfortable = True
            if not compatible:
                reasons.append(
                    f"{model.display_name} needs ~{minimum:g} GB VRAM, more than the available "
                    f"{vram:g} GB."
                )
        fit = _fit_score(model, vram, comfortable, compatible)
        capability = len(required & set(model.capabilities)) / len(required) if required else 1.0
        params = model.parameter_count_b or 8.0
        quality = min(1.0, params / 32.0)
        speed = 1.0 - min(1.0, params / 32.0)
        score = (
            1.2 * fit
            + 0.9 * capability
            + (0.5 + goal.quality_priority) * quality
            + (0.5 + goal.speed_priority) * speed
            + (0.35 if goal.cost_priority >= 0.5 else 0.0)
            + (0.35 if goal.privacy_priority >= 0.5 else 0.0)
            + (0.25 if goal.quality_priority < 0.5 else 0.0)
        )
        if comfortable:
            reasons.append(
                f"{model.display_name} fits comfortably within {vram:g} GB VRAM, leaving room for "
                "KV cache and agent tooling."
            )
        elif compatible and vram is not None:
            reasons.append(f"{model.display_name} fits only tightly within {vram:g} GB VRAM.")
        if required:
            reasons.append(
                "Supports " + ", ".join(sorted(required & set(model.capabilities))) + "."
            )
        if ram is not None and model.minimum_ram_gb and model.minimum_ram_gb > ram:
            score -= 0.5
            reasons.append(f"Prefers more than {ram:g} GB system RAM.")
        scored.append(_Scored(model, score, comfortable, compatible, reasons))

    scored.sort(key=lambda item: (-item.score, item.model.key))
    return scored


def _fit_score(
    model: ModelMetadata, vram: float | None, comfortable: bool, compatible: bool
) -> float:
    if vram is None:
        return 0.5
    if not compatible:
        return -1.0
    recommended = model.recommended_vram_gb or model.minimum_vram_gb or vram
    ratio = recommended / vram if vram else 1.0
    return 1.1 if comfortable and ratio <= 0.9 else 0.7


def _score_cloud(
    catalog: ModelCatalog,
    required: set[str],
    goal: UserGoal,
    providers: set[str],
) -> list[_Scored]:
    candidates = catalog.filter(cloud=True, providers=providers, capabilities=required)
    if not candidates:
        return []
    max_cost = max((_cost(model) for model in candidates), default=1.0) or 1.0
    scored: list[_Scored] = []
    for model in candidates:
        capability = len(required & set(model.capabilities)) / len(required) if required else 1.0
        quality = _cloud_quality(model)
        cost = _cost(model) / max_cost
        score = (
            (0.5 + goal.quality_priority) * quality
            - (0.5 + goal.cost_priority) * cost
            + 0.8 * capability
            + (0.2 if "long_context" in required and model.supports("long_context") else 0.0)
        )
        reasons = [f"Expert capability match for {', '.join(sorted(required)) or 'general'}."]
        scored.append(_Scored(model, score, True, True, reasons))
    scored.sort(key=lambda item: (-item.score, item.model.key))
    return scored


def _cloud_quality(model: ModelMetadata) -> float:
    quality = 0.5
    if model.supports("reasoning"):
        quality += 0.3
    if model.supports("coding"):
        quality += 0.2
    if model.supports("long_context"):
        quality += 0.1
    if model.supports("agentic"):
        quality += 0.1
    if model.supports("vision"):
        quality += 0.05
    return quality


def _cost(model: ModelMetadata) -> float:
    if not model.cost:
        return 1.0
    return model.cost.input_per_mtok + model.cost.output_per_mtok


def _preferences_from_goal(goal: UserGoal):
    from ..config.schema import UserPreferenceConfig

    return UserPreferenceConfig(
        local_first=goal.privacy_priority >= 0.5 or goal.cost_priority >= 0.5,
        cost_sensitive=goal.cost_priority >= 0.5,
        privacy_sensitive=goal.privacy_priority >= 0.5,
        cloud_allowed=goal.cloud_allowed,
        preferred_context=goal.preferred_context,
    )


def _recommend_policy(
    goal: UserGoal,
    cloud_allowed: bool,
    local_tier: ModelTierConfig | None,
    expert_tier: ModelTierConfig | None,
) -> str:
    if not cloud_allowed or expert_tier is None:
        return "local_only"
    if local_tier is None:
        return "expert_first"
    if goal.quality_priority >= 0.75 and goal.cost_priority < 0.5:
        return "adaptive"
    if goal.cost_priority >= 0.6 or goal.privacy_priority >= 0.6:
        return "local_first"
    return "adaptive"


def _confidence(
    hardware: HardwareProfile | None,
    local_best: _Scored | None,
    expert_best: _Scored | None,
    required: set[str],
) -> str:
    if hardware is None or hardware.gpu_vram_gb is None:
        return "medium"
    if local_best is not None and local_best.comfortable and required:
        return "high"
    if local_best is not None or expert_best is not None:
        return "medium"
    return "low"


def _alternatives(
    scored: list[_Scored], best: _Scored | None, vram: float | None
) -> list[ModelTierConfig]:
    alternatives: list[ModelTierConfig] = []
    for item in scored:
        if best is not None and item.model.key == best.model.key:
            continue
        if not item.compatible:
            continue
        alternatives.append(
            ModelTierConfig(
                provider=item.model.provider,
                model=item.model.model_id,
                context_limit=item.model.context_window,
            )
        )
        if len(alternatives) >= 2:
            break
    return alternatives


def _default_expert_endpoint(provider: str) -> str | None:
    if provider == "deepseek":
        return "https://api.deepseek.com/v1"
    return None


def _default_expert_key_env(provider: str) -> str | None:
    if provider == "deepseek":
        return "DEEPSEEK_API_KEY"
    if provider == "openai-compatible":
        return "OPENAI_API_KEY"
    return None
