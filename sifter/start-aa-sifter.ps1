# aa-sifter one-click launcher (Windows PowerShell).
$ErrorActionPreference = "Stop"
Set-Location -Path $PSScriptRoot

$python = if (Test-Path ".venv\Scripts\python.exe") { ".venv\Scripts\python.exe" } else { "python" }

Write-Host "Starting aa-sifter..."
& $python -m aa_sifter.cli.main launch @args

if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "aa-sifter failed to start."
    Write-Host "Logs: $HOME\.aa_sifter\logs"
    Write-Host "Try:  $python -m aa_sifter.cli.main doctor"
    Read-Host "Press Enter to close"
}
