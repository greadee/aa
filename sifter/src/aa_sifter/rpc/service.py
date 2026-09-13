"""Stable sifter RPC surface: ``sifter.route``, ``sifter.generate``, ``sifter.health``.

Routing decisions are deterministic and made by the classifier/preflight/policy
engine, never by a model. ``generate`` performs inference but still enforces cloud
egress redaction and the cloud-disabled profile (ADR-P4-003, ADR-P4-009).
"""

from __future__ import annotations

from typing import Any

from ..contracts import (
    CONTRACT_VERSION,
    ContractError,
    contracts_available,
    make_route_response,
    route_request_kwargs,
)
from ..model_defaults import DEFAULT_EXPERT_MAX_OUTPUT
from ..models.provider import Message, ProviderError, Role, Tier
from .envelope import (
    INCOMPATIBLE,
    INTERNAL_ERROR,
    INVALID_PARAMS,
    METHOD_NOT_FOUND,
    RPC_VERSION,
    UNAVAILABLE,
    failure,
    is_compatible,
    success,
)

_ROUTE_METHOD = "sifter.route"
_GENERATE_METHOD = "sifter.generate"
_HEALTH_METHOD = "sifter.health"


def _tier_name(tier: Tier | None) -> str:
    return "none" if tier is None else tier.value


class SifterService:
    """Transport-agnostic JSON-RPC service over an in-process ``ComputeSifter``."""

    def __init__(self, sifter: Any):
        self.sifter = sifter
        self._idempotent: dict[str, dict[str, Any]] = {}

    # -- dispatch ----------------------------------------------------------
    async def handle(self, message: dict[str, Any]) -> dict[str, Any]:
        request_id = message.get("id")
        method = message.get("method")
        params = message.get("params") or {}
        aa = message.get("aa") or {}

        if not is_compatible(aa):
            return failure(
                request_id,
                INCOMPATIBLE,
                "unsupported RPC major",
                data={"rpcVersion": aa.get("rpcVersion")},
            )
        try:
            if method == _ROUTE_METHOD:
                return success(request_id, self.route(params))
            if method == _GENERATE_METHOD:
                return success(request_id, await self.generate(params))
            if method == _HEALTH_METHOD:
                return success(request_id, self.health())
        except ContractError as exc:
            return failure(request_id, INVALID_PARAMS, str(exc))
        except ProviderError as exc:
            return failure(
                request_id,
                UNAVAILABLE,
                str(exc),
                data={"retryable": getattr(exc, "retryable", False)},
            )
        except Exception as exc:  # noqa: BLE001 - surfaced as JSON-RPC internal error
            return failure(request_id, INTERNAL_ERROR, str(exc))
        return failure(request_id, METHOD_NOT_FOUND, f"unknown method {method!r}")

    # -- methods -----------------------------------------------------------
    def route(self, request: dict[str, Any]) -> dict[str, Any]:
        kwargs = route_request_kwargs(request)
        config = self.sifter.config
        effective = kwargs["policy"] or config.routing_policy
        if not config.cloud_allowed:
            effective = "local_only"

        classification = self.sifter.classifier.classify(kwargs["prompt"])
        assessment = self.sifter.assessor.assess(
            kwargs["prompt"], kwargs["context"], classification=classification
        )
        decision = self.sifter.policy_engine.decide(assessment, policy=effective)

        route = decision.kind.value
        blocked_reason: str | None = None
        if decision.uses_cloud and not config.cloud_allowed:
            route = "blocked"
            blocked_reason = "cloud is disabled by the active profile"

        reasons = [*decision.reasons, *classification.reasons]
        return make_route_response(
            request,
            route=route,
            decision_level=classification.level.value,
            requires_human_approval=classification.requires_human_approval,
            reasons=reasons,
            planner_tier=_tier_name(decision.planner_tier),
            executor_tier=_tier_name(decision.executor_tier),
            review_tier=_tier_name(decision.review_tier),
            blocked_reason=blocked_reason,
            estimated_cost_usd=self._estimate_cost(assessment.context_requirement),
        )

    async def generate(self, params: dict[str, Any]) -> dict[str, Any]:
        key = params.get("idempotencyKey")
        if key is not None and key in self._idempotent:
            return self._idempotent[key]
        raw_messages = params.get("messages")
        if not isinstance(raw_messages, list) or not raw_messages:
            raise ContractError("messages: at least one message is required")
        messages = [self._message(item) for item in raw_messages]
        tier = str(params.get("tier", "local"))
        result = await self.sifter.generate(messages, tier=tier)
        payload = {
            "text": result.text,
            "usage": {
                "model": result.model,
                "tier": result.tier.value if hasattr(result.tier, "value") else str(result.tier),
                "inputTokens": result.input_tokens,
                "outputTokens": result.output_tokens,
            },
        }
        if key is not None:
            self._idempotent[key] = payload
        return payload

    def health(self) -> dict[str, Any]:
        config = self.sifter.config
        providers = [
            {
                "name": config.local_provider,
                "tier": "local",
                "available": True,
                "endpoint": config.local_endpoint or config.ollama_host,
            },
            {
                "name": config.expert_provider,
                "tier": "expert",
                "available": bool(config.cloud_allowed),
                "keyEnv": config.expert_api_key_env,
            },
        ]
        return {
            "available": True,
            "providers": providers,
            "rpcVersion": RPC_VERSION,
            "contractVersion": CONTRACT_VERSION,
            "contractsAvailable": contracts_available(),
        }

    # -- helpers -----------------------------------------------------------
    @staticmethod
    def _message(item: Any) -> Message:
        if not isinstance(item, dict) or "content" not in item:
            raise ContractError("messages: each message needs role and content")
        role = str(item.get("role", "user"))
        try:
            return Message(role=Role(role), content=str(item["content"]))
        except ValueError as exc:
            raise ContractError(f"messages: invalid role {role!r}") from exc

    def _estimate_cost(self, input_tokens: int) -> float:
        output = self.sifter.config.expert_max_output_tokens or DEFAULT_EXPERT_MAX_OUTPUT
        return self.sifter.cloud_config().estimate_cost(max(0, input_tokens), output)
