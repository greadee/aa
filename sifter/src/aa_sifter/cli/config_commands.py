from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from ..catalog.catalog import ModelCatalog
from ..config import (
    ApplicationConfig,
    ComputeProfile,
    HardwareProfile,
    ModelTierConfig,
    UserGoal,
    export_profile,
    import_profile,
    save_application_config,
    validate_profile,
)
from ..doctor import build_doctor_report, live_expert_check
from ..recommend.recommender import recommend_profile
from ..system.hardware import detect_hardware

CONFIG_COMMANDS = {"profile", "system", "model", "recommend", "setup", "doctor"}


# ---------------------------------------------------------------------------
# helpers
# ---------------------------------------------------------------------------
def _load(path: Path) -> ApplicationConfig:
    from ..config.store import load_application_config

    return load_application_config(path)


def _save(app: ApplicationConfig, path: Path) -> None:
    save_application_config(app, path)


def _print_tier(label: str, tier: ModelTierConfig | None) -> None:
    if tier is None:
        print(f"{label}: (not configured)")
        return
    print(f"{label}: {tier.provider} / {tier.model}")
    if tier.context_limit:
        print(f"    context: {tier.context_limit}")
    if tier.endpoint:
        print(f"    endpoint: {tier.endpoint}")


def _profile_summary(profile: ComputeProfile) -> None:
    _print_tier("Local", profile.local)
    _print_tier("Expert", profile.expert)
    print(f"Routing: {profile.routing_policy}")
    print(
        f"Approval: major={profile.approval.major} critical={profile.approval.critical} "
        f"significant={profile.approval.significant}"
    )
    print(
        "Preferences: "
        f"local_first={profile.preferences.local_first} "
        f"cost_sensitive={profile.preferences.cost_sensitive} "
        f"privacy_sensitive={profile.preferences.privacy_sensitive} "
        f"cloud_allowed={profile.preferences.cloud_allowed}"
    )


def _goal_from_args(args: argparse.Namespace) -> UserGoal:
    workload = list(args.goal or [])
    if not workload:
        workload = ["software development", "software engineering agents"]
    cloud_allowed = args.cloud if args.cloud is not None else True
    return UserGoal(
        workload=workload,
        quality_priority=args.quality if args.quality is not None else 0.5,
        speed_priority=args.speed if args.speed is not None else 0.5,
        cost_priority=args.cost if args.cost is not None else 0.5,
        privacy_priority=args.privacy if args.privacy is not None else 0.5,
        cloud_allowed=cloud_allowed,
        preferred_context=args.context_limit,
    )


def _hardware_from_args(args: argparse.Namespace, app: ApplicationConfig) -> HardwareProfile | None:
    if args.gpu or args.vram or args.ram or args.cpu:
        vendor = None
        name = args.gpu
        if name:
            lowered = name.lower()
            if "nvidia" in lowered or "rtx" in lowered or "gtx" in lowered:
                vendor = "NVIDIA"
            elif "radeon" in lowered or "amd" in lowered:
                vendor = "AMD"
            elif "apple" in lowered:
                vendor = "Apple"
        return HardwareProfile(
            cpu_name=args.cpu,
            gpu_name=name,
            gpu_vendor=vendor,
            gpu_vram_gb=args.vram,
            system_ram_gb=args.ram,
            cpu_cores=args.cores,
            cpu_threads=args.threads,
            source="manual",
        )
    if args.detect:
        return detect_hardware()
    if app.active_hardware and app.active_hardware in app.hardware:
        return app.hardware[app.active_hardware]
    return None


def _print_recommendation(rec) -> None:
    print("")
    print("Recommended configuration")
    print("=" * 26)
    _print_tier("Local", rec.profile.local)
    _print_tier("Expert", rec.profile.expert)
    print(f"Routing: {rec.profile.routing_policy}")
    print(f"Confidence: {rec.confidence}")
    if rec.reasons:
        print("")
        print("Why:")
        for reason in rec.reasons:
            print(f"  - {reason}")
    if rec.local_alternatives:
        print("")
        print("Local alternatives:")
        for alt in rec.local_alternatives:
            print(f"  - {alt.provider} / {alt.model}")
    if rec.expert_alternatives:
        print("")
        print("Expert alternatives:")
        for alt in rec.expert_alternatives:
            print(f"  - {alt.provider} / {alt.model}")
    if rec.notes:
        print("")
        print("Notes:")
        for note in rec.notes:
            print(f"  - {note}")


