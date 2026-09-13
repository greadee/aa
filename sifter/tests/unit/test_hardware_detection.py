from __future__ import annotations

import sys

from aa_sifter.system import hardware as hardware_module
from aa_sifter.system.hardware import (
    _clean_cpu,
    _detect_generic_gpu,
    _detect_nvidia,
    _detect_ram_gb,
    _run,
)


def test_run_returns_none_on_failure(monkeypatch):
    class _Result:
        returncode = 1
        stdout = ""

    monkeypatch.setattr(hardware_module.subprocess, "run", lambda *a, **k: _Result())
    assert _run(["nope"]) is None


def test_run_returns_none_on_oserror(monkeypatch):
    def boom(*args, **kwargs):
        raise OSError("missing")

    monkeypatch.setattr(hardware_module.subprocess, "run", boom)
    assert _run(["nope"]) is None


def test_run_returns_stdout(monkeypatch):
    class _Result:
        returncode = 0
        stdout = "hello\n"

    monkeypatch.setattr(hardware_module.subprocess, "run", lambda *a, **k: _Result())
    assert _run(["echo"]) == "hello"


def test_clean_cpu_strips_header():
    assert _clean_cpu("Name\nAMD Ryzen 5 9600X") == "AMD Ryzen 5 9600X"
    assert _clean_cpu(None) is None


def test_detect_nvidia_parses_first_gpu(monkeypatch):
    monkeypatch.setattr(
        hardware_module, "_run", lambda command: "NVIDIA RTX 3080, 10240\nNVIDIA RTX 3090, 24576"
    )
    name, vram = _detect_nvidia()
    assert name == "NVIDIA RTX 3080"
    assert vram == 10.0


def test_detect_nvidia_handles_bad_output(monkeypatch):
    monkeypatch.setattr(hardware_module, "_run", lambda command: "not,a,number")
    name, vram = _detect_nvidia()
    assert name == "not"
    assert vram is None


def test_detect_nvidia_missing(monkeypatch):
    monkeypatch.setattr(hardware_module, "_run", lambda command: None)
    assert _detect_nvidia() == (None, None)


def test_detect_generic_gpu_windows(monkeypatch):
    monkeypatch.setattr(sys, "platform", "win32")
    monkeypatch.setattr(hardware_module, "_run", lambda command: "Name\nIntel UHD Graphics 770")
    assert _detect_generic_gpu() == "Intel UHD Graphics 770"


def test_detect_ram_without_psutil_non_windows(monkeypatch):
    monkeypatch.setattr(sys, "platform", "linux")
    monkeypatch.setattr(hardware_module, "_run", lambda command: None)
    # /proc/meminfo may not exist on Windows; patch Path.exists/read_text via a fake file.
    fake = type(
        "F",
        (),
        {"exists": lambda self: True, "read_text": lambda self, **k: "MemTotal: 16777216 kB"},
    )
    monkeypatch.setattr(hardware_module, "Path", lambda p: fake())
    assert _detect_ram_gb() == 16.0
