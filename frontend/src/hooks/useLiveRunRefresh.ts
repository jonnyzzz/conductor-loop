import { useCallback, useEffect, useMemo, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { FlatRunItem, RunInfo, TaskDetail, TaskSummary } from '../types'
import { useSSE, type SSEConnectionState } from './useSSE'

// Legacy constant retained for test compatibility; log events no longer
// invalidate selected run-file queries directly.
export const LOG_REFRESH_DELAY_MS = 0
export const STATUS_REFRESH_DELAY_MS = 90

type Selection = {
  projectId?: string
  taskId?: string
  runId?: string
}

type UseLiveRunRefreshResult = {
  state: SSEConnectionState
  errorCount: number
}

type RunEventRef = {
  runId: string
  projectId: string
  taskId: string
  status: string
  exitCode: number | null
}

type StatusRefreshPlan = {
  refreshRunsFlat: boolean
  refreshTaskList: boolean
  refreshTask: boolean
  refreshRun: boolean
}

const noStatusRefresh: StatusRefreshPlan = {
  refreshRunsFlat: false,
  refreshTaskList: false,
  refreshTask: false,
  refreshRun: false,
}

function runsFlatProjectQueryKey(projectId: string | undefined) {
  return ['runs-flat', projectId] as const
}

function parseRunEventRef(event: MessageEvent): RunEventRef {
  try {
    const payload = JSON.parse(event.data) as {
      run_id?: unknown
      project_id?: unknown
      task_id?: unknown
      status?: unknown
      exit_code?: unknown
    }
    return {
      runId: typeof payload.run_id === 'string' ? payload.run_id : '',
      projectId: typeof payload.project_id === 'string' ? payload.project_id : '',
      taskId: typeof payload.task_id === 'string' ? payload.task_id : '',
      status: typeof payload.status === 'string' ? payload.status : '',
      exitCode: typeof payload.exit_code === 'number' ? payload.exit_code : null,
    }
  } catch {
    return { runId: '', projectId: '', taskId: '', status: '', exitCode: null }
  }
}

function hasRealEndTime(endTime: string | undefined): boolean {
  if (!endTime) {
    return false
  }
  const trimmed = endTime.trim()
  if (!trimmed || trimmed === '0001-01-01T00:00:00Z') {
    return false
  }
  return true
}

function shouldSetEndTime(status: string, endTime: string | undefined): boolean {
  if (!status) {
    return false
  }
  if (status === 'running' || status === 'queued') {
    return false
  }
  return !hasRealEndTime(endTime)
}

function patchFlatRunsStatus(runs: FlatRunItem[] | undefined, eventRef: RunEventRef): FlatRunItem[] | undefined {
  if (!runs || runs.length === 0 || !eventRef.runId) {
    return runs
  }
  const index = runs.findIndex((run) => run.id === eventRef.runId)
  if (index < 0) {
    return runs
  }
  const current = runs[index]
  const nextStatus = eventRef.status || current.status
  const nextExitCode = eventRef.exitCode ?? current.exit_code
  const nextEndTime = shouldSetEndTime(nextStatus, current.end_time)
    ? new Date().toISOString()
    : current.end_time
  if (
    nextStatus === current.status &&
    nextExitCode === current.exit_code &&
    nextEndTime === current.end_time
  ) {
    return runs
  }
  const next = [...runs]
  next[index] = {
    ...current,
    status: nextStatus as FlatRunItem['status'],
    exit_code: nextExitCode,
    end_time: nextEndTime,
  }
  return next
}

function deriveTaskStatusFromRuns(runs: TaskDetail['runs'], fallback: TaskDetail['status']): TaskDetail['status'] {
  if (!runs || runs.length === 0) {
    return fallback
  }
  for (const run of runs) {
    if (run.status === 'running') {
      return 'running'
    }
  }
  for (const run of runs) {
    if (run.status === 'queued') {
      return 'queued'
    }
  }
  let latest = runs[0]
  let latestTime = latest.end_time || latest.start_time
  for (const run of runs) {
    const runTime = run.end_time || run.start_time
    if (runTime > latestTime) {
      latest = run
      latestTime = runTime
    }
  }
  return latest.status
}

function patchTaskStatus(task: TaskDetail | undefined, eventRef: RunEventRef): TaskDetail | undefined {
  if (!task || !eventRef.runId || !task.runs || task.runs.length === 0) {
    return task
  }

  const now = new Date().toISOString()
  let changed = false
  const nextRuns = task.runs.map((run) => {
    if (run.id !== eventRef.runId) {
      return run
    }
    const nextStatus = eventRef.status || run.status
    const nextExitCode = eventRef.exitCode ?? run.exit_code
    const nextEndTime = shouldSetEndTime(nextStatus, run.end_time)
      ? now
      : run.end_time
    if (
      nextStatus === run.status &&
      nextExitCode === run.exit_code &&
      nextEndTime === run.end_time
    ) {
      return run
    }
    changed = true
    return {
      ...run,
      status: nextStatus as typeof run.status,
      exit_code: nextExitCode,
      end_time: nextEndTime,
    }
  })
  if (!changed) {
    return task
  }

  const nextTaskStatus = deriveTaskStatusFromRuns(nextRuns, task.status)
  return {
    ...task,
    status: nextTaskStatus,
    last_activity: now,
    runs: nextRuns,
  }
}

function patchRunInfoStatus(runInfo: RunInfo | undefined, eventRef: RunEventRef): RunInfo | undefined {
  if (!runInfo || !eventRef.runId || runInfo.run_id !== eventRef.runId) {
    return runInfo
  }
  const nextExitCode = eventRef.exitCode ?? runInfo.exit_code
  const nextEndTime = shouldSetEndTime(eventRef.status, runInfo.end_time)
    ? new Date().toISOString()
    : runInfo.end_time
  if (nextExitCode === runInfo.exit_code && nextEndTime === runInfo.end_time) {
    return runInfo
  }
  return {
    ...runInfo,
    exit_code: nextExitCode,
    end_time: nextEndTime,
  }
}

function findRunByID(runs: FlatRunItem[] | undefined, runId: string): FlatRunItem | undefined {
  if (!runs || runs.length === 0 || !runId) {
    return undefined
  }
  return runs.find((run) => run.id === runId)
}

function hasKnownTask(tasks: TaskSummary[] | undefined, taskId: string): boolean {
  if (!tasks || tasks.length === 0 || !taskId) {
    return false
  }
  return tasks.some((task) => task.id === taskId)
}

function resolveStatusRefreshPlan(
  selection: Selection,
  eventRef: RunEventRef,
  knownRuns: FlatRunItem[] | undefined,
  knownTasks: TaskSummary[] | undefined
): StatusRefreshPlan {
  const projectId = selection.projectId
  if (!projectId) {
    return noStatusRefresh
  }
  if (eventRef.projectId && eventRef.projectId !== projectId) {
    return noStatusRefresh
  }

  const knownRun = findRunByID(knownRuns, eventRef.runId)
  const refreshRunsFlat = (
    !eventRef.runId ||
    !knownRuns ||
    knownRuns.length === 0 ||
    !knownRun
  )
  const eventTaskID = eventRef.taskId || knownRun?.task_id || ''
  const refreshTaskList = Boolean(
    refreshRunsFlat &&
    (
      !eventTaskID ||
      !hasKnownTask(knownTasks, eventTaskID)
    )
  )

  const selectedTaskId = selection.taskId
  const selectedRunId = selection.runId
  if (!selectedTaskId) {
    return { refreshRunsFlat, refreshTaskList, refreshTask: false, refreshRun: false }
  }

  if (!eventRef.runId && !eventRef.taskId) {
    return {
      refreshRunsFlat,
      refreshTaskList,
      refreshTask: refreshRunsFlat,
      refreshRun: Boolean(refreshRunsFlat && selectedRunId),
    }
  }

  let refreshTask = false
  if (selectedRunId && eventRef.runId && eventRef.runId === selectedRunId) {
    refreshTask = true
  }
  if (eventTaskID && eventTaskID === selectedTaskId) {
    refreshTask = true
  } else if (!refreshTask && refreshRunsFlat) {
    // Unknown/new runs can alter selected-task state; reconcile selected task once.
    refreshTask = true
  }

  const refreshRun = Boolean(
    selectedRunId &&
      (
        eventRef.runId === selectedRunId ||
        (!eventRef.runId && refreshRunsFlat)
      )
  )

  return { refreshRunsFlat, refreshTaskList, refreshTask, refreshRun }
}

export function useLiveRunRefresh(selection: Selection): UseLiveRunRefreshResult {
  const queryClient = useQueryClient()
  const selectionRef = useRef(selection)

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const dueAtRef = useRef(0)
  const pendingRunsFlatRefreshRef = useRef(false)
  const pendingTaskListRefreshRef = useRef(false)
  const pendingTaskRefreshRef = useRef(false)
  const pendingRunRefreshRef = useRef(false)

  const invalidateRunsFlatQuery = useCallback(() => {
    const { projectId } = selectionRef.current
    if (!projectId) {
      return
    }
    queryClient.invalidateQueries({ queryKey: runsFlatProjectQueryKey(projectId) })
  }, [queryClient])

  const invalidateTaskListQuery = useCallback(() => {
    const { projectId } = selectionRef.current
    if (!projectId) {
      return
    }
    queryClient.invalidateQueries({ queryKey: ['tasks', projectId] })
  }, [queryClient])

  const invalidateTaskQuery = useCallback(() => {
    const { projectId, taskId } = selectionRef.current
    if (!projectId || !taskId) {
      return
    }

    queryClient.invalidateQueries({ queryKey: ['task', projectId, taskId] })
  }, [queryClient])

  const invalidateRunQuery = useCallback(() => {
    const { projectId, taskId, runId } = selectionRef.current
    if (!projectId || !taskId || !runId) {
      return
    }
    queryClient.invalidateQueries({ queryKey: ['run', projectId, taskId, runId] })
  }, [queryClient])

  const patchStatusCaches = useCallback((eventRef: RunEventRef) => {
    const { projectId, taskId, runId } = selectionRef.current
    if (!projectId || !eventRef.runId) {
      return
    }
    if (eventRef.projectId && eventRef.projectId !== projectId) {
      return
    }

    const runsFlatQueryKey = runsFlatProjectQueryKey(projectId)
    const knownRuns = queryClient.getQueryData<FlatRunItem[]>(runsFlatQueryKey)

    queryClient.setQueriesData<FlatRunItem[] | undefined>(
      { queryKey: runsFlatQueryKey },
      (current) => patchFlatRunsStatus(current, eventRef)
    )

    const runBelongsToSelectedTask = Boolean(
      taskId &&
      findRunByID(knownRuns, eventRef.runId)?.task_id === taskId
    )
    const mayAffectSelectedTask = Boolean(
      taskId &&
      (
        (eventRef.taskId && eventRef.taskId === taskId) ||
        (runId && eventRef.runId === runId) ||
        runBelongsToSelectedTask
      )
    )
    if (taskId && mayAffectSelectedTask) {
      queryClient.setQueryData<TaskDetail | undefined>(
        ['task', projectId, taskId],
        (current) => patchTaskStatus(current, eventRef)
      )
    }

    if (taskId && runId && eventRef.runId === runId) {
      queryClient.setQueryData<RunInfo | undefined>(
        ['run', projectId, taskId, runId],
        (current) => patchRunInfoStatus(current, eventRef)
      )
    }
  }, [queryClient])

  const flushInvalidate = useCallback(() => {
    timerRef.current = null
    dueAtRef.current = 0
    if (pendingRunsFlatRefreshRef.current) {
      invalidateRunsFlatQuery()
    }
    if (pendingTaskListRefreshRef.current) {
      invalidateTaskListQuery()
    }
    if (pendingTaskRefreshRef.current) {
      invalidateTaskQuery()
    }
    if (pendingRunRefreshRef.current) {
      invalidateRunQuery()
    }
    pendingRunsFlatRefreshRef.current = false
    pendingTaskListRefreshRef.current = false
    pendingTaskRefreshRef.current = false
    pendingRunRefreshRef.current = false
  }, [invalidateRunsFlatQuery, invalidateRunQuery, invalidateTaskListQuery, invalidateTaskQuery])

  const scheduleInvalidate = useCallback((delayMs: number) => {
    const dueAt = Date.now() + delayMs
    if (timerRef.current === null) {
      dueAtRef.current = dueAt
      timerRef.current = setTimeout(flushInvalidate, delayMs)
      return
    }
    if (dueAt >= dueAtRef.current) {
      return
    }
    clearTimeout(timerRef.current)
    dueAtRef.current = dueAt
    timerRef.current = setTimeout(flushInvalidate, delayMs)
  }, [flushInvalidate])

  const queueStatusRefresh = useCallback((plan: StatusRefreshPlan) => {
    if (!plan.refreshRunsFlat && !plan.refreshTaskList && !plan.refreshTask && !plan.refreshRun) {
      return
    }
    if (plan.refreshRunsFlat) {
      pendingRunsFlatRefreshRef.current = true
    }
    if (plan.refreshTaskList) {
      pendingTaskListRefreshRef.current = true
    }
    if (plan.refreshTask) {
      pendingTaskRefreshRef.current = true
    }
    if (plan.refreshRun) {
      pendingRunRefreshRef.current = true
    }
    scheduleInvalidate(STATUS_REFRESH_DELAY_MS)
  }, [scheduleInvalidate])

  const maybeRefreshForUnknownRunLog = useCallback((eventRef: RunEventRef) => {
    const { projectId } = selectionRef.current
    if (!projectId || !eventRef.runId || !eventRef.projectId) {
      return
    }
    if (eventRef.projectId !== projectId) {
      return
    }

    const runsFlatQueryKey = runsFlatProjectQueryKey(projectId)
    const knownRuns = queryClient.getQueryData<FlatRunItem[]>(runsFlatQueryKey)
    const knownTasks = queryClient.getQueryData<TaskSummary[]>(['tasks', projectId])
    const plan = resolveStatusRefreshPlan(selectionRef.current, eventRef, knownRuns, knownTasks)
    if (!plan.refreshRunsFlat) {
      return
    }
    queueStatusRefresh(plan)
  }, [queryClient, queueStatusRefresh])

  const sseHandlers = useMemo(
    () => ({
      status: (event: MessageEvent) => {
        const eventRef = parseRunEventRef(event)
        const projectId = selectionRef.current.projectId
        if (!projectId) {
          return
        }
        patchStatusCaches(eventRef)
        const knownRuns = queryClient.getQueryData<FlatRunItem[]>(runsFlatProjectQueryKey(projectId))
        const knownTasks = queryClient.getQueryData<TaskSummary[]>(['tasks', projectId])
        const plan = resolveStatusRefreshPlan(selectionRef.current, eventRef, knownRuns, knownTasks)
        queueStatusRefresh(plan)
      },
      log: (event: MessageEvent) => {
        const eventRef = parseRunEventRef(event)
        maybeRefreshForUnknownRunLog(eventRef)
      },
    }),
    [maybeRefreshForUnknownRunLog, patchStatusCaches, queryClient, queueStatusRefresh]
  )

  useEffect(() => {
    selectionRef.current = selection
  }, [selection])

  useEffect(() => {
    return () => {
      if (timerRef.current !== null) {
        clearTimeout(timerRef.current)
        timerRef.current = null
      }
      pendingRunsFlatRefreshRef.current = false
      pendingTaskListRefreshRef.current = false
      pendingTaskRefreshRef.current = false
      pendingRunRefreshRef.current = false
    }
  }, [])

  return useSSE(selection.projectId ? '/api/v1/runs/stream/all' : undefined, {
    events: sseHandlers,
  })
}
