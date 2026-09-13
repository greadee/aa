from __future__ import annotations

import argparse
import asyncio
import json
import sys
from pathlib import Path

from ..app import ComputeSifter, SifterResult
from ..config import default_config_path, load_runtime_config
from ..models.config import SifterConfig
from . import config_commands, desktop_commands


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="aa-sifter",
        description=(
            "aa-sifter: route one task across local and cloud models "
            "with human authority over major decisions."
        ),
        epilog=(
            "configuration commands: setup, recommend, system, profile, model, doctor. "
            "Run a task by passing a prompt, or use 'aa-sifter chat'."
        ),
    )
    parser.add_argument(
        "query",
        nargs="*",
        help="task prompt or a subcommand (setup|recommend|system|profile|model|doctor|chat|rules|stats|config)",
    )
    parser.add_argument("--profile", default=None, help="activate a named profile for this run")
    parser.add_argument("--policy", default=None, help="routing policy override for this task")
    parser.add_argument("--routing", default=None, help="routing policy override")
    parser.add_argument("--context", default=None, help="inline context")
    parser.add_argument("--context-file", default=None, help="path to a context file")
    parser.add_argument(
        "--max-cloud-cost", type=float, default=None, help="cloud cost cap for this task"
    )
    parser.add_argument("--debug", action="store_true", help="show routing trace and metrics")
    parser.add_argument(
        "--non-interactive", action="store_true", help="refuse major decisions automatically"
    )
    parser.add_argument("--json", action="store_true", help="emit machine-readable JSON")

    # model overrides
    parser.add_argument("--local-provider", default=None)
    parser.add_argument("--local-model", default=None)
    parser.add_argument("--local-endpoint", default=None)
    parser.add_argument("--local-api-key-env", default=None)
    parser.add_argument("--expert-provider", default=None)
    parser.add_argument("--expert-model", default=None)
    parser.add_argument("--expert-endpoint", default=None)
    parser.add_argument("--expert-api-key-env", default=None)
    parser.add_argument("--context-limit", type=int, default=None)
    parser.add_argument("--max-output-tokens", type=int, default=None)
    parser.add_argument("--endpoint", default=None, help="endpoint for `model set`")
    parser.add_argument("--api-key-env", default=None, help="api key env var for `model set`")

    # hardware / goal
    parser.add_argument("--detect", action="store_true", help="auto-detect hardware")
    parser.add_argument("--gpu", default=None)
    parser.add_argument("--vram", type=float, default=None)
    parser.add_argument("--ram", type=float, default=None)
    parser.add_argument("--cpu", default=None)
    parser.add_argument("--cores", type=int, default=None)
    parser.add_argument("--threads", type=int, default=None)
    parser.add_argument("--goal", action="append", default=None, help="workload goal (repeatable)")
    parser.add_argument("--quality", type=float, default=None)
    parser.add_argument("--speed", type=float, default=None)
    parser.add_argument("--cost", type=float, default=None)
    parser.add_argument("--privacy", type=float, default=None)
    parser.add_argument("--cloud", action=argparse.BooleanOptionalAction, default=None)
    parser.add_argument("--local-first", action=argparse.BooleanOptionalAction, default=None)

    # profile / system management
    parser.add_argument("--save", default=None, help="save under this profile/hardware name")
    parser.add_argument("--source", dest="source", default=None, help="source profile to copy")
    parser.add_argument("--name", default=None, help="name for imported/recommended profile")
    parser.add_argument("--output", default=None, help="output file")
    parser.add_argument("--force", action="store_true")
    parser.add_argument("--use", action="store_true", help="activate the saved profile")
    parser.add_argument("--tier", choices=["local", "expert"], default=None)
    parser.add_argument("--installed", action="store_true", help="check local model availability")
    parser.add_argument("--live", action="store_true", help="allow one paid live check")
    parser.add_argument("--yes", action="store_true", help="accept recommendations automatically")

    # desktop / launcher
    parser.add_argument("--host", default="127.0.0.1", help="backend bind host")
    parser.add_argument("--port", type=int, default=0, help="backend port (0 = auto)")
    parser.add_argument("--token", default=None, help="session token for the local API")
    parser.add_argument("--state-file", default=None, help="desktop state file path")
    parser.add_argument("--foreground", action="store_true", help="run the backend in-process")
    parser.add_argument("--headless", action="store_true", help="start without opening a UI")
    parser.add_argument("--no-browser", action="store_true", help="do not open a browser")
    parser.add_argument("--stop", action="store_true", help="stop a running desktop backend")
    return parser


def _load_context(args: argparse.Namespace) -> str | None:
    if args.context and args.context_file:
        raise SystemExit("Provide either --context or --context-file, not both")
    if args.context_file:
        return Path(args.context_file).read_text(encoding="utf-8")
    return args.context


def _print_result(result: SifterResult, args: argparse.Namespace) -> None:
    if args.json:
        print(json.dumps(result.model_dump(mode="json"), indent=2, default=str))
        return
    if result.status in {"completed", "done"}:
        print(result.answer)
    else:
        print(f"[status: {result.status}]")
        if result.answer:
            print(result.answer)
        if result.blocked_reason:
            print(f"Reason: {result.blocked_reason}")
    if args.debug:
        usage = result.usage
        print("", file=sys.stderr)
        print(
            f"[aa_sifter] route={result.route} decision={result.decision_level.value} "
            f"status={result.status}",
            file=sys.stderr,
        )
        print(
            f"[metrics] local_calls={usage.local_calls} cloud_calls={usage.cloud_calls} "
            f"tokens={usage.total_tokens} cloud_cost=${usage.cloud_cost:.6f} "
            f"retries={usage.retries} escalations={usage.escalations} "
            f"approvals={usage.human_approval_requests} duration_ms={usage.wall_clock_ms:.0f}",
            file=sys.stderr,
        )


