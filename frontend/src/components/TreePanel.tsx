import { useMemo, useState } from 'react'
import clsx from 'clsx'
import { useProjects, useProjectRunsFlat, useTasks } from '../hooks/useAPI'
import { buildTree, type TreeNode } from '../utils/treeBuilder'

interface TreePanelProps {
  projectId: string | undefined
  selectedProjectId: string | undefined
  selectedTaskId: string | undefined
  selectedRunId: string | undefined
  onSelectProject: (id: string) => void
  onSelectTask: (projectId: string, taskId: string) => void
  onSelectRun: (projectId: string, taskId: string, runId: string) => void
}

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
  defaultExpanded = true,
}: TreeNodeProps) {
  const [expanded, setExpanded] = useState(
    defaultExpanded || node.status === 'running'
  )

  const isSelected =
    (node.type === 'project' && selectedProjectId === node.id && !selectedTaskId && !selectedRunId) ||
    (node.type === 'task' && selectedTaskId === node.id && !selectedRunId) ||
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

  return (
    <div className="tree-node">
      <button
        type="button"
        className={clsx('tree-row', isSelected && 'tree-row-active')}
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
            {node.restartCount != null && node.restartCount > 0 && (
              <span className="tree-badge tree-badge-restart">×{node.restartCount + 1}</span>
            )}
          </>
        )}

        {node.type === 'run' && (
          <>
            <span className={clsx('tree-status-dot', statusClass(node.status))}>
              {statusDot(node.status)}
            </span>
            <span className={clsx('tree-label tree-label-run', node.status === 'running' && 'tree-label-active')}>
              [{node.agent ?? '?'}]
            </span>
            <span className="tree-time">{formatTime(node.startTime)}</span>
          </>
        )}
      </button>

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
              defaultExpanded={child.status === 'running' || depth < 1}
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

  const tree = useMemo(() => {
    if (!projectId) return null
    const tasks = tasksQuery.data ?? []
    const flatRuns = flatRunsQuery.data ?? []
    return buildTree(projectId, tasks, flatRuns)
  }, [projectId, tasksQuery.data, flatRunsQuery.data])

  const projects = projectsQuery.data ?? []

  return (
    <div className="panel panel-scroll tree-panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Tree</div>
          <div className="panel-subtitle">Project · Task · Run</div>
        </div>
      </div>

      {/* Project selector */}
      {projects.length > 1 && (
        <div className="panel-section">
          <div className="list">
            {projects.map((project) => (
              <button
                key={project.id}
                type="button"
                className={clsx('list-item list-item-compact', selectedProjectId === project.id && 'list-item-active')}
                onClick={() => onSelectProject(project.id)}
              >
                <span className="list-item-title">{project.id}</span>
              </button>
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
            defaultExpanded={true}
          />
        )}
      </div>
    </div>
  )
}