# ---------------------------------------------------------------------------
# profile
# ---------------------------------------------------------------------------
def cmd_profile(args: argparse.Namespace, query: list[str], config_path: Path) -> int:
    app = _load(config_path)
    action = query[1] if len(query) > 1 else "list"

    if action == "list":
        for name, prof in app.profiles.items():
            marker = "*" if name == app.active_profile else " "
            local = prof.local.model if prof.local else "-"
            expert = prof.expert.model if prof.expert and prof.expert.enabled else "-"
            print(f"{marker} {name}: local={local} expert={expert} routing={prof.routing_policy}")
        print(f"\nActive profile: {app.active_profile}")
        return 0

    if action == "show":
        name = query[2] if len(query) > 2 else app.active_profile
        profile = app.profiles.get(name)
        if profile is None:
            print(f"ERROR: profile '{name}' does not exist")
            return 1
        print(f"Profile: {name}{' (active)' if name == app.active_profile else ''}")
        _profile_summary(profile)
        return 0

    if action in {"create", "copy"}:
        if action == "create":
            if len(query) < 3:
                print("ERROR: usage: aa_sifter profile create <name>")
                return 1
            source_name = args.source or app.active_profile
            target_name = query[2]
        else:
            if len(query) < 4:
                print("ERROR: usage: aa_sifter profile copy <source> <target>")
                return 1
            source_name, target_name = query[2], query[3]
        if target_name in app.profiles and not args.force:
            print(f"ERROR: profile '{target_name}' already exists (use --force to overwrite)")
            return 1
        source = app.profiles.get(source_name)
        if source is None:
            print(f"ERROR: source profile '{source_name}' does not exist")
            return 1
        copied = source.model_copy(deep=True)
        copied.name = target_name
        app.profiles[target_name] = copied
        _save(app, config_path)
        print(f"Created profile '{target_name}' from '{source_name}'.")
        return 0

    if action == "use":
        if len(query) < 3:
            print("ERROR: usage: aa_sifter profile use <name>")
            return 1
        if query[2] not in app.profiles:
            print(f"ERROR: profile '{query[2]}' does not exist")
            return 1
        app.active_profile = query[2]
        _save(app, config_path)
        print(f"Active profile: {app.active_profile}")
        return 0

    if action == "delete":
        if len(query) < 3:
            print("ERROR: usage: aa_sifter profile delete <name>")
            return 1
        name = query[2]
        if name not in app.profiles:
            print(f"ERROR: profile '{name}' does not exist")
            return 1
        if len(app.profiles) == 1:
            print("ERROR: cannot delete the last profile")
            return 1
        del app.profiles[name]
        if app.active_profile == name:
            app.active_profile = next(iter(app.profiles))
            print(f"Active profile switched to '{app.active_profile}'.")
        _save(app, config_path)
        print(f"Deleted profile '{name}'.")
        return 0

    if action == "edit":
        name = query[2] if len(query) > 2 else app.active_profile
        profile = app.profiles.get(name)
        if profile is None:
            print(f"ERROR: profile '{name}' does not exist")
            return 1
        changed = _apply_profile_edits(profile, args)
        if changed:
            _save(app, config_path)
            print(f"Updated profile '{name}'.")
        else:
            print(f"Profile file: {config_path}")
            print(
                "Edit it directly, or pass flags such as --local-model, --expert-model, "
                "--routing, --context-limit, --no-cloud."
            )
            _profile_summary(profile)
        return 0

    if action == "validate":
        name = query[2] if len(query) > 2 else app.active_profile
        profile = app.profiles.get(name)
        if profile is None:
            print(f"ERROR: profile '{name}' does not exist")
            return 1
        hardware = profile.hardware or (
            app.hardware.get(app.active_hardware) if app.active_hardware else None
        )
        result = validate_profile(profile, hardware=hardware)
        print(f"Validating profile '{name}'")
        for error in result.errors:
            print(f"ERROR: {error}")
        for warning in result.warnings:
            print(f"WARNING: {warning}")
        print("PASS" if result.ok else "FAILED")
        return 0 if result.ok else 1

    if action == "export":
        if len(query) < 3:
            print("ERROR: usage: aa_sifter profile export <name> [--output file]")
            return 1
        try:
            text = export_profile(app, query[2])
        except KeyError:
            print(f"ERROR: profile '{query[2]}' does not exist")
            return 1
        if args.output:
            Path(args.output).write_text(text, encoding="utf-8")
            print(f"Exported profile '{query[2]}' to {args.output}")
        else:
            print(text)
        return 0

    if action == "import":
        if len(query) < 3:
            print("ERROR: usage: aa_sifter profile import <file>")
            return 1
        source_path = Path(query[2])
        if not source_path.exists():
            print(f"ERROR: file '{source_path}' not found")
            return 1
        try:
            profile = import_profile(source_path.read_text(encoding="utf-8"))
        except Exception as exc:  # noqa: BLE001 - report config problems
            print(f"ERROR: could not import profile: {exc}")
            return 1
        if args.name:
            profile.name = args.name
        if profile.name in app.profiles and not args.force:
            print(f"ERROR: profile '{profile.name}' already exists (use --force)")
            return 1
        app.profiles[profile.name] = profile
        _save(app, config_path)
        print(f"Imported profile '{profile.name}'.")
        return 0

    print(f"ERROR: unknown profile action '{action}'")
    return 1


