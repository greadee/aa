from __future__ import annotations

import pytest

from aa_sifter.config.defaults import default_profile
from aa_sifter.config.schema import ApplicationConfig, ApprovalConfig, HardwareProfile, UserGoal


def test_default_profile_is_ollama_local_and_deepseek_expert():
    profile = default_profile()
    assert profile.local is not None
    assert profile.local.provider == "ollama"
    assert profile.local.model == "qwen3.5:9b"
    assert profile.expert is not None
    assert profile.expert.provider == "deepseek"
    assert profile.expert.model == "deepseek-chat"
    assert profile.expert.api_key_env == "DEEPSEEK_API_KEY"
    assert profile.routing_policy == "adaptive"


def test_approval_gates_cannot_be_disabled():
    approval = ApprovalConfig(major=False, critical=False)
    assert approval.major is True
    assert approval.critical is True
    approval.major = False
    assert approval.major is True


def test_user_goal_capabilities_for_agent_workload():
    goal = UserGoal(workload=["software development", "software engineering agents"])
    required = goal.required_capabilities()
    assert {"coding", "tool_use", "agentic"} <= required


def test_user_goal_defaults_to_general():
    assert UserGoal(workload=[]).required_capabilities() == {"general"}


def test_hardware_has_gpu():
    assert HardwareProfile(gpu_name="RTX 3080", gpu_vram_gb=10).has_gpu is True
    assert HardwareProfile().has_gpu is False


def test_active_profile_missing_raises():
    config = ApplicationConfig(active_profile="nope", profiles={})
    with pytest.raises(KeyError):
        config.active()
