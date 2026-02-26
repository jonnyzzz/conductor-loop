'use strict';

// When opened via file://, connect to the conductor server (default port 8080).
// When served by the conductor server itself (/ui/), use relative paths automatically.
// Note: run-agent serve uses a different default port (14355); use its --host/--port flags
// to change it, and update this URL accordingly if opening the UI via file://.
const API_BASE = window.location.protocol === 'file:'
  ? 'http://localhost:8080'
  : '';

const REFRESH_MS = 5000;
const SSE_REFRESH_DEBOUNCE_MS = 250;
const MAX_STREAM_LINES = 1200;
const UI_REQUEST_HEADERS = { 'X-Conductor-Client': 'web-ui' };

// ── State ────────────────────────────────────────────────────────────────────

const state = {
  selectedProject: null,
  selectedTask:    null,
  selectedRun:     null,
  taskRuns:        [],   // runs for the currently selected task
  projects:        [],
  tasks:           [],
  activeTab:       'task.md',
};

let refreshTimer     = null;
let sseSource        = null;
let tabSseSource     = null;
let tabSseRunId      = null;
let tabSseTab        = null;
let projSseSource    = null;
let sseRefreshTimer  = null;
let refreshInFlight  = false;
let refreshQueued    = false;
let selectedRunLastStatus = null;

// ── API ──────────────────────────────────────────────────────────────────────

async function apiFetch(path) {
  const resp = await fetch(API_BASE + path, { headers: UI_REQUEST_HEADERS });
  if (!resp.ok) {
    const err = new Error(`HTTP ${resp.status}`);
    err.status = resp.status;
    throw err;
  }
  return resp.json();
}

// ── Status bar ───────────────────────────────────────────────────────────────

async function refreshStatusBar() {
  try {
    const [, status] = await Promise.all([
      apiFetch('/api/v1/health'),
      apiFetch('/api/v1/status'),
    ]);
    document.getElementById('health-dot').className   = 'dot ok';
    document.getElementById('health-label').textContent = `Health OK  ·  Active runs: ${status.active_runs_count}`;
    document.getElementById('status-uptime').textContent = `Uptime: ${fmtUptime(status.uptime_seconds)}`;
    if (status.version) {
      document.getElementById('header-version').textContent = status.version;
      document.getElementById('footer-version').textContent  = status.version;
    }
  } catch {
    document.getElementById('health-dot').className   = 'dot err';
    document.getElementById('health-label').textContent = 'Server unreachable';
    document.getElementById('status-uptime').textContent = '';
  }
}

// ── Projects ─────────────────────────────────────────────────────────────────

async function loadProjects() {
  try {
    const data = await apiFetch('/api/projects');
    state.projects = data.projects || [];
  } catch {
    // keep existing list on error
  }
  renderProjectList();
}

function renderProjectList() {
  const el = document.getElementById('project-list');
  if (!state.projects.length) {
    el.innerHTML = '<p class="empty">No projects found</p>';
    return;
  }
  el.innerHTML = state.projects.map(p => {
    const sel = p.id === state.selectedProject;
    return `<div class="project-item${sel ? ' selected' : ''}" onclick="selectProject(${js(p.id)})">
      <span class="proj-name">${h(p.id)}</span>
      <span class="badge">${p.task_count}</span>
    </div>`;
  }).join('');
}

async function selectProject(id) {
  if (state.selectedProject === id) return;
  state.selectedProject = id;
  state.selectedTask    = null;
  state.selectedRun     = null;
  state.taskRuns        = [];
  selectedRunLastStatus = null;
  renderProjectList();
  hideRunDetail();
  connectProjectSSE(id);
  await loadTasks();
}

