import { type FormEvent, useEffect, useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import Dialog from '@jetbrains/ring-ui-built/components/dialog/dialog'
import clsx from 'clsx'
import { useCreateProject, useHomeDirs, useProjects, useProjectRunsFlat, useStartTask, useTasks } from '../hooks/useAPI'
import { buildSelectionPathNodeIDs, buildTree, selectTreeRuns, type TreeNode } from '../utils/treeBuilder'
import type { ProjectCreateRequest, TaskStartRequest } from '../api/client'
import type { SSEConnectionState } from '../hooks/useSSE'

interface TreePanelProps {
  projectId: string | undefined
  runsStreamState?: SSEConnectionState
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

const emptyProjectForm: ProjectCreateRequest = {
  project_id: '',
  project_root: '',
}

const likelyProjectRootPattern = /^(~\/|\/|[a-zA-Z]:[\\/]|\\\\)/
const SELECTED_TASK_RUN_LIMIT = 1

function isLikelyProjectRoot(path: string): boolean {
  return likelyProjectRootPattern.test(path)
}

function statusDot(status: string): string {
  switch (status) {
    case 'running': return '●'
    case 'queued': return '◷'
    case 'blocked': return '⏸'
    case 'completed': return '✓'
    case 'failed': return '✗'
    default: return '○'
  }
}

function statusClass(status: string): string {
  switch (status) {
    case 'running': return 'tree-status-running'
    case 'queued': return 'tree-status-queued'
    case 'blocked': return 'tree-status-blocked'
    case 'completed': return 'tree-status-completed'
    case 'failed': return 'tree-status-failed'
    default: return 'tree-status-unknown'
  }
}

function isTerminalTaskStatus(status: string): boolean {
  return status === 'completed' || status === 'failed'
}

function isActiveRunStatus(status: string): boolean {
  return status === 'running' || status === 'queued'
}

function hasActiveSubtree(node: TreeNode): boolean {
  for (const child of node.children) {
    if (child.type === 'task' && !isTerminalTaskStatus(child.status)) {
      return true
    }
    if (child.type === 'run' && isActiveRunStatus(child.status)) {
      return true
    }
    if (hasActiveSubtree(child)) {
      return true
    }
  }
  return false
}

function shouldCollapseTerminalTaskNode(
  node: TreeNode
): boolean {
  if (node.type !== 'task' || !isTerminalTaskStatus(node.status)) {
    return false
  }
  return !hasActiveSubtree(node)
}

function formatTime(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function formatDurationBadge(startIso?: string, endIso?: string): string | null {
  if (!startIso) return null
  const start = Date.parse(startIso)
  if (Number.isNaN(start)) return null
  const end = endIso ? Date.parse(endIso) : Date.now()
  if (Number.isNaN(end) || end < start) return null

  const totalSeconds = Math.floor((end - start) / 1000)
  if (totalSeconds < 120) return `${totalSeconds}s`

  const totalMinutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (totalMinutes < 60) return `${totalMinutes}m${String(seconds).padStart(2, '0')}s`

  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (hours < 24) return `${hours}h${String(minutes).padStart(2, '0')}m`

  const days = Math.floor(hours / 24)
  const remHours = hours % 24
  return `${days}d${String(remHours).padStart(2, '0')}h`
}

interface TreeNodeProps {
  node: TreeNode
  depth: number
  selectionPathNodeIDs: ReadonlySet<string>
  selectedProjectId: string | undefined
  selectedTaskId: string | undefined
  selectedRunId: string | undefined
  onSelectProject: (id: string) => void
  onSelectTask: (projectId: string, taskId: string) => void
  onSelectRun: (projectId: string, taskId: string, runId: string) => void
  onCreateTask?: (projectId: string) => void
  defaultExpanded?: boolean
  terminalTaskCollapse?: {
    collapsed: boolean
    onToggle: () => void
  }
  taskLabelPrefix?: string
}

function TreeNodeRow({
  node,
  depth,
  selectionPathNodeIDs,
  selectedProjectId,
  selectedTaskId,
  selectedRunId,
  onSelectProject,
  onSelectTask,
  onSelectRun,
  onCreateTask,
  defaultExpanded = true,
  terminalTaskCollapse,
  taskLabelPrefix = '',
}: TreeNodeProps) {
  const [expanded, setExpanded] = useState(
    defaultExpanded || node.status === 'running'
  )
  const taskRepresentsSelectedLatestRun =
    node.type === 'task' &&
    node.inlineLatestRun &&
    selectedRunId != null &&
    selectedRunId === node.latestRunId

  const taskChildren = node.type === 'project'
    ? node.children.filter((child) => child.type === 'task')
    : []
  const nonTaskChildren = node.type === 'project'
    ? node.children.filter((child) => child.type !== 'task')
    : node.children
  const visibleTaskChildren: TreeNode[] = []
  const collapsedTerminalTaskChildren: TreeNode[] = []
  for (const child of taskChildren) {
    if (shouldCollapseTerminalTaskNode(child)) {
      collapsedTerminalTaskChildren.push(child)
    } else {
      visibleTaskChildren.push(child)
    }
  }
  const completedTaskCount = collapsedTerminalTaskChildren.filter((child) => child.status === 'completed').length
  const failedTaskCount = collapsedTerminalTaskChildren.filter((child) => child.status === 'failed').length
  const childNodes = node.type === 'project'
    ? [...visibleTaskChildren, ...nonTaskChildren]
    : nonTaskChildren
  const hasTerminalTaskSummary = node.type === 'project' && collapsedTerminalTaskChildren.length > 0
  const showTerminalTaskChildren = !terminalTaskCollapse?.collapsed
  const terminalTaskSummaryText = `... and ${collapsedTerminalTaskChildren.length} more tasks (${completedTaskCount} completed, ${failedTaskCount} failed)`

  const isSelected =
    (node.type === 'project' && selectedProjectId === node.id && !selectedTaskId && !selectedRunId) ||
    (
      node.type === 'task' &&
      selectedTaskId === node.id &&
      (
        !selectedRunId ||
        !selectionPathNodeIDs.has(selectedRunId) ||
        taskRepresentsSelectedLatestRun
      )
    ) ||
    (node.type === 'run' && selectedRunId === node.id)

  const hasChildren = childNodes.length > 0
  const hasExpandableContent = hasChildren || hasTerminalTaskSummary
  const subtreeHasSelection = selectionPathNodeIDs.has(node.id)

  useEffect(() => {
    if (hasExpandableContent && subtreeHasSelection) {
      setExpanded(true)
    }
  }, [hasExpandableContent, subtreeHasSelection])

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

  function handleTerminalTaskToggle(e: React.MouseEvent) {
    e.stopPropagation()
    terminalTaskCollapse?.onToggle()
  }

  const hasProjectAction = Boolean(node.type === 'project' && node.projectId && onCreateTask)
  const taskLabel = node.type === 'task'
    ? (taskLabelPrefix && node.label !== node.id ? `${taskLabelPrefix}${node.label}` : node.label)
    : ''
  const runLabel = node.id.length > 14 ? `${node.id.slice(0, 14)}…` : node.id
  const latestDuration =
    node.type === 'task'
      ? formatDurationBadge(node.latestRunStartTime ?? node.latestRunTime, node.latestRunEndTime)
      : null
  const runDuration =
    node.type === 'run'
      ? formatDurationBadge(node.startTime, node.endTime)
      : null
  const terminalTaskSummaryId = `tree-terminal-summary-${node.id.replace(/[^a-zA-Z0-9_-]/g, '-')}`

  return (
    <div className="tree-node">
      <div className={clsx('tree-row-shell', hasProjectAction && 'tree-row-shell-project')}>
        <button
          type="button"
          className={clsx('tree-row', isSelected && 'tree-row-active', hasProjectAction && 'tree-row-with-action')}
          style={{ paddingLeft: `${4 + depth * 10}px` }}
          onClick={handleClick}
        >
          <span
            className={clsx('tree-toggle', !hasExpandableContent && 'tree-toggle-empty')}
            onClick={hasExpandableContent ? handleToggle : undefined}
            aria-label={expanded ? 'Collapse' : 'Expand'}
          >
            {hasExpandableContent ? (expanded ? '▾' : '▸') : ' '}
          </span>

          {node.type === 'project' && (
            <>
              <span className="tree-icon">⬡</span>
              <span className="tree-label tree-label-project">{node.label}</span>
              <span className="tree-row-right">
                <span className="tree-badge tree-badge-count">{node.children.length}</span>
              </span>
            </>
          )}

          {node.type === 'task' && (
            <>
              <span className={clsx('tree-status-dot', statusClass(node.status))}>
                {statusDot(node.status)}
              </span>
              <span
                className={clsx('tree-label', node.status === 'running' && 'tree-label-active')}
                title={node.id}
              >
                {taskLabel}
              </span>
              <span className="tree-row-right">
                {node.restartCount != null && node.restartCount > 0 && (
                  <span
                    className="tree-badge tree-badge-restart"
                    title={`${node.restartCount} restart${node.restartCount === 1 ? '' : 's'} in chain`}
                  >
                    restart:{node.restartCount}
                  </span>
                )}
                {latestDuration && (
                  <span className="tree-badge tree-badge-duration" title="latest run duration">
                    dur:{latestDuration}
                  </span>
                )}
                {node.latestRunAgent && (
                  <span className={clsx('tree-badge tree-badge-agent', node.latestRunStatus === 'running' && 'tree-badge-agent-running')}>
                    [{node.latestRunAgent}]
                  </span>
                )}
                {node.latestRunTime && (
                  <span className="tree-time">{formatTime(node.latestRunTime)}</span>
                )}
                {node.status === 'blocked' && node.blockedBy && node.blockedBy.length > 0 && (
                  <span className="tree-badge tree-badge-blocked" title={`blocked by ${node.blockedBy.join(', ')}`}>
                    blocked:{node.blockedBy.length}
                  </span>
                )}
                {node.status === 'queued' && node.queuePosition != null && node.queuePosition > 0 && (
                  <span className="tree-badge tree-badge-queued" title="queue position">
                    queue:{node.queuePosition}
                  </span>
                )}
              </span>
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
              <span className="tree-row-right">
                {runDuration && (
                  <span className="tree-badge tree-badge-duration" title="run duration">
                    dur:{runDuration}
                  </span>
                )}
                {node.previousRunId && (
                  <span className="tree-badge tree-badge-restart" title={`restart of ${node.previousRunId}`}>
                    restart
                  </span>
                )}
                <span className="tree-badge tree-badge-agent">[{node.agent ?? '?'}]</span>
                <span className="tree-time">{formatTime(node.startTime)}</span>
              </span>
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

      {expanded && hasExpandableContent && (
        <div className="tree-children">
          {childNodes.map((child) => (
            <TreeNodeRow
              key={child.id}
              node={child}
              depth={depth + 1}
              selectionPathNodeIDs={selectionPathNodeIDs}
              selectedProjectId={selectedProjectId}
              selectedTaskId={selectedTaskId}
              selectedRunId={selectedRunId}
              onSelectProject={onSelectProject}
              onSelectTask={onSelectTask}
              onSelectRun={onSelectRun}
              onCreateTask={onCreateTask}
              defaultExpanded={child.type === 'project' || child.status === 'running'}
              terminalTaskCollapse={undefined}
            />
          ))}

          {hasTerminalTaskSummary && (
            <div className="tree-row-shell">
              <button
                type="button"
                className="tree-row tree-summary-row"
                style={{ paddingLeft: `${14 + (depth + 1) * 10}px` }}
                onClick={handleTerminalTaskToggle}
                aria-expanded={showTerminalTaskChildren}
                aria-controls={terminalTaskSummaryId}
                data-testid="tree-terminal-summary-toggle"
              >
                <span className="tree-summary-toggle" aria-hidden="true">
                  {showTerminalTaskChildren ? '▾' : '▸'}
                </span>
                <span className="tree-summary-label">{terminalTaskSummaryText}</span>
              </button>
            </div>
          )}

          {hasTerminalTaskSummary && showTerminalTaskChildren && (
            <div id={terminalTaskSummaryId}>
              {collapsedTerminalTaskChildren.map((child) => (
                <TreeNodeRow
                  key={child.id}
                  node={child}
                  depth={depth + 1}
                  selectionPathNodeIDs={selectionPathNodeIDs}
                  selectedProjectId={selectedProjectId}
                  selectedTaskId={selectedTaskId}
                  selectedRunId={selectedRunId}
                  onSelectProject={onSelectProject}
                  onSelectTask={onSelectTask}
                  onSelectRun={onSelectRun}
                  onCreateTask={onCreateTask}
                  defaultExpanded={child.type === 'project' || child.status === 'running'}
                  terminalTaskCollapse={undefined}
                  taskLabelPrefix="..."
                />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export function TreePanel({
  projectId,
  runsStreamState,
  selectedProjectId,
  selectedTaskId,
  selectedRunId,
  onSelectProject,
  onSelectTask,
  onSelectRun,
}: TreePanelProps) {
  const projectsQuery = useProjects()
  const tasksQuery = useTasks(projectId)
  const flatRunsQuery = useProjectRunsFlat(projectId, selectedTaskId, runsStreamState, SELECTED_TASK_RUN_LIMIT)
  const homeDirsQuery = useHomeDirs()
  const createProjectMutation = useCreateProject()
  const startTaskMutation = useStartTask(projectId)

  const [showCreate, setShowCreate] = useState(false)
  const [showCreateProject, setShowCreateProject] = useState(false)
  const [taskIdPrefix, setTaskIdPrefix] = useState(() => generateTaskIdPrefix())
  const [taskIdSuffix, setTaskIdSuffix] = useState(() => generateTaskSuffix())
  const [form, setForm] = useState<TaskStartRequest>(() => emptyForm(buildTaskId(taskIdPrefix, taskIdSuffix)))
  const [projectForm, setProjectForm] = useState<ProjectCreateRequest>(emptyProjectForm)
  const [dependsOnInput, setDependsOnInput] = useState('')
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [projectSubmitError, setProjectSubmitError] = useState<string | null>(null)
  const [submitFeedback, setSubmitFeedback] = useState<{ type: 'success' | 'error'; message: string } | null>(null)
  const [pendingCreatedTask, setPendingCreatedTask] = useState<{
    projectId: string
    taskId: string
    createdAt: string
    status: 'running' | 'queued'
    queuePosition?: number
  } | null>(null)
  const [terminalTaskCollapseByProject, setTerminalTaskCollapseByProject] = useState<Record<string, boolean>>({})

  const homeDirs = homeDirsQuery.data?.dirs ?? []

  useEffect(() => {
    if (!pendingCreatedTask || projectId !== pendingCreatedTask.projectId) {
      return
    }
    if (tasksQuery.data?.some((task) => task.id === pendingCreatedTask.taskId)) {
      setPendingCreatedTask(null)
    }
  }, [pendingCreatedTask, projectId, tasksQuery.data])

  const tree = useMemo(() => {
    if (!projectId) return null
    const tasks = tasksQuery.data ?? []
    const withPendingTask = pendingCreatedTask &&
      pendingCreatedTask.projectId === projectId &&
      !tasks.some((task) => task.id === pendingCreatedTask.taskId)
      ? [
          {
            id: pendingCreatedTask.taskId,
            project_id: projectId,
            status: pendingCreatedTask.status,
            queue_position: pendingCreatedTask.queuePosition,
            last_activity: pendingCreatedTask.createdAt,
            run_count: pendingCreatedTask.status === 'running' ? 1 : 0,
            run_counts: pendingCreatedTask.status === 'running' ? { running: 1 } : {},
          },
          ...tasks,
        ]
      : tasks
    const flatRuns = flatRunsQuery.data ?? []
    const visibleRuns = selectTreeRuns(withPendingTask, flatRuns, selectedTaskId)
    return buildTree(projectId, withPendingTask, visibleRuns)
  }, [projectId, tasksQuery.data, flatRunsQuery.data, pendingCreatedTask, selectedTaskId])
  const selectionPathNodeIDs = useMemo(() => {
    if (!tree) {
      return new Set<string>()
    }
    return buildSelectionPathNodeIDs(tree, selectedTaskId, selectedRunId)
  }, [selectedRunId, selectedTaskId, tree])
  const defaultTerminalTasksCollapsed = useMemo(() => {
    if (!tree || tree.type !== 'project') {
      return true
    }
    const rootTaskChildren = tree.children.filter((child) => child.type === 'task')
    if (rootTaskChildren.length === 0) {
      return true
    }
    const collapsibleTerminalCount = rootTaskChildren.filter((child) => shouldCollapseTerminalTaskNode(child)).length
    if (collapsibleTerminalCount === 0) {
      return true
    }
    // If every task is terminal/collapsible, expand them by default so tree content is visible.
    return collapsibleTerminalCount < rootTaskChildren.length
  }, [tree])

  const projects = projectsQuery.data ?? []
  const selectedProject = useMemo(
    () => projects.find((project) => project.id === projectId),
    [projects, projectId]
  )
  const derivedTaskId = useMemo(() => buildTaskId(taskIdPrefix, taskIdSuffix), [taskIdPrefix, taskIdSuffix])
  const terminalTasksCollapsed = projectId
    ? (terminalTaskCollapseByProject[projectId] ?? defaultTerminalTasksCollapsed)
    : true

  const toggleTerminalTasksCollapsed = () => {
    if (!projectId) {
      return
    }
    setTerminalTaskCollapseByProject((prev) => ({
      ...prev,
      [projectId]: !(prev[projectId] ?? true),
    }))
  }

  const openDialog = (targetProjectId?: string) => {
    if (targetProjectId && targetProjectId !== projectId) {
      onSelectProject(targetProjectId)
    }
    const nextPrefix = generateTaskIdPrefix()
    const nextSuffix = generateTaskSuffix()
    const nextTaskId = buildTaskId(nextPrefix, nextSuffix)
    const nextForm = emptyForm(nextTaskId)
    const projectForDialog = targetProjectId
      ? projects.find((project) => project.id === targetProjectId)
      : selectedProject
    if (projectForDialog?.project_root) {
      nextForm.project_root = projectForDialog.project_root
    }
    setTaskIdPrefix(nextPrefix)
    setTaskIdSuffix(nextSuffix)
    setForm(nextForm)
    setDependsOnInput('')
    setSubmitError(null)
    setSubmitFeedback(null)
    setShowCreate(true)
  }

  const closeDialog = () => {
    setShowCreate(false)
    setSubmitError(null)
  }

  const openProjectDialog = () => {
    setProjectForm({
      project_id: '',
      project_root: selectedProject?.project_root ?? homeDirs[0] ?? '',
    })
    setProjectSubmitError(null)
    setShowCreateProject(true)
  }

  const closeProjectDialog = () => {
    setShowCreateProject(false)
    setProjectSubmitError(null)
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSubmitError(null)
    if (!projectId) {
      const message = 'Select a project before creating a task'
      setSubmitError(message)
      setSubmitFeedback({ type: 'error', message })
      return
    }
    try {
      const dependsOn = dependsOnInput
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean)
      const created = await startTaskMutation.mutateAsync({
        ...form,
        task_id: derivedTaskId,
        depends_on: dependsOn.length > 0 ? dependsOn : undefined,
      })
      const createdTaskID = created.task_id || derivedTaskId
      const createdAt = new Date().toISOString()
      const createdStatus = created.status === 'queued' ? 'queued' : 'running'
      setPendingCreatedTask({
        projectId,
        taskId: createdTaskID,
        createdAt,
        status: createdStatus,
        queuePosition: created.queue_position,
      })
      onSelectTask(projectId, createdTaskID)
      setSubmitFeedback({
        type: 'success',
        message: created.status === 'queued'
          ? `Task ${createdTaskID} queued for root-task capacity.`
          : `Task ${createdTaskID} created and focused in the tree.`,
      })
      closeDialog()
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create task'
      setSubmitError(message)
      setSubmitFeedback({ type: 'error', message: `Task creation failed: ${message}` })
    }
  }

  const handleProjectSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setProjectSubmitError(null)

    const projectID = projectForm.project_id.trim()
    const projectRoot = projectForm.project_root.trim()
    if (!projectID) {
      setProjectSubmitError('Project ID is required')
      return
    }
    if (!projectRoot) {
      setProjectSubmitError('Home/work folder path is required')
      return
    }
    if (projects.some((project) => project.id.toLowerCase() === projectID.toLowerCase())) {
      setProjectSubmitError(`Project "${projectID}" already exists`)
      return
    }
    if (!isLikelyProjectRoot(projectRoot)) {
      setProjectSubmitError('Folder path must be absolute (or use ~/...)')
      return
    }
    if (projectRoot.includes('\u0000')) {
      setProjectSubmitError('Folder path contains invalid characters')
      return
    }

    try {
      const createdProject = await createProjectMutation.mutateAsync({
        project_id: projectID,
        project_root: projectRoot,
      })
      onSelectProject(createdProject.id)
      setSubmitFeedback({
        type: 'success',
        message: `Project ${createdProject.id} created and selected.`,
      })
      closeProjectDialog()
    } catch (err) {
      setProjectSubmitError(err instanceof Error ? err.message : 'Failed to create project')
    }
  }

  return (
    <div className="panel panel-scroll tree-panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Tree</div>
          <div className="panel-subtitle">Project · Task · Run</div>
        </div>
        <div className="panel-actions">
          <Button
            inline
            onClick={openProjectDialog}
            aria-label="Create new project"
            title="Create new project"
          >
            + New Project
          </Button>
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
      </div>
      {submitFeedback && (
        <div
          className={clsx('submit-feedback', submitFeedback.type === 'error' && 'submit-feedback-error')}
          role={submitFeedback.type === 'error' ? 'alert' : 'status'}
          aria-live="polite"
          data-testid="create-task-feedback"
        >
          {submitFeedback.message}
        </div>
      )}

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
            selectionPathNodeIDs={selectionPathNodeIDs}
            selectedProjectId={selectedProjectId}
            selectedTaskId={selectedTaskId}
            selectedRunId={selectedRunId}
            onSelectProject={onSelectProject}
            onSelectTask={onSelectTask}
            onSelectRun={onSelectRun}
            onCreateTask={openDialog}
            defaultExpanded={true}
            terminalTaskCollapse={
              projectId
                ? {
                    collapsed: terminalTasksCollapsed,
                    onToggle: toggleTerminalTasksCollapsed,
                  }
                : undefined
            }
          />
        )}
      </div>

      <Dialog
        show={showCreateProject}
        label="Create new project"
        showCloseButton
        onCloseAttempt={closeProjectDialog}
      >
        <div className="dialog-content">
          <div className="dialog-title">New Project</div>
          <form onSubmit={handleProjectSubmit}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span className="form-label">Project ID / Name *</span>
                <input
                  className="input"
                  style={{ width: '100%' }}
                  value={projectForm.project_id}
                  onChange={(event) => setProjectForm((formState) => ({ ...formState, project_id: event.target.value }))}
                  placeholder="my-project"
                  required
                />
                <span className="form-hint">
                  Unique identifier used in API paths and storage directories.
                </span>
              </label>

              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <span className="form-label">Home / Work Folder *</span>
                <input
                  className="input"
                  style={{ width: '100%' }}
                  list="home-dir-suggestions"
                  value={projectForm.project_root}
                  onChange={(event) => setProjectForm((formState) => ({ ...formState, project_root: event.target.value }))}
                  placeholder="~/Work/my-project  or  /absolute/path/to/project"
                  autoComplete="off"
                  required
                />
                <span className="form-hint">
                  Absolute path where newly created tasks for this project should run.
                </span>
              </label>

              {projectSubmitError && (
                <div className="form-error">{projectSubmitError}</div>
              )}

              <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end', marginTop: '4px' }}>
                <Button inline onClick={closeProjectDialog} type="button">
                  Cancel
                </Button>
                <Button
                  primary
                  type="submit"
                  disabled={createProjectMutation.isPending}
                >
                  {createProjectMutation.isPending ? 'Creating…' : 'Create Project'}
                </Button>
              </div>
            </div>
          </form>
        </div>
      </Dialog>

      <Dialog
        show={showCreate}
        label="Create new task"
        showCloseButton
        onCloseAttempt={closeDialog}
      >
        <div className="dialog-content dialog-content-wide">
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
                  className="input new-task-prompt"
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
                onChange={(event) => setDependsOnInput(event.target.value)}
                placeholder="task-a, task-b"
              />
              <span className="form-hint">Comma-separated task IDs that must complete first.</span>
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
