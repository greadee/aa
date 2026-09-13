from __future__ import annotations

import re
from typing import Any

from pydantic import BaseModel, Field

from ..history.store import HistoryStore

_COST_WAIVER = re.compile(r"\$\s*([0-9]+(?:\.[0-9]+)?)")
_APPROVAL_FRAMING = re.compile(
    r"\b(approved|approve|use\s+only|must\s+use|stick\s+with|do\s+not|don't|never|no\s+other|forbid)\b",
    re.IGNORECASE,
)

_TECH_FAMILIES: dict[str, set[str]] = {
    "persistence": {
        "sqlite",
        "postgres",
        "postgresql",
        "mysql",
        "mariadb",
        "mongodb",
        "dynamodb",
        "oracle",
        "redis",
        "filesystem",
    },
    "frontend": {"react", "vue", "svelte", "angular", "next.js", "nextjs", "solid"},
    "backend": {"django", "fastapi", "flask", "rails", "laravel", "express", "nestjs"},
    "auth": {
        "oauth",
        "oidc",
        "saml",
        "sso",
        "jwt",
        "session",
        "basic auth",
        "auth",
        "authentication",
    },
    "provider": {"ollama", "openai", "anthropic", "deepseek", "bedrock", "azure openai"},
    "deployment": {"kubernetes", "k8s", "docker", "ecs", "lambda", "serverless", "bare metal"},
}

_KNOWN_GRANTS: dict[str, list[str]] = {
    "cloud_debug_failed_tests": [
        r"debug\s+(failed\s+)?tests\s+automatically",
        r"deepseek\s+may\s+debug",
    ],
    "cloud_auto_escalation": [r"automatically\s+escalate", r"may\s+use\s+cloud\s+automatically"],
}


class StandingRule(BaseModel):
    id: int | None = None
    text: str
    forbid_terms: list[str] = Field(default_factory=list)
    grants: list[str] = Field(default_factory=list)
    tags: list[str] = Field(default_factory=list)
    active: bool = True

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> StandingRule:
        return cls(
            id=data.get("id"),
            text=data.get("text", ""),
            forbid_terms=list(data.get("forbid_terms") or []),
            grants=list(data.get("grants") or []),
            tags=list(data.get("tags") or []),
            active=bool(data.get("active", True)),
        )


def derive_rule_metadata(text: str) -> dict[str, list[str]]:
    lower = text.lower()
    forbid: list[str] = []
    grants: list[str] = []
    if _APPROVAL_FRAMING.search(text):
        for terms in _TECH_FAMILIES.values():
            for term in terms:
                if term in lower:
                    forbid.append(term)
    for grant, patterns in _KNOWN_GRANTS.items():
        if any(re.search(pattern, lower) for pattern in patterns):
            grants.append(grant)
    return {"forbid_terms": sorted(set(forbid)), "grants": sorted(set(grants))}


class StandingRuleEngine:
    """Durable project constraints set explicitly by the human.

    Standing rules may *reduce* approval prompts (e.g. a cost waiver) or *force*
    a major-decision review when a proposal conflicts with an approved choice.
    """

    def __init__(self, store: HistoryStore | None = None, rules: list[StandingRule] | None = None):
        self._store = store
        self._rules: list[StandingRule] = []
        if rules is not None:
            for rule in rules:
                if not rule.forbid_terms and not rule.grants:
                    metadata = derive_rule_metadata(rule.text)
                    rule.forbid_terms = metadata["forbid_terms"]
                    rule.grants = metadata["grants"]
                self._rules.append(rule)
        else:
            self.reload()

    def reload(self) -> None:
        if self._store is None:
            return
        self._rules = [StandingRule.from_dict(row) for row in self._store.list_standing_rules()]

    @property
    def rules(self) -> list[StandingRule]:
        return list(self._rules)

    def add_rule(
        self,
        text: str,
        *,
        forbid_terms: list[str] | None = None,
        grants: list[str] | None = None,
        tags: list[str] | None = None,
    ) -> StandingRule:
        metadata = derive_rule_metadata(text)
        final_forbid = forbid_terms if forbid_terms is not None else metadata["forbid_terms"]
        final_grants = grants if grants is not None else metadata["grants"]
        rule = StandingRule(
            text=text,
            forbid_terms=final_forbid,
            grants=final_grants,
            tags=list(tags or []),
        )
        if self._store is not None:
            rule.id = self._store.add_standing_rule(
                text,
                forbid_terms=final_forbid,
                grants=final_grants,
                tags=tags,
            )
        self._rules.append(rule)
        return rule

    def detect_conflicts(self, proposal_text: str) -> list[str]:
        lower = proposal_text.lower()
        conflicts: list[str] = []
        for rule in self._rules:
            if not rule.active:
                continue
            allowed: set[str] = set()
            families: set[str] = set()
            for term in rule.forbid_terms:
                family = _family_for(term)
                if family is not None:
                    families.add(family)
                    allowed.add(term)
            conflict = False
            for family in families:
                for candidate in _TECH_FAMILIES[family]:
                    if candidate in lower and candidate not in allowed:
                        conflict = True
                        break
                if conflict:
                    break
            if conflict:
                conflicts.append(rule.text)
        return conflicts

    def has_grant(self, grant: str) -> bool:
        return any(grant in rule.grants for rule in self._rules if rule.active)

    def cost_waiver_threshold(self) -> float | None:
        threshold: float | None = None
        for rule in self._rules:
            if not rule.active:
                continue
            lower = rule.text.lower()
            if "under" in lower and ("approval" in lower or "ask" in lower):
                match = _COST_WAIVER.search(rule.text)
                if match:
                    value = float(match.group(1))
                    threshold = max(threshold or 0.0, value)
        return threshold


def _family_for(term: str) -> str | None:
    for family, terms in _TECH_FAMILIES.items():
        if term in terms:
            return family
    return None