function connectProjectSSE(projectId) {
  if (projSseSource) {
    projSseSource.close();
    projSseSource = null;
  }
  const section = document.getElementById('proj-messages');
  if (!section) return;
  if (!projectId) {
    section.style.display = 'none';
    section.innerHTML = '';
    return;
  }
  section.style.display = '';
  section.innerHTML =
    '<div style="padding:3px 8px;font-size:10px;color:var(--text-dim);text-transform:uppercase;' +
    'letter-spacing:0.5px;background:var(--bg3);border-bottom:1px solid var(--border)">Project Messages</div>';
  const msgArea = document.createElement('div');
  msgArea.className = 'proj-messages';
  section.appendChild(msgArea);

  const sseUrl = `${API_BASE}/api/projects/${enc(projectId)}/messages/stream`;
  const source = new EventSource(sseUrl);
  projSseSource = source;

  source.addEventListener('message', (event) => {
    try {
      const m = JSON.parse(event.data);
      const cls  = msgTypeClass(m.type);
      const text = `[${shortTime(m.timestamp)}] [${m.type || ''}] ${m.content || ''}`;
      appendMessageLine(msgArea, cls, text);
    } catch { /* ignore parse errors */ }
  });

  source.addEventListener('heartbeat', () => { /* keep-alive */ });

  source.onerror = () => {
    if (projSseSource !== source) return;
    // browser auto-reconnects EventSource
  };
}

// ── Tasks ─────────────────────────────────────────────────────────────────────

async function loadTasks() {
  if (!state.selectedProject) {
    state.tasks = [];
    renderMainPanel();
    return;
  }
  try {
    const data = await apiFetch(`/api/projects/${enc(state.selectedProject)}/tasks`);
    state.tasks = data.tasks || [];
  } catch {
    // keep existing
  }
  // Also refresh runs for the currently selected task
  if (state.selectedTask) {
    try {
      const task = await apiFetch(
        `/api/projects/${enc(state.selectedProject)}/tasks/${enc(state.selectedTask)}?include_files=0`
      );
      state.taskRuns = task.runs || [];
    } catch { /* keep */ }
  }
  renderMainPanel();
}

async function selectTask(id) {
  state.selectedTask = id;
  state.selectedRun  = null;
  state.taskRuns     = [];
  selectedRunLastStatus = null;
  hideRunDetail();
  renderMainPanel();
  try {
    const task = await apiFetch(
      `/api/projects/${enc(state.selectedProject)}/tasks/${enc(id)}?include_files=0`
    );
    state.taskRuns = task.runs || [];
  } catch {
    state.taskRuns = [];
  }
  renderMainPanel();
}

// ── Main panel ────────────────────────────────────────────────────────────────

function renderMainPanel() {
  const hdrEl = document.getElementById('main-header');
  const panEl = document.getElementById('main-panel');

  if (!state.selectedProject) {
    hdrEl.textContent = 'Tasks';
    panEl.innerHTML   = '<p class="empty">Select a project</p>';
    return;
  }

  hdrEl.textContent = `Tasks — ${state.selectedProject}`;

  if (!state.tasks.length) {
    panEl.innerHTML = '<p class="empty">No tasks found</p>';
    return;
  }

  panEl.innerHTML = state.tasks.map(t => {
    const sel  = t.id === state.selectedTask;
    const icon = stIcon(t.status);
    const ago  = timeAgo(t.last_activity);
    let runLabel;
    if (sel && state.taskRuns.length > 0) {
      const runningCount = state.taskRuns.filter(r => r.status === 'running').length;
      runLabel = runningCount > 0
        ? `${state.taskRuns.length} runs, ${runningCount} running`
        : `${state.taskRuns.length} runs`;
    } else {
      runLabel = `Runs: ${t.run_count}`;
    }
    const runs = sel ? renderRunsSection() : '';
    return `<div class="task-card${sel ? ' selected' : ''}">
      <div class="task-row" onclick="selectTask(${js(t.id)})">
        ${icon}
        <span class="task-id">${h(t.id)}</span>
        <span class="task-meta">${runLabel} · ${ago}</span>
      </div>
      ${runs}
    </div>`;
  }).join('');
}

function renderRunsSection() {
  if (!state.taskRuns.length) {
    return '<div class="runs-section"><p class="empty">No runs yet</p></div>';
  }
  // Show most-recent first
  const items = [...state.taskRuns].reverse().map(r => {
    const sel = r.id === state.selectedRun;
    const icon = stIcon(r.status);
    const dur  = fmtDuration(r.start_time, r.end_time || Date.now());
    return `<div class="run-row${sel ? ' selected' : ''}" onclick="selectRun(${js(r.id)})">
      ${icon}
      <span class="run-id">${h(r.id)}</span>
      <span class="run-agent">${h(r.agent || '')}</span>
      <span class="run-meta">exit:${r.exit_code} · ${dur} · ${shortTime(r.start_time)}</span>
    </div>`;
  }).join('');
  return `<div class="runs-section">${items}</div>`;
}

