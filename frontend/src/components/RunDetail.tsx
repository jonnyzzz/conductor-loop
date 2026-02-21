import { useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import clsx from 'clsx'
import type { FileContent, RunInfo, RunStatus, TaskDetail } from '../types'
import { FileViewer } from './FileViewer'
import { RunTree } from './RunTree'

const runStatusFilters = ['all', 'running', 'completed', 'failed'] as const
type RunStatusFilter = (typeof runStatusFilters)[number]

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
  const [runFilter, setRunFilter] = useState<RunStatusFilter>('all')

  const filteredRuns = useMemo(() => {
    if (!task?.runs) return []
    if (runFilter === 'all') return task.runs
    if (runFilter === 'failed') return task.runs.filter((r) => r.status === 'failed' || (r.status as RunStatus) === 'stopped')
    return task.runs.filter((r) => r.status === runFilter)
  }, [task?.runs, runFilter])

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
                  <span className={clsx('status-pill', `status-${task?.status ?? 'unknown'}`)}>
                    {task?.status ?? 'unknown'}
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
          <div className="filters">
            {runStatusFilters.map((f) => (
              <Button
                key={f}
                inline
                className={clsx('filter-button', runFilter === f && 'filter-button-active')}
                onClick={() => setRunFilter(f)}
              >
                {f}
              </Button>
            ))}
          </div>
          {task ? (
            <RunTree runs={filteredRuns} selectedRunId={selectedRunId} onSelect={onSelectRun} />
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
