from __future__ import annotations

from collections.abc import Callable, Sequence
from pathlib import Path
from typing import Protocol

from pydantic import BaseModel, Field

from ..metrics.trace import Trace
from .command import CommandResult, run_command


class VerificationCheck(BaseModel):
    name: str
    passed: bool
    detail: str = ""
    duration_ms: float = 0.0


class VerificationResult(BaseModel):
    passed: bool
    checks: list[VerificationCheck] = Field(default_factory=list)

    @property
    def summary(self) -> str:
        if not self.checks:
            return "no checks were run"
        return ", ".join(f"{c.name}={'pass' if c.passed else 'FAIL'}" for c in self.checks)


class Verifier(Protocol):
    name: str

    async def verify(
        self, *, answer: str, workdir: str | Path | None = None
    ) -> VerificationResult: ...


class NullVerifier:
    name = "null"

    async def verify(self, *, answer: str, workdir: str | Path | None = None) -> VerificationResult:
        return VerificationResult(passed=True)


class CommandVerifier:
    """Runs objective checks such as pytest, lint, typecheck or build."""

    name = "command"

    def __init__(
        self,
        commands: Sequence[str],
        *,
        cwd: str | Path | None = None,
        timeout: float = 300.0,
        trace: Trace | None = None,
    ):
        self.commands = list(commands)
        self.cwd = cwd
        self.timeout = timeout
        self.trace = trace

    async def verify(self, *, answer: str, workdir: str | Path | None = None) -> VerificationResult:
        checks: list[VerificationCheck] = []
        for command in self.commands:
            result: CommandResult = await run_command(
                command, cwd=workdir or self.cwd, timeout=self.timeout
            )
            if self.trace is not None:
                self.trace.log(
                    "verify",
                    f"$ {command} -> exit {result.exit_code}",
                    passed=result.passed,
                    duration_ms=round(result.duration_ms, 1),
                )
            checks.append(
                VerificationCheck(
                    name=command,
                    passed=result.passed,
                    detail=(result.stderr or result.stdout)[-500:],
                    duration_ms=result.duration_ms,
                )
            )
        return VerificationResult(
            passed=all(c.passed for c in checks) if checks else True, checks=checks
        )


class FileChangeVerifier:
    name = "files"

    def __init__(
        self,
        expected_files: Sequence[str],
        *,
        root: str | Path | None = None,
        trace: Trace | None = None,
    ):
        self.expected_files = list(expected_files)
        self.root = Path(root) if root else None
        self.trace = trace

    async def verify(self, *, answer: str, workdir: str | Path | None = None) -> VerificationResult:
        root = Path(workdir) if workdir else (self.root or Path.cwd())
        checks: list[VerificationCheck] = []
        for name in self.expected_files:
            exists = (root / name).exists()
            checks.append(VerificationCheck(name=f"exists:{name}", passed=exists))
        if self.trace is not None:
            self.trace.log("verify", "expected files checked", passed=all(c.passed for c in checks))
        return VerificationResult(passed=all(c.passed for c in checks), checks=checks)


CustomValidator = Callable[[str], "tuple[bool, str] | bool"]


class CallableVerifier:
    name = "callable"

    def __init__(self, name: str, validator: CustomValidator, *, trace: Trace | None = None):
        self.name = name
        self.validator = validator
        self.trace = trace

    async def verify(self, *, answer: str, workdir: str | Path | None = None) -> VerificationResult:
        outcome = self.validator(answer)
        if isinstance(outcome, tuple):
            passed, detail = outcome
        else:
            passed, detail = bool(outcome), ""
        check = VerificationCheck(name=self.name, passed=passed, detail=detail)
        if self.trace is not None:
            self.trace.log("verify", f"{self.name}={'pass' if passed else 'FAIL'}", detail=detail)
        return VerificationResult(passed=passed, checks=[check])


class CompositeVerifier:
    name = "composite"

    def __init__(self, verifiers: Sequence[Verifier], *, trace: Trace | None = None):
        self.verifiers = list(verifiers)
        self.trace = trace

    async def verify(self, *, answer: str, workdir: str | Path | None = None) -> VerificationResult:
        if not self.verifiers:
            return VerificationResult(passed=True)
        checks: list[VerificationCheck] = []
        for verifier in self.verifiers:
            result = await verifier.verify(answer=answer, workdir=workdir)
            checks.extend(result.checks)
        return VerificationResult(
            passed=all(c.passed for c in checks) if checks else True, checks=checks
        )