def _apply_profile_edits(profile: ComputeProfile, args: argparse.Namespace) -> bool:
    changed = False
    if profile.local is None:
        profile.local = ModelTierConfig(provider="ollama", model="")
    if profile.expert is None:
        profile.expert = ModelTierConfig(provider="deepseek", model="")

    mapping = {
        "local_provider": profile.local,
        "local_model": profile.local,
        "local_endpoint": profile.local,
        "local_api_key_env": profile.local,
        "expert_provider": profile.expert,
        "expert_model": profile.expert,
        "expert_endpoint": profile.expert,
        "expert_api_key_env": profile.expert,
    }
    field_map = {
        "local_provider": "provider",
        "local_model": "model",
        "local_endpoint": "endpoint",
        "local_api_key_env": "api_key_env",
        "expert_provider": "provider",
        "expert_model": "model",
        "expert_endpoint": "endpoint",
        "expert_api_key_env": "api_key_env",
    }
    for arg_name, field in field_map.items():
        value = getattr(args, arg_name, None)
        if value is not None:
            setattr(mapping[arg_name], field, value)
            changed = True
    if args.context_limit is not None:
        profile.local.context_limit = args.context_limit
        changed = True
    if args.max_output_tokens is not None:
        profile.local.max_output_tokens = args.max_output_tokens
        changed = True
    if args.routing:
        profile.routing_policy = args.routing
        changed = True
    if args.cloud is not None:
        profile.preferences.cloud_allowed = args.cloud
        if profile.expert is not None:
            profile.expert.enabled = args.cloud
        changed = True
    if args.local_first is not None:
        profile.preferences.local_first = args.local_first
        changed = True
    return changed


# ---------------------------------------------------------------------------
# system
# ---------------------------------------------------------------------------
def cmd_system(args: argparse.Namespace, query: list[str], config_path: Path) -> int:
    app = _load(config_path)
    action = query[1] if len(query) > 1 else "show"

    if action == "detect":
        hardware = detect_hardware()
        _print_hardware(hardware)
        if args.save:
            app.hardware[args.save] = hardware
            app.active_hardware = args.save
            _save(app, config_path)
            print(f"Saved hardware profile '{args.save}'.")
        return 0

    if action == "show":
        name = query[2] if len(query) > 2 else app.active_hardware
        if name and name in app.hardware:
            print(f"Hardware profile: {name}")
            _print_hardware(app.hardware[name])
            return 0
        print("No active hardware profile. Run 'aa_sifter system detect --save desktop'.")
        return 0

    if action == "list":
        if not app.hardware:
            print("No hardware profiles.")
            return 0
        for name, hardware in app.hardware.items():
            marker = "*" if name == app.active_hardware else " "
            print(
                f"{marker} {name}: {hardware.gpu_name or 'no GPU'} "
                f"({hardware.gpu_vram_gb or '?'} GB VRAM), "
                f"{hardware.system_ram_gb or '?'} GB RAM"
            )
        return 0

    if action == "use":
        if len(query) < 3:
            print("ERROR: usage: aa_sifter system use <name>")
            return 1
        if query[2] not in app.hardware:
            print(f"ERROR: hardware profile '{query[2]}' does not exist")
            return 1
        app.active_hardware = query[2]
        _save(app, config_path)
        print(f"Active hardware profile: {app.active_hardware}")
        return 0

    if action == "configure":
        hardware = HardwareProfile(
            cpu_name=args.cpu,
            cpu_cores=args.cores,
            cpu_threads=args.threads,
            gpu_name=args.gpu,
            gpu_vram_gb=args.vram,
            system_ram_gb=args.ram,
            source="manual",
        )
        name = args.save or "default"
        app.hardware[name] = hardware
        app.active_hardware = name
        _save(app, config_path)
        print(f"Saved manual hardware profile '{name}'.")
        _print_hardware(hardware)
        return 0

    print(f"ERROR: unknown system action '{action}'")
    return 1


