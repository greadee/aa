from __future__ import annotations

import asyncio
import time
from pathlib import Path

from pydantic import BaseModel


class CommandResult(BaseModel):
    command: str
    exit_code: int
    stdout: str = ""
    stderr: str = ""
    duration_ms: float = 0.0
    timed_out: bool = False

    @property
    def passed(self) -> bool:
        return self.exit_code == 0 and not self.timed_out


async def run_command(
    command: str,
    *,
    cwd: str | Path | None = None,
    timeout: float = 300.0,
) -> CommandResult:
    start = time.perf_counter()
    process = await asyncio.create_subprocess_shell(
        command,
        cwd=str(cwd) if cwd else None,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )
    try:
        stdout, stderr = await asyncio.wait_for(process.communicate(), timeout=timeout)
        timed_out = False
    except TimeoutError:
        process.kill()
        await process.wait()
        stdout, stderr = b"", b""
        timed_out = True
    duration = (time.perf_counter() - start) * 1000.0
    return CommandResult(
        command=command,
        exit_code=process.returncode if process.returncode is not None else -1,
        stdout=stdout.decode(errors="replace")[-8000:],
        stderr=stderr.decode(errors="replace")[-8000:],
        duration_ms=duration,
        timed_out=timed_out,
    )
