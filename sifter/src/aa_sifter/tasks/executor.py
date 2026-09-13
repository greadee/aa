from __future__ import annotations

import asyncio
from collections.abc import Awaitable, Callable

from ..metrics.trace import Trace
from ..models.provider import Tier
from .graph import TaskGraph
from .task import Task, TaskStatus

WorkerFn = Callable[[Task], Awaitable[Task]]


class ApprovalRequiredError(RuntimeError):
    def __init__(self, task_id: str, reason: str):
        super().__init__(reason)
        self.task_id = task_id
        self.reason = reason


class InferenceQueue:
    """Separates concurrent agent/tool activity from simultaneous model inference.

    On a single 10 GB GPU only one local model runs at a time, so local
    inference is serialized while other agents may continue filesystem/tool work.
    """

    def __init__(
        self,
        *,
        max_parallel_local: int = 1,
        max_parallel_cloud: int = 2,
        trace: Trace | None = None,
    ):
        self.local = asyncio.Semaphore(max(1, max_parallel_local))
        self.cloud = asyncio.Semaphore(max(1, max_parallel_cloud))
        self.trace = trace

    def semaphore(self, tier: Tier) -> asyncio.Semaphore:
        return self.local if tier == Tier.LOCAL else self.cloud


class TaskExecutor:
    def __init__(
        self,
        worker: WorkerFn,
        *,
        max_concurrency: int = 4,
        queue: InferenceQueue | None = None,
        trace: Trace | None = None,
    ):
        self.worker = worker
        self.max_concurrency = max_concurrency
        self.queue = queue or InferenceQueue(trace=trace)
        self.trace = trace

    async def run(self, graph: TaskGraph) -> TaskGraph:
        gate = asyncio.Semaphore(self.max_concurrency)
        while graph.has_pending():
            ready = graph.ready_tasks()
            if not ready:
                if graph.blocked_tasks():
                    if self.trace is not None:
                        self.trace.log(
                            "tasks",
                            "execution paused: branch blocked awaiting user approval",
                            blocked=[t.id for t in graph.blocked_tasks()],
                        )
                break
            await asyncio.gather(*(self._run_one(task, gate) for task in ready))
            graph.invalidate_dependencies()
        return graph

    async def _run_one(self, task: Task, gate: asyncio.Semaphore) -> None:
        async with gate:
            task.status = TaskStatus.RUNNING
            task.attempt_count += 1
            try:
                await self.worker(task)
            except ApprovalRequiredError as exc:
                task.status = TaskStatus.BLOCKED_APPROVAL
                task.verification_state = "blocked_approval"
                task.result = exc.reason
                if self.trace is not None:
                    self.trace.log(
                        "tasks", f"task {task.id} blocked on approval", reason=exc.reason
                    )
            except Exception as exc:  # noqa: BLE001 - surface as task failure
                task.status = TaskStatus.FAILED
                task.result = f"error: {exc}"
                if self.trace is not None:
                    self.trace.log("tasks", f"task {task.id} failed", level="error", error=str(exc))
            else:
                if task.status == TaskStatus.RUNNING:
                    task.status = TaskStatus.DONE