def _print_hardware(hardware: HardwareProfile) -> None:
    print("Detected System")
    print("")
    print("CPU:")
    print(hardware.cpu_name or "unknown")
    print("")
    print("GPU:")
    print(hardware.gpu_name or "none detected")
    print("")
    print("VRAM:")
    print(f"{hardware.gpu_vram_gb} GB" if hardware.gpu_vram_gb else "unknown")
    print("")
    print("RAM:")
    print(f"{hardware.system_ram_gb} GB" if hardware.system_ram_gb else "unknown")
    print("")
    print("OS:")
    print(hardware.operating_system or "unknown")


# ---------------------------------------------------------------------------
# model
# ---------------------------------------------------------------------------
def cmd_model(args: argparse.Namespace, query: list[str], config_path: Path) -> int:
    app = _load(config_path)
    action = query[1] if len(query) > 1 else "list"
    catalog = ModelCatalog()

    if action == "list":
        installed = _ollama_installed() if args.installed else None
        local_only = args.tier == "local"
        expert_only = args.tier == "expert"
        for model in catalog.all():
            if local_only and not model.local:
                continue
            if expert_only and not model.cloud:
                continue
            status = ""
            if installed is not None and model.local and model.provider == "ollama":
                status = " [installed]" if model.model_id in installed else " [not installed]"
            tier_label = "local " if model.local else "expert"
            print(f"{tier_label} {model.provider}/{model.model_id}  {model.display_name}{status}")
        return 0

    if action == "show":
        profile = app.active()
        _print_tier("Local", profile.local)
        _print_tier("Expert", profile.expert)
        return 0

    if action == "set":
        if len(query) < 5:
            print("ERROR: usage: aa_sifter model set <local|expert> <provider> <model>")
            return 1
        tier_name, provider, model_id = query[2], query[3], query[4]
        profile = app.active()
        if tier_name == "local":
            if profile.local is None:
                profile.local = ModelTierConfig(provider=provider, model=model_id)
            else:
                profile.local.provider = provider
                profile.local.model = model_id
            selected_tier = profile.local
        elif tier_name == "expert":
            if profile.expert is None:
                profile.expert = ModelTierConfig(provider=provider, model=model_id)
            else:
                profile.expert.provider = provider
                profile.expert.model = model_id
            selected_tier = profile.expert
        else:
            print(f"ERROR: unknown tier '{tier_name}'")
            return 1
        if args.context_limit is not None:
            selected_tier.context_limit = args.context_limit
        if args.max_output_tokens is not None:
            selected_tier.max_output_tokens = args.max_output_tokens
        if args.endpoint:
            selected_tier.endpoint = args.endpoint
        if args.api_key_env:
            selected_tier.api_key_env = args.api_key_env
        if catalog.get(provider, model_id) is None:
            print(
                f"WARNING: '{provider}/{model_id}' is not in the Compute Sifter catalog; "
                "configuration is still allowed."
            )
        _save(app, config_path)
        print(f"Set {tier_name} tier to {provider}/{model_id} in profile '{profile.name}'.")
        return 0

    print(f"ERROR: unknown model action '{action}'")
    return 1


def _ollama_installed() -> set[str] | None:
    try:
        import httpx

        response = httpx.get("http://localhost:11434/api/tags", timeout=2.0)
        if response.status_code != 200:
            return None
        data = response.json()
        return {item.get("name", "") for item in data.get("models", [])}
    except Exception:  # noqa: BLE001 - availability check is best-effort
        return None


# ---------------------------------------------------------------------------
# recommend
# ---------------------------------------------------------------------------
def cmd_recommend(args: argparse.Namespace, query: list[str], config_path: Path) -> int:
    app = _load(config_path)
    hardware = _hardware_from_args(args, app)
    goal = _goal_from_args(args)
    rec = recommend_profile(hardware=hardware, goal=goal, name=args.name or "recommended")

    if args.json:
        print(json.dumps(rec.model_dump(mode="json"), indent=2, default=str))
    else:
        _print_recommendation(rec)

    if args.save:
        profile = rec.profile.model_copy(deep=True)
        profile.name = args.save
        if args.save in app.profiles and not args.force:
            print(f"ERROR: profile '{args.save}' exists (use --force)")
            return 1
        app.profiles[args.save] = profile
        if args.use:
            app.active_profile = args.save
        _save(app, config_path)
        print(f"\nSaved profile '{args.save}'.")
    return 0


