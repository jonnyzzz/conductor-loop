/* eslint-disable react-refresh/only-export-components */
import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type {
  BusMessage,
  FileContent,
  FlatRunItem,
  Project,
  ProjectDetail,
  ProjectStats,
  RunInfo,
  TaskDetail,
  TaskSummary,
} from '../types'
import { APIClient, createClient } from '../api/client'
import type { ProjectCreateRequest, TaskStartRequest } from '../api/client'
import type { SSEConnectionState } from './useSSE'
import { mergeMessagesByID } from '../utils/messageStore'

const APIClientContext = createContext<APIClient | null>(null)
const RUNS_FLAT_EMPTY_REFETCH_MS = 1500
const RUNS_FLAT_ACTIVE_REFETCH_MS = 800
const RUNS_FLAT_IDLE_REFETCH_MS = 2500
const RUNS_FLAT_STREAM_SYNC_EMPTY_REFETCH_MS = 1000
const RUNS_FLAT_STREAM_SYNC_ACTIVE_REFETCH_MS = 810
const RUNS_FLAT_STREAM_SYNC_IDLE_REFETCH_MS = 3000
const RUNS_FLAT_DEFAULT_LIMIT_KEY_PART = 0
const RUN_FILE_ACTIVE_REFETCH_MS = 2500
export const MESSAGE_FALLBACK_REFETCH_MS = 3000
export const PROJECT_STATS_REFETCH_INTERVAL_MS = 10000

type MessageQueryOptions = {
  fallbackPollIntervalMs?: number | false
}

function flatRunItemsEqual(previous: FlatRunItem, incoming: FlatRunItem): boolean {
  return previous.id === incoming.id &&
    previous.task_id === incoming.task_id &&
    previous.agent === incoming.agent &&
    previous.status === incoming.status &&
    previous.exit_code === incoming.exit_code &&
    previous.start_time === incoming.start_time &&
    previous.end_time === incoming.end_time &&
    previous.parent_run_id === incoming.parent_run_id &&
    previous.previous_run_id === incoming.previous_run_id
}

export function stabilizeFlatRuns(
  previous: FlatRunItem[] | undefined,
  incoming: FlatRunItem[]
): FlatRunItem[] {
  if (!previous || previous.length !== incoming.length) {
    return incoming
  }
  for (let i = 0; i < incoming.length; i += 1) {
    const previousItem = previous[i]
    const incomingItem = incoming[i]
    if (previousItem === incomingItem) {
      continue
    }
    if (!flatRunItemsEqual(previousItem, incomingItem)) {
      return incoming
    }
  }
  return previous
}

function compareFlatRunsByStartTime(a: FlatRunItem, b: FlatRunItem): number {
  if (a.start_time === b.start_time) {
    if (a.id === b.id) {
      return 0
    }
    return a.id < b.id ? -1 : 1
  }
  return a.start_time < b.start_time ? -1 : 1
}

function flatRunLineageEdgeCount(run: FlatRunItem): number {
  let count = 0
  if (run.parent_run_id) {
    count += 1
  }
  if (run.previous_run_id) {
    count += 1
  }
  return count
}

function flatRunActivityTime(run: FlatRunItem): string {
  if (run.end_time && run.end_time > run.start_time) {
    return run.end_time
  }
  return run.start_time
}

function preferRicherAncestryRun(existing: FlatRunItem | undefined, candidate: FlatRunItem): FlatRunItem {
  if (!existing) {
    return candidate
  }

  const existingLineageEdges = flatRunLineageEdgeCount(existing)
  const candidateLineageEdges = flatRunLineageEdgeCount(candidate)
  if (candidateLineageEdges > existingLineageEdges) {
    return candidate
  }
  if (candidateLineageEdges < existingLineageEdges) {
    return existing
  }

  const existingActivity = flatRunActivityTime(existing)
  const candidateActivity = flatRunActivityTime(candidate)
  if (candidateActivity > existingActivity) {
    return candidate
  }
  if (candidateActivity < existingActivity) {
    return existing
  }
  if (candidate.start_time > existing.start_time) {
    return candidate
  }
  if (candidate.start_time < existing.start_time) {
    return existing
  }

  return existing
}

