from __future__ import annotations

import asyncio
import queue
import threading
import time
import uuid
from collections.abc import Callable
from contextvars import ContextVar
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from .. import __version__
from ..app import ComputeSifter
from ..catalog.catalog import ModelCatalog
from ..config.defaults import default_application_config
from ..config.profiles import (
    activate_profile,
    create_profile,
    delete_profile,
    get_profile,
    list_profiles,
    profile_summary,
    set_tier,
    update_profile,
)
from ..config.resolve import load_runtime_config
from ..config.schema import ApplicationConfig
from ..config.store import default_config_path, load_application_config, save_application_config
from ..config.validation import validate_profile
from ..decisions.approval import (
    ApprovalAction,
    ApprovalProvider,
    ApprovalRequest,
    HumanDecision,
)
from ..doctor import build_doctor_report, ollama_installed
from ..models.config import SifterConfig
from ..models.provider import ModelProvider
from ..system.hardware import detect_hardware
from .errors import translate_startup_issues

_CURRENT_TASK: ContextVar[TaskRecord | None] = ContextVar("sifter_current_task", default=None)

_TERMINAL_STATUSES = {"completed", "failed", "cancelled", "stopped", "error"}


@dataclass
class TaskRecord:
    id: str
    prompt: str
    context: str | None = None
    policy: str | None = None
    execution_mode: str = "adaptive"
    status: str = "queued"
    created_at: float = field(default_factory=time.time)
    started_at: float | None = None
    finished_at: float | None = None
    route: str | None = None
    decision_level: str | None = None
    result: dict[str, Any] | None = None
    error: str | None = None
    usage: dict[str, Any] = field(default_factory=dict)
    events: list[dict[str, Any]] = field(default_factory=list)
    approval_request: dict[str, Any] | None = None
    _subscribers: list[queue.Queue] = field(default_factory=list)
    _lock: threading.Lock = field(default_factory=threading.Lock)
    _future: Any = None
    _approval_future: Any = None

    @property
    def terminal(self) -> bool:
        return self.status in _TERMINAL_STATUSES

    def emit(self, event: str, data: dict[str, Any]) -> None:
        payload = {"event": event, "task_id": self.id, "ts": time.time(), "data": data}
        with self._lock:
            self.events.append(payload)
            subscribers = list(self._subscribers)
        for subscriber in subscribers:
            subscriber.put(payload)

    def subscribe(self) -> queue.Queue:
        subscriber: queue.Queue = queue.Queue()
        with self._lock:
            self._subscribers.append(subscriber)
        return subscriber

    def unsubscribe(self, subscriber: queue.Queue) -> None:
        with self._lock:
            if subscriber in self._subscribers:
                self._subscribers.remove(subscriber)

    def summary(self) -> dict[str, Any]:
        return {
            "id": self.id,
            "prompt": self.prompt,
            "status": self.status,
            "execution_mode": self.execution_mode,
            "created_at": self.created_at,
            "started_at": self.started_at,
            "finished_at": self.finished_at,
            "route": self.route,
            "decision_level": self.decision_level,
            "usage": self.usage,
            "error": self.error,
            "approval_request": self.approval_request,
            "result": self.result,
        }


class ApprovalRouter(ApprovalProvider):
    """Routes human approval requests to the task that triggered them."""

    def __init__(self, service: ComputeSifterService):
        self.service = service

    async def request(self, request: ApprovalRequest) -> HumanDecision:
        record = _CURRENT_TASK.get()
        if record is None:
            return HumanDecision(action=ApprovalAction.STOP, note="no active desktop task")
        return await self.service._await_approval(record, request)


