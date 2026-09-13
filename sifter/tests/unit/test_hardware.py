from __future__ import annotations

from aa_sifter.config.schema import HardwareProfile
from aa_sifter.system import hardware as hardware_module
from aa_sifter.system.hardware import detect_hardware


def test_detects_nvidia_gpu(monkeypatch):
    def fake_run(command):
        joined = " ".join(command)
        if "nvidia-smi" in joined:
            return "NVIDIA GeForce RTX 3080, 10240"
        if "wmic" in joined and "cpu" in joined:
            return "Name\nAMD Ryzen 5 9600X"
        if "ComputerSystem" in joined:
            return "TotalPhysicalMemory\n34359738368"
        return None

    monkeypatch.setattr(hardware_module, "_run", fake_run)
    hardware = detect_hardware()
    assert hardware.gpu_name == "NVIDIA GeForce RTX 3080"
    assert hardware.gpu_vendor == "NVIDIA"
    assert hardware.gpu_vram_gb == 10.0
    # CPU detection is platform-specific (procfs vs wmic); assert it reports something.
    assert hardware.cpu_name
    assert hardware.source == "detected"


def test_detection_is_best_effort_when_tools_missing(monkeypatch):
    monkeypatch.setattr(hardware_module, "_run", lambda command: None)
    hardware = detect_hardware()
    assert hardware.gpu_name is None
    assert hardware.source == "detected"
    assert hardware.operating_system is not None


def test_manual_hardware_has_gpu():
    hardware = HardwareProfile(gpu_name="RTX 3080", gpu_vram_gb=10, source="manual")
    assert hardware.has_gpu is True