// ── Run detail ────────────────────────────────────────────────────────────────

async function selectRun(id) {
  state.selectedRun = id;
  const run = state.taskRuns.find(r => r.id === id);
  selectedRunLastStatus = run ? run.status : null;
  renderMainPanel(); // highlight selected run
  showRunDetail();
  await loadRunMeta();
  await loadTabContent();
}

async function loadRunMeta() {
  const prefix = runPrefix();
  try {
    const run = await apiFetch(`${prefix}/runs/${enc(state.selectedRun)}`);
    const dur  = fmtDuration(run.start_time, run.end_time || Date.now());
    document.getElementById('run-meta').innerHTML =
      `<span>Run: <b>${h(run.id)}</b></span>` +
      `<span>Agent: ${h(run.agent || '—')}</span>` +
      `<span>Status: <b class="${stClass(run.status)}">${h(run.status)}</b></span>` +
      `<span>Exit: ${run.exit_code}</span>` +
      `<span>Duration: ${dur}</span>` +
      `<span>${shortTime(run.start_time)}</span>`;
    const stopBtn = document.getElementById('stop-run-btn');
    if (stopBtn) {
      if (run.status === 'running') {
        stopBtn.classList.remove('hidden');
      } else {
        stopBtn.classList.add('hidden');
      }
    }
  } catch (e) {
    document.getElementById('run-meta').innerHTML =
      `<span class="error-msg">Error loading run: ${h(e.message)}</span>`;
  }
}

async function stopCurrentRun() {
  if (!state.selectedRun) return;
  const prefix = runPrefix();
  try {
    const resp = await fetch(API_BASE + `${prefix}/runs/${enc(state.selectedRun)}/stop`, {
      method: 'POST',
      headers: UI_REQUEST_HEADERS,
    });
    if (!resp.ok) {
      const data = await resp.json().catch(() => ({}));
      const msg = (data.error && data.error.message) || `HTTP ${resp.status}`;
      showToast(`Stop agent failed: ${msg}`, true);
      return;
    }
    showToast('Stop agent signal sent');
    await loadRunMeta();
  } catch (e) {
    showToast(`Stop agent error: ${e.message}`, true);
  }
}

function stopTabSSE() {
  if (tabSseSource) {
    tabSseSource.close();
    tabSseSource = null;
    tabSseRunId  = null;
    tabSseTab    = null;
  }
}

