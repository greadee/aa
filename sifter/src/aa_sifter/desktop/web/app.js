const TOKEN = window.__SIFTER_TOKEN__ || "";
const headers = { "X-Sifter-Token": TOKEN };

async function api(path, options = {}) {
  const opts = { headers: { ...headers }, ...options };
  if (opts.body && typeof opts.body !== "string") {
    opts.body = JSON.stringify(opts.body);
    opts.headers["Content-Type"] = "application/json";
  }
  const response = await fetch(path, opts);
  const text = await response.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch (e) { data = { raw: text }; }
  if (!response.ok) throw new Error((data && (data.error || data.detail)) || response.statusText);
  return data;
}

const $ = (id) => document.getElementById(id);

// ---------------------------------------------------------------- navigation
const views = ["task", "history", "profiles", "models", "system", "doctor", "usage", "settings", "trace"];
function showView(name) {
  views.forEach((v) => $(`view-${v}`).classList.toggle("hidden", v !== name));
  document.querySelectorAll(".nav").forEach((b) => b.classList.toggle("active", b.dataset.view === name));
  localStorage.setItem("aa_sifter.view", name);
  if (name === "history") loadHistory();
  if (name === "profiles") loadProfiles();
  if (name === "models") loadModels();
  if (name === "system") loadSystem();
  if (name === "usage") loadUsage();
  if (name === "settings") loadSettings();
}
document.querySelectorAll(".nav").forEach((b) => b.addEventListener("click", () => showView(b.dataset.view)));

// ------------------------------------------------------------------- theme
function applyTheme(theme) {
  document.documentElement.setAttribute("data-theme", theme);
  localStorage.setItem("aa_sifter.theme", theme);
}
applyTheme(localStorage.getItem("aa_sifter.theme") || "light");
$("theme-toggle").addEventListener("click", () => {
  const next = document.documentElement.getAttribute("data-theme") === "dark" ? "light" : "dark";
  applyTheme(next);
});

// -------------------------------------------------------------- connection
function statusClass(value) { return `status status-${value || "unknown"}`; }

async function refreshStatus() {
  try {
    const status = await api("/api/status");
    const profile = status.profile;
    $("profile-label").textContent = `Profile: ${profile ? profile.name : "-"}`;
    $("settings-profile").textContent = profile ? profile.name : "-";
    const local = status.providers.local;
    const expert = status.providers.expert;
    $("local-status").className = statusClass(local);
    $("local-status").innerHTML = `Local <b>&#9679;</b> ${labelStatus(local)}`;
    $("expert-status").className = statusClass(expert);
    $("expert-status").innerHTML = `Expert <b>&#9679;</b> ${labelStatus(expert)}`;
    renderIssues(status.issues || []);
  } catch (e) {
    $("local-status").textContent = "Local unreachable";
  }
}

function labelStatus(value) {
  return ({
    ready: "Ready",
    configured: "Configured",
    unavailable: "Unavailable",
    model_missing: "Model missing",
    missing_credentials: "No credentials",
    not_configured: "Not configured",
  })[value] || value;
}

function renderIssues(issues) {
  const target = $("issues-list");
  if (!issues.length) { target.innerHTML = '<p class="muted">No startup issues.</p>'; return; }
  target.innerHTML = issues.map((issue) => `
    <div class="card">
      <b class="${issue.severity === "error" ? "check-fail" : "check-skip"}">${escapeHtml(issue.title)}</b>
      <p class="muted">${escapeHtml(issue.detail || "")}</p>
      <div class="row">${(issue.options || []).map((o) => `<span class="chip">${escapeHtml(o.label)}</span>`).join("")}</div>
    </div>`).join("");
}

// ------------------------------------------------------------------- task
let currentTaskId = null;
let eventSource = null;
let traceEvents = [];
let routeLabel = null;

function addStatus(text) {
  const li = document.createElement("li");
  li.textContent = text;
  $("status-list").appendChild(li);
}
function clearStatus() { $("status-list").innerHTML = ""; }
function addTrace(event) {
  traceEvents.push(event);
  const lines = traceEvents.map((e) => {
    const time = new Date(e.ts * 1000).toLocaleTimeString();
    const data = e.data && Object.keys(e.data).length ? " " + JSON.stringify(e.data) : "";
    return `${time} ${e.event}${data}`;
  });
  $("trace-output").textContent = lines.join("\n");
}

