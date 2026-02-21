import { useEffect, useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import clsx from 'clsx'
import type { FileContent, RunInfo, TaskDetail } from '../types'
import { FileViewer } from './FileViewer'
import { RunTree } from './RunTree'

function formatDateTime(value?: string): string {
  const raw = (value ?? '').trim()
  if (!raw || raw === '0001-01-01T00:00:00Z') {
    return '—'
  }
  const parsed = new Date(raw)
  if (Number.isNaN(parsed.getTime())) {
    return '—'
  }
  return parsed.toLocaleString()
}

export function RunDetail({
  task,
  runInfo,
  selectedRunId,
  onSelectRun,
  fileName,
  onSelectFile,
  fileContent,
  taskState,
  onDeleteRun,
  onDeleteTask,
  onStopRun,
  onResumeTask,
}: {
  task?: TaskDetail
  runInfo?: RunInfo
  selectedRunId?: string
  onSelectRun: (runId: string) => void
  fileName: string
  onSelectFile: (name: string) => void
  fileContent?: FileContent
  taskState?: string
  onDeleteRun?: (runId: string) => void
  onDeleteTask?: (taskId: string) => void
  onStopRun?: (runId: string) => void
  onResumeTask?: (taskId: string) => void
}) {
  const [showCompletedRuns, setShowCompletedRuns] = useState(false)

  const completedRuns = useMemo(() => {
    if (!task?.runs) return []
    return task.runs.filter((r) => r.status === 'completed')
  }, [task?.runs])

  const visibleRuns = useMemo(() => {
    if (!task?.runs) return []
    if (showCompletedRuns) return task.runs
    return task.runs.filter((r) => r.status !== 'completed')
  }, [task?.runs, showCompletedRuns])

  const runMix = useMemo(() => {
    if (!task?.runs?.length) {
      return 'No runs yet'
    }
    const counts: Record<string, number> = {}
    task.runs.forEach((run) => {
      counts[run.status] = (counts[run.status] ?? 0) + 1
    })

    const parts = ['running', 'failed', 'completed', 'stopped', 'unknown']
      .filter((status) => (counts[status] ?? 0) > 0)
      .map((status) => `${counts[status]} ${status}`)
    return parts.join(' | ')
  }, [task?.runs])

  useEffect(() => {
    setShowCompletedRuns(false)
  }, [task?.id])

  useEffect(() => {
    if (!task?.runs || !selectedRunId) {
      return
    }
    const selectedRun = task.runs.find((run) => run.id === selectedRunId)
    if (selectedRun?.status === 'completed') {
      setShowCompletedRuns(true)
    }
  }, [task?.runs, selectedRunId])

  const restartHint = useMemo(() => {
    if (!task) {
      return null
    }
    if (task.status === 'blocked') {
      const blockedBy = task.blocked_by && task.blocked_by.length > 0
        ? task.blocked_by.join(', ')
        : 'dependencies'
      return {
        state: 'task-restart-hint-open',
        title: 'Blocked by dependencies',
        detail: `Task is waiting for: ${blockedBy}.`,
      }
    }
    if (task.done) {
      return {
        state: 'task-restart-hint-done',
        title: 'Restart disabled',
        detail: 'DONE marker is present. Failed and ended runs stay final until you click Resume task.',
      }
    }
    if (task.status === 'running') {
      return {
        state: 'task-restart-hint-running',
        title: 'Auto-restart active',
        detail: 'Task loop is active. Failed runs can restart automatically until a DONE marker is created.',
      }
    }
    return {
      state: 'task-restart-hint-open',
      title: 'Restart possible',
      detail: 'Task loop is stopped now, but DONE marker is missing so this task can be resumed and run again.',
    }
  }, [task])

  const runStatus = useMemo(() => {
    if (!runInfo) {
      return task?.status ?? 'unknown'
    }
    if (runInfo.exit_code === 0) {
      return 'completed'
    }
    if (runInfo.exit_code > 0) {
      return 'failed'
    }
    return task?.status ?? 'unknown'
  }, [runInfo, task?.status])

  if (!task) {
    return (
      <div className="panel">
        <div className="panel-header">
          <div>
            <div className="panel-title">Run detail</div>
            <div className="panel-subtitle">Select a task</div>
          </div>
        </div>
        <div className="panel-section panel-section-tight">
          <div className="section-title">Task details</div>
          <div className="empty-state">Select a task from the tree to inspect metadata and run files.</div>
          <div className="panel-subtitle">Project message bus remains visible below with live updates.</div>
        </div>
      </div>
    )
  }

  return (
    <div className="panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Run detail</div>
          <div className="panel-subtitle">{`${task.project_id} / ${task.id}`}</div>
        </div>
        {task.status !== 'running' && (onDeleteTask || (onResumeTask && task.done)) && (
          <div className="panel-actions">
            {onResumeTask && task.done && (
              <Button
                inline
                onClick={() => {
                  if (window.confirm(`Resume task ${task.id}? This will clear the DONE marker.`)) {
                    onResumeTask(task.id)
                  }
                }}
              >
                Resume task
              </Button>
            )}
            {onDeleteTask && (
              <Button
                inline
                danger
                onClick={() => {
                  if (window.confirm(`Delete task ${task.id} and all its runs? This cannot be undone.`)) {
                    onDeleteTask(task.id)
                  }
                }}
              >
                Delete task
              </Button>
            )}
          </div>
        )}
      </div>
      <div className="panel-section panel-section-tight task-overview">
        <div className="section-title">Task at a glance</div>
        <div className="task-overview-grid">
          <div className="task-overview-item">
            <div className="metadata-label">Task status</div>
            <div className="metadata-value">
              <span className={clsx('status-pill', `status-${task.status}`)}>{task.status}</span>
            </div>
          </div>
          <div className="task-overview-item">
            <div className="metadata-label">Restart policy</div>
            <div className="task-overview-restart-value">{restartHint?.title ?? 'Unknown'}</div>
            {restartHint && <div className="task-overview-note">{restartHint.detail}</div>}
          </div>
          <div className="task-overview-item">
            <div className="metadata-label">Runs</div>
            <div className="metadata-value">{task.runs.length}</div>
            <div className="task-overview-note">{runMix}</div>
          </div>
          <div className="task-overview-item">
            <div className="metadata-label">Last activity</div>
            <div className="metadata-value">{formatDateTime(task.last_activity)}</div>
          </div>
          {runInfo && (
            <>
              <div className="task-overview-item">
                <div className="metadata-label">Selected run</div>
                <div className="metadata-value">{runInfo.run_id}</div>
              </div>
              <div className="task-overview-item">
                <div className="metadata-label">Agent</div>
                <div className="metadata-value">{runInfo.agent}</div>
              </div>
              <div className="task-overview-item">
                <div className="metadata-label">Run start</div>
                <div className="metadata-value">{formatDateTime(runInfo.start_time)}</div>
              </div>
            </>
          )}
        </div>
        {restartHint && (
          <div className={clsx('task-restart-hint', restartHint.state)}>
            <span className="task-restart-title">{restartHint.title}</span>
            <span>{restartHint.detail}</span>
          </div>
        )}
      </div>
      <div className="panel-section panel-split">
        <div className="panel-column">
          <div className="section-title">Selected run metadata</div>
          {taskState && <div className="task-state">{taskState}</div>}
          {runInfo && task?.status === 'running' && onStopRun && (
            <div className="panel-actions" style={{ marginBottom: '8px' }}>
              <Button
                inline
                danger
                onClick={() => {
                  if (window.confirm(`Stop run ${runInfo.run_id}?`)) {
                    onStopRun(runInfo.run_id)
                  }
                }}
              >
                Stop run
              </Button>
            </div>
          )}
          {runInfo && task?.status !== 'running' && onDeleteRun && (
            <div className="panel-actions" style={{ marginBottom: '8px' }}>
              <Button
                inline
                danger
                onClick={() => {
                  if (window.confirm(`Delete run ${runInfo.run_id}? This cannot be undone.`)) {
                    onDeleteRun(runInfo.run_id)
                  }
                }}
              >
                Delete run
              </Button>
            </div>
          )}
          {runInfo ? (
            <div className="metadata-grid">
              <div>
                <div className="metadata-label">Run ID</div>
                <div className="metadata-value">{runInfo.run_id}</div>
              </div>
              <div>
                <div className="metadata-label">Agent</div>
                <div className="metadata-value">{runInfo.agent}</div>
              </div>
              {runInfo.agent_version && (
                <div>
                  <div className="metadata-label">Agent version</div>
                  <div className="metadata-value">{runInfo.agent_version}</div>
                </div>
              )}
              <div>
                <div className="metadata-label">Status</div>
                <div className="metadata-value">
                  <span className={clsx('status-pill', `status-${runStatus}`)}>
                    {runStatus}
                  </span>
                </div>
              </div>
              <div>
                <div className="metadata-label">Exit code</div>
                <div className="metadata-value">{runInfo.exit_code}</div>
              </div>
              <div>
                <div className="metadata-label">Start</div>
                <div className="metadata-value">{formatDateTime(runInfo.start_time)}</div>
              </div>
              <div>
                <div className="metadata-label">End</div>
                <div className="metadata-value">{formatDateTime(runInfo.end_time)}</div>
              </div>
              <div>
                <div className="metadata-label">Parent</div>
                <div className="metadata-value">{runInfo.parent_run_id || '—'}</div>
              </div>
              <div>
                <div className="metadata-label">Previous</div>
                <div className="metadata-value">{runInfo.previous_run_id || '—'}</div>
              </div>
              <div className="metadata-span">
                <div className="metadata-label">Working dir</div>
                <div className="metadata-value">{runInfo.cwd}</div>
              </div>
              {runInfo.error_summary && (
                <div className="metadata-span">
                  <div className="metadata-label">Error summary</div>
                  <div className="metadata-value metadata-error">{runInfo.error_summary}</div>
                </div>
              )}
            </div>
          ) : (
            <div className="empty-state">Select a run to see details.</div>
          )}
        </div>
        <div className="panel-column">
          <div className="section-title">Run tree</div>
          {completedRuns.length > 0 && (
            <div className="runs-completed-controls">
              <button
                type="button"
                className="runs-completed-toggle"
                onClick={() => setShowCompletedRuns((value) => !value)}
              >
                {showCompletedRuns ? `Hide ${completedRuns.length} completed` : `... ${completedRuns.length} completed`}
              </button>
              {!showCompletedRuns && (
                <div className="runs-completed-hint">Archived history is hidden. Click to include completed runs.</div>
              )}
            </div>
          )}
          {task ? (
            visibleRuns.length > 0 ? (
              <RunTree
                runs={visibleRuns}
                selectedRunId={selectedRunId}
                onSelect={onSelectRun}
                restartHint={restartHint}
              />
            ) : (
              <div className="empty-state">
                No running or failed runs. Expand completed runs to inspect history.
                {restartHint ? ` Restart policy: ${restartHint.title}.` : ''}
              </div>
            )
          ) : (
            <div className="empty-state">No task loaded.</div>
          )}
        </div>
      </div>
      <div className="panel-divider" />
      <div className="panel-header">
        <div>
          <div className="panel-title">Run files</div>
          <div className="panel-subtitle">Default view: output.md</div>
        </div>
        <div className="panel-actions">
          {(task?.runs?.find((r) => r.id === selectedRunId)?.files ?? []).map((option) => (
            <Button
              key={option.name}
              inline
              className={clsx('filter-button', fileName === option.name && 'filter-button-active')}
              onClick={() => onSelectFile(option.name)}
            >
              {option.label}
            </Button>
          ))}
        </div>
      </div>
      <div className="panel-section panel-section-tight">
        <FileViewer fileName={fileName} content={fileContent?.content ?? ''} />
      </div>
    </div>
  )
}