class ComputeSifterService:
    """Application/service boundary shared by CLI, desktop and future aa-sync."""

    def __init__(
        self,
        aa_sifter: ComputeSifter,
        config: SifterConfig,
        app_config: ApplicationConfig,
        *,
        config_path: Path | None = None,
        profile_name: str | None = None,
        ollama_installed_fn: Callable[[], set[str] | None] | None = None,
        live_check_fn: Callable[[Any], tuple[bool, str]] | None = None,
        catalog: ModelCatalog | None = None,
        start_loop: bool = True,
    ):
        self.sifter = aa_sifter
        self.config = config
        self.app_config = app_config
        self.config_path = config_path or default_config_path()
        self.profile_name = profile_name or app_config.active_profile
        self.ollama_installed_fn = ollama_installed_fn
        self.live_check_fn = live_check_fn
        self.catalog = catalog or ModelCatalog()
        # Desktop approvals always flow through the service so the UI can respond.
        self.sifter.approval_provider = ApprovalRouter(self)
        self._tasks: dict[str, TaskRecord] = {}
        self._tasks_lock = threading.Lock()
        self._loop: asyncio.AbstractEventLoop | None = None
        self._loop_thread: threading.Thread | None = None
        self._status_cache: tuple[float, tuple[str, str]] | None = None
        if start_loop:
            self.start()

    # -- lifecycle ---------------------------------------------------------
    def start(self) -> None:
        if self._loop is not None:
            return
        self._loop = asyncio.new_event_loop()
        self._loop_thread = threading.Thread(
            target=self._loop.run_forever, name="aa-sifter-loop", daemon=True
        )
        self._loop_thread.start()

    def close(self) -> None:
        if self._loop is not None:
            try:
                future = asyncio.run_coroutine_threadsafe(self.sifter.aclose(), self._loop)
                future.result(timeout=5)
            except Exception:  # noqa: BLE001 - best-effort shutdown
                pass
            with self._tasks_lock:
                pending = [
                    record._future
                    for record in self._tasks.values()
                    if record._future is not None and not record.terminal
                ]
            for pending_future in pending:
                pending_future.cancel()
            self._loop.call_soon_threadsafe(self._loop.stop)
            if self._loop_thread is not None:
                self._loop_thread.join(timeout=5)
            self._loop = None

    def __enter__(self) -> ComputeSifterService:
        self.start()
        return self

    def __exit__(self, *exc: object) -> None:
        self.close()

    # -- construction ------------------------------------------------------
    @classmethod
    def create(
        cls,
        *,
        config_path: Path | None = None,
        profile_name: str | None = None,
        provider: ModelProvider | None = None,
        app_config: ApplicationConfig | None = None,
        **kwargs: Any,
    ) -> ComputeSifterService:
        path = config_path or default_config_path()
        app = app_config or load_application_config(path)
        config, _ = load_runtime_config(profile_name=profile_name, path=path)
        aa_sifter = ComputeSifter(config, provider=provider)
        return cls(
            aa_sifter,
            config,
            app,
            config_path=path,
            profile_name=profile_name,
            **kwargs,
        )

    # -- tasks -------------------------------------------------------------
    def submit(
        self,
        prompt: str,
        *,
        context: str | None = None,
        policy: str | None = None,
        execution_mode: str = "adaptive",
    ) -> TaskRecord:
        record = TaskRecord(
            id=uuid.uuid4().hex[:12],
            prompt=prompt,
            context=context,
            policy=_policy_for_mode(execution_mode, policy),
            execution_mode=execution_mode,
        )
        with self._tasks_lock:
            self._tasks[record.id] = record
        assert self._loop is not None
        record._future = asyncio.run_coroutine_threadsafe(self._execute(record), self._loop)
        return record

    def get(self, task_id: str) -> TaskRecord | None:
        with self._tasks_lock:
            return self._tasks.get(task_id)

    def list_tasks(self) -> list[dict[str, Any]]:
        with self._tasks_lock:
            records = sorted(self._tasks.values(), key=lambda r: r.created_at, reverse=True)
        return [record.summary() for record in records]

    def cancel(self, task_id: str) -> bool:
        record = self.get(task_id)
        if record is None or record.terminal or record._future is None:
            return False
        record.emit("status", {"status": "cancelling"})
        return bool(record._future.cancel())

    def resolve_approval(
        self,
        task_id: str,
        *,
        action: str,
        selected_options: list[str] | None = None,
        constraints: list[str] | None = None,
        note: str = "",
    ) -> bool:
        record = self.get(task_id)
        if record is None or record._approval_future is None or self._loop is None:
            return False
        try:
            decision = HumanDecision(
                action=ApprovalAction(action),
                selected_options=selected_options or [],
                constraints=constraints or [],
                note=note,
            )
        except ValueError:
            return False
        future = record._approval_future
        self._loop.call_soon_threadsafe(_set_future, future, decision)
        return True

    async def wait(self, task_id: str, timeout: float | None = None) -> TaskRecord | None:
        record = self.get(task_id)
        if record is None or record._future is None:
            return record
        try:
            await asyncio.wrap_future(record._future)
        except asyncio.CancelledError:
            pass
        except Exception:  # noqa: BLE001 - error recorded on the task
            pass
        # Cancellation can resolve the concurrent future before the task's finally
        # block updates its status, so wait briefly for the record to settle.
        deadline = time.time() + (timeout if timeout is not None else 5.0)
        while not record.terminal and time.time() < deadline:
            await asyncio.sleep(0.02)
        return record

    async def _execute(self, record: TaskRecord) -> None:
        _CURRENT_TASK.set(record)
        record.status = "running"
        record.started_at = time.time()
        record.emit("status", {"status": "running"})

        def progress(event: str, data: dict[str, Any]) -> None:
            record.emit(event, data)

        try:
            result = await self.sifter.run(
                record.prompt,
                context=record.context,
                policy=record.policy,
                progress=progress,
            )
            record.result = result.model_dump(mode="json")
            record.route = result.route
            record.decision_level = result.decision_level.value
            record.usage = result.usage.model_dump()
            record.status = result.status
            record.emit("result", {"status": result.status, "route": result.route})
        except asyncio.CancelledError:
            record.status = "cancelled"
            record.emit("status", {"status": "cancelled"})
        except Exception as exc:  # noqa: BLE001 - surface as task failure
            record.status = "failed"
            record.error = str(exc)
            record.emit("error", {"message": str(exc)})
        finally:
            record.finished_at = time.time()
            record.emit("done", {"status": record.status})

    async def _await_approval(self, record: TaskRecord, request: ApprovalRequest) -> HumanDecision:
        record.approval_request = request.model_dump(mode="json")
        record.status = "waiting_approval"
        record.emit("approval_required", record.approval_request)
        loop = asyncio.get_running_loop()
        future: asyncio.Future[HumanDecision] = loop.create_future()
        record._approval_future = future
        try:
            decision = await future
        finally:
            record._approval_future = None
            record.approval_request = None
            if not record.terminal:
                record.status = "running"
                record.emit("status", {"status": "running"})
        return decision

    # -- status / config ---------------------------------------------------
    def health(self) -> dict[str, Any]:
        local, expert = self._provider_status()
        return {
            "status": "ready",
            "version": __version__,
            "profile": self.app_config.active_profile,
            "local_provider": local,
            "expert_provider": expert,
        }

    def startup_issues(self) -> list[dict[str, Any]]:
        installed_fn = self.ollama_installed_fn or ollama_installed
        issues = translate_startup_issues(
            self.app_config, ollama_installed_fn=lambda: self._installed(installed_fn)
        )
        return [issue.model_dump() for issue in issues]

    def _installed(self, fn: Callable[[], set[str] | None]) -> set[str] | None:
        return fn()

    def status(self) -> dict[str, Any]:
        profile = self.app_config.profiles.get(self.app_config.active_profile)
        local, expert = self._provider_status()
        return {
            "version": __version__,
            "profile": (
                profile_summary(
                    self.app_config.active_profile, profile, self.app_config.active_profile
                )
                if profile
                else None
            ),
            "profiles": list_profiles(self.app_config),
            "providers": {"local": local, "expert": expert},
            "models": self.models(),
            "system": self.system(),
            "issues": self.startup_issues(),
        }

    def _provider_status(self) -> tuple[str, str]:
        now = time.time()
        if self._status_cache and now - self._status_cache[0] < 5.0:
            return self._status_cache[1]
        local = self._local_status()
        expert = self._expert_status()
        self._status_cache = (now, (local, expert))
        return local, expert

    def _local_status(self) -> str:
        profile = self.app_config.profiles.get(self.app_config.active_profile)
        local = profile.local if profile else None
        if local is None or not local.enabled:
            return "not_configured"
        if local.provider != "ollama":
            return "configured"
        installed_fn = self.ollama_installed_fn or ollama_installed
        installed = installed_fn()
        if installed is None:
            return "unavailable"
        return "ready" if local.model in installed else "model_missing"

    def _expert_status(self) -> str:
        import os

        profile = self.app_config.profiles.get(self.app_config.active_profile)
        if profile is None or profile.expert is None or not profile.expert.enabled:
            return "not_configured"
        if not profile.preferences.cloud_allowed:
            return "not_configured"
        expert = profile.expert
        if expert.provider in {"deepseek", "openai-compatible"}:
            if not (expert.api_key_env and os.environ.get(expert.api_key_env)):
                return "missing_credentials"
        return "configured"

    def models(self) -> dict[str, Any]:
        profile = self.app_config.profiles.get(self.app_config.active_profile)
        local_meta = None
        expert_meta = None
        if profile and profile.local:
            local_meta = self.catalog.get(profile.local.provider, profile.local.model)
        if profile and profile.expert:
            expert_meta = self.catalog.get(profile.expert.provider, profile.expert.model)
        local, expert = self._provider_status()
        return {
            "local": {
                "configured": profile.local.model_dump(exclude_none=True)
                if profile and profile.local
                else None,
                "status": local,
                "metadata": local_meta.model_dump() if local_meta else None,
            },
            "expert": {
                "configured": profile.expert.model_dump(exclude_none=True)
                if profile and profile.expert
                else None,
                "status": expert,
                "metadata": expert_meta.model_dump() if expert_meta else None,
            },
        }

    def system(self) -> dict[str, Any] | None:
        if self.app_config.active_hardware:
            hardware = self.app_config.hardware.get(self.app_config.active_hardware)
            if hardware is not None:
                return {
                    "name": self.app_config.active_hardware,
                    "hardware": hardware.model_dump(),
                    "profiles": {
                        name: value.model_dump() for name, value in self.app_config.hardware.items()
                    },
                }
        return {
            "name": None,
            "hardware": None,
            "profiles": {
                name: value.model_dump() for name, value in self.app_config.hardware.items()
            },
        }

    def detect_system(self, *, save_as: str | None = None) -> dict[str, Any]:
        hardware = detect_hardware()
        if save_as:
            self.app_config.hardware[save_as] = hardware
            self.app_config.active_hardware = save_as
            save_application_config(self.app_config, self.config_path)
        return hardware.model_dump()

    def doctor(self, *, live: bool = False) -> dict[str, Any]:
        live_fn = self.live_check_fn
        return build_doctor_report(
            self.app_config,
            config_path=self.config_path,
            live=live,
            ollama_installed_fn=self.ollama_installed_fn or ollama_installed,
            live_check_fn=live_fn,
        )

    def metrics(self) -> dict[str, Any]:
        usage = {
            "local_calls": 0,
            "cloud_calls": 0,
            "cloud_tokens": 0,
            "cloud_cost": 0.0,
            "escalations": 0,
        }
        with self._tasks_lock:
            records = list(self._tasks.values())
        for record in records:
            usage["local_calls"] += int(record.usage.get("local_calls", 0))
            usage["cloud_calls"] += int(record.usage.get("cloud_calls", 0))
            usage["cloud_tokens"] += int(record.usage.get("cloud_input_tokens", 0)) + int(
                record.usage.get("cloud_output_tokens", 0)
            )
            usage["cloud_cost"] += float(record.usage.get("cloud_cost", 0.0))
            usage["escalations"] += int(record.usage.get("escalations", 0))
        return {"tasks": len(records), "usage": usage}

    # -- profiles ----------------------------------------------------------
    def profiles(self) -> list[dict[str, Any]]:
        return list_profiles(self.app_config)

    def activate_profile(self, name: str) -> dict[str, Any]:
        profile = activate_profile(self.app_config, name, path=self.config_path)
        self._reload()
        return profile_summary(name, profile, self.app_config.active_profile)

    def create_profile(
        self, name: str, *, source: str | None = None, force: bool = False
    ) -> dict[str, Any]:
        profile = create_profile(
            self.app_config, name, source=source, force=force, path=self.config_path
        )
        return profile_summary(name, profile, self.app_config.active_profile)

    def delete_profile(self, name: str) -> str:
        active = delete_profile(self.app_config, name, path=self.config_path)
        self._reload()
        return active

    def set_model(
        self,
        tier: str,
        provider: str,
        model: str,
        *,
        context_limit: int | None = None,
        endpoint: str | None = None,
        api_key_env: str | None = None,
    ) -> dict[str, Any]:
        set_tier(
            self.app_config,
            tier=tier,
            provider=provider,
            model=model,
            context_limit=context_limit,
            endpoint=endpoint,
            api_key_env=api_key_env,
            path=self.config_path,
        )
        self._reload()
        return self.models()

    def update_active_profile(self, **changes: Any) -> list[dict[str, Any]]:
        update_profile(self.app_config, path=self.config_path, **changes)
        self._reload()
        return self.profiles()

    def validate_profile(self, name: str | None = None) -> dict[str, Any]:
        profile = get_profile(self.app_config, name)
        hardware = profile.hardware or (
            self.app_config.hardware.get(self.app_config.active_hardware)
            if self.app_config.active_hardware
            else None
        )
        result = validate_profile(profile, hardware=hardware)
        return {"ok": result.ok, "errors": result.errors, "warnings": result.warnings}

    def _reload(self) -> None:
        self._status_cache = None
        self.profile_name = self.app_config.active_profile
        config, _ = load_runtime_config(profile_name=self.profile_name, path=self.config_path)
        self.config = config


def _set_future(future: asyncio.Future, decision: HumanDecision) -> None:
    if not future.done():
        future.set_result(decision)


def _policy_for_mode(execution_mode: str, policy: str | None) -> str | None:
    if policy:
        return policy
    mapping = {"local_only": "local_only", "expert_only": "expert_first", "adaptive": None}
    return mapping.get(execution_mode)


def service_from_app_config(
    app: ApplicationConfig,
    *,
    config_path: Path | None = None,
    provider: ModelProvider | None = None,
    **kwargs: Any,
) -> ComputeSifterService:
    return ComputeSifterService.create(
        config_path=config_path,
        provider=provider,
        app_config=app or default_application_config(),
        **kwargs,
    )
