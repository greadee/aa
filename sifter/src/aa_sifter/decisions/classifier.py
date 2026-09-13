from __future__ import annotations

import re
from enum import StrEnum

from pydantic import BaseModel, Field


class DecisionLevel(StrEnum):
    ROUTINE = "routine"
    SIGNIFICANT = "significant"
    MAJOR = "major"
    CRITICAL = "critical"


_SEVERITY: dict[DecisionLevel, int] = {
    DecisionLevel.ROUTINE: 0,
    DecisionLevel.SIGNIFICANT: 1,
    DecisionLevel.MAJOR: 2,
    DecisionLevel.CRITICAL: 3,
}


def max_level(*levels: DecisionLevel) -> DecisionLevel:
    result = DecisionLevel.ROUTINE
    for level in levels:
        if _SEVERITY[level] > _SEVERITY[result]:
            result = level
    return result


def level_at_least(level: DecisionLevel, floor: DecisionLevel) -> bool:
    return _SEVERITY[level] >= _SEVERITY[floor]


class DecisionSignals(BaseModel):
    architecture_change: bool = False
    framework_change: bool = False
    database_change: bool = False
    persistence_change: bool = False
    auth_change: bool = False
    new_external_service: bool = False
    provider_change: bool = False
    api_incompatible: bool = False
    schema_migration: bool = False
    security_boundary: bool = False
    deployment_change: bool = False
    scope_expansion: bool = False
    remove_functionality: bool = False
    new_major_dependency: bool = False
    destructive: bool = False
    data_deletion: bool = False
    irreversible: bool = False
    credential_change: bool = False
    production_action: bool = False
    large_expenditure: bool = False
    confidential_exposure: bool = False
    security_control_removal: bool = False
    cross_file: bool = False
    minor_dependency: bool = False
    refactor: bool = False
    migration_required: bool = False

    def triggered(self) -> dict[str, bool]:
        return {k: v for k, v in self.model_dump().items() if v}

    def names(self) -> list[str]:
        return list(self.triggered().keys())


class DecisionClassification(BaseModel):
    level: DecisionLevel
    signals: DecisionSignals = Field(default_factory=DecisionSignals)
    reasons: list[str] = Field(default_factory=list)
    requires_human_approval: bool = False
    model_assisted: bool = False
    conflicts_with_rules: list[str] = Field(default_factory=list)

    @property
    def is_major(self) -> bool:
        return level_at_least(self.level, DecisionLevel.MAJOR)

    @property
    def signal_names(self) -> list[str]:
        return self.signals.names()


_CRITICAL_PATTERNS: dict[str, list[str]] = {
    "destructive": [
        r"\bdrop\s+(table|database|schema|column)\b",
        r"\btruncate\b",
        r"\bdelete\s+from\b",
        r"\brm\s+-rf\b",
        r"\bdestroy\b",
        r"\bwipe\b",
        r"\bpurge\b(?!\s+cache)",
        r"\breset\s+--hard\b",
        r"\bforce[- ]push\b",
    ],
    "data_deletion": [
        r"\bdelete\s+(all|every|the)\b.*\b(data|records|rows|users|database)\b",
        r"\berase\b.*\b(data|records|history)\b",
        r"\bremove\s+all\s+(data|records|users)\b",
    ],
    "irreversible": [
        r"\birreversible\b",
        r"\bcannot be undone\b",
        r"\bcan't be undone\b",
        r"\bno rollback\b",
        r"\bunrecoverable\b",
        r"\bpermanent(ly)?\s+(delete|remove|migrat)",
    ],
    "credential_change": [
        r"\brotate\s+(credentials|keys|secrets)\b",
        r"\bcredential\s+model\b",
        r"\bapi\s+key(s)?\b",
        r"\bprivate\s+key\b",
        r"\bsecret(s)?\b.*\b(config|store|rotate)\b",
    ],
    "production_action": [
        r"\bproduction\b",
        r"\bprod\s+(environment|server|deploy|database)\b",
        r"\blive\s+(system|traffic|users)\b",
    ],
    "large_expenditure": [
        r"\bbudget\s+increase\b",
        r"\blarge\s+(unexpected\s+)?(cost|expenditure|spend)",
        r"\bspend\s+\$?\d{3,}",
        r"\bcost\s+spike\b",
    ],
    "confidential_exposure": [
        r"\bexpose\s+(confidential|secrets|private)",
        r"\bupload\b.*\b(externally|to\s+the\s+cloud|third[- ]party)\b",
        r"\bsend\b.*\b(secrets|credentials|keys)\b",
        r"\bleak\b",
    ],
    "security_control_removal": [
        r"\bdisable\s+(auth|authentication|authorization|tls|ssl|https|security)\b",
        r"\bremove\s+(auth|authentication|authorization|security)\b",
        r"\bbypass\s+(auth|security|authentication)\b",
        r"\b(no|without)\s+authentication\b",
    ],
}

