import { useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import { TaskList } from './components/TaskList'
import { RunDetail } from './components/RunDetail'
import { LogViewer } from './components/LogViewer'
import { MessageBus } from './components/MessageBus'
import { useProjects, useRunFile, useRunInfo, useTask, useTaskFile, useTasks } from './hooks/useAPI'

const defaultRunFile = 'output.md'

export function App() {
  const [selectedProjectId, setSelectedProjectId] = useState<string | undefined>(undefined)
  const [selectedTaskId, setSelectedTaskId] = useState<string | undefined>(undefined)
  const [selectedRunId, setSelectedRunId] = useState<string | undefined>(undefined)
  const [runFileName, setRunFileName] = useState(defaultRunFile)
  const [busScope, setBusScope] = useState<'project' | 'task'>('task')

  const projectsQuery = useProjects()
  const effectiveProjectId = selectedProjectId ?? projectsQuery.data?.[0]?.id

  const tasksQuery = useTasks(effectiveProjectId)
  const effectiveTaskId = selectedTaskId ?? tasksQuery.data?.[0]?.id

  const taskQuery = useTask(effectiveProjectId, effectiveTaskId)
  const effectiveRunId = selectedRunId ?? taskQuery.data?.runs?.[taskQuery.data.runs.length - 1]?.id

  const runInfoQuery = useRunInfo(effectiveProjectId, effectiveTaskId, effectiveRunId)
  const taskStateQuery = useTaskFile(effectiveProjectId, effectiveTaskId, 'TASK_STATE.md')
  const runFileQuery = useRunFile(effectiveProjectId, effectiveTaskId, effectiveRunId, runFileName, 5000)

  const logStreamUrl = useMemo(() => {
    if (!effectiveProjectId || !effectiveTaskId) {
      return undefined
    }
    return `/api/projects/${effectiveProjectId}/tasks/${effectiveTaskId}/logs/stream`
  }, [effectiveProjectId, effectiveTaskId])

  const busStreamUrl = useMemo(() => {
    if (!effectiveProjectId) {
      return undefined
    }
    if (busScope === 'project') {
      return `/api/projects/${effectiveProjectId}/bus/stream`
    }
    if (!effectiveTaskId) {
      return undefined
    }
    return `/api/projects/${effectiveProjectId}/tasks/${effectiveTaskId}/bus/stream`
  }, [busScope, effectiveProjectId, effectiveTaskId])

  return (
    <div className="app-shell">
      <header className="app-header">
        <div>
          <div className="app-title">Conductor Loop Monitor</div>
          <div className="app-subtitle">Multi-agent orchestration dashboard</div>
        </div>
        <div className="app-header-actions">
          <Button inline onClick={() => projectsQuery.refetch()}>Refresh all</Button>
        </div>
      </header>

      <main className="app-grid">
        <section className="app-panel">
          <TaskList
            projects={projectsQuery.data ?? []}
            tasks={tasksQuery.data ?? []}
            selectedProjectId={effectiveProjectId}
            selectedTaskId={effectiveTaskId}
            onSelectProject={(projectId) => {
              setSelectedProjectId(projectId)
              setSelectedTaskId(undefined)
              setSelectedRunId(undefined)
              setRunFileName(defaultRunFile)
            }}
            onSelectTask={(taskId) => {
              setSelectedTaskId(taskId)
              setSelectedRunId(undefined)
              setRunFileName(defaultRunFile)
            }}
            onRefresh={() => {
              projectsQuery.refetch()
              if (effectiveProjectId) {
                tasksQuery.refetch()
              }
            }}
          />
        </section>

        <section className="app-panel">
          <div className="panel">
            <div className="panel-header">
              <div>
                <div className="panel-title">Message bus</div>
                <div className="panel-subtitle">Project + task feeds</div>
              </div>
              <div className="panel-actions">
                <Button
                  inline
                  className={busScope === 'task' ? 'filter-button-active' : undefined}
                  onClick={() => setBusScope('task')}
                >
                  Task
                </Button>
                <Button
                  inline
                  className={busScope === 'project' ? 'filter-button-active' : undefined}
                  onClick={() => setBusScope('project')}
                >
                  Project
                </Button>
              </div>
            </div>
          </div>
          <MessageBus streamUrl={busStreamUrl} title={`${busScope} bus`} />
        </section>

        <section className="app-panel app-panel-wide">
          <RunDetail
            task={taskQuery.data}
            runInfo={runInfoQuery.data}
            selectedRunId={effectiveRunId}
            onSelectRun={(runId) => {
              setSelectedRunId(runId)
              setRunFileName(defaultRunFile)
            }}
            fileName={runFileName}
            onSelectFile={setRunFileName}
            fileContent={runFileQuery.data}
            taskState={taskStateQuery.data?.content}
          />
        </section>

        <section className="app-panel app-panel-full">
          <LogViewer streamUrl={logStreamUrl} />
        </section>
      </main>
    </div>
  )
}
