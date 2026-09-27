@echo off
setlocal
cd /d "%~dp0"

if exist ".venv\Scripts\python.exe" (
  set "PY=.venv\Scripts\python.exe"
) else (
  set "PY=python"
)

echo Starting aa-inference...
"%PY%" -m aa_inference.cli.main launch %*

if errorlevel 1 (
  echo.
  echo aa-inference failed to start.
  echo Logs: %USERPROFILE%\.aa_inference\logs
  echo Try:  "%PY%" -m aa_inference.cli.main doctor
  pause
)

endlocal
