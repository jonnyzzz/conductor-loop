import type {
  BusMessage,
  FileContent,
  FlatRunItem,
  FlatRunsResponse,
  MessageResponse,
  Project,
  ProjectDetail,
  ProjectStats,
  ProjectsResponse,
  RunInfo,
  RunSummary,
  RunsResponse,
  TaskDetail,
  TaskSummary,
  TasksResponse,
} from '../types'

type RequestOptions = Omit<RequestInit, 'body'> & { body?: unknown }

export interface TaskStartRequest {
  task_id: string
  prompt: string
  project_root: string
  attach_mode: 'create' | 'attach' | 'resume'
  depends_on?: string[]
  thread_parent?: {
    project_id: string
    task_id: string
    run_id: string
    message_id: string
  }
  thread_message_type?: 'USER_REQUEST'
  project_id?: string  // injected by startTask
  agent_type?: string  // set to 'claude' as default
}

export interface ProjectCreateRequest {
  project_id: string
  project_root: string
}

export interface TaskStartResponse {
  task_id: string
  status: string
  run_id: string
  queue_position?: number
}

const BUS_MESSAGE_FETCH_LIMIT = 500
const TASKS_PAGE_LIMIT = 500

interface MessageListRequestOptions {
  since?: string
  limit?: number
}

function normalizeStringArray(value: string[] | null | undefined): string[] | undefined {
  return Array.isArray(value) ? value : undefined
}

function normalizeRunSummary<T extends { files?: unknown }>(run: T): T {
  if (run.files === undefined || Array.isArray(run.files)) {
    return run
  }
  return {
    ...run,
    files: undefined,
  }
}

function normalizeTaskSummary(task: TaskSummary): TaskSummary {
  const dependsOn = normalizeStringArray(task.depends_on)
  const blockedBy = normalizeStringArray(task.blocked_by)
  if (dependsOn === task.depends_on && blockedBy === task.blocked_by) {
    return task
  }
  return {
    ...task,
    depends_on: dependsOn,
    blocked_by: blockedBy,
  }
}

function buildMessageListQuery(options?: MessageListRequestOptions): string {
  const params = new URLSearchParams()
  const since = options?.since?.trim()
  const limit = options?.limit && options.limit > 0
    ? options.limit
    : BUS_MESSAGE_FETCH_LIMIT

  params.set('limit', String(limit))
  if (since) {
    params.set('since', since)
  }
  return params.toString()
}

function isSinceCursorMiss(err: unknown): boolean {
  if (!(err instanceof Error)) {
    return false
  }
  const normalized = err.message.toLowerCase()
  return normalized.includes('api 404') && normalized.includes('message id not found')
}

export class APIClient {
  private baseURL: string

  constructor(baseURL: string) {
    this.baseURL = baseURL.replace(/\/$/, '')
  }

  private async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const url = `${this.baseURL}${path}`
    const headers = new Headers(options.headers)
    headers.set('X-Conductor-Client', 'web-ui')
    if (options.body !== undefined) {
      headers.set('Content-Type', 'application/json')
    }

    const response = await fetch(url, {
      ...options,
      headers,
      body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    })

    if (!response.ok) {
      let details = ''
      try {
        const payload = await response.json()
        details = payload?.error?.message ?? JSON.stringify(payload)
      } catch {
        details = await response.text()
      }
      throw new Error(`api ${response.status}: ${details || response.statusText}`)
    }

    if (response.status === 204) {
      return undefined as T
    }

