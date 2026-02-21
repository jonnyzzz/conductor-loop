import { useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import { TreePanel } from './components/TreePanel'
import { RunDetail } from './components/RunDetail'
import { LogViewer } from './components/LogViewer'
import { MessageBus } from './components/MessageBus'
import { useDeleteRun, useDeleteTask, useProjects, useResumeTask, useRunFile, useRunInfo, useStopRun, useTask, useTaskFile, useTasks } from './hooks/useAPI'

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
  const taskStateQuery = useTaskFile(effectiveProjectId, effectiveTaskId, 'TASK.md')
  const runFileQuery = useRunFile(effectiveProjectId, effectiveTaskId, effectiveRunId, runFileName, 5000)
  const deleteRunMutation = useDeleteRun(effectiveProjectId, effectiveTaskId)
  const deleteTaskMutation = useDeleteTask(effectiveProjectId)
  const stopRunMutation = useStopRun(effectiveProjectId, effectiveTaskId)
  const resumeTaskMutation = useResumeTask(effectiveProjectId)

  const logStreamUrl = effectiveProjectId && effectiveTaskId
    ? `/api/projects/${effectiveProjectId}/tasks/${effectiveTaskId}/runs/stream`
    : undefined

  const busStreamUrl = useMemo(() => {
    if (!effectiveProjectId) {
      return undefined
    }
    if (busScope === 'project') {
      return `/api/projects/${effectiveProjectId}/messages/stream`
    }
    if (!effectiveTaskId) {
      return undefined
    }
    return `/api/projects/${effectiveProjectId}/tasks/${effectiveTaskId}/messages/stream`
  }, [busScope, effectiveProjectId, effectiveTaskId])

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="app-brand">
          <img src="/logo.svg" className="app-logo" alt="Conductor Loop Logo" />
          <div>
            <div className="app-title">Conductor Loop Monitor</div>
            <div className="app-subtitle">Multi-agent orchestration dashboard</div>
          </div>
        </div>
        <div className="app-header-actions">
          <Button inline onClick={() => projectsQuery.refetch()}>Refresh all</Button>
        </div>
      </header>

      <main className="app-grid">
        <section className="app-panel">
          <TreePanel
            projectId={effectiveProjectId}
            selectedProjectId={effectiveProjectId}
            selectedTaskId={selectedTaskId}
            selectedRunId={selectedRunId}
            onSelectProject={(pid) => {
              setSelectedProjectId(pid)
              setSelectedTaskId(undefined)
              setSelectedRunId(undefined)
              setRunFileName(defaultRunFile)
            }}
            onSelectTask={(pid, tid) => {
              setSelectedProjectId(pid)
              setSelectedTaskId(tid)
              setSelectedRunId(undefined)
              setRunFileName(defaultRunFile)
            }}
            onSelectRun={(pid, tid, rid) => {
              setSelectedProjectId(pid)
              setSelectedTaskId(tid)
              setSelectedRunId(rid)
              setRunFileName(defaultRunFile)
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
          <MessageBus
            streamUrl={busStreamUrl}
            title={`${busScope} bus`}
            projectId={effectiveProjectId}
            taskId={effectiveTaskId}
            scope={busScope}
          />
        </section>

        <section className="app-panel app-panel-wide">
          <RunDetail
            task={taskQuery.data}
            runInfo={runInfoQuery.data}
            selectedRunId={effectiveRunId}
            onSelectRun={(runId) => {
              const run = taskQuery.data?.runs?.find((r) => r.id === runId)
              const firstFile = run?.files?.[0]?.name ?? defaultRunFile
              setSelectedRunId(runId)
              setRunFileName(firstFile)
            }}
            fileName={runFileName}
            onSelectFile={setRunFileName}
            fileContent={runFileQuery.data}
            taskState={taskStateQuery.data?.content}
            onDeleteRun={(runId) => {
              deleteRunMutation.mutate(runId, {
                onSuccess: () => {
                  setSelectedRunId(undefined)
                  setRunFileName(defaultRunFile)
                },
              })
            }}
            onDeleteTask={(taskId) => {
              deleteTaskMutation.mutate(taskId, {
                onSuccess: () => {
                  setSelectedTaskId(undefined)
                  setSelectedRunId(undefined)
                  setRunFileName(defaultRunFile)
                },
              })
            }}
            onStopRun={(runId) => {
              stopRunMutation.mutate(runId)
            }}
            onResumeTask={(taskId) => {
              resumeTaskMutation.mutate(taskId)
            }}
          />
        </section>

        <section className="app-panel app-panel-full">
          <LogViewer streamUrl={logStreamUrl} />
        </section>
      </main>
    </div>
  )
}