export function mergeFlatRunsForTree(
  previousProjectRuns: FlatRunItem[] | undefined,
  incomingRuns: FlatRunItem[]
): FlatRunItem[] {
  if (!previousProjectRuns || previousProjectRuns.length === 0) {
    return incomingRuns
  }
  if (incomingRuns.length === 0) {
    // Keep known ancestor context when scoped refreshes come back empty.
    return previousProjectRuns
  }

  const mergedRuns = [...previousProjectRuns]
  const indexByRunID = new Map<string, number>()
  for (let i = 0; i < previousProjectRuns.length; i += 1) {
    indexByRunID.set(previousProjectRuns[i].id, i)
  }

  let changed = false
  for (const incomingRun of incomingRuns) {
    const index = indexByRunID.get(incomingRun.id)
    if (index === undefined) {
      indexByRunID.set(incomingRun.id, mergedRuns.length)
      mergedRuns.push(incomingRun)
      changed = true
      continue
    }
    const currentRun = mergedRuns[index]
    if (!flatRunItemsEqual(currentRun, incomingRun)) {
      mergedRuns[index] = incomingRun
      changed = true
    }
  }

  if (!changed) {
    return previousProjectRuns
  }
  mergedRuns.sort(compareFlatRunsByStartTime)
  return mergedRuns
}

export function scopedRunsForTree(
  previousScopedRuns: FlatRunItem[] | undefined,
  incomingScopedRuns: FlatRunItem[],
  selectedTaskId: string | undefined,
  selectedTaskLimit?: number,
  previousProjectRuns?: FlatRunItem[]
): FlatRunItem[] {
  const stableIncomingScopedRuns = stabilizeFlatRuns(previousScopedRuns, incomingScopedRuns)
  if (!selectedTaskId) {
    // Keep root tree refreshes bounded to the active/scoped payload returned by API.
    return stableIncomingScopedRuns
  }
  if (selectedTaskLimit && selectedTaskLimit > 0) {
    return scopedRunsForSelectedTaskLimit(
      previousScopedRuns,
      stableIncomingScopedRuns,
      previousProjectRuns
    )
  }
  // Selected-task views may need temporary ancestry continuity across short polling races.
  const mergedScopedRuns = mergeFlatRunsForTree(previousScopedRuns, stableIncomingScopedRuns)
  return stabilizeFlatRuns(previousScopedRuns, mergedScopedRuns)
}

function scopedRunsForSelectedTaskLimit(
  previousScopedRuns: FlatRunItem[] | undefined,
  incomingScopedRuns: FlatRunItem[],
  previousProjectRuns?: FlatRunItem[]
): FlatRunItem[] {
  if (incomingScopedRuns.length === 0) {
    return previousScopedRuns ?? incomingScopedRuns
  }

  const ancestryByRunID = new Map<string, FlatRunItem>()
  const indexAncestryRuns = (runs: FlatRunItem[] | undefined) => {
    if (!runs || runs.length === 0) {
      return
    }
    for (const run of runs) {
      const existing = ancestryByRunID.get(run.id)
      const preferred = preferRicherAncestryRun(existing, run)
      if (preferred !== existing) {
        ancestryByRunID.set(run.id, preferred)
      }
    }
  }
  indexAncestryRuns(previousScopedRuns)
  indexAncestryRuns(previousProjectRuns)
  if (ancestryByRunID.size === 0) {
    return incomingScopedRuns
  }

  const nextRuns: FlatRunItem[] = [...incomingScopedRuns]
  const includedRunIDs = new Set(nextRuns.map((run) => run.id))

  const includeRun = (run: FlatRunItem | undefined) => {
    if (!run || includedRunIDs.has(run.id)) {
      return
    }
    includedRunIDs.add(run.id)
    nextRuns.push(run)
  }

  const includeParentChain = (run: FlatRunItem) => {
    let parentRunID = run.parent_run_id
    while (parentRunID && !includedRunIDs.has(parentRunID)) {
      const parent = ancestryByRunID.get(parentRunID)
      if (!parent) {
        break
      }
      includeRun(parent)
      parentRunID = parent.parent_run_id
    }
  }

  for (const run of incomingScopedRuns) {
    includeParentChain(run)
    if (run.previous_run_id && !includedRunIDs.has(run.previous_run_id)) {
      const previousRun = ancestryByRunID.get(run.previous_run_id)
      includeRun(previousRun)
      if (previousRun) {
        includeParentChain(previousRun)
      }
    }
  }

  if (nextRuns.length === incomingScopedRuns.length) {
    return incomingScopedRuns
  }
  nextRuns.sort(compareFlatRunsByStartTime)
  return stabilizeFlatRuns(previousScopedRuns, nextRuns)
}

