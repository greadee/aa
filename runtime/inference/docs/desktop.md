# aa-inference Desktop

aa-inference can run as an ordinary desktop application. The desktop app is a
**client of the same backend** used by the CLI and (future) aa-sync. It does not
reimplement routing, profiles, providers, budgets, or the human approval gate.

```
Desktop UI  ──HTTP(localhost)──►  aa-inference service  ──►  Router
   CLI      ─────────────────────►        (same service)        │
 aa-sync    ─────────────────────►                    Ollama ───┴─── DeepSeek
```

## Launch

```powershell
# One command (starts the backend and opens the UI)
aa-inference launch

# One click on Windows
start-aa-inference.cmd
# or
powershell -ExecutionPolicy Bypass -File start-aa-inference.ps1
```

Options:

```powershell
aa-inference launch --headless      # start the backend, do not open a browser
aa-inference launch --no-browser    # start and print the URL only
aa-inference launch --foreground    # run the backend in this process (Ctrl+C to stop)
aa-inference launch --debug         # verbose logging
aa-inference launch --stop          # stop a backend that aa-inference started
aa-inference stop                   # equivalent to launch --stop
```

By default the launcher starts a detached backend process, waits for a health
handshake, records ownership in the state file, then opens the UI and returns.
A second launch focuses the existing instance instead of starting another stack.

## Development

```powershell
# Run the backend in the foreground with debug logging
python -m aa_inference.cli.main launch --foreground --debug

# Backend only (no browser); useful when iterating on the frontend
python -m aa_inference.cli.main serve --host 127.0.0.1 --port 8765
```

The frontend is plain HTML/CSS/JS served from `aa-inference/desktop/web/` with no build
step and no CDN dependencies, so local-only profiles work offline.

## Architecture

- `aa-inference/desktop/service.py` — `AaInferenceService`, the shared application/service
  boundary (`submit`, `cancel`, `get`, `profiles`, `models`, `system`, `doctor`, `metrics`).
- `aa-inference/desktop/server.py` — localhost-only HTTP JSON API plus Server-Sent Events.
- `aa-inference/desktop/launcher.py` — process ownership, single-instance, health checks, state.
- `aa-inference/desktop/errors.py` — startup error translation.
- `aa-inference/desktop/web/` — the UI (Task console, Approvals, Profiles, Models, System,
  Doctor, Usage, Settings, Advanced Trace).
- `aa-inference/doctor.py` — shared diagnostics used by both `aa-inference doctor` and the UI.

The CLI, desktop UI, and future aa-sync all call the same core. Removing the
desktop UI does not affect the CLI or headless operation.

## Local API

Bound to `127.0.0.1` only. Every `/api/*` request requires a per-session token
(sent in the `X-Inference-Token` header, `Authorization: Bearer`, or `token` query
parameter). The UI receives the token when `index.html` is served. Provider API
keys never reach the frontend.

```
GET  /api/health           GET  /api/status        GET  /api/profiles
POST /api/profiles         POST /api/profiles/{name}/activate
GET  /api/models           POST /api/models/set
GET  /api/system           POST /api/system/detect
GET  /api/doctor           GET  /api/metrics
GET  /api/tasks            POST /api/tasks
GET  /api/tasks/{id}       GET  /api/tasks/{id}/events   (SSE)
POST /api/tasks/{id}/approval
POST /api/tasks/{id}/cancel
POST /api/shutdown
```

## Human approval

MAJOR and CRITICAL decisions pause execution and surface a modal in the UI.
The backend enforces this in application code; the UI cannot bypass it. If the
user permits expert analysis, a second approval is required before implementing
the recommendation. Cloud models are never called before the first approval.

## Profiles, models, and system

The UI lists profiles, activates them, and shows local/expert status. Model
changes go through the same profile system as `aa-inference model set`. System detects
hardware with the same best-effort detector used by the CLI.

## Processes and shutdown

The launcher only terminates processes it started (`started_by` ownership). A
pre-existing Ollama process is never stopped. Closing the app or `aa-inference stop`
shuts down the owned backend cleanly.

## Logs and data

```
~/.aa_inference/
├── config.toml
├── aa_inference.db
├── logs/
│   ├── application.log
│   └── backend.log
└── state/desktop.json
```

Logs never contain provider API keys. Override the base directory with
`INFERENCE_DATA_DIR`.

## First run

If no profile exists, `aa-inference setup` (or the recommend/setup CLI) creates one.
The UI surfaces the same startup issues (Ollama unreachable, model not
installed, missing expert key) with recovery options. Missing cloud credentials
never prevent local use.

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| "Ollama does not appear to be running" | start Ollama, or use a cloud-only or local-only profile |
| "Local model is not installed" | `ollama pull qwen3.5:9b` |
| Backend did not become ready | check `~/.aa_inference/logs/backend.log` |
| Port occupied | the launcher picks another port automatically |
| Stale state | `aa-inference stop` or delete `~/.aa_inference/state/desktop.json` |

## Packaging

The Python package ships the backend and web assets. The repo includes portable
launcher scripts; the release workflow also produces a `AaInference-portable.zip`
containing the wheel and launcher scripts. Ollama and model weights are **not**
bundled and remain external dependencies.

A native Tauri shell is a future enhancement; the current UI is a localhost web
shell launched automatically, which keeps the core portable and CI-friendly.
