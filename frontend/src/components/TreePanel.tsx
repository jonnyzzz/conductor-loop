import { type FormEvent, useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import Dialog from '@jetbrains/ring-ui-built/components/dialog/dialog'
import clsx from 'clsx'
import { useHomeDirs, useProjects, useProjectRunsFlat, useStartTask, useTasks } from '../hooks/useAPI'
import { buildTree, type TreeNode } from '../utils/treeBuilder'
import type { TaskStartRequest } from '../api/client'

interface TreePanelProps {
  projectId: string | undefined
  selectedProjectId: string | undefined
  selectedTaskId: string | undefined
  selectedRunId: string | undefined
  onSelectProject: (id: string) => void
  onSelectTask: (projectId: string, taskId: string) => void
  onSelectRun: (projectId: string, taskId: string, runId: string) => void
}

function generateTaskIdPrefix(): string {
  const now = new Date()
  const date = now.toISOString().slice(0, 10).replace(/-/g, '')
  const time = now.toTimeString().slice(0, 8).replace(/:/g, '')
  return `task-${date}-${time}`
}

function generateTaskSuffix(): string {
  return Math.random().toString(36).slice(2, 7)
}

function sanitizeTaskSuffix(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
    .slice(0, 48)
}

function buildTaskId(prefix: string, suffix: string): string {
  const normalized = sanitizeTaskSuffix(suffix)
  return normalized ? `${prefix}-${normalized}` : prefix
}

const emptyForm = (taskId: string): TaskStartRequest => ({
  task_id: taskId,
  prompt: '',
  project_root: '',
  attach_mode: 'create',
  agent_type: 'claude',
})

function statusDot(status: string): string {
  switch (status) {
    case 'running': return '●'
    case 'completed': return '✓'
    case 'failed': return '✗'
    default: return '○'
  }
}

function statusClass(status: string): string {
  switch (status) {
    case 'running': return 'tree-status-running'
    case 'completed': return 'tree-status-completed'
    case 'failed': return 'tree-status-failed'
    default: return 'tree-status-unknown'
  }
}

function formatTime(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

interface TreeNodeProps {
  node: TreeNode
  depth: number
  selectedProjectId: string | undefined
  selectedTaskId: string | undefined
  selectedRunId: string | undefined
  onSelectProject: (id: string) => void
  onSelectTask: (projectId: string, taskId: string) => void
  onSelectRun: (projectId: string, taskId: string, runId: string) => void
  onCreateTask?: (projectId: string) => void
  defaultExpanded?: boolean
}

function TreeNodeRow({
  node,
  depth,
  selectedProjectId,
  selectedTaskId,
  selectedRunId,
  onSelectProject,
  onSelectTask,
  onSelectRun,
  onCreateTask,
  defaultExpanded = true,
}: TreeNodeProps) {
  const [expanded, setExpanded] = useState(
    defaultExpanded || node.status === 'running'
  )
  const taskRepresentsSelectedLatestRun =
    node.type === 'task' &&
    node.inlineLatestRun &&
    selectedRunId != null &&
    selectedRunId === node.latestRunId

  const isSelected =
    (node.type === 'project' && selectedProjectId === node.id && !selectedTaskId && !selectedRunId) ||
    (node.type === 'task' && selectedTaskId === node.id && (!selectedRunId || taskRepresentsSelectedLatestRun)) ||
    (node.type === 'run' && selectedRunId === node.id)

  const hasChildren = node.children.length > 0

  function handleClick() {
    if (node.type === 'project' && node.projectId) {
      onSelectProject(node.projectId)
    } else if (node.type === 'task' && node.projectId && node.taskId) {
      onSelectTask(node.projectId, node.taskId)
    } else if (node.type === 'run' && node.projectId && node.taskId) {
      onSelectRun(node.projectId, node.taskId, node.id)
    }
  }

  function handleToggle(e: React.MouseEvent) {
    e.stopPropagation()
    setExpanded((v) => !v)
  }

  const hasProjectAction = Boolean(node.type === 'project' && node.projectId && onCreateTask)
  const runLabel = node.id.length > 18 ? `${node.id.slice(0, 18)}…` : node.id

  return (
    <div className="tree-node">
      <div className={clsx('tree-row-shell', hasProjectAction && 'tree-row-shell-project')}>
        <button
          type="button"
          className={clsx('tree-row', isSelected && 'tree-row-active', hasProjectAction && 'tree-row-with-action')}
          style={{ paddingLeft: `${8 + depth * 16}px` }}
          onClick={handleClick}
        >
          <span
            className={clsx('tree-toggle', !hasChildren && 'tree-toggle-empty')}
            onClick={hasChildren ? handleToggle : undefined}
            aria-label={expanded ? 'Collapse' : 'Expand'}
          >
            {hasChildren ? (expanded ? '▾' : '▸') : ' '}
          </span>

          {node.type === 'project' && (
            <>
              <span className="tree-icon">⬡</span>
              <span className="tree-label tree-label-project">{node.label}</span>
              <span className="tree-badge tree-badge-count">{node.children.length}</span>
            </>
          )}

          {node.type === 'task' && (
            <>
              <span className={clsx('tree-status-dot', statusClass(node.status))}>
                {statusDot(node.status)}
              </span>
              <span className={clsx('tree-label', node.status === 'running' && 'tree-label-active')}>
                {node.label}
              </span>
              {node.latestRunAgent && (
                <span className={clsx('tree-badge tree-badge-agent', node.latestRunStatus === 'running' && 'tree-badge-agent-running')}>
                  [{node.latestRunAgent}]
                </span>
              )}
              {node.latestRunTime && (
                <span className="tree-time">{formatTime(node.latestRunTime)}</span>
              )}
              {node.restartCount != null && node.restartCount > 0 && (
                <span
                  className="tree-badge tree-badge-restart"
                  title={`${node.restartCount} restart${node.restartCount === 1 ? '' : 's'} in chain`}
                >
                  ↻{node.restartCount}
                </span>
              )}
            </>
          )}

          {node.type === 'run' && (
            <>
              <span className={clsx('tree-status-dot', statusClass(node.status))}>
                {statusDot(node.status)}
              </span>
              <span className={clsx('tree-label tree-label-run', node.status === 'running' && 'tree-label-active')}>
                {runLabel}
              </span>
              <span className="tree-badge tree-badge-agent">[{node.agent ?? '?'}]</span>
              <span className="tree-time">{formatTime(node.startTime)}</span>
            </>
          )}
        </button>
        {hasProjectAction && (
          <button
            type="button"
            className="tree-inline-action"
            onClick={(event) => {
              event.stopPropagation()
              onCreateTask?.(node.projectId!)
            }}
          >
            + New Task
          </button>
        )}
      </div>

      {expanded && hasChildren && (
        <div className="tree-children">
          {node.children.map((child) => (
            <TreeNodeRow
              key={child.id}
              node={child}
              depth={depth + 1}
              selectedProjectId={selectedProjectId}
              selectedTaskId={selectedTaskId}
              selectedRunId={selectedRunId}
              onSelectProject={onSelectProject}
              onSelectTask={onSelectTask}
              onSelectRun={onSelectRun}
              onCreateTask={onCreateTask}
              defaultExpanded={child.type === 'project' || child.status === 'running'}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export function TreePanel({
  projectId,
  selectedProjectId,
  selectedTaskId,
  selectedRunId,
  onSelectProject,
  onSelectTask,
  onSelectRun,
}: TreePanelProps) {
  const projectsQuery = useProjects()
  const tasksQuery = useTasks(projectId)
  const flatRunsQuery = useProjectRunsFlat(projectId)
  const homeDirsQuery = useHomeDirs()
  const startTaskMutation = useStartTask(projectId)

  const [showCreate, setShowCreate] = useState(false)
  const [taskIdPrefix, setTaskIdPrefix] = useState(() => generateTaskIdPrefix())
  const [taskIdSuffix, setTaskIdSuffix] = useState(() => generateTaskSuffix())
  const [form, setForm] = useState<TaskStartRequest>(() => emptyForm(buildTaskId(taskIdPrefix, taskIdSuffix)))
  const [submitError, setSubmitError] = useState<string | null>(null)

  const homeDirs = homeDirsQuery.data?.dirs ?? []

  const tree = useMemo(() => {
    if (!projectId) return null
    const tasks = tasksQuery.data ?? []
    const flatRuns = flatRunsQuery.data ?? []
    return buildTree(projectId, tasks, flatRuns)
  }, [projectId, tasksQuery.data, flatRunsQuery.data])

  const projects = projectsQuery.data ?? []
  const derivedTaskId = useMemo(() => buildTaskId(taskIdPrefix, taskIdSuffix), [taskIdPrefix, taskIdSuffix])

  const openDialog = (targetProjectId?: string) => {
    if (targetProjectId && targetProjectId !== projectId) {
      onSelectProject(targetProjectId)
    }
    const nextPrefix = generateTaskIdPrefix()
    const nextSuffix = generateTaskSuffix()
    const nextTaskId = buildTaskId(nextPrefix, nextSuffix)
    setTaskIdPrefix(nextPrefix)
    setTaskIdSuffix(nextSuffix)
    setForm(emptyForm(nextTaskId))
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
      await startTaskMutation.mutateAsync({ ...form, task_id: derivedTaskId })
      closeDialog()
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : 'Failed to create task')
    }
  }

  return (
    <div className="panel panel-scroll tree-panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Tree</div>
          <div className="panel-subtitle">Project · Task · Run</div>
        </div>
        <Button
          inline
          onClick={() => openDialog()}
          disabled={!projectId}
          aria-label="Create new task"
          title={projectId ? 'Create new task' : 'Select a project first'}
        >
          + New Task
        </Button>
      </div>

      {/* Project selector */}
      {projects.length > 1 && (
        <div className="panel-section">
          <div className="list">
            {projects.map((project) => (
              <div key={project.id} className="tree-project-picker-row">
                <button
                  type="button"
                  className={clsx(
                    'list-item list-item-compact tree-project-picker-item',
                    selectedProjectId === project.id && 'list-item-active'
                  )}
                  onClick={() => onSelectProject(project.id)}
                >
                  <span className="list-item-title">{project.id}</span>
                </button>
                <button
                  type="button"
                  className="tree-inline-action tree-inline-action-list"
                  onClick={(event) => {
                    event.stopPropagation()
                    openDialog(project.id)
                  }}
                >
                  + New Task
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tree */}
      <div className="panel-section panel-section-tight tree-content">
        {!projectId && (
          <div className="empty-state">Select a project to see the tree.</div>
        )}
        {projectId && !tree && (
          <div className="empty-state">Loading…</div>
        )}
        {tree && (
          <TreeNodeRow
            node={tree}
            depth={0}
            selectedProjectId={selectedProjectId}
            selectedTaskId={selectedTaskId}
            selectedRunId={selectedRunId}
            onSelectProject={onSelectProject}
            onSelectTask={onSelectTask}
            onSelectRun={onSelectRun}
            onCreateTask={openDialog}
            defaultExpanded={true}
          />
        )}
      </div>

      <Dialog
        show={showCreate}
        label="Create new task"
        showCloseButton
        onCloseAttempt={closeDialog}
      >
        <div className="dialog-content">
          <div className="dialog-title">
            New Task — {projectId}
          </div>
          <form onSubmit={handleSubmit}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span className="form-label">Task ID Prefix</span>
                <input
                  className="input"
                  style={{ width: '100%' }}
                  value={taskIdPrefix}
                  readOnly
                />
              </label>

              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span className="form-label">Task Modifier</span>
                <input
                  className="input"
                  style={{ width: '100%' }}
                  value={taskIdSuffix}
                  onChange={(e) => setTaskIdSuffix(sanitizeTaskSuffix(e.target.value))}
                  placeholder="ux-batch"
                />
                <span className="form-hint">
                  Full task id is derived as <code>{taskIdPrefix}-modifier</code>. Leave empty to use prefix only.
                </span>
              </label>

              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span className="form-label">Full Task ID</span>
                <input
                  className="input"
                  style={{ width: '100%' }}
                  value={derivedTaskId}
                  readOnly
                  required
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
