import { useProjectStats } from '../hooks/useAPI'

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function ProjectStats({ projectId }: { projectId: string }) {
  const { data, isLoading, isError } = useProjectStats(projectId)

  if (isLoading) {
    return (
      <div className="project-stats-bar">
        <span className="project-stats-loading">Loading stats…</span>
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="project-stats-bar">
        <span className="project-stats-unavailable">Stats unavailable</span>
      </div>
    )
  }

  return (
    <div className="project-stats-bar">
      <span className="project-stats-item">
        <span className="project-stats-label">Tasks</span>
        <span className="project-stats-value">{data.total_tasks}</span>
      </span>
      <span className="project-stats-sep" />
      <span className="project-stats-item">
        <span className="project-stats-label">Runs</span>
        <span className="project-stats-value">{data.total_runs}</span>
      </span>
      <span className="project-stats-sep" />
      {data.running_runs > 0 && (
        <>
          <span className="project-stats-item">
            <span className="project-stats-label">Running</span>
            <span className="project-stats-value project-stats-running">{data.running_runs}</span>
          </span>
          <span className="project-stats-sep" />
        </>
      )}
      <span className="project-stats-item">
        <span className="project-stats-label">Done</span>
        <span className="project-stats-value project-stats-completed">{data.completed_runs}</span>
      </span>
      {(data.failed_runs > 0 || data.crashed_runs > 0) && (
        <>
          <span className="project-stats-sep" />
          <span className="project-stats-item">
            <span className="project-stats-label">Failed</span>
            <span className="project-stats-value project-stats-failed">
              {data.failed_runs + data.crashed_runs}
            </span>
          </span>
        </>
      )}
      <span className="project-stats-sep" />
      <span className="project-stats-item">
        <span className="project-stats-label">Bus</span>
        <span className="project-stats-value">{formatBytes(data.message_bus_total_bytes)}</span>
      </span>
    </div>
  )
}
