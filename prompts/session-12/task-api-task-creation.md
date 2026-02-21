# Task: Implement Task Creation API Enhancement (monitoring-ui Q3)

## Context
The project is Conductor Loop — a Go multi-agent orchestration framework.
Working directory: /Users/jonnyzzz/Work/conductor-loop

## Background
The human (Eugene Petrenko) answered monitoring-ui Q3 as "yes":
> Q3: Task creation payload and response shape
> Issue: Specs include `project_root`, `attach_mode`, and `run_id` in the task creation flow,
> but `internal/api/handlers.go` expects `{project_id, task_id, agent_type, prompt, config}`
> and returns `{project_id, task_id, status}` only.
> Question: Should the API add `project_root` and `attach_mode` handling and return `run_id`?
> Answer: yes

## Your Job

### 1. Read the relevant code first
- /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go (task creation handler)
- /Users/jonnyzzz/Work/conductor-loop/internal/api/routes.go (routing)
- /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go (project handlers)
- /Users/jonnyzzz/Work/conductor-loop/internal/storage/runinfo.go (RunInfo struct)
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/task.go (how tasks are run)
- /Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-monitoring-ui.md (spec)

### 2. Understand the spec
The monitoring-ui spec defines:
- Task creation payload should include optional `project_root` (working directory for the task)
- `attach_mode` field: "create" (new task), "attach" (attach to existing task), "resume" (resume)
- Response should include `run_id` (the ID of the run that was created)

### 3. Implement the enhancement
In `internal/api/handlers.go`, update the task creation request/response:

**Request struct changes:**
```go
type createTaskRequest struct {
    ProjectID   string `json:"project_id"`
    TaskID      string `json:"task_id"`
    AgentType   string `json:"agent_type"`
    Prompt      string `json:"prompt"`
    Config      string `json:"config,omitempty"`
    ProjectRoot string `json:"project_root,omitempty"`  // ADD: working dir for task
    AttachMode  string `json:"attach_mode,omitempty"`   // ADD: "create"|"attach"|"resume"
}
```

**Response struct changes:**
```go
type createTaskResponse struct {
    ProjectID string `json:"project_id"`
    TaskID    string `json:"task_id"`
    RunID     string `json:"run_id"`   // ADD: the run ID that was created
    Status    string `json:"status"`
}
```

**Handler logic:**
- If `project_root` is set, use it as the CWD for the agent run
- `attach_mode` semantics:
  - "create" (default): start new task (current behavior)
  - "attach": attach to existing task, start new run in that task's directory
  - "resume": same as attach but with restart prefix in prompt
- Return `run_id` in the response (the run ID that was allocated)
- If `project_root` is invalid (doesn't exist), return 400 error

### 4. Update tests
- Update existing task creation tests to check `run_id` in response
- Add test cases for `project_root` validation
- Add test cases for `attach_mode` values

### 5. Quality gates
- `go build ./...` must pass
- `go test ./...` must pass (all 18 packages)
- `go test -race ./internal/...` must pass

### 6. Commit format
```
feat(api): add project_root, attach_mode, run_id to task creation endpoint
```

## Done Criteria
Create /Users/jonnyzzz/Work/conductor-loop/DONE when complete.
Write summary to /Users/jonnyzzz/Work/conductor-loop/output.md.
