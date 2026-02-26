import { useEffect, useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import clsx from 'clsx'
import type { FileContent, RunInfo, TaskDetail } from '../types'
import { FileViewer } from './FileViewer'
import { RunTree } from './RunTree'

const COMPLETED_RUNS_INITIAL_LIMIT = 200
const COMPLETED_RUNS_LOAD_STEP = 200

type DetailTab = 'metadata' | 'files' | 'tree'

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

function runActivityTimestamp(run: { start_time: string; end_time?: string }): number {
  const end = Date.parse(run.end_time ?? '')
  if (!Number.isNaN(end)) {
    return end
  }
  const start = Date.parse(run.start_time)
  return Number.isNaN(start) ? 0 : start
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
  onStopRun?: (runId: string) => void
  onResumeTask?: (taskId: string) => void
}) {
  const [showCompletedRuns, setShowCompletedRuns] = useState(false)
  const [completedRunsLimit, setCompletedRunsLimit] = useState(COMPLETED_RUNS_INITIAL_LIMIT)
  const [detailTab, setDetailTab] = useState<DetailTab>('files')
  const taskRuns = useMemo(() => (
    Array.isArray(task?.runs) ? task.runs : []
  ), [task?.runs])

  const completedRuns = useMemo(() => {
    return taskRuns.filter((r) => r.status === 'completed')
  }, [taskRuns])

  const nonCompletedRuns = useMemo(() => {
    return taskRuns.filter((r) => r.status !== 'completed')
  }, [taskRuns])

  const completedRunsNewestFirst = useMemo(() => {
    const sorted = [...completedRuns]
    sorted.sort((a, b) => {
      const byTime = runActivityTimestamp(b) - runActivityTimestamp(a)
      if (byTime !== 0) {
        return byTime
      }
      return b.id.localeCompare(a.id)
    })
    return sorted
  }, [completedRuns])

  const selectedCompletedRun = useMemo(() => {
    if (!selectedRunId) {
      return undefined
    }
    return completedRuns.find((run) => run.id === selectedRunId)
  }, [completedRuns, selectedRunId])

  const visibleCompletedRuns = useMemo(() => {
    if (!showCompletedRuns) {
      return []
    }
    if (completedRunsNewestFirst.length <= completedRunsLimit) {
      return completedRunsNewestFirst
    }
    const capped = completedRunsNewestFirst.slice(0, completedRunsLimit)
    if (selectedCompletedRun && !capped.some((run) => run.id === selectedCompletedRun.id)) {
      return [...capped, selectedCompletedRun]
    }
    return capped
  }, [completedRunsLimit, completedRunsNewestFirst, selectedCompletedRun, showCompletedRuns])

  const visibleCompletedCount = useMemo(() => {
    return new Set(visibleCompletedRuns.map((run) => run.id)).size
  }, [visibleCompletedRuns])

  const hiddenCompletedRuns = useMemo(() => {
    return Math.max(0, completedRuns.length - visibleCompletedCount)
  }, [completedRuns.length, visibleCompletedCount])

  const visibleRuns = useMemo(() => {
    if (!showCompletedRuns) {
      return nonCompletedRuns
    }
    return [...nonCompletedRuns, ...visibleCompletedRuns]
  }, [nonCompletedRuns, showCompletedRuns, visibleCompletedRuns])

  const runMix = useMemo(() => {
    if (taskRuns.length === 0) {
      return 'No runs yet'
    }
    const counts: Record<string, number> = {}
    taskRuns.forEach((run) => {
      counts[run.status] = (counts[run.status] ?? 0) + 1
    })

    const parts = ['running', 'failed', 'completed', 'stopped', 'unknown']
      .filter((status) => (counts[status] ?? 0) > 0)
      .map((status) => `${counts[status]} ${status}`)
    return parts.join(' | ')
  }, [taskRuns])

  useEffect(() => {
    setShowCompletedRuns(false)
    setCompletedRunsLimit(COMPLETED_RUNS_INITIAL_LIMIT)
    setDetailTab('files')
  }, [task?.id])

  useEffect(() => {
    if (!selectedRunId) {
      return
    }
    const selectedRun = taskRuns.find((run) => run.id === selectedRunId)
    if (selectedRun?.status === 'completed') {
      setShowCompletedRuns(true)
    }
  }, [selectedRunId, taskRuns])

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
    if (task.status === 'queued') {
      const queueSuffix = task.queue_position && task.queue_position > 0
        ? ` (#${task.queue_position})`
        : ''
      return {
        state: 'task-restart-hint-open',
        title: `Queued for root slot${queueSuffix}`,
        detail: 'Task is waiting for root-task scheduler capacity and will start automatically when a slot is free.',
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

  const selectedRunFiles = useMemo(
    () => taskRuns.find((r) => r.id === selectedRunId)?.files ?? [],
    [taskRuns, selectedRunId]
  )

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
        {task.status !== 'running' && onResumeTask && task.done && (
          <div className="panel-actions">
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
            <div className="metadata-value">{taskRuns.length}</div>
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

      <div className="detail-tabs" role="tablist" aria-label="Detail sections">
        <Button
          inline
          className={clsx('filter-button detail-tab-button', detailTab === 'metadata' && 'filter-button-active')}
          onClick={() => setDetailTab('metadata')}
          role="tab"
          aria-selected={detailTab === 'metadata'}
        >
          Run metadata
        </Button>
        <Button
          inline
          className={clsx('filter-button detail-tab-button', detailTab === 'files' && 'filter-button-active')}
          onClick={() => setDetailTab('files')}
          role="tab"
          aria-selected={detailTab === 'files'}
        >
          Files
        </Button>
        <Button
          inline
          className={clsx('filter-button detail-tab-button', detailTab === 'tree' && 'filter-button-active')}
          onClick={() => setDetailTab('tree')}
          role="tab"
          aria-selected={detailTab === 'tree'}
        >
          Run tree
        </Button>
      </div>

      <div className="detail-tab-content">
        {detailTab === 'metadata' && (
          <div className="panel-section panel-section-tight">
            {taskState && <div className="task-state">{taskState}</div>}
            {runInfo && task?.status === 'running' && onStopRun && (
              <div className="panel-actions" style={{ marginBottom: '8px' }}>
                <Button
                  inline
                  danger
                  onClick={() => {
                    if (window.confirm(`Stop agent for run ${runInfo.run_id}?`)) {
                      onStopRun(runInfo.run_id)
                    }
                  }}
                >
                  Stop agent
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
        )}

        {detailTab === 'files' && (
          <>
            {selectedRunFiles.length > 0 && (
              <div className="detail-tab-files-header">
                {selectedRunFiles.map((option) => (
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
            )}
            <div className="panel-section panel-section-tight panel-section-files">
              <FileViewer fileName={fileName} content={fileContent?.content ?? ''} />
            </div>
          </>
        )}

        {detailTab === 'tree' && (
          <div className="panel-section panel-section-tight">
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
                {showCompletedRuns && (
                  <div className="runs-completed-hint">
                    {hiddenCompletedRuns > 0
                      ? `Showing latest ${visibleCompletedCount} of ${completedRuns.length} completed runs.`
                      : `Showing ${completedRuns.length} completed runs.`}
                  </div>
                )}
                {showCompletedRuns && hiddenCompletedRuns > 0 && (
                  <button
                    type="button"
                    className="runs-completed-toggle"
                    onClick={() => setCompletedRunsLimit((value) => value + COMPLETED_RUNS_LOAD_STEP)}
                  >
                    {`Load older completed (+${Math.min(COMPLETED_RUNS_LOAD_STEP, hiddenCompletedRuns)})`}
                  </button>
                )}
              </div>
            )}
            {visibleRuns.length > 0 ? (
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
            )}
          </div>
        )}
      </div>
    </div>
  )
}
