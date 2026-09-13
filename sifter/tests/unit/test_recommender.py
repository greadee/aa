from __future__ import annotations

from aa_sifter.catalog import ModelCatalog
from aa_sifter.config.schema import HardwareProfile, UserGoal
from aa_sifter.recommend.recommender import recommend_context, recommend_profile


def _desktop_10gb() -> HardwareProfile:
    return HardwareProfile(
        cpu_name="AMD Ryzen 5 9600X",
        gpu_name="NVIDIA GeForce RTX 3080",
        gpu_vendor="NVIDIA",
        gpu_vram_gb=10.0,
        system_ram_gb=32.0,
        source="manual",
    )


def _coding_goal(**overrides) -> UserGoal:
    defaults = {
        "workload": ["software development", "software engineering agents"],
        "quality_priority": 0.5,
        "cost_priority": 0.7,
        "privacy_priority": 0.6,
        "cloud_allowed": True,
    }
    defaults.update(overrides)
    return UserGoal(**defaults)


def test_10gb_coding_recommends_comfortable_9b_over_oversized():
    rec = recommend_profile(hardware=_desktop_10gb(), goal=_coding_goal(), catalog=ModelCatalog())
    assert rec.profile.local is not None
    assert rec.profile.local.model == "qwen3.5:9b"
    assert rec.profile.local.model != "qwen3.5:32b"
    assert rec.profile.expert is not None
    assert rec.profile.expert.provider == "deepseek"
    assert rec.confidence in {"medium", "high"}
    assert any("comfortab" in reason.lower() for reason in rec.reasons)


def test_context_recommendation_for_9b_on_10gb_is_operational_not_max():
    catalog = ModelCatalog()
    model = catalog.get("ollama", "qwen3.5:9b")
    assert model is not None
    context, notes = recommend_context(model, vram=10.0)
    assert 32768 <= context <= 131072
    assert context < model.context_window
    assert notes


def test_no_gpu_high_quality_cloud_profile():
    hardware = HardwareProfile(system_ram_gb=16.0, source="manual")
    goal = UserGoal(
        workload=["general assistant"],
        quality_priority=0.9,
        cost_priority=0.2,
        cloud_allowed=True,
    )
    rec = recommend_profile(hardware=hardware, goal=goal, catalog=ModelCatalog())
    assert rec.profile.expert is not None
    assert rec.profile.routing_policy != "local_only"


def test_privacy_goal_excludes_cloud():
    goal = _coding_goal(cloud_allowed=False, privacy_priority=1.0)
    rec = recommend_profile(hardware=_desktop_10gb(), goal=goal, catalog=ModelCatalog())
    assert rec.profile.expert is None
    assert rec.profile.routing_policy == "local_only"
    assert any("cloud" in note.lower() for note in rec.notes)


def test_recommendation_has_explanation_and_confidence():
    rec = recommend_profile(hardware=_desktop_10gb(), goal=_coding_goal(), catalog=ModelCatalog())
    assert rec.reasons
    assert rec.confidence in {"low", "medium", "high"}
    assert rec.profile.preferences.local_first is True
