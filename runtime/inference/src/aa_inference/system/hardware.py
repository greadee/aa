from __future__ import annotations

import os
import platform
import subprocess
import sys
from pathlib import Path

from ..config.schema import HardwareProfile

_TIMEOUT = 5


def _run(command: list[str]) -> str | None:
    try:
        completed = subprocess.run(
            command,
            capture_output=True,
            text=True,
            timeout=_TIMEOUT,
            check=False,
        )
    except (OSError, subprocess.SubprocessError):
        return None
    if completed.returncode != 0:
        return None
    return completed.stdout.strip()


def detect_hardware() -> HardwareProfile:
    """Best-effort hardware detection. Never raises."""
    gpu_name, gpu_vendor, vram_gb = _detect_gpu()
    return HardwareProfile(
        cpu_name=_detect_cpu_name(),
        cpu_cores=_detect_physical_cores(),
        cpu_threads=os.cpu_count(),
        gpu_name=gpu_name,
        gpu_vendor=gpu_vendor,
        gpu_vram_gb=vram_gb,
        system_ram_gb=_detect_ram_gb(),
        operating_system=platform.system() or None,
        architecture=platform.machine() or None,
        source="detected",
    )


def _detect_cpu_name() -> str | None:
    if sys.platform == "win32":
        for command in (
            ["wmic", "cpu", "get", "name"],
            [
                "powershell",
                "-NoProfile",
                "-Command",
                "(Get-CimInstance Win32_Processor).Name",
            ],
        ):
            output = _run(command)
            name = _clean_cpu(output)
            if name:
                return name
    elif sys.platform == "darwin":
        output = _run(["sysctl", "-n", "machdep.cpu.brand_string"])
        if output:
            return output
    else:
        cpuinfo = Path("/proc/cpuinfo")
        if cpuinfo.exists():
            for line in cpuinfo.read_text(errors="ignore").splitlines():
                if line.lower().startswith("model name"):
                    return line.split(":", 1)[1].strip()
    processor = platform.processor()
    return processor or None


def _clean_cpu(output: str | None) -> str | None:
    if not output:
        return None
    lines = [line.strip() for line in output.splitlines() if line.strip()]
    lines = [line for line in lines if line.lower() != "name"]
    return lines[0] if lines else None


def _detect_physical_cores() -> int | None:
    try:
        import psutil

        return psutil.cpu_count(logical=False)
    except Exception:
        return None


def _detect_ram_gb() -> float | None:
    try:
        import psutil

        return round(psutil.virtual_memory().total / (1024**3), 1)
    except Exception:
        pass
    if sys.platform == "win32":
        output = _run(["wmic", "ComputerSystem", "get", "TotalPhysicalMemory"])
        for token in (output or "").split():
            if token.isdigit() and len(token) > 6:
                return round(int(token) / (1024**3), 1)
    else:
        meminfo = Path("/proc/meminfo")
        if meminfo.exists():
            for line in meminfo.read_text(errors="ignore").splitlines():
                if line.startswith("MemTotal"):
                    kb = int(line.split()[1])
                    return round(kb / (1024**2), 1)
    return None


def _detect_gpu() -> tuple[str | None, str | None, float | None]:
    name, vram = _detect_nvidia()
    if name:
        return name, "NVIDIA", vram
    generic = _detect_generic_gpu()
    if generic:
        return generic, None, None
    return None, None, None


def _detect_nvidia() -> tuple[str | None, float | None]:
    output = _run(
        [
            "nvidia-smi",
            "--query-gpu=name,memory.total",
            "--format=csv,noheader,nounits",
        ]
    )
    if not output:
        return None, None
    first = output.splitlines()[0]
    parts = [part.strip() for part in first.split(",")]
    if len(parts) < 2:
        return parts[0] or None, None
    name = parts[0] or None
    try:
        vram_gb = round(float(parts[1]) / 1024, 1)
    except ValueError:
        vram_gb = None
    return name, vram_gb


def _detect_generic_gpu() -> str | None:
    if sys.platform == "win32":
        output = _run(["wmic", "path", "win32_VideoController", "get", "name"])
        if output:
            lines = [line.strip() for line in output.splitlines() if line.strip()]
            lines = [line for line in lines if line.lower() != "name"]
            if lines:
                return lines[0]
    else:
        output = _run(["sh", "-c", "lspci 2>/dev/null | grep -i vga | head -n1"])
        if output:
            return output.split(":", 2)[-1].strip()
    return None