$("run").addEventListener("click", runTask);
$("cancel").addEventListener("click", cancelTask);

async function runTask() {
  const prompt = $("prompt").value.trim();
  if (!prompt) return;
  clearStatus();
  traceEvents = [];
  $("trace-output").textContent = "";
  $("response").classList.add("hidden");
  $("response").textContent = "";
  $("route-indicator").classList.add("hidden");
  routeLabel = null;
  $("run").disabled = true;
  $("cancel").disabled = false;
  addStatus("Submitting task...");
  try {
    const task = await api("/api/tasks", {
      method: "POST",
      body: { prompt, execution_mode: $("execution-mode").value },
    });
    currentTaskId = task.id;
    streamTask(task.id);
  } catch (e) {
    addStatus(`Error: ${e.message}`);
    $("run").disabled = false;
    $("cancel").disabled = true;
  }
}

function streamTask(taskId) {
  if (eventSource) eventSource.close();
  eventSource = new EventSource(`/api/tasks/${taskId}/events?token=${encodeURIComponent(TOKEN)}`);
  eventSource.onmessage = async (message) => {
    const event = JSON.parse(message.data);
    addTrace(event);
    handleEvent(event);
    if (event.event === "done") {
      eventSource.close();
      eventSource = null;
      await finishTask(taskId);
    }
  };
  eventSource.onerror = () => {
    if (eventSource) { eventSource.close(); eventSource = null; }
  };
}

function handleEvent(event) {
  const d = event.data || {};
  switch (event.event) {
    case "status": addStatus(`Status: ${d.status}`); break;
    case "analyzing": addStatus(`Analyzing task (${d.decision_level})...`); break;
    case "routing": {
      routeLabel = d.route;
      showRoute();
      break;
    }
    case "local_call": addStatus(`Running locally (${d.purpose})...`); break;
    case "expert_call": addStatus(`Consulting expert model (${d.purpose})...`); break;
    case "escalating": addStatus("Local worker requested escalation..."); break;
    case "verifying": addStatus(`Verifying result (${d.passed ? "pass" : "retry"})...`); break;
    case "approval_required": showApproval(d); break;
    case "error": addStatus(`Error: ${d.message}`); break;
  }
}

function showRoute() {
  const el = $("route-indicator");
  const friendly = ({ local: "Local only", hybrid: "Local \u2192 Expert \u2192 Local", cloud: "Expert" })[routeLabel] || routeLabel;
  el.textContent = `Route: ${friendly}`;
  el.classList.remove("hidden");
}

async function finishTask(taskId) {
  $("run").disabled = false;
  $("cancel").disabled = true;
  const task = await api(`/api/tasks/${taskId}`);
  const result = task.result;
  const response = $("response");
  response.classList.remove("hidden");
  if (result) {
    const meta = [
      `Status: ${result.status}`,
      `Route: ${result.route}`,
      `Decision: ${result.decision_level}`,
      `Duration: ${Math.round((result.usage.wall_clock_ms || 0))} ms`,
      `Cloud calls: ${result.usage.cloud_calls}`,
      `Estimated cloud cost: $${(result.usage.cloud_cost || 0).toFixed(6)}`,
    ];
    response.textContent = meta.join("\n") + "\n\n" + (result.answer || "");
  } else {
    response.textContent = task.error || "Task finished without a result.";
  }
  refreshStatus();
}

async function cancelTask() {
  if (!currentTaskId) return;
  addStatus("Cancelling...");
  await api(`/api/tasks/${currentTaskId}/cancel`, { method: "POST", body: {} });
}

// --------------------------------------------------------------- approval
function actionForLabel(label) {
  const l = (label || "").toLowerCase();
  if (l.includes("keep") || l.includes("current")) return "continue_with_current_architecture";
  if (l.includes("analysis") || l.includes("analyze") || l.includes("deepseek") || l.includes("expert")) return "allow_expert_analysis";
  if (l.includes("different") || l.includes("instruction")) return "modify";
  if (l.includes("stop")) return "stop_task";
  if (l.includes("approve")) return "approve";
  return "choose_option";
}