# ---------------------------------------------------------------------------
# setup
# ---------------------------------------------------------------------------
def cmd_setup(args: argparse.Namespace, query: list[str], config_path: Path) -> int:
    interactive = not args.non_interactive and sys.stdin is not None and sys.stdin.isatty()
    app = _load(config_path)

    print("Compute Sifter Setup")
    print("")
    hardware = _hardware_from_args(args, app)
    if hardware is None and interactive:
        answer = _ask("Detect hardware automatically? [Y/n]", "y")
        if answer.strip().lower() in {"", "y", "yes"}:
            hardware = detect_hardware()
    if hardware is not None:
        print("")
        _print_hardware(hardware)

    goal = _goal_from_args(args)
    if interactive:
        raw = _ask(
            "Goals (comma separated) [software development, software engineering agents]",
            "",
        )
        if raw.strip():
            goal.workload = [item.strip() for item in raw.split(",") if item.strip()]
        cloud = _ask("Can Compute Sifter use paid cloud models? [Y/n]", "y")
        goal.cloud_allowed = cloud.strip().lower() in {"", "y", "yes"}
        local = _ask("Should local compute be preferred? [Y/n]", "y")
        goal.privacy_priority = 0.8 if local.strip().lower() in {"", "y", "yes"} else 0.3

    print("")
    print("Analyzing compatible models...")
    rec = recommend_profile(hardware=hardware, goal=goal, name=args.name or "default")
    _print_recommendation(rec)

    if interactive and not args.yes:
        choice = _ask("Would you like to [A]ccept, [C]ustomize, or [V]iew reasoning?", "a")
        lowered = choice.strip().lower()
        if lowered.startswith("v"):
            print("")
            print("Reasoning:")
            for reason in rec.reasons:
                print(f"  - {reason}")
            choice = _ask("Accept? [Y/n]", "y")
            lowered = choice.strip().lower()
        if lowered.startswith("c"):
            print("Customize with: aa_sifter model set local <provider> <model>")
            print(f"Then: aa_sifter profile use {args.name or 'default'}")
            return 0

    profile = rec.profile.model_copy(deep=True)
    profile.name = args.name or "default"
    if hardware is not None:
        hw_name = args.save or "detected"
        app.hardware[hw_name] = hardware
        app.active_hardware = hw_name
        profile.hardware = hardware
    app.profiles[profile.name] = profile
    app.active_profile = profile.name
    _save(app, config_path)
    print("")
    print(f"Saved profile '{profile.name}' to {config_path}")
    print('Run: aa_sifter "your task"')
    return 0


def _ask(prompt: str, default: str) -> str:
    try:
        value = input(f"{prompt} ")
    except (EOFError, KeyboardInterrupt):
        return default
    return value.strip() or default


# ---------------------------------------------------------------------------
# doctor (profile-aware)
# ---------------------------------------------------------------------------
def cmd_doctor(args: argparse.Namespace, query: list[str], config_path: Path) -> int:
    app = _load(config_path)
    report = build_doctor_report(
        app,
        config_path=config_path,
        live=args.live,
        ollama_installed_fn=_ollama_installed,
        live_check_fn=_live_expert_check,
    )
    return _emit_doctor(report["checks"], args.json)


def _live_expert_check(expert: ModelTierConfig) -> tuple[bool, str]:
    """One minimal paid request. Only runs with `aa_sifter doctor --live`."""
    return live_expert_check(expert)


def _emit_doctor(checks: list[dict[str, object]], as_json: bool) -> int:
    required_ok = all(bool(check["ok"]) for check in checks if not check["optional"])
    if as_json:
        print(json.dumps({"ok": required_ok, "checks": checks}, indent=2, default=str))
    else:
        print("Compute Sifter Doctor")
        print("")
        for check in checks:
            mark = "PASS" if check["ok"] else ("SKIP" if check["optional"] else "FAIL")
            detail = f" - {check['detail']}" if check["detail"] else ""
            print(f"[{mark}] {check['name']}{detail}")
        print("doctor: healthy" if required_ok else "doctor: problems detected")
    return 0 if required_ok else 1


def dispatch(command: str, args: argparse.Namespace, query: list[str], config_path: Path) -> int:
    handlers = {
        "profile": cmd_profile,
        "system": cmd_system,
        "model": cmd_model,
        "recommend": cmd_recommend,
        "setup": cmd_setup,
        "doctor": cmd_doctor,
    }
    return handlers[command](args, query, config_path)
