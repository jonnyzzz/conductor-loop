import { useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import clsx from 'clsx'
import type { Project, TaskSummary } from '../types'

const statusFilters = ['all', 'running', 'completed', 'failed'] as const
export type StatusFilter = (typeof statusFilters)[number]

function parseDate(value?: string) {
  if (!value) {
    return 0
  }
  return new Date(value).getTime()
}

export function TaskList({
  projects,
  tasks,
  selectedProjectId,
  selectedTaskId,
  onSelectProject,
  onSelectTask,
  onRefresh,
}: {
  projects: Project[]
  tasks: TaskSummary[]
  selectedProjectId?: string
  selectedTaskId?: string
  onSelectProject: (projectId: string) => void
  onSelectTask: (taskId: string) => void
  onRefresh?: () => void
}) {
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')

  const filteredTasks = useMemo(() => {
    const filtered = statusFilter === 'all' ? tasks : tasks.filter((task) => task.status === statusFilter)
    return [...filtered].sort((a, b) => parseDate(b.last_activity) - parseDate(a.last_activity))
  }, [statusFilter, tasks])

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
          <div className="panel-subtitle">Filter by status</div>
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
              </div>
            </button>
          ))}
          {filteredTasks.length === 0 && <div className="empty-state">No tasks match the filter.</div>}
        </div>
      </div>
    </div>
  )
}