export function stabilizeProjectStats(
  previous: ProjectStats | undefined,
  incoming: ProjectStats
): ProjectStats {
  if (!previous) {
    return incoming
  }
  if (
    previous.project_id === incoming.project_id &&
    previous.total_tasks === incoming.total_tasks &&
    previous.total_runs === incoming.total_runs &&
    previous.running_runs === incoming.running_runs &&
    previous.completed_runs === incoming.completed_runs &&
    previous.failed_runs === incoming.failed_runs &&
    previous.crashed_runs === incoming.crashed_runs &&
    previous.message_bus_files === incoming.message_bus_files &&
    previous.message_bus_total_bytes === incoming.message_bus_total_bytes
  ) {
    return previous
  }
  return incoming
}

export function runsFlatProjectQueryKey(projectId: string | undefined) {
  return ['runs-flat', projectId] as const
}

export function runsFlatScopedQueryKey(
  projectId: string | undefined,
  selectedTaskId: string | undefined,
  selectedTaskLimit?: number
) {
  const normalizedLimit = selectedTaskLimit && selectedTaskLimit > 0
    ? selectedTaskLimit
    : RUNS_FLAT_DEFAULT_LIMIT_KEY_PART
  return [...runsFlatProjectQueryKey(projectId), selectedTaskId ?? '', normalizedLimit] as const
}

export function APIProvider({
  children,
  client,
}: {
  children: React.ReactNode
  client?: APIClient
}) {
  const value = useMemo(() => client ?? createClient(), [client])
  return <APIClientContext.Provider value={value}>{children}</APIClientContext.Provider>
}

export function useAPIClient(): APIClient {
  const client = useContext(APIClientContext)
  if (!client) {
    throw new Error('API client not available; wrap with APIProvider')
  }
  return client
}

export function useProjects() {
  const api = useAPIClient()
  return useQuery<Project[]>({
    queryKey: ['projects'],
    queryFn: () => api.getProjects(),
    staleTime: 2000,
  })
}

export function useProject(projectId?: string) {
  const api = useAPIClient()
  return useQuery<ProjectDetail>({
    queryKey: ['project', projectId],
    queryFn: () => api.getProject(projectId ?? ''),
    enabled: Boolean(projectId),
    staleTime: 2000,
  })
}

export function useTasks(projectId?: string) {
  const api = useAPIClient()
  return useQuery<TaskSummary[]>({
    queryKey: ['tasks', projectId],
    queryFn: () => api.getTasks(projectId ?? ''),
    enabled: Boolean(projectId),
    staleTime: 2000,
  })
}

export function useTask(projectId?: string, taskId?: string) {
  const api = useAPIClient()
  return useQuery<TaskDetail>({
    queryKey: ['task', projectId, taskId],
    queryFn: () => api.getTask(projectId ?? '', taskId ?? ''),
    enabled: Boolean(projectId && taskId),
    staleTime: 2000,
  })
}

export function messageFallbackRefetchIntervalFor(
  streamState?: SSEConnectionState
): number | false {
  if (!streamState || streamState === 'open' || streamState === 'connecting') {
    return false
  }
  return MESSAGE_FALLBACK_REFETCH_MS
}

