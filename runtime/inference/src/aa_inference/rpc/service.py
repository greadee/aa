"""Stable inference RPC surface: ``inference.route``, ``inference.generate``, ``inference.health``.

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
from ..decision import RoutingBlockedError, parse_routing_decision
from ..model_defaults import DEFAULT_EXPERT_MAX_OUTPUT
from ..models.provider import Message, Role
from .envelope import (
    INCOMPATIBLE,
    METHOD_NOT_FOUND,
    RPC_VERSION,
    failure,
    is_compatible,
    success,
)
from .errors import map_exception
from .idempotency import IdempotencyStore, request_fingerprint
from .identity import IdentityError, StoreBinding, requested_store_id

_ROUTE_METHOD = "inference.route"
_GENERATE_METHOD = "inference.generate"
_HEALTH_METHOD = "inference.health"
_RECOMMEND_METHOD = "inference.recommend"


class InferenceService:
    """Transport-agnostic JSON-RPC service over an in-process ``ComputeInference``."""

    def __init__(
        self,
        inference: Any,
        *,
        store_id: str | None = None,
        idempotency: IdempotencyStore | None = None,
    ):
        self.inference = inference
        self._store = StoreBinding(store_id)
        self.idempotency = idempotency or IdempotencyStore()

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
            self._store.check(requested_store_id(message))
        except IdentityError as exc:
            return exc.as_error(request_id)
        try:
            if method == _ROUTE_METHOD:
                return success(request_id, self.route(params))
            if method == _GENERATE_METHOD:
                return success(request_id, await self.generate(params))
            if method == _HEALTH_METHOD:
                return success(request_id, self.health())
            if method == _RECOMMEND_METHOD:
                return success(request_id, self.recommend(params))
        except Exception as exc:  # noqa: BLE001 - mapped to the governed v1 error set
            return map_exception(exc, request_id)
        return failure(request_id, METHOD_NOT_FOUND, f"unknown method {method!r}")

    # -- methods -----------------------------------------------------------
    def route(self, request: dict[str, Any]) -> dict[str, Any]:
        route_request_kwargs(request)
        decision = parse_routing_decision(request.get("routingDecision"))
        config = self.inference.config
        blocked_reason: str | None = None
        route: str = decision.locality
        if decision.blocked:
            route = "blocked"
            blocked_reason = decision.reason or "blocked by the allocator"
        elif decision.locality == "cloud" and not config.cloud_allowed:
            route = "blocked"
            blocked_reason = "cloud is disabled by the active profile"

        reasons = [decision.reason] if decision.reason else []
        return make_route_response(
            request,
            route=route,
            decision_level=decision.decision_level,
            requires_human_approval=decision.requires_human_approval,
            reasons=reasons,
            planner_tier=decision.tier,
            executor_tier=decision.tier,
            review_tier=decision.tier if decision.locality == "cloud" else "none",
            blocked_reason=blocked_reason,
            estimated_cost_usd=decision.estimated_cost_usd,
        )

    async def generate(self, params: dict[str, Any]) -> dict[str, Any]:
        key = params.get("idempotencyKey")
        fingerprint = request_fingerprint(params) if key is not None else ""
        if key is not None:
            cached = self.idempotency.resolve(str(key), fingerprint)
            if cached is not None:
                return cached
        decision = parse_routing_decision(params.get("routingDecision"))
        if decision.blocked:
            raise RoutingBlockedError(decision)
        requested_tier = params.get("tier")
        if requested_tier is not None and str(requested_tier) != decision.tier:
            raise ContractError(
                f"tier: {requested_tier!r} conflicts with the routing decision "
                f"tier {decision.tier!r}"
            )
        raw_messages = params.get("messages")
        if not isinstance(raw_messages, list) or not raw_messages:
            raise ContractError("messages: at least one message is required")
        messages = [self._message(item) for item in raw_messages]
        result = await self.inference.generate(messages, tier=decision.provider_tier)
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
            self.idempotency.remember(str(key), fingerprint, payload)
        return payload

    def health(self) -> dict[str, Any]:
        config = self.inference.config
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
            "capabilities": ["route", "generate", "health", "recommend"],
        }

    def recommend(self, params: dict[str, Any]) -> dict[str, Any]:
        """Deterministic hardware/goal recommendation (advisory only)."""
        from ..config.schema import HardwareProfile, UserGoal
        from ..recommend.recommender import recommend_profile
        from ..system.hardware import detect_hardware

        hardware_data = params.get("hardware")
        hardware = HardwareProfile(**hardware_data) if hardware_data else detect_hardware()
        raw_goal = params.get("goal")
        if isinstance(raw_goal, str):
            goal = UserGoal(workload=[raw_goal])
        elif isinstance(raw_goal, dict):
            goal = UserGoal(**raw_goal)
        else:
            goal = UserGoal(workload=["coding"])
        recommendation = recommend_profile(hardware=hardware, goal=goal)
        return {
            "hardware": hardware.model_dump(mode="json"),
            "recommendation": recommendation.model_dump(mode="json"),
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
        output = self.inference.config.expert_max_output_tokens or DEFAULT_EXPERT_MAX_OUTPUT
        return self.inference.cloud_config().estimate_cost(max(0, input_tokens), output)