function showApproval(request) {
  const body = $("approval-body");
  const options = request.options || [];
  body.innerHTML = `
    <h2>${escapeHtml((request.category || "major").toUpperCase())} DECISION REQUIRED</h2>
    <p><b>${escapeHtml(request.issue || "")}</b></p>
    <p class="muted">${escapeHtml(request.why_it_matters || "")}</p>
    ${request.current_approach ? `<p class="muted">Current: ${escapeHtml(request.current_approach)}</p>` : ""}
    ${request.recommended_action ? `<p class="muted">Recommended next step: ${escapeHtml(request.recommended_action)}</p>` : ""}
    <p class="muted">${request.cloud_reasoning_invoked ? "Cloud reasoning has been used for analysis." : "Cloud reasoning has NOT been invoked yet."}</p>
    <div class="row">
      <input id="approval-constraints" placeholder="optional constraints (; separated)" style="flex:1" />
    </div>
    <div class="options">
      ${options.map((o) => `<button data-label="${escapeAttr(o.label)}">${escapeHtml(o.key)}. ${escapeHtml(o.label)}</button>`).join("")}
      <button data-action="modify">Give different instructions</button>
      <button data-action="stop_task">Stop this work</button>
    </div>`;
  body.querySelectorAll("button").forEach((button) => {
    button.addEventListener("click", () => submitApproval(request, button));
  });
  $("approval-overlay").classList.remove("hidden");
}

async function submitApproval(request, button) {
  const constraintsRaw = ($("approval-constraints") || {}).value || "";
  const constraints = constraintsRaw.split(";").map((s) => s.trim()).filter(Boolean);
  const action = button.dataset.action || actionForLabel(button.dataset.label);
  $("approval-overlay").classList.add("hidden");
  await api(`/api/tasks/${currentTaskId}/approval`, {
    method: "POST",
    body: { action, selected_options: [], constraints },
  });
}

// --------------------------------------------------------------- profiles
async function loadProfiles() {
  const data = await api("/api/profiles");
  const list = $("profile-list");
  list.innerHTML = data.profiles.map((p) => `
    <div class="card">
      <div class="row">
        <b>${p.active ? "\u25cf " : "\u25cb "}${escapeHtml(p.name)}</b>
        <span class="chip">routing: ${escapeHtml(p.routing_policy)}</span>
        ${p.active ? "" : `<button data-activate="${escapeAttr(p.name)}" class="ghost">Activate</button>`}
      </div>
      <div class="muted">Local: ${p.local ? escapeHtml(p.local.provider + " / " + p.local.model) : "none"}</div>
      <div class="muted">Expert: ${p.expert ? escapeHtml(p.expert.provider + " / " + p.expert.model) : "none"}</div>
    </div>`).join("");
  list.querySelectorAll("[data-activate]").forEach((b) =>
    b.addEventListener("click", async () => {
      await api(`/api/profiles/${encodeURIComponent(b.dataset.activate)}/activate`, { method: "POST", body: {} });
      await loadProfiles();
      await refreshStatus();
    }));
}

$("create-profile").addEventListener("click", async () => {
  const name = $("new-profile-name").value.trim();
  if (!name) return;
  await api("/api/profiles", { method: "POST", body: { name } });
  $("new-profile-name").value = "";
  await loadProfiles();
});

// ----------------------------------------------------------------- models
async function loadModels() {
  const models = await api("/api/models");
  $("local-model").innerHTML = renderModel(models.local);
  $("expert-model").innerHTML = renderModel(models.expert);
}

function renderModel(entry) {
  if (!entry || !entry.configured) return '<p class="muted">Not configured</p>';
  const meta = entry.metadata;
  const rows = [
    `Provider: ${escapeHtml(entry.configured.provider)}`,
    `Model: ${escapeHtml(entry.configured.model)}`,
    `Status: ${labelStatus(entry.status)}`,
  ];
  if (entry.configured.context_limit) rows.push(`Context: ${entry.configured.context_limit}`);
  if (meta) {
    if (meta.recommended_vram_gb) rows.push(`Recommended VRAM: ${meta.recommended_vram_gb} GB`);
    if (meta.capabilities) rows.push(`Capabilities: ${meta.capabilities.join(", ")}`);
  }
  return rows.map((r) => `<div>${r}</div>`).join("");
}

