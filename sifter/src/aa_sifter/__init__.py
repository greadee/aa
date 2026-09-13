"""aa-sifter: one adaptive local/cloud compute interface with a human approval gate."""

from .app import ComputeSifter, SifterResult
from .decisions.approval import (
    ApprovalAction,
    ApprovalOption,
    ApprovalRequest,
    HumanDecision,
)
from .decisions.classifier import DecisionClassification, DecisionLevel
from .memory import MemorySink, NopMemorySink
from .models.config import ModelConfig, SifterConfig
from .models.provider import GenerationResult, Message, Tier
from .recommend.recommender import recommend_profile
from .system.hardware import detect_hardware

Sifter = ComputeSifter

__version__ = "0.1.0"

__all__ = [
    "ComputeSifter",
    "Sifter",
    "SifterResult",
    "SifterConfig",
    "ModelConfig",
    "DecisionLevel",
    "DecisionClassification",
    "ApprovalAction",
    "ApprovalOption",
    "ApprovalRequest",
    "HumanDecision",
    "GenerationResult",
    "Message",
    "Tier",
    "MemorySink",
    "NopMemorySink",
    "detect_hardware",
    "recommend_profile",
    "__version__",
]
