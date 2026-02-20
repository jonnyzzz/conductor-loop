'use strict';

// When served via file:// (opened directly), point at the conductor server.
// When served from the conductor server itself (/ui/), use relative paths.
const API_BASE = window.location.protocol === 'file:'
  ? 'http://localhost:8080'
  : '';

const REFRESH_MS = 5000;

// ── State ────────────────────────────────────────────────────────────────────

const state = {
  selectedProject: null,
  selectedTask:    null,
  selectedRun:     null,
  taskRuns:        [],   // runs for the currently selected task
  projects:        [],
  tasks:           [],
  activeTab:       'output.md',
};

let refreshTimer = null;
let sseSource    = null;

// ── API ──────────────────────────────────────────────────────────────────────

async function apiFetch(path) {
  const resp = await fetch(API_BASE + path);
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
  renderProjectList();
  hideRunDetail();
  await loadTasks();
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
  renderMainPanel();
}

async function selectTask(id) {
  state.selectedTask = id;
  state.selectedRun  = null;
  state.taskRuns     = [];
  hideRunDetail();
  renderMainPanel();
  try {
    const task = await apiFetch(
      `/api/projects/${enc(state.selectedProject)}/tasks/${enc(id)}`
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
    const runs = sel ? renderRunsSection() : '';
    return `<div class="task-card${sel ? ' selected' : ''}">
      <div class="task-row" onclick="selectTask(${js(t.id)})">
        ${icon}
        <span class="task-id">${h(t.id)}</span>
        <span class="task-meta">Runs: ${t.run_count} · ${ago}</span>
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
    const dur  = r.end_time ? fmtDuration(r.start_time, r.end_time) : 'running';
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
  renderMainPanel(); // highlight selected run
  showRunDetail();
  await loadRunMeta();
  await loadTabContent();
}

async function loadRunMeta() {
  const prefix = runPrefix();
  try {
    const run = await apiFetch(`${prefix}/runs/${enc(state.selectedRun)}`);
    const dur  = run.end_time ? fmtDuration(run.start_time, run.end_time) : 'running';
    document.getElementById('run-meta').innerHTML =
      `<span>Run: <b>${h(run.id)}</b></span>` +
      `<span>Agent: ${h(run.agent || '—')}</span>` +
      `<span>Status: <b class="${stClass(run.status)}">${h(run.status)}</b></span>` +
      `<span>Exit: ${run.exit_code}</span>` +
      `<span>Duration: ${dur}</span>` +
      `<span>${shortTime(run.start_time)}</span>`;
  } catch (e) {
    document.getElementById('run-meta').innerHTML =
      `<span class="error-msg">Error loading run: ${h(e.message)}</span>`;
  }
}

async function loadTabContent() {
  const tab = state.activeTab;
  const el  = document.getElementById('tab-content');
  el.textContent = 'Loading…';

  const prefix = runPrefix();
  try {
    if (tab === 'messages') {
      const data = await apiFetch(
        `/api/v1/messages?project_id=${enc(state.selectedProject)}&task_id=${enc(state.selectedTask)}`
      );
      const msgs = (data.messages || []).slice(-50);
      if (msgs.length) {
        el.innerHTML = msgs.map(m => {
          const cls  = msgTypeClass(m.type);
          const text = `[${h(shortTime(m.timestamp))}] [${h(m.type)}] ${h(m.body || m.content || '')}`;
          return cls ? `<span class="${cls}">${text}</span>` : text;
        }).join('\n');
      } else {
        el.textContent = '(no messages)';
      }
    } else {
      const data = await apiFetch(
        `${prefix}/runs/${enc(state.selectedRun)}/file?name=${enc(tab)}`
      );
      el.textContent = data.content || '(empty)';
    }
    el.scrollTop = el.scrollHeight;
  } catch (e) {
    el.textContent = (e.status === 404) ? `(${tab} not available)` : `Error: ${e.message}`;
  }
}

function switchTab(name) {
  state.activeTab = name;
  document.querySelectorAll('.tab-btn').forEach(b => {
    b.classList.toggle('active', b.dataset.tab === name);
  });
  if (state.selectedRun) loadTabContent();
}

function showRunDetail() {
  document.getElementById('run-detail').classList.remove('hidden');
}

function hideRunDetail() {
  document.getElementById('run-detail').classList.add('hidden');
}

function closeRun() {
  state.selectedRun = null;
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
      headers: { 'Content-Type': 'application/json' },
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
  sseSource.onmessage = () => fullRefresh();
  // onerror: browser auto-reconnects EventSource
}

// ── Full refresh ──────────────────────────────────────────────────────────────

async function fullRefresh() {
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
        `/api/projects/${enc(state.selectedProject)}/tasks/${enc(state.selectedTask)}`
      );
      state.taskRuns = task.runs || [];
    } catch { /* keep */ }
  }

  renderMainPanel();

  // Refresh run detail if visible
  if (state.selectedRun) {
    await loadRunMeta();
    await loadTabContent();
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
    default:             return '';
  }
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
