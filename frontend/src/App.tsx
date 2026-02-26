import { useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import { TreePanel } from './components/TreePanel'
import { RunDetail } from './components/RunDetail'
import { LogViewer } from './components/LogViewer'
import { MessageBus } from './components/MessageBus'
import { useProjects, useResumeTask, useRunFile, useRunInfo, useStopRun, useTask, useTaskFile, useVersion } from './hooks/useAPI'
import { useLiveRunRefresh } from './hooks/useLiveRunRefresh'

const defaultRunFile = 'output.md'
const defaultTaskSection = 'details'
const RUN_FILE_TAIL_LINES = 1500

type TaskSection = 'details' | 'messages' | 'logs'

export function App() {
  const [selectedProjectId, setSelectedProjectId] = useState<string | undefined>(undefined)
  const [selectedTaskId, setSelectedTaskId] = useState<string | undefined>(undefined)
  const [selectedRunId, setSelectedRunId] = useState<string | undefined>(undefined)
  const [runFileName, setRunFileName] = useState(defaultRunFile)
  const [busScope, setBusScope] = useState<'project' | 'task'>('task')
  const [taskSection, setTaskSection] = useState<TaskSection>(defaultTaskSection)
  const [focusedMessage, setFocusedMessage] = useState<{
    projectId: string
    taskId: string
    messageId: string
  } | null>(null)

  const projectsQuery = useProjects()
  const effectiveProjectId = selectedProjectId ?? projectsQuery.data?.[0]?.id

  const explicitTaskId = selectedTaskId
  const effectiveTaskId = explicitTaskId
  const effectiveBusScope: 'project' | 'task' = explicitTaskId ? busScope : 'project'
  const detailsSectionActive = taskSection === 'details'

  const taskQuery = useTask(
    detailsSectionActive ? effectiveProjectId : undefined,
    detailsSectionActive ? effectiveTaskId : undefined
  )
  const taskRuns = useMemo(() => (
    Array.isArray(taskQuery.data?.runs) ? taskQuery.data.runs : []
  ), [taskQuery.data])
  const effectiveRunId = selectedRunId ?? taskRuns[taskRuns.length - 1]?.id
  const effectiveRun = useMemo(
    () => taskRuns.find((run) => run.id === effectiveRunId),
    [effectiveRunId, taskRuns]
  )
  const effectiveRunFileName = useMemo(() => {
    if (!effectiveRun?.files || effectiveRun.files.length === 0) {
      return runFileName
    }
    const hasSelectedFile = effectiveRun.files.some((file) => file.name === runFileName)
    if (hasSelectedFile) {
      return runFileName
    }
    return effectiveRun.files[0].name
  }, [effectiveRun, runFileName])

  const runInfoQuery = useRunInfo(
    detailsSectionActive ? effectiveProjectId : undefined,
    detailsSectionActive ? effectiveTaskId : undefined,
    detailsSectionActive ? effectiveRunId : undefined
  )
  const taskStateQuery = useTaskFile(
    detailsSectionActive ? effectiveProjectId : undefined,
    detailsSectionActive ? effectiveTaskId : undefined,
    detailsSectionActive ? 'TASK.md' : undefined
  )
  const runFileQuery = useRunFile(
    detailsSectionActive ? effectiveProjectId : undefined,
    detailsSectionActive ? effectiveTaskId : undefined,
    detailsSectionActive ? effectiveRunId : undefined,
    detailsSectionActive ? effectiveRunFileName : undefined,
    RUN_FILE_TAIL_LINES,
    detailsSectionActive ? effectiveRun?.status : undefined
  )
  const stopRunMutation = useStopRun(effectiveProjectId, effectiveTaskId)
  const resumeTaskMutation = useResumeTask(effectiveProjectId)

  const liveRunRefresh = useLiveRunRefresh({
    projectId: effectiveProjectId,
    taskId: detailsSectionActive ? effectiveTaskId : undefined,
    runId: detailsSectionActive ? effectiveRunId : undefined,
  })

  const versionQuery = useVersion()

  const logStreamUrl = effectiveProjectId && explicitTaskId
    ? `/api/projects/${effectiveProjectId}/tasks/${effectiveTaskId}/runs/stream`
    : undefined

  const busStreamUrl = useMemo(() => {
    if (!effectiveProjectId) {
      return undefined
    }
    if (effectiveBusScope === 'project') {
      return `/api/projects/${effectiveProjectId}/messages/stream`
    }
    if (!explicitTaskId) {
      return undefined
    }
    return `/api/projects/${effectiveProjectId}/tasks/${explicitTaskId}/messages/stream`
  }, [effectiveBusScope, effectiveProjectId, explicitTaskId])

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
          {versionQuery.data?.version && (
            <span className="app-version" data-testid="app-version">v{versionQuery.data.version}</span>
          )}
          <a
            href="https://linkedin.com/in/jonnyzzz"
            target="_blank"
            rel="noopener noreferrer"
            className="app-author-link"
            title="Support · Donate · Follow @jonnyzzz"
          >
            @jonnyzzz
          </a>
          <Button inline onClick={() => projectsQuery.refetch()}>Refresh all</Button>
        </div>
      </header>

      <main className="app-grid">
        <section className="app-panel app-panel-tree">
          <TreePanel
            projectId={effectiveProjectId}
            runsStreamState={liveRunRefresh.state}
            selectedProjectId={effectiveProjectId}
            selectedTaskId={selectedTaskId}
            selectedRunId={selectedRunId}
            onSelectProject={(pid) => {
              setSelectedProjectId(pid)
              setSelectedTaskId(undefined)
              setSelectedRunId(undefined)
              setRunFileName(defaultRunFile)
              setTaskSection(defaultTaskSection)
              setFocusedMessage(null)
            }}
            onSelectTask={(pid, tid) => {
              setSelectedProjectId(pid)
              setSelectedTaskId(tid)
              setSelectedRunId(undefined)
              setRunFileName(defaultRunFile)
              setTaskSection(defaultTaskSection)
              setFocusedMessage(null)
            }}
            onSelectRun={(pid, tid, rid) => {
              setSelectedProjectId(pid)
              setSelectedTaskId(tid)
              setSelectedRunId(rid)
              setRunFileName(defaultRunFile)
              setTaskSection(defaultTaskSection)
              setFocusedMessage(null)
            }}
          />
        </section>

        <section className="app-panel app-panel-run">
          <div className="app-task-tabs" role="tablist" aria-label="Task sections">
            <Button
              inline
              className={`filter-button app-tab-button${taskSection === 'details' ? ' filter-button-active' : ''}`}
              onClick={() => setTaskSection('details')}
              role="tab"
              aria-selected={taskSection === 'details'}
            >
              Task details
            </Button>
            <Button
              inline
              className={`filter-button app-tab-button${taskSection === 'messages' ? ' filter-button-active' : ''}`}
              onClick={() => setTaskSection('messages')}
              role="tab"
              aria-selected={taskSection === 'messages'}
            >
              Message bus
            </Button>
            <Button
              inline
              className={`filter-button app-tab-button${taskSection === 'logs' ? ' filter-button-active' : ''}`}
              onClick={() => setTaskSection('logs')}
              role="tab"
              aria-selected={taskSection === 'logs'}
            >
              Live logs
            </Button>
          </div>
          <div className="app-task-tab-content">
            {taskSection === 'details' && (
              <RunDetail
                task={taskQuery.data}
                runInfo={runInfoQuery.data}
                selectedRunId={effectiveRunId}
                onSelectRun={(runId) => {
                  const run = taskRuns.find((r) => r.id === runId)
                  const firstFile = run?.files?.[0]?.name ?? defaultRunFile
                  setSelectedRunId(runId)
                  setRunFileName(firstFile)
                }}
                fileName={effectiveRunFileName}
                onSelectFile={setRunFileName}
                fileContent={runFileQuery.data}
                taskState={taskStateQuery.data?.content}
                onStopRun={(runId) => {
                  stopRunMutation.mutate(runId)
                }}
                onResumeTask={(taskId) => {
                  resumeTaskMutation.mutate(taskId)
                }}
              />
            )}
            {taskSection === 'messages' && (
              <MessageBus
                key={busStreamUrl ?? `bus-${effectiveBusScope}-none`}
                streamUrl={busStreamUrl}
                title={effectiveBusScope === 'project' ? 'Project message bus' : 'Task message bus'}
                projectId={effectiveProjectId}
                taskId={explicitTaskId}
                scope={effectiveBusScope}
                focusedMessageId={
                  focusedMessage &&
                  focusedMessage.projectId === effectiveProjectId &&
                  focusedMessage.taskId === explicitTaskId
                    ? focusedMessage.messageId
                    : undefined
                }
                onNavigateToTask={(pid, tid) => {
                  setSelectedProjectId(pid)
                  setSelectedTaskId(tid)
                  setSelectedRunId(undefined)
                  setRunFileName(defaultRunFile)
                  setTaskSection('messages')
                  setBusScope('task')
                  setFocusedMessage(null)
                }}
                onNavigateToMessage={(pid, tid, msgID) => {
                  setSelectedProjectId(pid)
                  setSelectedTaskId(tid)
                  setSelectedRunId(undefined)
                  setRunFileName(defaultRunFile)
                  setTaskSection('messages')
                  setBusScope('task')
                  setFocusedMessage({ projectId: pid, taskId: tid, messageId: msgID })
                }}
                headerActions={(
                  <>
                    <Button
                      inline
                      className={effectiveBusScope === 'task' ? 'filter-button-active' : undefined}
                      onClick={() => {
                        if (explicitTaskId) {
                          setBusScope('task')
                        }
                      }}
                      disabled={!explicitTaskId}
                    >
                      Task
                    </Button>
                    <Button
                      inline
                      className={effectiveBusScope === 'project' ? 'filter-button-active' : undefined}
                      onClick={() => setBusScope('project')}
                    >
                      Project
                    </Button>
                  </>
                )}
              />
            )}
            {taskSection === 'logs' && (
              <LogViewer key={logStreamUrl ?? 'logs-none'} streamUrl={logStreamUrl} />
            )}
          </div>
        </section>
      </main>
    </div>
  )
}
