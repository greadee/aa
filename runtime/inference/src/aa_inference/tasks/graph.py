from __future__ import annotations

from .task import Task, TaskStatus


class TaskGraph:
    """A task DAG used for decomposed work.

    Nodes blocked on an approval dependency do not stop independent branches.
    """

    def __init__(self, tasks: list[Task] | None = None):
        self._tasks: dict[str, Task] = {}
        for task in tasks or []:
            self.add(task)

    @property
    def tasks(self) -> dict[str, Task]:
        return self._tasks

    def add(self, task: Task) -> Task:
        if task.id in self._tasks:
            raise ValueError(f"duplicate task id: {task.id}")
        self._tasks[task.id] = task
        return task

    def get(self, task_id: str) -> Task:
        return self._tasks[task_id]

    def invalidate_dependencies(self) -> None:
        for task in self._tasks.values():
            if task.status == TaskStatus.PENDING and task.dependencies:
                if any(
                    self._tasks[dep].status in {TaskStatus.FAILED, TaskStatus.SKIPPED}
                    for dep in task.dependencies
                ):
                    task.status = TaskStatus.SKIPPED

    def ready_tasks(self) -> list[Task]:
        ready: list[Task] = []
        for task in self._tasks.values():
            if task.status not in {TaskStatus.PENDING, TaskStatus.READY}:
                continue
            deps = [self._tasks[dep] for dep in task.dependencies]
            if any(dep.status in {TaskStatus.FAILED, TaskStatus.SKIPPED} for dep in deps):
                task.status = TaskStatus.SKIPPED
                continue
            if any(dep.status != TaskStatus.DONE for dep in deps):
                continue
            if task.status == TaskStatus.PENDING:
                task.status = TaskStatus.READY
            ready.append(task)
        return ready

    def blocked_tasks(self) -> list[Task]:
        return [t for t in self._tasks.values() if t.status == TaskStatus.BLOCKED_APPROVAL]

    def descendants(self, task_id: str) -> list[str]:
        seen: set[str] = set()
        stack = [task_id]
        while stack:
            current = stack.pop()
            for task in self._tasks.values():
                if current in task.dependencies and task.id not in seen:
                    seen.add(task.id)
                    stack.append(task.id)
        return list(seen)

    def has_pending(self) -> bool:
        return any(not task.terminal for task in self._tasks.values())

    def all_terminal(self) -> bool:
        return all(task.terminal for task in self._tasks.values())

    def topological_order(self) -> list[Task]:
        ordered: list[Task] = []
        visited: set[str] = set()
        temp: set[str] = set()

        def visit(task_id: str) -> None:
            if task_id in visited:
                return
            if task_id in temp:
                raise ValueError("task graph contains a cycle")
            temp.add(task_id)
            task = self._tasks[task_id]
            for dep in task.dependencies:
                visit(dep)
            temp.discard(task_id)
            visited.add(task_id)
            ordered.append(task)

        for task_id in self._tasks:
            visit(task_id)
        return ordered
