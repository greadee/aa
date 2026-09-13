@echo off
setlocal
cd /d "%~dp0"

if exist ".venv\Scripts\python.exe" (
  set "PY=.venv\Scripts\python.exe"
) else (
  set "PY=python"
)

echo Starting aa-sifter...
"%PY%" -m aa_sifter.cli.main launch %*

if errorlevel 1 (
  echo.
  echo aa-sifter failed to start.
  echo Logs: %USERPROFILE%\.aa_sifter\logs
  echo Try:  "%PY%" -m aa_sifter.cli.main doctor
  pause
)

endlocal