export function useProjectMessages(projectId?: string, options?: MessageQueryOptions) {
  const api = useAPIClient()
  const queryClient = useQueryClient()
  const queryKey = ['messages', 'project', projectId] as const
  return useQuery<BusMessage[]>({
    queryKey,
    queryFn: async () => {
      const cached = queryClient.getQueryData<BusMessage[]>(queryKey) ?? []
      const since = cached[0]?.msg_id
      const incoming = await api.getProjectMessages(projectId ?? '', since ? { since } : undefined)
      if (!since || incoming.length === 0) {
        return since ? cached : incoming
      }
      return mergeMessagesByID(cached, incoming)
    },
    enabled: Boolean(projectId),
    staleTime: 0,
    refetchOnMount: 'always',
    refetchInterval: options?.fallbackPollIntervalMs ?? false,
  })
}

export function useTaskMessages(
  projectId?: string,
  taskId?: string,
  options?: MessageQueryOptions
) {
  const api = useAPIClient()
  const queryClient = useQueryClient()
  const queryKey = ['messages', 'task', projectId, taskId] as const
  return useQuery<BusMessage[]>({
    queryKey,
    queryFn: async () => {
      const cached = queryClient.getQueryData<BusMessage[]>(queryKey) ?? []
      const since = cached[0]?.msg_id
      const incoming = await api.getTaskMessages(projectId ?? '', taskId ?? '', since ? { since } : undefined)
      if (!since || incoming.length === 0) {
        return since ? cached : incoming
      }
      return mergeMessagesByID(cached, incoming)
    },
    enabled: Boolean(projectId && taskId),
    staleTime: 0,
    refetchOnMount: 'always',
    refetchInterval: options?.fallbackPollIntervalMs ?? false,
  })
}

export function useRunInfo(projectId?: string, taskId?: string, runId?: string) {
  const api = useAPIClient()
  return useQuery<RunInfo>({
    queryKey: ['run', projectId, taskId, runId],
    queryFn: () => api.getRunInfo(projectId ?? '', taskId ?? '', runId ?? ''),
    enabled: Boolean(projectId && taskId && runId),
    staleTime: 2000,
  })
}

export function useTaskFile(projectId?: string, taskId?: string, name?: string) {
  const api = useAPIClient()
  return useQuery<FileContent>({
    queryKey: ['task-file', projectId, taskId, name],
    queryFn: () => api.getTaskFile(projectId ?? '', taskId ?? '', name ?? ''),
    enabled: Boolean(projectId && taskId && name),
    staleTime: 1000,
  })
}

function isActiveRunStatus(runStatus?: string): boolean {
  return runStatus === 'running' || runStatus === 'queued'
}

export function runFileRefetchIntervalFor(runStatus?: string, streamState?: SSEConnectionState): number | false {
  if (!isActiveRunStatus(runStatus)) {
    return false
  }
  // Active run-file streams should not be polled while stream is healthy.
  if (streamState === 'open' || streamState === 'connecting') {
    return false
  }
  if (streamState === 'reconnecting') {
    return RUN_FILE_ACTIVE_REFETCH_MS
  }
  if (streamState === 'error' || streamState === 'disabled' || streamState === undefined) {
    return RUN_FILE_ACTIVE_REFETCH_MS
  }
  return RUN_FILE_ACTIVE_REFETCH_MS
}

