"""aa-inference: one adaptive local/cloud compute interface with a human approval gate."""

from .app import ComputeInference, InferenceResult
from .decisions.approval import (
    ApprovalAction,
    ApprovalOption,
    ApprovalRequest,
    HumanDecision,
)
from .decisions.classifier import DecisionClassification, DecisionLevel
from .memory import MemorySink, NopMemorySink
from .models.config import InferenceConfig, ModelConfig
from .models.provider import GenerationResult, Message, Tier
from .recommend.recommender import recommend_profile
from .system.hardware import detect_hardware

Inference = ComputeInference

__version__ = "0.1.0"

__all__ = [
    "ComputeInference",
    "Inference",
    "InferenceResult",
    "InferenceConfig",
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
