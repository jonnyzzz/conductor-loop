export type RunStatus = 'running' | 'queued' | 'completed' | 'failed' | 'blocked' | 'stopped' | 'unknown'

export interface Project {
  id: string
  last_activity: string
  task_count: number
  project_root?: string
}

export interface ProjectDetail extends Project {
  home_folders?: {
    project_root?: string
    source_folders?: string[]
    additional_folders?: string[]
  }
  tasks?: TaskSummary[]
}

export interface TaskSummary {
  id: string
  name?: string
  project_id?: string
  status: RunStatus
  queue_position?: number
  last_activity: string
  run_count?: number
  run_counts?: Partial<Record<RunStatus, number>>
  depends_on?: string[]
  blocked_by?: string[]
  thread_parent?: ThreadParentReference
}

export interface TaskDetail {
  id: string
  name?: string
  project_id: string
  status: RunStatus
  queue_position?: number
  last_activity: string
  created_at: string
  done: boolean
  state: string
  depends_on?: string[]
  blocked_by?: string[]
  thread_parent?: ThreadParentReference
  runs: RunSummary[]
}

export interface RunFile {
  name: string
  label: string
}

export interface RunSummary {
  id: string
  agent: string
  agent_version?: string
  status: RunStatus
  exit_code: number
  start_time: string
  end_time?: string
  parent_run_id?: string
  previous_run_id?: string
  error_summary?: string
  files?: RunFile[]
}

export interface RunInfo {
  version: number
  run_id: string
  project_id: string
  task_id: string
  parent_run_id: string
  previous_run_id: string
  agent: string
  agent_version?: string
  pid: number
  pgid: number
  start_time: string
  end_time: string
  exit_code: number
  cwd: string
  backend_provider?: string
  backend_model?: string
  backend_endpoint?: string
  commandline?: string
  error_summary?: string
}

export interface FileContent {
  name: string
  content: string
  modified: string
  size_bytes?: number
}

export interface BusMessage {
  msg_id: string
  timestamp: string
  type: string
  body: string
  parents?: Array<string | { msg_id: string; kind?: string; meta?: Record<string, unknown> }>
  run_id?: string
  attachment_path?: string
  issue_id?: string
  meta?: Record<string, string>
  project_id?: string
  task_id?: string
  // Backward-compat aliases retained while APIs converge on project_id/task_id.
  project?: string
  task?: string
}

export interface LogEvent {
  run_id: string
  stream: 'stdout' | 'stderr'
  line: string
  ts?: string
}

export interface RunStartEvent {
  run_id: string
  agent: string
  start_time: string
}

export interface RunEndEvent {
  run_id: string
  exit_code: number
  end_time: string
}

export interface ProjectsResponse {
  projects: Project[]
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  limit: number
  offset: number
  has_more: boolean
}

export type TasksResponse = PaginatedResponse<TaskSummary>

export type RunsResponse = PaginatedResponse<RunSummary>

export type TaskResponse = TaskDetail

export type RunInfoResponse = RunInfo

export interface MessageResponse {
  msg_id: string
}

export interface ProjectStats {
  project_id: string
  total_tasks: number
  total_runs: number
  running_runs: number
  completed_runs: number
  failed_runs: number
  crashed_runs: number
  message_bus_files: number
  message_bus_total_bytes: number
}

export interface FlatRunItem {
  id: string
  task_id: string
  agent: string
  status: RunStatus
  exit_code: number
  start_time: string
  end_time?: string
  parent_run_id?: string
  previous_run_id?: string
}

export interface FlatRunsResponse {
  runs: FlatRunItem[]
}

export interface ThreadParentReference {
  project_id: string
  task_id: string
  run_id: string
  message_id: string
  message_type?: string
}
