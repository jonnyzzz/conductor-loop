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
  project_id?: string  // injected by startTask
  agent_type?: string  // set to 'claude' as default
}

export class APIClient {
  private baseURL: string

  constructor(baseURL: string) {
    this.baseURL = baseURL.replace(/\/$/, '')
  }

  private async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const url = `${this.baseURL}${path}`
    const headers = new Headers(options.headers)
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

  async getProject(projectId: string): Promise<ProjectDetail> {
    return this.request<ProjectDetail>(`/api/projects/${encodeURIComponent(projectId)}`)
  }

  async getTasks(projectId: string): Promise<TaskSummary[]> {
    const data = await this.request<TasksResponse>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks?limit=500`
    )
    return data.items
  }

  async getTask(projectId: string, taskId: string): Promise<TaskDetail> {
    return this.request<TaskDetail>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}`
    )
  }

  async getRuns(projectId: string, taskId: string): Promise<RunSummary[]> {
    const data = await this.request<RunsResponse>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/runs`
    )
    return data.items
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

  async getProjectMessages(projectId: string): Promise<BusMessage[]> {
    const data = await this.request<{ messages: BusMessage[] }>(
      `/api/projects/${encodeURIComponent(projectId)}/messages`
    )
    return data.messages
  }

  async getTaskMessages(projectId: string, taskId: string): Promise<BusMessage[]> {
    const data = await this.request<{ messages: BusMessage[] }>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/messages`
    )
    return data.messages
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

  async startTask(projectId: string, payload: TaskStartRequest): Promise<{ task_id: string; status: string; run_id: string }> {
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

  async deleteRun(projectId: string, taskId: string, runId: string): Promise<void> {
    await this.request<void>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}/runs/${encodeURIComponent(runId)}`,
      { method: 'DELETE' }
    )
  }

  async deleteTask(projectId: string, taskId: string): Promise<void> {
    await this.request<void>(
      `/api/projects/${encodeURIComponent(projectId)}/tasks/${encodeURIComponent(taskId)}`,
      { method: 'DELETE' }
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

  async getProjectRunsFlat(projectId: string): Promise<FlatRunItem[]> {
    const data = await this.request<FlatRunsResponse>(
      `/api/projects/${encodeURIComponent(projectId)}/runs/flat`
    )
    return data.runs
  }

  async getHomeDirs(): Promise<{ dirs: string[] }> {
    return this.request<{ dirs: string[] }>('/api/projects/home-dirs')
  }
}

export const createClient = () => {
  const baseURL = import.meta.env.VITE_API_BASE_URL ?? ''
  return new APIClient(baseURL)
}
