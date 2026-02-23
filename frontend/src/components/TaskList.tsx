import { type FormEvent, useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import Dialog from '@jetbrains/ring-ui-built/components/dialog/dialog'
import clsx from 'clsx'
import type { Project, RunStatus, TaskSummary } from '../types'
import { useStartTask } from '../hooks/useAPI'
import type { TaskStartRequest } from '../api/client'
import { ProjectStats } from './ProjectStats'

const STATUS_BADGE_ORDER: RunStatus[] = ['running', 'failed', 'completed']

const statusFilters = ['all', 'running', 'queued', 'blocked', 'completed', 'failed'] as const
export type StatusFilter = (typeof statusFilters)[number]

function parseDate(value?: string) {
  if (!value) {
    return 0
  }
  return new Date(value).getTime()
}

function generateTaskId(): string {
  const now = new Date()
  const date = now.toISOString().slice(0, 10).replace(/-/g, '')
  const time = now.toTimeString().slice(0, 8).replace(/:/g, '')
  const rand = Math.random().toString(36).slice(2, 8)
  return `task-${date}-${time}-${rand}`
}

const emptyForm = (): TaskStartRequest => ({
  task_id: generateTaskId(),
  prompt: '',
  project_root: '',
  attach_mode: 'create',
  agent_type: 'claude',
})

export function TaskList({
  projects,
  tasks,
  selectedProjectId,
  selectedTaskId,
  onSelectProject,
  onSelectTask,
  onRefresh,
  homeDirs = [],
}: {
  projects: Project[]
  tasks: TaskSummary[]
  selectedProjectId?: string
  selectedTaskId?: string
  onSelectProject: (projectId: string) => void
  onSelectTask: (taskId: string) => void
  onRefresh?: () => void
  homeDirs?: string[]
}) {
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [searchText, setSearchText] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState<TaskStartRequest>(emptyForm)
  const [dependsOnInput, setDependsOnInput] = useState('')
  const [submitError, setSubmitError] = useState<string | null>(null)

  const startTaskMutation = useStartTask(selectedProjectId)

  const filteredTasks = useMemo(() => {
    let filtered = tasks
    if (searchText.trim()) {
      const q = searchText.trim().toLowerCase()
      filtered = filtered.filter((task) => task.id.toLowerCase().includes(q))
    }
    if (statusFilter !== 'all') {
      filtered = filtered.filter((task) => task.status === statusFilter)
    }
    return [...filtered].sort((a, b) => parseDate(b.last_activity) - parseDate(a.last_activity))
  }, [statusFilter, searchText, tasks])

  const openDialog = () => {
    setForm({ ...emptyForm(), task_id: generateTaskId() })
    setDependsOnInput('')
    setSubmitError(null)
    setShowCreate(true)
  }

  const closeDialog = () => {
    setShowCreate(false)
    setSubmitError(null)
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSubmitError(null)
    try {
      const dependsOn = dependsOnInput
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean)
      await startTaskMutation.mutateAsync({
        ...form,
        depends_on: dependsOn.length > 0 ? dependsOn : undefined,
      })
      closeDialog()
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : 'Failed to create task')
    }
  }

  return (
    <div className="panel panel-scroll">
      <div className="panel-header">
        <div>
          <div className="panel-title">Projects</div>
          <div className="panel-subtitle">Last activity first</div>
        </div>
        <Button inline onClick={onRefresh} aria-label="Refresh projects">
          Refresh
        </Button>
      </div>
      <div className="panel-section">
        <div className="list">
          {projects.map((project) => (
            <button
              key={project.id}
              type="button"
              className={clsx('list-item', selectedProjectId === project.id && 'list-item-active')}
              onClick={() => onSelectProject(project.id)}
            >
              <div className="list-item-title">{project.id}</div>
              <div className="list-item-meta">
                {project.task_count} tasks · {new Date(project.last_activity).toLocaleString()}
              </div>
            </button>
          ))}
          {projects.length === 0 && <div className="empty-state">No projects yet.</div>}
        </div>
      </div>
      <div className="panel-divider" />
      <div className="panel-header">
        <div>
          <div className="panel-title">Tasks</div>
          <div className="panel-subtitle">{selectedProjectId ? `Project: ${selectedProjectId}` : 'No project selected'}</div>
        </div>
        <Button
          inline
          onClick={openDialog}
          disabled={!selectedProjectId}
          aria-label="Create new task"
          title={selectedProjectId ? 'Create new task' : 'Select a project first'}
        >
          + New Task
        </Button>
      </div>
      {selectedProjectId && <ProjectStats projectId={selectedProjectId} />}
      <div className="panel-section">
        <div className="task-search-row">
          <input
            className="input task-search-input"
            type="text"
            placeholder="Search tasks..."
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            aria-label="Search tasks"
          />
          {searchText && (
            <button
              type="button"
              className="task-search-clear"
              onClick={() => setSearchText('')}
              aria-label="Clear search"
            >
              ×
            </button>
          )}
        </div>
      </div>
      <div className="panel-section filters">
        {statusFilters.map((filter) => (
          <Button
            key={filter}
            inline
            className={clsx('filter-button', statusFilter === filter && 'filter-button-active')}
            onClick={() => setStatusFilter(filter)}
            data-testid={`task-filter-${filter}`}
          >
            {filter}
          </Button>
        ))}
      </div>
      {(searchText.trim() || statusFilter !== 'all') && (
        <div className="panel-section task-count-row">
          Showing {filteredTasks.length} of {tasks.length} tasks
        </div>
      )}
      <div className="panel-section panel-section-tight">
        <div className="list">
          {filteredTasks.map((task) => (
            <button
              key={task.id}
              type="button"
              className={clsx('list-item', selectedTaskId === task.id && 'list-item-active')}
              onClick={() => onSelectTask(task.id)}
              data-testid={`task-item-${task.id}`}
            >
              <div className="list-item-title">{task.name ?? task.id}</div>
              <div className="list-item-meta">
                <span className={clsx('status-dot', `status-${task.status}`)} />
                {task.status} · {new Date(task.last_activity).toLocaleString()}
                {task.status === 'queued' && task.queue_position && task.queue_position > 0
                  ? ` · queue #${task.queue_position}`
                  : ''}
                {task.status === 'blocked' && task.blocked_by && task.blocked_by.length > 0
                  ? ` · blocked by ${task.blocked_by.join(', ')}`
                  : ''}
              </div>
              {task.depends_on && task.depends_on.length > 0 && (
                <div className="list-item-meta">depends on: {task.depends_on.join(', ')}</div>
              )}
              {task.run_counts && (
                <div className="run-count-badges">
                  {STATUS_BADGE_ORDER.filter((s) => (task.run_counts?.[s] ?? 0) > 0).map((s) => (
                    <span key={s} className={`run-count-badge run-count-${s}`}>
                      {task.run_counts![s]} {s}
                    </span>
                  ))}
                </div>
              )}
            </button>
          ))}
          {filteredTasks.length === 0 && <div className="empty-state">No tasks match the filter.</div>}
        </div>
      </div>

      <Dialog
        show={showCreate}
        label="Create new task"
        showCloseButton
        onCloseAttempt={closeDialog}
      >
        <div className="dialog-content">
          <div className="dialog-title">
            New Task — {selectedProjectId}
          </div>
          <form onSubmit={handleSubmit}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span className="form-label">Task ID</span>
                <input
                  className="input"
                  style={{ width: '100%' }}
                  value={form.task_id}
                  onChange={(e) => setForm((f) => ({ ...f, task_id: e.target.value }))}
                  required
                  placeholder="task-20260220-120000-my-task"
                />
              </label>

              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span className="form-label">Agent</span>
                <select
                  className="input"
                  style={{ width: '100%' }}
                  value={form.agent_type ?? 'claude'}
                  onChange={(e) => setForm((f) => ({ ...f, agent_type: e.target.value }))}
                >
                  <option value="claude">claude</option>
                  <option value="codex">codex</option>
                  <option value="gemini">gemini</option>
                </select>
              </label>

              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span className="form-label">Prompt *</span>
                <textarea
                  className="input"
                  style={{ width: '100%', minHeight: '100px', resize: 'vertical', fontFamily: 'inherit' }}
                  value={form.prompt}
                  onChange={(e) => setForm((f) => ({ ...f, prompt: e.target.value }))}
                  required
                  placeholder="Describe what the agent should do..."
                />
              </label>

              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span className="form-label">Project Home Directory</span>
                <div style={{ position: 'relative' }}>
                  <input
                    className="input"
                    style={{ width: '100%' }}
                    list="home-dir-suggestions"
                    value={form.project_root}
                    onChange={(e) => setForm((f) => ({ ...f, project_root: e.target.value }))}
                    placeholder="~/Work/my-project  or  /absolute/path/to/project"
                    autoComplete="off"
                  />
                  <datalist id="home-dir-suggestions">
                    {homeDirs.map((dir) => (
                      <option key={dir} value={dir} />
                    ))}
                  </datalist>
                </div>
                <span className="form-hint">
                  Working directory where the agent will run. Use ~ for home directory.
                  The conductor-loop task folder is managed separately.
                </span>
              </label>

              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span className="form-label">Attach Mode</span>
                <select
                  className="input"
                  style={{ width: '100%' }}
                  value={form.attach_mode}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, attach_mode: e.target.value as TaskStartRequest['attach_mode'] }))
                  }
                >
                  <option value="create">create</option>
                  <option value="attach">attach</option>
                  <option value="resume">resume</option>
                </select>
              </label>

              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span className="form-label">Depends on</span>
                <input
                  className="input"
                  style={{ width: '100%' }}
                  value={dependsOnInput}
                  onChange={(e) => setDependsOnInput(e.target.value)}
                  placeholder="task-a, task-b"
                />
                <span className="form-hint">Comma-separated task IDs that must complete before this task can start.</span>
              </label>

              {submitError && (
                <div className="form-error">{submitError}</div>
              )}

              <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end', marginTop: '4px' }}>
                <Button inline onClick={closeDialog} type="button">
                  Cancel
                </Button>
                <Button
                  primary
                  type="submit"
                  disabled={startTaskMutation.isPending}
                >
                  {startTaskMutation.isPending ? 'Creating…' : 'Create Task'}
                </Button>
              </div>
            </div>
          </form>
        </div>
      </Dialog>
    </div>
  )
}