export function useRunFile(
  projectId?: string,
  taskId?: string,
  runId?: string,
  name?: string,
  tail?: number,
  runStatus?: string
) {
  const api = useAPIClient()
  const queryClient = useQueryClient()
  const queryKey = useMemo(
    () => ['run-file', projectId, taskId, runId, name, tail] as const,
    [name, projectId, runId, tail, taskId]
  )
  const [streamState, setStreamState] = useState<SSEConnectionState>('disabled')
  const replaceStreamChunkRef = useRef(true)

  const streamURL = useMemo(() => {
    if (!projectId || !taskId || !runId || !name || !isActiveRunStatus(runStatus)) {
      return undefined
    }
    const params = new URLSearchParams({ name })
    return (
      `/api/projects/${encodeURIComponent(projectId)}` +
      `/tasks/${encodeURIComponent(taskId)}` +
      `/runs/${encodeURIComponent(runId)}` +
      `/stream?${params.toString()}`
    )
  }, [name, projectId, runId, runStatus, taskId])

  useEffect(() => {
    replaceStreamChunkRef.current = true
  }, [streamURL])

  useEffect(() => {
    if (!streamURL || !name) {
      setStreamState('disabled')
      return undefined
    }

    setStreamState('connecting')
    const source = new EventSource(streamURL)
    let closed = false

    const handleOpen = () => {
      setStreamState('open')
    }

    const handleMessage = (event: Event) => {
      const messageEvent = event as MessageEvent
      const incomingChunk = typeof messageEvent.data === 'string' ? messageEvent.data : ''
      queryClient.setQueryData<FileContent | undefined>(queryKey, (current) => {
        const currentPayload = current as (FileContent & Record<string, unknown>) | undefined
        const currentContent = typeof currentPayload?.content === 'string' ? currentPayload.content : ''
        const nextContent = replaceStreamChunkRef.current
          ? incomingChunk
          : `${currentContent}${incomingChunk}`
        replaceStreamChunkRef.current = false
        return {
          ...currentPayload,
          name: currentPayload?.name ?? name,
          content: nextContent,
          modified: new Date().toISOString(),
          size_bytes: nextContent.length,
        }
      })
    }

    const handleDone = () => {
      close('disabled')
      queryClient.invalidateQueries({ queryKey })
    }

    const handleError = () => {
      // Close stream to avoid duplicate replay on auto-reconnect; fallback polling resumes.
      close('error')
    }

    function close(nextState: SSEConnectionState) {
      if (closed) {
        return
      }
      closed = true
      source.removeEventListener('open', handleOpen)
      source.removeEventListener('message', handleMessage)
      source.removeEventListener('done', handleDone)
      source.removeEventListener('error', handleError)
      source.close()
      setStreamState(nextState)
    }

    source.addEventListener('open', handleOpen)
    source.addEventListener('message', handleMessage)
    source.addEventListener('done', handleDone)
    source.addEventListener('error', handleError)

    return () => {
      close('disabled')
    }
  }, [name, queryClient, queryKey, streamURL])

  return useQuery<FileContent>({
    queryKey,
    queryFn: () => api.getRunFile(projectId ?? '', taskId ?? '', runId ?? '', name ?? '', tail),
    enabled: Boolean(projectId && taskId && runId && name),
    staleTime: 1000,
    refetchInterval: runFileRefetchIntervalFor(runStatus, streamState),
  })
}

export function useStartTask(projectId?: string) {
  const api = useAPIClient()
  const client = useQueryClient()
  return useMutation({
    mutationFn: (payload: TaskStartRequest) => api.startTask(projectId ?? '', payload),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['tasks', projectId] })
    },
  })
}

export function useCreateProject() {
  const api = useAPIClient()
  const client = useQueryClient()
  return useMutation({
    mutationFn: (payload: ProjectCreateRequest) => api.createProject(payload),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['projects'] })
      client.invalidateQueries({ queryKey: ['home-dirs'] })
    },
  })
}

export function useProjectStats(projectId?: string) {
  const api = useAPIClient()
  const queryClient = useQueryClient()
  const queryKey = ['project-stats', projectId] as const
  return useQuery<ProjectStats>({
    queryKey,
    queryFn: async () => {
      const incoming = await api.getProjectStats(projectId ?? '')
      const previous = queryClient.getQueryData<ProjectStats>(queryKey)
      return stabilizeProjectStats(previous, incoming)
    },
    enabled: Boolean(projectId),
    staleTime: 5000,
    refetchInterval: PROJECT_STATS_REFETCH_INTERVAL_MS,
  })
}

export function useStopRun(projectId?: string, taskId?: string) {
  const api = useAPIClient()
  const client = useQueryClient()
  return useMutation({
    mutationFn: (runId: string) => api.stopRun(projectId ?? '', taskId ?? '', runId),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['task', projectId, taskId] })
      client.invalidateQueries({ queryKey: ['tasks', projectId] })
    },
  })
}

export function usePostTaskMessage(projectId?: string, taskId?: string) {
  const api = useAPIClient()
  return useMutation({
    mutationFn: (payload: { type: string; body: string }) =>
      api.postTaskMessage(projectId ?? '', taskId ?? '', { type: payload.type, body: payload.body }),
  })
}