_MAJOR_PATTERNS: dict[str, list[str]] = {
    "architecture_change": [
        r"\barchitectur(e|al)\b",
        r"\bmicro[- ]?service",
        r"\bmonolith",
        r"\bre[- ]?architect\b",
        r"\bsystem\s+design\b",
    ],
    "framework_change": [
        r"\b(change|switch|replace|migrate|adopt)\b.*\bframework\b",
        r"\bnew\s+framework\b",
        r"\b(react|vue|angular|svelte|next\.?js|django|rails|laravel)\b",
    ],
    "database_change": [
        r"\b(changing|change|switch|replace|swap|migrate\s+to)\b.*\b(database|db)\b",
        r"\b(database|db)\s+technology\b",
        r"\b(postgres|postgresql|mysql|mariadb|mongodb|oracle|dynamodb)\b",
        r"\blarge\s+database\s+migration\b",
        r"\bschema\s+redesign\b",
        r"\bmajor\s+schema\b",
    ],
    "persistence_change": [
        r"\bpersistence\s+(architecture|layer|strategy|design)\b",
        r"\bre[- ]?design\b.*\bpersistence\b",
        r"\breplace\b.*\b(sqlite|persistence|storage\s+layer)\b",
        r"\bstorage\s+architecture\b",
    ],
    "auth_change": [
        r"\boauth\b",
        r"\boidc\b",
        r"\bsaml\b",
        r"\bsso\b",
        r"\bauth(entication)?\s+(architecture|provider|system|strategy|model)\b",
        r"\b(change|replace|switch|redesign|rework)\b.*\bauth(entication|orization)?\b",
        r"\bauth(entication|orization)?\b.*\b(architecture|provider|strategy)\b",
    ],
    "new_external_service": [
        r"\bnew\s+(external|third[- ]party|major)\s+(service|provider|api|vendor)\b",
        r"\bintroduc(e|ing)\b.*\b(service|vendor|provider)\b",
        r"\b(stripe|twilio|sendgrid|auth0|okta|datadog|sentry)\b",
    ],
    "provider_change": [
        r"\b(cloud|model|inference|llm)\s+provider\b",
        r"\bprovider\s+(layer|strategy|change|switch)\b",
        r"\breplace\s+(ollama|openai|anthropic|deepseek)\b",
        r"\bswitch\s+(to\s+)?(openai|anthropic|deepseek|bedrock|azure)\b",
    ],
    "api_incompatible": [
        r"\bbreaking\s+(change|api)\b",
        r"\bbackward(s)?[- ]incompatible\b",
        r"\bpublic\s+api\b.*\b(change|break)\b",
        r"\bchange\s+the\s+public\s+api\b",
    ],
    "schema_migration": [
        r"\b(schema|data)\s+migration\b",
        r"\bmigrate\s+(the\s+)?(schema|database)\b",
        r"\bschema\s+(change|version)\b",
    ],
    "security_boundary": [
        r"\bsecurity\s+(boundary|boundaries|model|posture)\b",
        r"\btrust\s+boundary\b",
        r"\bpermission\s+model\b",
    ],
    "deployment_change": [
        r"\bdeployment\s+architecture\b",
        r"\bkubernetes\b",
        r"\bk8s\b",
        r"\b(docker\s+swarm|infrastructure|hosting\s+provider)\b",
        r"\bdeploy(ment)?\s+(to|on)\b.*\b(cloud|aws|azure|gcp)\b",
    ],
    "scope_expansion": [
        r"\bexpanding?\s+(the\s+)?scope\b",
        r"\bmajor\s+scope\b",
        r"\bsubstantially\s+(expand|increase)\b",
        r"\bnew\s+major\s+(feature|capability|component)\b",
    ],
    "remove_functionality": [
        r"\bremove\s+(major|core|existing)\s+(functionality|feature|module)\b",
        r"\bdelete\s+(the\s+)?(feature|module|subsystem)\b",
    ],
    "new_major_dependency": [
        r"\bmajor\s+(third[- ]party\s+)?(dependency|library|package)\b",
        r"\badd\s+(a\s+)?(major|new\s+foundational)\s+(dependency|framework)\b",
    ],
}