async function loadTabContent() {
  const tab = state.activeTab;
  const el  = document.getElementById('tab-content');

  // For messages tab, SSE is task-scoped; check task+tab instead of run+tab.
  if (tab === 'messages' && tabSseSource && tabSseTab === 'messages' && tabSseRunId === state.selectedTask) {
    return;
  }

  // If SSE is already streaming this exact run/tab, let it continue.
  if (tabSseSource && tabSseRunId === state.selectedRun && tabSseTab === tab) {
    return;
  }

  stopTabSSE();

  const prefix = runPrefix();

  if (tab === 'messages') {
    el.innerHTML = '';
    el._lineCount = 0;
    const sseUrl = `${API_BASE}/api/projects/${enc(state.selectedProject)}/tasks/${enc(state.selectedTask)}/messages/stream`;
    const source = new EventSource(sseUrl);
    tabSseSource = source;
    tabSseRunId  = state.selectedTask; // task-scoped; not run-scoped
    tabSseTab    = 'messages';

    source.addEventListener('message', (event) => {
      try {
        const m = JSON.parse(event.data);
        const cls  = msgTypeClass(m.type);
        const text = `[${shortTime(m.timestamp)}] [${m.type || ''}] ${m.content || ''}`;
        appendMessageLine(el, cls, text);
      } catch { /* ignore parse errors */ }
    });

    source.addEventListener('heartbeat', () => { /* keep-alive, no-op */ });

    source.onerror = () => {
      if (tabSseSource !== source) return;
      // browser auto-reconnects EventSource; no cleanup needed here
    };

    return;
  }

  // Task-scoped tab: fetch TASK.md from the task directory (not run-scoped).
  if (tab === 'task.md') {
    el.textContent = 'Loading…';
    try {
      const data = await apiFetch(`${prefix}/file?name=TASK.md`);
      el.textContent = data.content || '(empty)';
    } catch (e) {
      el.textContent = (e.status === 404) ? 'No TASK.md found' : `Error: ${e.message}`;
    }
    return;
  }

  // File tab — check if the run is currently active.
  const run = state.taskRuns.find(r => r.id === state.selectedRun);
  const isRunning = run && run.status === 'running';

  // Load current content via API for immediate display.
  el.textContent = 'Loading…';
  try {
    const data = await apiFetch(
      `${prefix}/runs/${enc(state.selectedRun)}/file?name=${enc(tab)}`
    );
    let content = data.content || (isRunning ? '' : '(empty)');
    if (data.fallback) {
      content = `[Note: output.md not found, showing ${data.fallback}]\n\n` + content;
    }
    el.textContent = content;
  } catch (e) {
    if (e.status === 404 && isRunning) {
      el.textContent = ''; // file not yet created — SSE will populate it
    } else {
      el.textContent = (e.status === 404) ? `(${tab} not available)` : `Error: ${e.message}`;
      el.scrollTop = el.scrollHeight;
      return;
    }
  }
  el.scrollTop = el.scrollHeight;

  if (!isRunning) return;

  // Start SSE for live streaming of growing file content.
  const sseUrl = `${API_BASE}${prefix}/runs/${enc(state.selectedRun)}/stream?name=${enc(tab)}`;
  const source = new EventSource(sseUrl);
  tabSseSource = source;
  tabSseRunId  = state.selectedRun;
  tabSseTab    = tab;
  let sseFirst = true;

  source.onmessage = (event) => {
    if (sseFirst) {
      // First message contains all file content from offset 0; replace
      // the API snapshot to avoid drift if the file grew in between.
      sseFirst = false;
      el.textContent = event.data;
    } else {
      el.textContent += event.data;
    }
    el.scrollTop = el.scrollHeight;
  };

  source.addEventListener('done', () => {
    source.close();
    if (tabSseSource === source) {
      tabSseSource = null;
      tabSseRunId  = null;
      tabSseTab    = null;
    }
    // Reload once via API to display the definitive final content.
    loadTabContent();
  });

  source.onerror = () => {
    if (tabSseSource !== source) return; // already superseded
    source.close();
    tabSseSource = null;
    tabSseRunId  = null;
    tabSseTab    = null;
    el.textContent += '\n(stream ended or error)';
  };
}

function switchTab(name) {
  state.activeTab = name;
  document.querySelectorAll('.tab-btn').forEach(b => {
    b.classList.toggle('active', b.dataset.tab === name);
  });
  updateMsgCompose();
  if (state.selectedRun) loadTabContent();
}

function showRunDetail() {
  document.getElementById('run-detail').classList.remove('hidden');
  updateMsgCompose();
}

function hideRunDetail() {
  stopTabSSE();
  document.getElementById('run-detail').classList.add('hidden');
  updateMsgCompose();
}

function closeRun() {
  state.selectedRun = null;
  selectedRunLastStatus = null;
  hideRunDetail();
  renderMainPanel();
}

// ── New Task modal ─────────────────────────────────────────────────────────────

function openNewTaskModal() {
  if (state.selectedProject) {
    document.getElementById('nt-project-id').value = state.selectedProject;
  }
  document.getElementById('nt-error').classList.add('hidden');
  document.getElementById('new-task-dialog').showModal();
}

function closeNewTaskModal() {
  document.getElementById('new-task-dialog').close();
}

function generateTaskId() {
  const now = new Date();
  const p = (n, w = 2) => String(n).padStart(w, '0');
  const date = `${now.getFullYear()}${p(now.getMonth() + 1)}${p(now.getDate())}`;
  const time = `${p(now.getHours())}${p(now.getMinutes())}${p(now.getSeconds())}`;
  const rand = Math.random().toString(36).slice(2, 7);
  return `task-${date}-${time}-${rand}`;
}