$("apply-model").addEventListener("click", async () => {
  const result = $("set-model-result");
  try {
    await api("/api/models/set", {
      method: "POST",
      body: {
        tier: $("set-tier").value,
        provider: $("set-provider").value.trim(),
        model: $("set-model").value.trim(),
      },
    });
    result.textContent = "Model updated.";
    await loadModels();
    await refreshStatus();
  } catch (e) {
    result.textContent = `Error: ${e.message}`;
  }
});

// ----------------------------------------------------------------- system
async function loadSystem() {
  const data = await api("/api/system");
  const hw = data.hardware;
  if (!hw) {
    $("system-info").innerHTML = '<p class="muted">No hardware profile yet. Use Re-detect.</p>';
    return;
  }
  const rows = [
    ["CPU", hw.cpu_name], ["GPU", hw.gpu_name], ["VRAM", hw.gpu_vram_gb ? hw.gpu_vram_gb + " GB" : null],
    ["RAM", hw.system_ram_gb ? hw.system_ram_gb + " GB" : null], ["OS", hw.operating_system],
  ];
  $("system-info").innerHTML = `<div class="card">${rows
    .map(([k, v]) => `<div><b>${k}:</b> ${escapeHtml(String(v == null ? "unknown" : v))}</div>`)
    .join("")}</div>`;
}

$("detect-system").addEventListener("click", async () => {
  await api("/api/system/detect", { method: "POST", body: { save_as: "detected" } });
  await loadSystem();
});

// ----------------------------------------------------------------- doctor
$("run-doctor").addEventListener("click", async () => {
  const live = $("doctor-live").checked;
  const report = await api(`/api/doctor?live=${live ? "true" : "false"}`);
  $("doctor-output").innerHTML = report.checks.map((c) => {
    const cls = c.ok ? "check-pass" : (c.optional ? "check-skip" : "check-fail");
    const mark = c.ok ? "\u2713" : (c.optional ? "\u25cb" : "\u2717");
    return `<div class="doctor-check"><span class="${cls}">${mark} ${escapeHtml(c.name)}</span>
      <span class="muted"> ${escapeHtml(c.detail || "")}</span></div>`;
  }).join("") + `<p><b>Overall: ${report.ok ? "READY" : "ATTENTION NEEDED"}</b></p>`;
});

// ------------------------------------------------------------------ usage
async function loadUsage() {
  const data = await api("/api/metrics");
  $("usage-output").innerHTML = `<div class="card">
    <div>Tasks: ${data.tasks}</div>
    <div>Local calls: ${data.usage.local_calls}</div>
    <div>Cloud calls: ${data.usage.cloud_calls}</div>
    <div>Cloud tokens: ${data.usage.cloud_tokens}</div>
    <div>Estimated cloud cost: $${(data.usage.cloud_cost || 0).toFixed(6)}</div>
    <div>Escalations: ${data.usage.escalations}</div>
  </div>`;
}

// ----------------------------------------------------------------- settings
async function loadSettings() {
  $("default-mode").value = localStorage.getItem("aa_sifter.mode") || "adaptive";
}
$("default-mode").addEventListener("change", () => {
  localStorage.setItem("aa_sifter.mode", $("default-mode").value);
  $("execution-mode").value = $("default-mode").value;
});

// ------------------------------------------------------------------ history
async function loadHistory() {
  const data = await api("/api/tasks");
  $("history-list").innerHTML = data.tasks.map((t) => `
    <div class="card">
      <div><b>${escapeHtml((t.prompt || "").slice(0, 120))}</b></div>
      <div class="muted">status: ${escapeHtml(t.status)} route: ${escapeHtml(t.route || "-")}
        cost: $${(t.usage && t.usage.cloud_cost || 0).toFixed(6)}</div>
    </div>`).join("") || '<p class="muted">No tasks yet.</p>';
}

// -------------------------------------------------------------------- util
function escapeHtml(value) {
  return String(value == null ? "" : value)
    .replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
function escapeAttr(value) { return escapeHtml(value); }

// ------------------------------------------------------------------- init
$("execution-mode").value = localStorage.getItem("aa_sifter.mode") || "adaptive";
showView(localStorage.getItem("aa_sifter.view") || "task");
refreshStatus();
setInterval(refreshStatus, 15000);