_SIGNIFICANT_PATTERNS: dict[str, list[str]] = {
    "refactor": [r"\brefactor\b", r"\brestructure\b", r"\bmoderate\s+refactor\b"],
    "cross_file": [
        r"\b(several|multiple|many)\s+(related\s+)?files\b",
        r"\bacross\s+files\b",
        r"\bmultiple\s+modules\b",
    ],
    "minor_dependency": [
        r"\bminor\s+dependency\b",
        r"\badd\s+(a\s+)?(small|minor)\s+(dependency|library|package)\b",
    ],
    "migration_required": [r"\bmigration\b"],
}


def _search(patterns: list[str], text: str) -> bool:
    return any(re.search(pattern, text, flags=re.IGNORECASE) for pattern in patterns)


class DecisionClassifier:
    """Deterministic decision significance classifier.

    The classifier never *lowers* a level proposed by another signal source.
    The policy engine treats a higher level as authoritative.
    """

    def detect_signals(self, text: str) -> DecisionSignals:
        signals = DecisionSignals()
        for field, patterns in _CRITICAL_PATTERNS.items():
            if _search(patterns, text):
                setattr(signals, field, True)
        for field, patterns in _MAJOR_PATTERNS.items():
            if _search(patterns, text):
                setattr(signals, field, True)
        for field, patterns in _SIGNIFICANT_PATTERNS.items():
            if _search(patterns, text):
                setattr(signals, field, True)
        return signals

    def level_for_signals(self, signals: DecisionSignals) -> DecisionLevel:
        critical = [
            signals.destructive,
            signals.data_deletion,
            signals.irreversible,
            signals.credential_change,
            signals.production_action,
            signals.large_expenditure,
            signals.confidential_exposure,
            signals.security_control_removal,
        ]
        if any(critical):
            return DecisionLevel.CRITICAL
        major = [
            signals.architecture_change,
            signals.framework_change,
            signals.database_change,
            signals.persistence_change,
            signals.auth_change,
            signals.new_external_service,
            signals.provider_change,
            signals.api_incompatible,
            signals.schema_migration,
            signals.security_boundary,
            signals.deployment_change,
            signals.scope_expansion,
            signals.remove_functionality,
            signals.new_major_dependency,
        ]
        if any(major):
            return DecisionLevel.MAJOR
        if any(
            [
                signals.refactor,
                signals.cross_file,
                signals.minor_dependency,
                signals.migration_required,
            ]
        ):
            return DecisionLevel.SIGNIFICANT
        return DecisionLevel.ROUTINE

    def classify(
        self,
        text: str,
        *,
        model_level: DecisionLevel | None = None,
        model_reason: str | None = None,
        rule_conflicts: list[str] | None = None,
    ) -> DecisionClassification:
        signals = self.detect_signals(text)
        level = self.level_for_signals(signals)
        reasons: list[str] = []
        triggered = signals.triggered()
        if triggered:
            reasons.append("deterministic signals: " + ", ".join(triggered))
        if model_level is not None:
            reasons.append(
                f"model assessment suggested {model_level.value}"
                + (f" ({model_reason})" if model_reason else "")
            )
            level = max_level(level, model_level)
        if rule_conflicts:
            level = max_level(level, DecisionLevel.MAJOR)
            reasons.append("conflicts with standing rule(s): " + "; ".join(rule_conflicts))
        if level_at_least(level, DecisionLevel.MAJOR):
            requires_approval = True
        else:
            requires_approval = False
            reasons.append("routine or significant; autonomous execution permitted by default")
        return DecisionClassification(
            level=level,
            signals=signals,
            reasons=reasons,
            requires_human_approval=requires_approval,
            model_assisted=model_level is not None,
            conflicts_with_rules=list(rule_conflicts or []),
        )
