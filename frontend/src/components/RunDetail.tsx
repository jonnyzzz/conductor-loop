import { useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import clsx from 'clsx'
import type { FileContent, RunInfo, RunStatus, TaskDetail } from '../types'
import { RunTree } from './RunTree'

const runStatusFilters = ['all', 'running', 'completed', 'failed'] as const
type RunStatusFilter = (typeof runStatusFilters)[number]

const fileOptions = [
  { label: 'output.md', name: 'output.md' },
  { label: 'stdout', name: 'agent-stdout.txt' },
  { label: 'stderr', name: 'agent-stderr.txt' },
  { label: 'prompt', name: 'prompt.md' },
]

export function RunDetail({
  task,
  runInfo,
  selectedRunId,
  onSelectRun,
  fileName,
  onSelectFile,
  fileContent,
  taskState,
}: {
  task?: TaskDetail
  runInfo?: RunInfo
  selectedRunId?: string
  onSelectRun: (runId: string) => void
  fileName: string
  onSelectFile: (name: string) => void
  fileContent?: FileContent
  taskState?: string
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
          <div className="panel-subtitle">{task?.id ?? 'Select a task'}</div>
        </div>
      </div>
      <div className="panel-section panel-split">
        <div className="panel-column">
          <div className="section-title">Metadata</div>
          {taskState && <div className="task-state">{taskState}</div>}
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
          {fileOptions.map((option) => (
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
        <pre className="file-viewer">{fileContent?.content ?? 'No file loaded yet.'}</pre>
      </div>
    </div>
  )
}