async def _run_once(
    aa_sifter: ComputeSifter, args: argparse.Namespace, prompt: str, context: str | None
) -> SifterResult:
    return await aa_sifter.run(
        prompt,
        context=context,
        policy=args.policy,
        max_cloud_cost=args.max_cloud_cost,
    )


async def _chat(aa_sifter: ComputeSifter, args: argparse.Namespace, context: str | None) -> int:
    print("aa-sifter interactive mode. Type 'exit' to quit.")
    while True:
        try:
            prompt = input("\naa-sifter> ").strip()
        except (EOFError, KeyboardInterrupt):
            print()
            break
        if not prompt:
            continue
        if prompt.lower() in {"exit", "quit", ":q"}:
            break
        result = await _run_once(aa_sifter, args, prompt, context)
        _print_result(result, args)
    return 0


def _handle_rules(aa_sifter: ComputeSifter, query: list[str]) -> int:
    action = query[1] if len(query) > 1 else "list"
    if action == "list":
        rules = aa_sifter.rules.rules
        if not rules:
            print("No standing rules.")
            return 0
        for rule in rules:
            status = "active" if rule.active else "inactive"
            print(f"[{rule.id}] ({status}) {rule.text}")
            if rule.forbid_terms:
                print(f"      forbid: {', '.join(rule.forbid_terms)}")
            if rule.grants:
                print(f"      grants: {', '.join(rule.grants)}")
        return 0
    if action == "add":
        text = " ".join(query[2:]).strip()
        if not text:
            raise SystemExit('Usage: aa_sifter rules add "rule text"')
        rule = aa_sifter.rules.add_rule(text)
        print(f"Added standing rule [{rule.id}]: {rule.text}")
        return 0
    if action == "remove":
        if len(query) < 3:
            raise SystemExit("Usage: aa_sifter rules remove <id>")
        ok = (
            aa_sifter.history.deactivate_standing_rule(int(query[2]))
            if aa_sifter.history is not None
            else False
        )
        aa_sifter.rules.reload()
        print("Removed." if ok else "No such rule.")
        return 0
    raise SystemExit(f"Unknown rules action: {action}")


def _show_stats(aa_sifter: ComputeSifter) -> int:
    if aa_sifter.history is None:
        print("No history store available.")
        return 1
    print(json.dumps(aa_sifter.history.stats(), indent=2, default=str))
    return 0


def _show_config(config: SifterConfig) -> int:
    print(json.dumps(config.model_dump(), indent=2, default=str))
    return 0


def _runtime_overrides(args: argparse.Namespace) -> dict[str, object]:
    overrides: dict[str, object] = {}
    if args.max_cloud_cost is not None:
        overrides["max_cloud_cost_per_task"] = args.max_cloud_cost
    if args.local_provider:
        overrides["local_provider"] = args.local_provider
    if args.local_model:
        overrides["local_model"] = args.local_model
    if args.local_endpoint:
        overrides["local_endpoint"] = args.local_endpoint
    if args.expert_provider:
        overrides["expert_provider"] = args.expert_provider
    if args.expert_model:
        overrides["cloud_model"] = args.expert_model
    if args.expert_endpoint:
        overrides["expert_endpoint"] = args.expert_endpoint
    if args.context_limit:
        overrides["local_context_limit"] = args.context_limit
    if args.routing:
        overrides["routing_policy"] = args.routing
    if args.cloud is not None:
        overrides["cloud_allowed"] = args.cloud
    return overrides


async def _main_async(args: argparse.Namespace) -> int:
    config_path = default_config_path()
    query: list[str] = list(args.query)
    command = query[0] if query else None

    if command in desktop_commands.DESKTOP_COMMANDS:
        return desktop_commands.dispatch(command, args)

    if command in config_commands.CONFIG_COMMANDS:
        return config_commands.dispatch(command, args, query, config_path)

    config, warnings = load_runtime_config(
        profile_name=args.profile,
        overrides=_runtime_overrides(args),
    )
    if args.debug:
        config = config.model_copy(update={"debug": True})
    if args.non_interactive:
        config = config.model_copy(update={"non_interactive": True})
    for warning in warnings:
        print(f"[config] {warning}", file=sys.stderr)

    aa_sifter = ComputeSifter(config)
    try:
        if command == "rules":
            return _handle_rules(aa_sifter, query)
        if command == "stats":
            return _show_stats(aa_sifter)
        if command == "config":
            return _show_config(aa_sifter.config)
        context = _load_context(args)
        if command == "chat":
            return await _chat(aa_sifter, args, context)
        if not query:
            build_parser().print_help()
            return 1
        prompt = " ".join(query)
        result = await _run_once(aa_sifter, args, prompt, context)
        _print_result(result, args)
        return 0 if result.status != "error" else 1
    finally:
        await aa_sifter.aclose()


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    try:
        return asyncio.run(_main_async(args))
    except KeyboardInterrupt:
        return 130


if __name__ == "__main__":
    raise SystemExit(main())
