from __future__ import annotations

import pytest

from aa_sifter.tasks.executor import (
    ApprovalRequiredError,
    InferenceQueue,
    TaskExecutor,
)
from aa_sifter.tasks.graph import TaskGraph
from aa_sifter.tasks.task import Task, TaskStatus


def _task(task_id: str, deps: list[str] | None = None) -> Task:
    return Task(id=task_id, description=f"task {task_id}", dependencies=deps or [])


def test_topological_order():
    graph = TaskGraph([_task("c", ["a", "b"]), _task("a"), _task("b", ["a"])])
    order = [t.id for t in graph.topological_order()]
    assert order.index("a") < order.index("b") < order.index("c")


def test_ready_tasks_respect_dependencies():
    graph = TaskGraph([_task("a"), _task("b", ["a"])])
    ready = graph.ready_tasks()
    assert [t.id for t in ready] == ["a"]
    graph.get("a").status = TaskStatus.DONE
    assert [t.id for t in graph.ready_tasks()] == ["b"]


def test_blocked_branch_does_not_freeze_independent_branch():
    graph = TaskGraph([_task("a"), _task("b", ["a"]), _task("e")])
    graph.get("a").status = TaskStatus.BLOCKED_APPROVAL
    ready_ids = {t.id for t in graph.ready_tasks()}
    assert "e" in ready_ids
    assert "b" not in ready_ids
    assert graph.blocked_tasks()[0].id == "a"


@pytest.mark.asyncio
async def test_executor_runs_worker_and_marks_done():
    graph = TaskGraph([_task("a"), _task("b", ["a"])])
    seen: list[str] = []

    async def worker(task: Task) -> Task:
        seen.append(task.id)
        task.result = "ok"
        return task

    await TaskExecutor(worker, max_concurrency=2).run(graph)
    assert seen == ["a", "b"]
    assert all(t.status == TaskStatus.DONE for t in graph.tasks.values())


@pytest.mark.asyncio
async def test_executor_marks_approval_blocked():
    graph = TaskGraph([_task("a"), _task("b", ["a"]), _task("e")])

    async def worker(task: Task) -> Task:
        if task.id == "a":
            raise ApprovalRequiredError("a", "major decision")
        task.result = "ok"
        return task

    await TaskExecutor(worker, max_concurrency=2).run(graph)
    assert graph.get("a").status == TaskStatus.BLOCKED_APPROVAL
    assert graph.get("e").status == TaskStatus.DONE


def test_inference_queue_serializes_local():
    queue = InferenceQueue(max_parallel_local=1, max_parallel_cloud=2)
    from aa_sifter.models.provider import Tier

    assert queue.semaphore(Tier.LOCAL) is queue.local
    assert queue.semaphore(Tier.EXPERT) is queue.cloud
    assert queue.local._value == 1


def test_descendants_and_cycle_detection():
    graph = TaskGraph([_task("a"), _task("b", ["a"]), _task("c", ["b"])])
    assert set(graph.descendants("a")) == {"b", "c"}

    cyclic = TaskGraph([_task("x", ["y"]), _task("y", ["x"])])
    with pytest.raises(ValueError):
        cyclic.topological_order()


def test_invalidate_dependencies_skips_children_of_failed():
    graph = TaskGraph([_task("a"), _task("b", ["a"]), _task("c", ["b"])])
    graph.get("a").status = TaskStatus.FAILED
    graph.invalidate_dependencies()
    assert graph.get("b").status == TaskStatus.SKIPPED
    assert graph.get("c").status == TaskStatus.SKIPPED


def test_duplicate_task_id_rejected():
    graph = TaskGraph([_task("a")])
    with pytest.raises(ValueError):
        graph.add(_task("a"))


def test_all_terminal_and_get():
    graph = TaskGraph([_task("a"), _task("b", ["a"])])
    graph.get("a").status = TaskStatus.DONE
    graph.get("b").status = TaskStatus.DONE
    assert graph.all_terminal() is True
    assert graph.get("a").id == "a"