async function submitNewTask() {
  const projectId   = document.getElementById('nt-project-id').value.trim();
  const taskIdVal   = document.getElementById('nt-task-id').value.trim();
  const agentType   = document.getElementById('nt-agent-type').value;
  const prompt      = document.getElementById('nt-prompt').value.trim();
  const projectRoot = document.getElementById('nt-project-root').value.trim();
  const attachMode  = document.getElementById('nt-attach-mode').value;
  const errEl       = document.getElementById('nt-error');

  errEl.classList.add('hidden');
  if (!projectId) {
    errEl.textContent = 'Project ID is required';
    errEl.classList.remove('hidden');
    return;
  }
  if (!prompt) {
    errEl.textContent = 'Prompt is required';
    errEl.classList.remove('hidden');
    return;
  }

  const taskId = taskIdVal || generateTaskId();
  const body = { project_id: projectId, task_id: taskId, agent_type: agentType, prompt, attach_mode: attachMode };
  if (projectRoot) body.project_root = projectRoot;

  try {
    const resp = await fetch(API_BASE + '/api/v1/tasks', {
      method: 'POST',
      headers: { ...UI_REQUEST_HEADERS, 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(`HTTP ${resp.status}: ${text}`);
    }
    const data = await resp.json();
    closeNewTaskModal();
    showToast(`Task created: ${data.task_id}`);
    await fullRefresh();
  } catch (e) {
    errEl.textContent = e.message;
    errEl.classList.remove('hidden');
  }
}

function showToast(msg, isError = false) {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = `toast${isError ? ' toast-error' : ''}`;
  clearTimeout(el._timer);
  el._timer = setTimeout(() => { el.className = 'toast hidden'; }, 4000);
}

// ── SSE live updates ──────────────────────────────────────────────────────────

function connectSSE() {
  if (sseSource) sseSource.close();
  sseSource = new EventSource(API_BASE + '/api/v1/runs/stream/all');
  sseSource.onmessage = () => {
    if (sseRefreshTimer) return;
    sseRefreshTimer = setTimeout(() => {
      sseRefreshTimer = null;
      fullRefresh();
    }, SSE_REFRESH_DEBOUNCE_MS);
  };
  // onerror: browser auto-reconnects EventSource
}

// ── Full refresh ──────────────────────────────────────────────────────────────

async function fullRefresh() {
  if (refreshInFlight) {
    refreshQueued = true;
    return;
  }
  refreshInFlight = true;
  try {
    await refreshStatusBar();

    // Refresh project list
    try {
      const data = await apiFetch('/api/projects');
      state.projects = data.projects || [];
    } catch { /* keep */ }
    renderProjectList();

    // Refresh task list
    if (state.selectedProject) {
      try {
        const data = await apiFetch(`/api/projects/${enc(state.selectedProject)}/tasks`);
        state.tasks = data.tasks || [];
      } catch { /* keep */ }
    }

    // Refresh runs for selected task
    if (state.selectedProject && state.selectedTask) {
      try {
        const task = await apiFetch(
          `/api/projects/${enc(state.selectedProject)}/tasks/${enc(state.selectedTask)}?include_files=0`
        );
        state.taskRuns = task.runs || [];
      } catch { /* keep */ }
    }

    renderMainPanel();

    // Refresh run detail if visible
    if (state.selectedRun) {
      const selectedRun = state.taskRuns.find(r => r.id === state.selectedRun);
      const selectedStatus = selectedRun ? selectedRun.status : null;
      const statusChanged = selectedStatus !== selectedRunLastStatus;
      if (selectedStatus === 'running' || statusChanged) {
        await loadRunMeta();
        await loadTabContent();
      }
      selectedRunLastStatus = selectedStatus;
    } else {
      selectedRunLastStatus = null;
    }
  } finally {
    refreshInFlight = false;
    if (refreshQueued) {
      refreshQueued = false;
      setTimeout(() => fullRefresh(), 0);
    }
  }
}

// ── Message compose ───────────────────────────────────────────────────────────

function updateMsgCompose() {
  const el = document.getElementById('msg-compose');
  if (!el) return;
  if (state.selectedTask && state.activeTab === 'messages') {
    el.classList.remove('hidden');
  } else {
    el.classList.add('hidden');
  }
}

async function postMessage() {
  const type = document.getElementById('msg-type').value;
  const bodyEl = document.getElementById('msg-body');
  const body = bodyEl.value.trim();
  if (!body) return;
  try {
    const url = state.selectedTask
      ? `${API_BASE}/api/projects/${enc(state.selectedProject)}/tasks/${enc(state.selectedTask)}/messages`
      : `${API_BASE}/api/projects/${enc(state.selectedProject)}/messages`;
    const resp = await fetch(url, {
      method: 'POST',
      headers: { ...UI_REQUEST_HEADERS, 'Content-Type': 'application/json' },
      body: JSON.stringify({ type, body }),
    });
    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(`HTTP ${resp.status}: ${text}`);
    }
    bodyEl.value = '';
    showToast('Message posted');
  } catch (e) {
    showToast(`Error: ${e.message}`, true);
  }
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function runPrefix() {
  return `/api/projects/${enc(state.selectedProject)}/tasks/${enc(state.selectedTask)}`;
}

function stIcon(status) {
  switch (status) {
    case 'running':   return '<span class="st-running">●</span>';
    case 'completed': return '<span class="st-ok">✓</span>';
    case 'failed':    return '<span class="st-err">✗</span>';
    default:          return '<span class="st-idle">○</span>';
  }
}

function msgTypeClass(type) {
  if (!type) return '';
  switch (type.toUpperCase()) {
    case 'RUN_CRASH':    return 'msg-crash';
    case 'RUN_START':
    case 'RUN_COMPLETE': return 'msg-run';
    case 'TASK_DONE':    return 'msg-ok';
    case 'USER':
    case 'QUESTION':     return 'msg-user';
    default:             return '';
  }
}

function appendMessageLine(el, cls, text) {
  if (!el) return;
  const span = document.createElement('span');
  if (cls) span.className = cls;
  span.textContent = text;
  el.appendChild(span);
  el.appendChild(document.createTextNode('\n'));

  if (typeof el._lineCount !== 'number') {
    el._lineCount = 0;
  }
  el._lineCount += 1;

  while (el._lineCount > MAX_STREAM_LINES && el.firstChild) {
    el.removeChild(el.firstChild);
    if (el.firstChild) {
      el.removeChild(el.firstChild);
    }
    el._lineCount -= 1;
  }

  el.scrollTop = el.scrollHeight;
}

function stClass(status) {
  switch (status) {
    case 'running':   return 'st-running';
    case 'completed': return 'st-ok';
    case 'failed':    return 'st-err';
    default:          return '';
  }
}

function timeAgo(ts) {
  if (!ts) return '—';
  const s = Math.floor((Date.now() - new Date(ts)) / 1000);
  if (s < 60)   return `${s}s ago`;
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return new Date(ts).toLocaleDateString();
}

function shortTime(ts) {
  if (!ts) return '—';
  return new Date(ts).toLocaleTimeString('en-US', { hour12: false });
}

function fmtDuration(start, end) {
  const s = Math.floor((new Date(end) - new Date(start)) / 1000);
  const m = Math.floor(s / 60);
  return m > 0 ? `${m}m${s % 60}s` : `${s}s`;
}

function fmtUptime(secs) {
  const s = Math.floor(secs);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (h > 0) return `${h}h${m}m`;
  return `${m}m${s % 60}s`;
}

// Escape for HTML content
function h(str) {
  if (str == null) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

// Encode for URL
function enc(str) {
  return encodeURIComponent(str || '');
}

// Serialize for inline onclick attribute (safe JSON string)
function js(val) {
  return JSON.stringify(String(val));
}

// ── Init ──────────────────────────────────────────────────────────────────────

async function init() {
  // Wire up tab buttons
  document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => switchTab(btn.dataset.tab));
  });

  // Initial load
  await fullRefresh();

  // Periodic auto-refresh
  refreshTimer = setInterval(fullRefresh, REFRESH_MS);

  // SSE for live run updates
  connectSSE();
}

document.addEventListener('DOMContentLoaded', init);