export function usePostProjectMessage(projectId?: string) {
  const api = useAPIClient()
  return useMutation({
    mutationFn: (payload: { type: string; body: string }) =>
      api.postProjectMessage(projectId ?? '', { type: payload.type, body: payload.body }),
  })
}

export function useResumeTask(projectId?: string) {
  const api = useAPIClient()
  const client = useQueryClient()
  return useMutation({
    mutationFn: (taskId: string) => api.resumeTask(projectId ?? '', taskId),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['tasks', projectId] })
    },
  })
}

export function useProjectRunsFlat(
  projectId: string | undefined,
  selectedTaskId: string | undefined,
  liveState?: SSEConnectionState,
  selectedTaskLimit?: number
) {
  const api = useAPIClient()
  const queryClient = useQueryClient()
  const normalizedSelectedTaskLimit = selectedTaskLimit && selectedTaskLimit > 0
    ? selectedTaskLimit
    : undefined
  const scopedQueryKey = runsFlatScopedQueryKey(projectId, selectedTaskId, normalizedSelectedTaskLimit)
  const projectQueryKey = runsFlatProjectQueryKey(projectId)
  return useQuery<FlatRunItem[]>({
    queryKey: scopedQueryKey,
    queryFn: async () => {
      const incomingRuns = await api.getProjectRunsFlat(projectId!, {
        activeOnly: true,
        selectedTaskId: selectedTaskId ?? undefined,
        selectedTaskLimit: normalizedSelectedTaskLimit,
      })
      const previousScopedRuns = queryClient.getQueryData<FlatRunItem[]>(scopedQueryKey)
      const previousProjectRuns = queryClient.getQueryData<FlatRunItem[]>(projectQueryKey)
      const treeScopedRuns = scopedRunsForTree(
        previousScopedRuns,
        incomingRuns,
        selectedTaskId,
        normalizedSelectedTaskLimit,
        previousProjectRuns
      )

      const stableProjectRuns = mergeFlatRunsForTree(previousProjectRuns, treeScopedRuns)
      if (stableProjectRuns !== previousProjectRuns) {
        queryClient.setQueryData(projectQueryKey, stableProjectRuns)
      }

      return treeScopedRuns
    },
    enabled: Boolean(projectId),
    staleTime: 1000,
    // Keep the previous scoped data as a placeholder while the new query (different
    // selectedTaskId) is fetching.  Without this, the query key change causes a brief
    // empty-data window that makes the tree lose all run-based nesting and jump.
    placeholderData: keepPreviousData,
    refetchInterval: (query) => {
      const runs = query.state.data as FlatRunItem[] | undefined
      return runsFlatRefetchIntervalFor(runs, liveState)
    },
    notifyOnChangeProps: ['data', 'error'],
  })
}

export function runsFlatRefetchIntervalFor(
  runs: FlatRunItem[] | undefined,
  liveState?: SSEConnectionState
): number {
  const isStreamHealthy = liveState === 'open'
  if (!runs || runs.length === 0) {
    return isStreamHealthy ? RUNS_FLAT_STREAM_SYNC_EMPTY_REFETCH_MS : RUNS_FLAT_EMPTY_REFETCH_MS
  }

  const hasActiveRun = runs.some((run) => run.status === 'running' || run.status === 'queued')
  if (isStreamHealthy) {
    return hasActiveRun
      ? RUNS_FLAT_STREAM_SYNC_ACTIVE_REFETCH_MS
      : RUNS_FLAT_STREAM_SYNC_IDLE_REFETCH_MS
  }
  return hasActiveRun ? RUNS_FLAT_ACTIVE_REFETCH_MS : RUNS_FLAT_IDLE_REFETCH_MS
}

export function useHomeDirs() {
  const api = useAPIClient()
  return useQuery({
    queryKey: ['home-dirs'],
    queryFn: () => api.getHomeDirs(),
    staleTime: 30_000,
  })
}

export function useVersion() {
  const api = useAPIClient()
  return useQuery({
    queryKey: ['version'],
    queryFn: () => api.getVersion(),
    staleTime: Infinity,
  })
}