    return (await response.json()) as T
  }

  async getProjects(): Promise<Project[]> {
    const data = await this.request<ProjectsResponse>('/api/projects')
    return data.projects
  }

  async createProject(payload: ProjectCreateRequest): Promise<Project> {
    return this.request<Project>('/api/projects', {
      method: 'POST',
      body: payload,
    })
  }

  async getProject(projectId: string): Promise<ProjectDetail> {
    return this.request<ProjectDetail>(`/api/projects/${encodeURIComponent(projectId)}`)
  }

  async getTasks(projectId: string): Promise<TaskSummary[]> {
    const items: TaskSummary[] = []
    let offset = 0

    for (;;) {
      const data = await this.request<TasksResponse>(
        `/api/projects/${encodeURIComponent(projectId)}/tasks?limit=${TASKS_PAGE_LIMIT}&offset=${offset}`
      )
      const pageItems = data.items ?? []
      items.push(...pageItems)

      if (!data.has_more || pageItems.length === 0) {
        break
      }
      offset += pageItems.length
    }

    return items.map((item) => normalizeTaskSummary(item))
  }

  async getTask(projectId: string, taskId: string): Promise<TaskDetail> {
    const task = await this.request<TaskDetail>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}`
    )
    const dependsOn = normalizeStringArray(task.depends_on)
    const blockedBy = normalizeStringArray(task.blocked_by)
    const hasRunsArray = Array.isArray(task.runs)
    const sourceRuns = hasRunsArray ? task.runs : []
    let runsChanged = !hasRunsArray
    const normalizedRuns = sourceRuns.map((run, index) => {
      const normalized = normalizeRunSummary(run)
      if (normalized !== sourceRuns[index]) {
        runsChanged = true
      }
      return normalized
    })
    if (!runsChanged && dependsOn === task.depends_on && blockedBy === task.blocked_by) {
      return task
    }
    return {
      ...task,
      depends_on: dependsOn,
      blocked_by: blockedBy,
      runs: normalizedRuns,
    }
  }

  async getRuns(projectId: string, taskId: string): Promise<RunSummary[]> {
    const data = await this.request<RunsResponse>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/runs`
    )
    const runs = data.items ?? []
    let runsChanged = false
    const normalized = runs.map((run, index) => {
      const nextRun = normalizeRunSummary(run)
      if (nextRun !== runs[index]) {
        runsChanged = true
      }
      return nextRun
    })
    return runsChanged ? normalized : runs
  }

  async getRunInfo(projectId: string, taskId: string, runId: string): Promise<RunInfo> {
    return this.request<RunInfo>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/runs/${encodeURIComponent(runId)}`
    )
  }

  async getTaskFile(projectId: string, taskId: string, name: string): Promise<FileContent> {
    const params = new URLSearchParams({ name })
    return this.request<FileContent>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/file?${params.toString()}`
    )
  }

  async getRunFile(
    projectId: string,
    taskId: string,
    runId: string,
    name: string,
    tail?: number
  ): Promise<FileContent> {
    const params = new URLSearchParams({ name })
    if (tail) {
      params.set('tail', String(tail))
    }
    return this.request<FileContent>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/runs/${encodeURIComponent(runId)}/file?${params.toString()}`
    )
  }

  async getProjectMessages(
    projectId: string,
    options?: MessageListRequestOptions
  ): Promise<BusMessage[]> {
    return this.listBusMessages(
      `/api/projects/${encodeURIComponent(projectId)}/messages`,
      options
    )
  }

  async getTaskMessages(
    projectId: string,
    taskId: string,
    options?: MessageListRequestOptions
  ): Promise<BusMessage[]> {
    return this.listBusMessages(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/messages`,
      options
    )
  }

  async postProjectMessage(projectId: string, body: Omit<BusMessage, 'msg_id' | 'timestamp'>): Promise<MessageResponse> {
    return this.request<MessageResponse>(`/api/projects/${encodeURIComponent(projectId)}/messages`, {
      method: 'POST',
      body,
    })
  }

  async postTaskMessage(
    projectId: string,
    taskId: string,
    body: Omit<BusMessage, 'msg_id' | 'timestamp'>
  ): Promise<MessageResponse> {
    return this.request<MessageResponse>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/messages`,
      { method: 'POST', body }
    )
  }

  async startTask(projectId: string, payload: TaskStartRequest): Promise<TaskStartResponse> {
    return this.request(`/api/v1/tasks`, {
      method: 'POST',
      body: { ...payload, project_id: projectId },
    })
  }

  async stopRun(projectId: string, taskId: string, runId: string): Promise<void> {
    // Endpoint is expected to be added by the backend; keep read-only by default.
    await this.request<void>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/runs/${encodeURIComponent(runId)}/stop`,
      { method: 'POST' }
    )
  }

  async getProjectStats(projectId: string): Promise<ProjectStats> {
    return this.request<ProjectStats>(`/api/projects/${encodeURIComponent(projectId)}/stats`)
  }

  async resumeTask(projectId: string, taskId: string): Promise<{ task_id: string; resumed: boolean }> {
    return this.request<{ task_id: string; resumed: boolean }>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/resume`,
      { method: 'POST' }
    )
  }

  async getProjectRunsFlat(
    projectId: string,
    options?: {
      activeOnly?: boolean
      selectedTaskId?: string
      selectedTaskLimit?: number
    }
  ): Promise<FlatRunItem[]> {
    const params = new URLSearchParams()
    if (options?.activeOnly) {
      params.set('active_only', '1')
    }
    if (options?.selectedTaskId) {
      params.set('selected_task_id', options.selectedTaskId)
    }
    if (options?.selectedTaskLimit && options.selectedTaskLimit > 0) {
      params.set('selected_task_limit', String(options.selectedTaskLimit))
    }
    const query = params.toString()
    const path = query.length > 0
      ? `/api/projects/${encodeURIComponent(projectId)}/runs/flat?${query}`
      : `/api/projects/${encodeURIComponent(projectId)}/runs/flat`

    const data = await this.request<FlatRunsResponse>(path)
    return data.runs
  }

  async getHomeDirs(): Promise<{ dirs: string[] }> {
    return this.request<{ dirs: string[] }>('/api/projects/home-dirs')
  }

  private async listBusMessages(path: string, options?: MessageListRequestOptions): Promise<BusMessage[]> {
    const query = buildMessageListQuery(options)
    try {
      const data = await this.request<{ messages: BusMessage[] }>(`${path}?${query}`)
      return data.messages
    } catch (err) {
      const since = options?.since?.trim()
      if (!since || !isSinceCursorMiss(err)) {
        throw err
      }
      const fallbackQuery = buildMessageListQuery({ limit: options?.limit })
      const data = await this.request<{ messages: BusMessage[] }>(`${path}?${fallbackQuery}`)
      return data.messages
    }
  }
}

export const createClient = () => {
  const baseURL = import.meta.env.VITE_API_BASE_URL ?? ''
  return new APIClient(baseURL)
}
