import { useEffect, useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import clsx from 'clsx'
import type { FileContent, RunInfo, TaskDetail } from '../types'
import { FileViewer } from './FileViewer'
import { RunTree } from './RunTree'

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
    if (task.done) {
      return {
        state: 'task-restart-hint-done',
        title: 'Restart disabled',
        detail: 'DONE marker is present. This task will not restart until you click Resume task.',
      }
    }
    if (task.status === 'running') {
      return {
        state: 'task-restart-hint-running',
        title: 'Auto-restart active',
        detail: 'Ralph loop can keep restarting runs until a DONE marker is created.',
      }
    }
    return {
      state: 'task-restart-hint-open',
      title: 'Restart possible',
      detail: 'DONE marker is missing. This task can be resumed and run again.',
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

  return (
    <div className="panel">
      <div className="panel-header">
        <div>
          <div className="panel-title">Run detail</div>
          <div className="panel-subtitle">{task ? `${task.project_id} / ${task.id}` : 'Select a task'}</div>
        </div>
        {task && task.status !== 'running' && (onDeleteTask || (onResumeTask && task.done)) && (
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
      {task && (
        <div className="panel-section panel-section-tight task-overview">
          <div className="task-overview-grid">
            <div className="task-overview-item">
              <div className="metadata-label">Task status</div>
              <div className="metadata-value">
                <span className={clsx('status-pill', `status-${task.status}`)}>{task.status}</span>
              </div>
            </div>
            <div className="task-overview-item">
              <div className="metadata-label">Last activity</div>
              <div className="metadata-value">{new Date(task.last_activity).toLocaleString()}</div>
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
                  <div className="metadata-value">{new Date(runInfo.start_time).toLocaleString()}</div>
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
      )}
      <div className="panel-section panel-split">
        <div className="panel-column">
          <div className="section-title">Metadata</div>
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
                <div className="metadata-value">{new Date(runInfo.start_time).toLocaleString()}</div>
              </div>
              <div>
                <div className="metadata-label">End</div>
                <div className="metadata-value">{new Date(runInfo.end_time).toLocaleString()}</div>
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
            <button
              type="button"
              className="runs-completed-toggle"
              onClick={() => setShowCompletedRuns((value) => !value)}
            >
              {showCompletedRuns ? `Hide ${completedRuns.length} completed` : `... ${completedRuns.length} completed`}
            </button>
          )}
          {task ? (
            visibleRuns.length > 0 ? (
              <RunTree runs={visibleRuns} selectedRunId={selectedRunId} onSelect={onSelectRun} />
            ) : (
              <div className="empty-state">No running or failed runs. Expand completed runs to inspect history.</div>
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
