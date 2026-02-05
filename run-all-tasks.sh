#!/bin/bash
# Conductor Loop - Parallel Implementation Orchestration Script
# Based on THE_PROMPT_v5.md workflow
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
RUNS_DIR="${PROJECT_ROOT}/runs"
MESSAGE_BUS="${PROJECT_ROOT}/MESSAGE-BUS.md"
ISSUES_FILE="${PROJECT_ROOT}/ISSUES.md"
PROMPTS_DIR="${PROJECT_ROOT}/prompts"

export PROJECT_ROOT RUNS_DIR MESSAGE_BUS ISSUES_FILE

# Configuration
MAX_PARALLEL_AGENTS=16
STAGE_TIMEOUT=3600  # 1 hour per stage

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Initialize files
mkdir -p "$PROMPTS_DIR"
touch "$MESSAGE_BUS" "$ISSUES_FILE"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$MESSAGE_BUS"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" | tee -a "$ISSUES_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

# Initialize MESSAGE-BUS.md
cat > "$MESSAGE_BUS" <<'EOF'
# Conductor Loop Implementation - Message Bus

**Project**: Conductor Loop
**Start**: $(date '+%Y-%m-%d %H:%M:%S')
**Plan**: THE_PLAN_v5.md
**Workflow**: THE_PROMPT_v5.md

---

EOF

log "DECISION: Starting parallel implementation orchestration"
log "DECISION: Max parallel agents: $MAX_PARALLEL_AGENTS"
log "DECISION: Agent assignment: Codex (implementation), Claude (research/docs), Multi-agent (review)"

#############################################################################
# STAGE 0: CREATE ALL PROMPT FILES
#############################################################################

create_prompt_bootstrap_01() {
    cat > "$PROMPTS_DIR/bootstrap-01-structure.md" <<'EOF'
# Task: Create Project Structure

**Task ID**: bootstrap-01
**Phase**: Bootstrap
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Set up Go project structure with proper module initialization and build tooling.

## Required Actions

1. **Initialize Go Module**
   ```bash
   cd ~/Work/conductor-loop
   go mod init github.com/jonnyzzz/conductor-loop
   ```

2. **Create go.mod with Dependencies**
   Add these dependencies:
   - gopkg.in/yaml.v3 (YAML parsing)
   - github.com/spf13/cobra (CLI framework)
   - github.com/spf13/viper (configuration)
   - golang.org/x/sync (sync primitives)

3. **Create Makefile**
   Targets:
   - `make build` - Build all binaries
   - `make test` - Run all tests
   - `make lint` - Run linters
   - `make docker` - Build Docker image
   - `make clean` - Clean build artifacts

4. **Create .gitignore**
   Ignore: binaries, test outputs, IDE files, runs/ directory

5. **Create cmd/conductor/main.go**
   Basic CLI skeleton with cobra:
   - `conductor task` command (placeholder)
   - `conductor job` command (placeholder)
   - Version flag

## Success Criteria
- `go mod download` works
- `make build` succeeds
- `./conductor --version` works
- All directories from THE_PLAN_v5.md exist

## References
- THE_PLAN_v5.md: Phase 0, Task bootstrap-01
- THE_PROMPT_v5.md: Standard Workflow, Phase 0

## Output
Log completion to MESSAGE-BUS.md with:
- FACT: Go module initialized
- FACT: Makefile targets working
- FACT: Basic CLI runs
EOF
}

create_prompt_bootstrap_02() {
    cat > "$PROMPTS_DIR/bootstrap-02-documentation.md" <<'EOF'
# Task: Create Documentation Structure

**Task ID**: bootstrap-02
**Phase**: Bootstrap
**Agent Type**: Research/Documentation (Claude preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Set up project documentation following THE_PROMPT_v5.md conventions.

## Required Actions

1. **Create AGENTS.md**
   Define:
   - Project conventions (Go style, commit format)
   - Agent types (Orchestrator, Implementation, Review, Test, Debug)
   - Permissions (file access, tool access)
   - Subsystem ownership

2. **Create Instructions.md**
   Document:
   - Repository structure
   - Build commands
   - Test commands
   - Tool paths (go, docker, make)
   - Environment setup

3. **Create Role Prompt Files**
   Copy from THE_PROMPT_v5.md template and adapt:
   - THE_PROMPT_v5_orchestrator.md
   - THE_PROMPT_v5_research.md
   - THE_PROMPT_v5_implementation.md
   - THE_PROMPT_v5_review.md
   - THE_PROMPT_v5_test.md
   - THE_PROMPT_v5_debug.md

4. **Create DEVELOPMENT.md**
   - Local development setup
   - Running tests
   - Debugging tips
   - Contributing guidelines

## Success Criteria
- All role prompt files exist
- AGENTS.md defines clear conventions
- Instructions.md has all tool paths

## References
- THE_PROMPT_v5.md: Role-Specific Prompts section
- docs/specifications/ for technical details

## Output
Log to MESSAGE-BUS.md:
- FACT: Documentation structure created
- FACT: Role prompts ready
EOF
}

create_prompt_bootstrap_03() {
    cat > "$PROMPTS_DIR/bootstrap-03-tooling.md" <<'EOF'
# Task: Set Up Tooling and CI/CD

**Task ID**: bootstrap-03
**Phase**: Bootstrap
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Set up development tooling, Docker, and CI/CD pipelines.

## Required Actions

1. **Docker Setup**
   Create Dockerfile:
   - Multi-stage build (Go builder + minimal runtime)
   - Copy conductor binary
   - Expose API port (default: 8080)
   - Volume for /data (run storage)

2. **Docker Compose**
   Services:
   - conductor: Main service
   - postgres: (optional, for future metadata storage)
   - nginx: (optional, for frontend)

3. **GitHub Actions Workflows**
   Create .github/workflows/:
   - test.yml: Run tests on push/PR
   - build.yml: Build binaries on release
   - docker.yml: Build and push Docker image
   - lint.yml: Run golangci-lint

4. **Monitoring Scripts**
   Create:
   - watch-agents.sh (60s polling)
   - monitor-agents.py (live console monitor)

## Success Criteria
- Docker builds successfully
- docker-compose up works
- GitHub Actions validate

## References
- THE_PROMPT_v5.md: Agent Execution and Traceability

## Output
Log to MESSAGE-BUS.md:
- FACT: Docker image builds
- FACT: CI/CD pipelines configured
EOF
}

create_prompt_bootstrap_04() {
    cat > "$PROMPTS_DIR/bootstrap-04-arch-review.md" <<'EOF'
# Task: Architecture Review

**Task ID**: bootstrap-04
**Phase**: Bootstrap
**Agent Type**: Multi-Agent Review (2+ agents)
**Project Root**: ~/Work/conductor-loop

## Objective
Multi-agent review of architecture and implementation plan.

## Required Actions

1. **Specification Review**
   Read all files in docs/specifications/:
   - Validate 8 subsystems are complete
   - Check for missing details
   - Verify consistency across specs

2. **Dependency Analysis**
   Map dependencies between components:
   - Storage → Message Bus
   - Message Bus → Runner
   - Runner → Agent Backends
   - API → All components

3. **Risk Assessment**
   Identify:
   - Platform-specific risks (Windows, macOS, Linux)
   - Concurrency risks (race conditions)
   - Integration risks (agent CLIs)

4. **Implementation Strategy**
   Validate THE_PLAN_v5.md:
   - Correct phase ordering
   - Sufficient parallelism
   - Realistic timelines

## Success Criteria
- 2+ agents provide independent reviews
- Consensus on approach or documented differences
- Issues logged to ISSUES.md

## References
- THE_PLAN_v5.md: Full implementation plan
- docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md

## Output
Log to MESSAGE-BUS.md:
- REVIEW: Architecture assessment
- DECISION: Any plan adjustments
- ERROR: Critical issues to ISSUES.md
EOF
}

#############################################################################
# STAGE 1: CORE INFRASTRUCTURE PROMPTS
#############################################################################

create_prompt_infra_storage() {
    cat > "$PROMPTS_DIR/infra-storage.md" <<'EOF'
# Task: Implement Storage Layer

**Task ID**: infra-storage
**Phase**: Core Infrastructure
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Implement file-based storage with atomic operations and run-info.yaml handling.

## Specifications
Read: docs/specifications/subsystem-storage-layout-run-info-schema.md

## Required Implementation

### 1. Package Structure
Location: `internal/storage/`
Files:
- runinfo.go - RunInfo struct and operations
- atomic.go - Atomic file operations
- storage.go - Storage interface and impl

### 2. RunInfo Struct
```go
type RunInfo struct {
    RunID       string    `yaml:"run_id"`
    ParentRunID string    `yaml:"parent_run_id,omitempty"`
    ProjectID   string    `yaml:"project_id"`
    TaskID      string    `yaml:"task_id"`
    AgentType   string    `yaml:"agent_type"`
    PID         int       `yaml:"pid"`
    PGID        int       `yaml:"pgid"`
    StartTime   time.Time `yaml:"start_time"`
    EndTime     time.Time `yaml:"end_time,omitempty"`
    ExitCode    int       `yaml:"exit_code,omitempty"`
    Status      string    `yaml:"status"` // running, completed, failed
}
```

### 3. Atomic Operations
Implement per Problem #3 decision:
- WriteRunInfo(path, info) - temp + fsync + rename
- ReadRunInfo(path) - direct read
- UpdateRunInfo(path, updates) - atomic rewrite

### 4. Storage Interface
```go
type Storage interface {
    CreateRun(projectID, taskID, agentType string) (*RunInfo, error)
    UpdateRunStatus(runID string, status string, exitCode int) error
    GetRunInfo(runID string) (*RunInfo, error)
    ListRuns(projectID, taskID string) ([]*RunInfo, error)
}
```

## Tests Required
Location: `test/unit/storage_test.go`
- TestRunInfoSerialization
- TestAtomicWrite
- TestConcurrentWrites (10 goroutines × 100 writes)
- TestUpdateRunInfo

## Success Criteria
- All tests pass
- IntelliJ MCP review: no warnings
- Atomic operations verified with race detector: `go test -race`

## References
- docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md: Problem #3
- THE_PROMPT_v5.md: Stage 5 (Implement changes and tests)

## Output
Log to MESSAGE-BUS.md:
- FACT: Storage layer implemented
- FACT: N unit tests passing
- FACT: Race detector clean
EOF
}

create_prompt_infra_config() {
    cat > "$PROMPTS_DIR/infra-config.md" <<'EOF'
# Task: Implement Configuration Management

**Task ID**: infra-config
**Phase**: Core Infrastructure
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Implement YAML-based configuration with secure token handling and validation.

## Specifications
Read: docs/specifications/subsystem-runner-orchestration-config-schema.md

## Required Implementation

### 1. Package Structure
Location: `internal/config/`
Files:
- config.go - Config struct and loading
- validation.go - Config validation
- tokens.go - Token/token_file handling

### 2. Config Struct
```go
type Config struct {
    Agents map[string]AgentConfig `yaml:"agents"`
    Defaults DefaultConfig `yaml:"defaults"`
}

type AgentConfig struct {
    Type      string `yaml:"type"`      // claude, codex, gemini, perplexity, xai
    Token     string `yaml:"token,omitempty"`
    TokenFile string `yaml:"token_file,omitempty"`
    BaseURL   string `yaml:"base_url,omitempty"`
    Model     string `yaml:"model,omitempty"`
}

type DefaultConfig struct {
    Agent   string `yaml:"agent"`
    Timeout int    `yaml:"timeout"`
}
```

### 3. Loading Logic
Implement:
- LoadConfig(path string) (*Config, error)
- Load from YAML file
- Resolve token_file to token value
- Apply environment variable overrides (CONDUCTOR_AGENT_<NAME>_TOKEN)
- Validate configuration

### 4. Token Handling
Per specification:
- If token is set, use it directly
- If token_file is set, read token from file
- Support both relative and absolute paths
- Error if both token and token_file are set
- Support environment variable override

### 5. Validation
Validate:
- At least one agent configured
- Agent type is valid (claude, codex, gemini, perplexity, xai)
- Either token or token_file is set (not both)
- Token files exist and are readable
- Timeouts are positive

## Tests Required
Location: `test/unit/config_test.go`
- TestLoadConfig
- TestTokenFileResolution
- TestTokenFromEnv
- TestConfigValidation (invalid configs)
- TestAgentDefaults

## Success Criteria
- All tests pass
- IntelliJ MCP review: no warnings
- Example config.yaml works

## Example Config
```yaml
agents:
  claude:
    type: claude
    token_file: ~/.config/claude/token

  codex:
    type: codex
    token_file: ~/.config/codex/token

  gemini:
    type: gemini
    token: ${GEMINI_API_KEY}

defaults:
  agent: claude
  timeout: 3600
```

## References
- docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md
- THE_PROMPT_v5.md: Stage 5 (Implement changes and tests)

## Output
Log to MESSAGE-BUS.md:
- FACT: Configuration system implemented
- FACT: N unit tests passing
- FACT: Token handling secure
EOF
}

create_prompt_infra_messagebus() {
    cat > "$PROMPTS_DIR/infra-messagebus.md" <<'EOF'
# Task: Implement Message Bus

**Task ID**: infra-messagebus
**Phase**: Core Infrastructure
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Implement O_APPEND + flock message bus with fsync-always policy.

## Specifications
Read:
- docs/specifications/subsystem-message-bus-tools.md
- docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md (Problem #1)

## Required Implementation

### 1. Package Structure
Location: `internal/messagebus/`
Files:
- messagebus.go - Message struct and operations
- msgid.go - Message ID generation
- lock.go - File locking primitives

### 2. Message Struct
```go
type Message struct {
    MsgID        string            `yaml:"msg_id"`
    Timestamp    time.Time         `yaml:"ts"`
    Type         string            `yaml:"type"` // FACT, PROGRESS, DECISION, REVIEW, ERROR
    ProjectID    string            `yaml:"project_id"`
    TaskID       string            `yaml:"task_id"`
    RunID        string            `yaml:"run_id"`
    ParentMsgIDs []string          `yaml:"parents,omitempty"`
    Attachment   string            `yaml:"attachment_path,omitempty"`
    Body         string            `yaml:"-"` // After second ---
}
```

### 3. Message ID Generation
Format: `MSG-YYYYMMDD-HHMMSS-NNNNNNNNN-PIDXXXXX-SSSS`
- Nanosecond timestamp
- PID (5 digits)
- Atomic counter (4 digits, per-process)

### 4. Write Algorithm
```go
func (mb *MessageBus) AppendMessage(msg *Message) (string, error) {
    // 1. Generate msg_id
    msg.MsgID = generateMessageID()

    // 2. Serialize to YAML with --- delimiters
    data := serializeMessage(msg)

    // 3. Open with O_APPEND
    fd, err := os.OpenFile(mb.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

    // 4. Acquire exclusive lock (10s timeout)
    err = flockExclusive(fd, 10*time.Second)

    // 5. Write
    n, err := fd.Write(data)

    // 6. fsync
    err = fd.Sync()

    // 7. Unlock (defer)
    return msg.MsgID, nil
}
```

### 5. Read Operations
```go
func (mb *MessageBus) ReadMessages(sinceID string) ([]*Message, error)
func (mb *MessageBus) PollForNew(lastID string) ([]*Message, error)
```

## Tests Required
Location: `test/integration/messagebus_test.go`
- TestMessageIDUniqueness (1000 IDs)
- TestConcurrentAppend (10 processes × 100 messages = 1000 total)
- TestLockTimeout
- TestCrashRecovery (SIGKILL during write)
- TestReadWhileWriting

## Success Criteria
- All 1000 messages present (no data loss)
- All msg_ids unique
- Lock timeout works
- IntelliJ MCP review clean
- Race detector clean

## References
- THE_PROMPT_v5.md: Stage 7 (Re-run tests in IntelliJ MCP)

## Output
Log to MESSAGE-BUS.md:
- FACT: Message bus implemented
- FACT: Concurrency tests pass (1000/1000 messages)
- FACT: Zero data loss verified
EOF
}

create_prompt_agent_protocol() {
    cat > "$PROMPTS_DIR/agent-protocol.md" <<'EOF'
# Task: Implement Agent Protocol Interface

**Task ID**: agent-protocol
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Implement the common agent protocol interface that all agent backends will implement.

## Specifications
Read: docs/specifications/subsystem-agent-protocol.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/`
Files:
- agent.go - Agent interface and context types
- executor.go - Execution helper functions
- stdio.go - Stdout/stderr capture utilities

### 2. Agent Interface
```go
type Agent interface {
    // Execute runs the agent with the given context
    Execute(ctx context.Context, runCtx *RunContext) error

    // Type returns the agent type (claude, codex, etc.)
    Type() string
}

type RunContext struct {
    RunID       string
    ProjectID   string
    TaskID      string
    Prompt      string
    WorkingDir  string
    StdoutPath  string
    StderrPath  string
    Environment map[string]string
}
```

### 3. Executor Functions
- SpawnProcess(cmd, args, stdin, stdout, stderr) - Process spawning with setsid
- CaptureOutput(stdout, stderr, files) - Stdio redirection
- CreateOutputMD(runDir, fallback) - Runner fallback for output.md

### 4. Tests Required
Location: `test/unit/agent_test.go`
- TestAgentInterface
- TestRunContext
- TestSpawnProcess
- TestCaptureOutput

## Success Criteria
- All tests pass
- Interface documented
- IntelliJ MCP review: no warnings

## References
- docs/specifications/subsystem-agent-protocol.md
- docs/decisions/problem-5-DECISION.md (output.md responsibility)

## Output
Log to MESSAGE-BUS.md:
- FACT: Agent protocol interface implemented
- FACT: N unit tests passing
EOF
}

create_prompt_agent_claude() {
    cat > "$PROMPTS_DIR/agent-claude.md" <<'EOF'
# Task: Implement Claude Agent Backend

**Task ID**: agent-claude
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol

## Objective
Implement Claude agent backend adapter.

## Specifications
Read: docs/specifications/subsystem-agent-backend-claude.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/claude/`
Files:
- claude.go - Claude agent implementation

### 2. Claude Agent
```go
type ClaudeAgent struct {
    token string
    model string
}

func (a *ClaudeAgent) Execute(ctx context.Context, runCtx *agent.RunContext) error {
    // Execute claude CLI with prompt
    // Redirect stdio to files
    // Return on completion
}

func (a *ClaudeAgent) Type() string {
    return "claude"
}
```

### 3. CLI Integration
- Use `claude` CLI binary
- Pass prompt via stdin
- Set working directory with `-C` flag
- Capture stdout/stderr to files

### 4. Tests Required
Location: `test/integration/agent_claude_test.go`
- TestClaudeExecution (requires claude CLI)
- TestClaudeStdioCapture

## Success Criteria
- All tests pass
- Claude CLI integration working
- Stdio properly captured

## References
- docs/specifications/subsystem-agent-backend-claude.md

## Output
Log to MESSAGE-BUS.md:
- FACT: Claude agent backend implemented
- FACT: Integration tests passing
EOF
}

create_prompt_agent_codex() {
    cat > "$PROMPTS_DIR/agent-codex.md" <<'EOF'
# Task: Implement Codex Agent Backend

**Task ID**: agent-codex
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol

## Objective
Implement Codex (IntelliJ MCP) agent backend adapter.

## Specifications
Read: docs/specifications/subsystem-agent-backend-codex.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/codex/`
Files:
- codex.go - Codex agent implementation

### 2. Codex Agent
Similar to Claude but using `codex exec` CLI.

### 3. CLI Integration
- Use `codex exec` CLI binary
- Pass prompt via stdin
- Dangerously bypass approvals for automation
- Set working directory with `-C` flag

### 4. Tests Required
Location: `test/integration/agent_codex_test.go`
- TestCodexExecution

## Success Criteria
- All tests pass
- Codex CLI integration working

## Output
Log to MESSAGE-BUS.md:
- FACT: Codex agent backend implemented
EOF
}

create_prompt_agent_gemini() {
    cat > "$PROMPTS_DIR/agent-gemini.md" <<'EOF'
# Task: Implement Gemini Agent Backend

**Task ID**: agent-gemini
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol

## Objective
Implement Gemini agent backend adapter.

## Specifications
Read: docs/specifications/subsystem-agent-backend-gemini.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/gemini/`
Files:
- gemini.go - Gemini agent implementation

### 2. Gemini Agent
REST API integration for Gemini.

### 3. API Integration
- Use Google Gemini REST API
- Handle authentication via token
- Stream response to stdout

### 4. Tests Required
Location: `test/integration/agent_gemini_test.go`
- TestGeminiExecution

## Success Criteria
- All tests pass
- API integration working

## Output
Log to MESSAGE-BUS.md:
- FACT: Gemini agent backend implemented
EOF
}

create_prompt_agent_perplexity() {
    cat > "$PROMPTS_DIR/agent-perplexity.md" <<'EOF'
# Task: Implement Perplexity Agent Backend

**Task ID**: agent-perplexity
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol

## Objective
Implement Perplexity agent backend adapter.

## Specifications
Read: docs/specifications/subsystem-agent-backend-perplexity.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/perplexity/`
Files:
- perplexity.go - Perplexity agent implementation

### 2. Perplexity Agent
REST API + SSE integration.

### 3. API Integration
- Use Perplexity REST API
- Handle SSE streaming
- Unified stdout-only output (per Problem #6)

### 4. Tests Required
Location: `test/integration/agent_perplexity_test.go`
- TestPerplexityExecution

## Success Criteria
- All tests pass
- SSE streaming working

## Output
Log to MESSAGE-BUS.md:
- FACT: Perplexity agent backend implemented
EOF
}

create_prompt_agent_xai() {
    cat > "$PROMPTS_DIR/agent-xai.md" <<'EOF'
# Task: Implement xAI Agent Backend

**Task ID**: agent-xai
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol

## Objective
Implement xAI agent backend adapter.

## Specifications
Read: docs/specifications/subsystem-agent-backend-xai.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/xai/`
Files:
- xai.go - xAI agent implementation

### 2. xAI Agent
REST API integration for xAI.

### 3. API Integration
- Use xAI REST API
- Handle authentication
- Stream response to stdout

### 4. Tests Required
Location: `test/integration/agent_xai_test.go`
- TestXAIExecution

## Success Criteria
- All tests pass
- API integration working

## Output
Log to MESSAGE-BUS.md:
- FACT: xAI agent backend implemented
EOF
}

create_prompt_runner_process() {
    cat > "$PROMPTS_DIR/runner-process.md" <<'EOF'
# Task: Implement Process Management

**Task ID**: runner-process
**Phase**: Runner Orchestration
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol, storage, config

## Objective
Implement process spawning with setsid() and stdio redirection.

## Specifications
Read: docs/specifications/subsystem-runner-orchestration.md
Read: docs/decisions/problem-7-DECISION.md (setsid not daemonization)

## Required Implementation

### 1. Package Structure
Location: `internal/runner/`
Files:
- process.go - Process spawning with setsid
- stdio.go - Stdio redirection to files
- pgid.go - Process group ID management

### 2. Process Spawning
```go
type ProcessManager struct {
    runDir string
}

func (pm *ProcessManager) SpawnAgent(ctx context.Context, agentType string, opts SpawnOptions) (*Process, error) {
    // Use exec.Cmd with SysProcAttr.Setsid = true
    // Redirect stdin/stdout/stderr to files
    // Track PID and PGID
    // Return Process handle
}
```

### 3. Setsid Implementation
- Use syscall.SysProcAttr{Setsid: true} on Unix
- Use CREATE_NEW_PROCESS_GROUP on Windows
- Do NOT daemonize (no double-fork)
- Terminal detachment only

### 4. Stdio Redirection
- Create agent-stdout.txt, agent-stderr.txt in run dir
- Open with O_APPEND for concurrent writes
- Use io.MultiWriter for tee-style logging if needed

### 5. Tests Required
Location: `test/unit/process_test.go`
- TestSpawnProcess
- TestProcessSetsid
- TestStdioRedirection
- TestProcessGroupManagement

## Success Criteria
- All tests pass
- Processes properly detached from terminal
- Stdio correctly captured to files

## References
- docs/decisions/problem-7-DECISION.md

## Output
Log to MESSAGE-BUS.md:
- FACT: Process management implemented
- FACT: Setsid working on Unix and Windows
EOF
}

create_prompt_runner_ralph() {
    cat > "$PROMPTS_DIR/runner-ralph.md" <<'EOF'
# Task: Implement Ralph Loop

**Task ID**: runner-ralph
**Phase**: Runner Orchestration
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: runner-process, messagebus

## Objective
Implement the Ralph Loop (Root Agent Loop) with wait-without-restart pattern.

## Specifications
Read: docs/specifications/subsystem-runner-orchestration.md
Read: docs/decisions/problem-2-FINAL-DECISION.md (DONE + children handling)

## Required Implementation

### 1. Package Structure
Location: `internal/runner/`
Files:
- ralph.go - Ralph Loop implementation
- wait.go - Child process waiting logic

### 2. Ralph Loop
```go
type RalphLoop struct {
    runDir        string
    messagebus    *messagebus.MessageBus
    maxRestarts   int
    waitTimeout   time.Duration // 300s default
}

func (rl *RalphLoop) Run(ctx context.Context) error {
    // 1. Check for DONE file
    // 2. If DONE exists:
    //    - Wait for children (up to waitTimeout)
    //    - Check children with kill(-pgid, 0)
    //    - Return when all children exit or timeout
    // 3. If no DONE:
    //    - Check if process should restart
    //    - Respect maxRestarts limit
    //    - Restart if needed
}
```

### 3. DONE File Detection
- Check for DONE file in run directory
- File presence signals "don't restart"
- Must still wait for children

### 4. Child Waiting
- Use kill(-pgid, 0) to detect children
- Poll every 1 second
- Timeout after 300 seconds (configurable)
- Return early if all children exit

### 5. Restart Logic
- Count restarts, enforce maxRestarts limit
- Log restart events to message bus
- Exponential backoff optional

### 6. Tests Required
Location: `test/unit/ralph_test.go`
- TestRalphLoopDONEWithChildren
- TestRalphLoopDONEWithoutChildren
- TestRalphLoopRestartLogic
- TestChildWaitTimeout

## Success Criteria
- All tests pass
- DONE + children scenario working
- 300s timeout enforced

## References
- docs/decisions/problem-2-FINAL-DECISION.md

## Output
Log to MESSAGE-BUS.md:
- FACT: Ralph Loop implemented
- FACT: Wait-without-restart working
EOF
}

create_prompt_runner_orchestration() {
    cat > "$PROMPTS_DIR/runner-orchestration.md" <<'EOF'
# Task: Implement Run Orchestration

**Task ID**: runner-orchestration
**Phase**: Runner Orchestration
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: runner-ralph, storage, config, agent-protocol

## Objective
Implement run-agent task and job commands for orchestration.

## Specifications
Read: docs/specifications/subsystem-runner-orchestration.md

## Required Implementation

### 1. Package Structure
Location: `internal/runner/`
Files:
- orchestrator.go - Main orchestration logic
- task.go - Task command implementation
- job.go - Job command implementation

### 2. Task Command
```go
func RunTask(projectID, taskID string, opts TaskOptions) error {
    // 1. Create run directory
    // 2. Write run-info.yaml (status: running)
    // 3. Load config, select agent
    // 4. Spawn agent process
    // 5. Start Ralph Loop
    // 6. Update run-info.yaml (status: completed/failed)
}
```

### 3. Job Command
- Similar to task but with job-specific metadata
- Support parent-child run relationships
- Track parent_run_id in run-info.yaml

### 4. Run Directory Structure
Create:
- run-info.yaml
- agent-stdout.txt
- agent-stderr.txt
- DONE (created by agent when finished)

### 5. Parent-Child Relationships
- Child runs set parent_run_id
- Parent waits for children via Ralph Loop
- Message bus allows inter-run communication

### 6. Tests Required
Location: `test/integration/orchestration_test.go`
- TestRunTask
- TestRunJob
- TestParentChildRuns
- TestNestedRuns

## Success Criteria
- All tests pass
- run-agent task command working
- run-agent job command working
- Parent-child relationships functional

## Output
Log to MESSAGE-BUS.md:
- FACT: Orchestration implemented
- FACT: Task and job commands working
EOF
}

#############################################################################
# STAGE 4: API AND FRONTEND PROMPTS
#############################################################################

create_prompt_api_rest() {
    cat > "$PROMPTS_DIR/api-rest.md" <<'EOF'
# Task: Implement REST API

**Task ID**: api-rest
**Phase**: API and Frontend
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: storage, config

## Objective
Implement REST API server for task management and run monitoring.

## Specifications
Read: docs/specifications/subsystem-api-rest.md

## Required Implementation

### 1. Package Structure
Location: `internal/api/`
Files:
- server.go - HTTP server setup
- routes.go - Route definitions
- handlers.go - Request handlers
- middleware.go - Auth, logging, CORS

### 2. HTTP Framework Decision
Research and choose:
- net/http (stdlib, minimal)
- gin (fast, popular)
- echo (lightweight, good middleware)

Recommendation: Start with net/http for simplicity, can refactor to gin/echo later if needed.

### 3. API Endpoints

#### Task Management
```
POST   /api/v1/tasks         - Create new task
GET    /api/v1/tasks         - List all tasks
GET    /api/v1/tasks/:id     - Get task details
DELETE /api/v1/tasks/:id     - Cancel task
```

#### Run Management
```
GET    /api/v1/runs          - List all runs
GET    /api/v1/runs/:id      - Get run details
GET    /api/v1/runs/:id/info - Get run-info.yaml
POST   /api/v1/runs/:id/stop - Stop running task
```

#### Message Bus
```
GET    /api/v1/messages      - Get all messages
GET    /api/v1/messages?after=<msg_id> - Get messages after ID
```

#### Health
```
GET    /api/v1/health        - Health check
GET    /api/v1/version       - Version info
```

### 4. Request/Response Models
```go
type TaskCreateRequest struct {
    ProjectID string            `json:"project_id"`
    TaskID    string            `json:"task_id"`
    AgentType string            `json:"agent_type"`
    Prompt    string            `json:"prompt"`
    Config    map[string]string `json:"config,omitempty"`
}

type RunResponse struct {
    RunID      string    `json:"run_id"`
    ProjectID  string    `json:"project_id"`
    TaskID     string    `json:"task_id"`
    Status     string    `json:"status"`
    StartTime  time.Time `json:"start_time"`
    EndTime    time.Time `json:"end_time,omitempty"`
    ExitCode   int       `json:"exit_code,omitempty"`
}
```

### 5. Middleware
- Logging (request/response times)
- CORS (allow frontend origin)
- Error handling (consistent JSON errors)
- Authentication stub (token validation placeholder)

### 6. Configuration
Add to config.yaml:
```yaml
api:
  host: "0.0.0.0"
  port: 8080
  cors_origins:
    - "http://localhost:3000"
  auth_enabled: false  # stub for future
```

### 7. Tests Required
Location: `test/integration/api_test.go`
- TestCreateTask
- TestListRuns
- TestGetRunInfo
- TestMessageBusEndpoint
- TestCORSHeaders
- TestErrorResponses

## Implementation Steps

1. **Research Phase** (10 minutes)
   - Compare net/http vs gin vs echo
   - Document decision in MESSAGE-BUS.md

2. **Implementation Phase** (45 minutes)
   - Create server.go with HTTP server setup
   - Implement all endpoint handlers
   - Add middleware (logging, CORS, errors)
   - Wire up storage and config dependencies

3. **Testing Phase** (30 minutes)
   - Write integration tests for all endpoints
   - Test CORS with curl
   - Test error handling

4. **IntelliJ Checks** (15 minutes)
   - Run all inspections
   - Fix any warnings
   - Ensure >80% test coverage

## Success Criteria
- All endpoints functional
- All tests passing
- CORS properly configured
- Error handling consistent
- IntelliJ checks clean

## Output
Log to MESSAGE-BUS.md:
- DECISION: HTTP framework choice and rationale
- FACT: REST API implemented
- FACT: All endpoints tested
EOF
}

create_prompt_api_sse() {
    cat > "$PROMPTS_DIR/api-sse.md" <<'EOF'
# Task: Implement SSE Log Streaming

**Task ID**: api-sse
**Phase**: API and Frontend
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: storage

## Objective
Implement Server-Sent Events (SSE) streaming for real-time log tailing.

## Specifications
Read: docs/specifications/subsystem-api-streaming.md

## Required Implementation

### 1. Package Structure
Location: `internal/api/`
Files:
- sse.go - SSE handler and streaming logic
- tailer.go - Log file tailing implementation
- discovery.go - Run discovery polling

### 2. SSE Endpoints

```
GET /api/v1/runs/:id/stream     - Stream logs for specific run
GET /api/v1/runs/stream/all     - Stream logs from all runs
GET /api/v1/messages/stream     - Stream message bus updates
```

### 3. SSE Event Format
```
event: log
data: {"run_id": "...", "line": "...", "timestamp": "..."}

event: status
data: {"run_id": "...", "status": "completed", "exit_code": 0}

event: message
data: {"msg_id": "...", "content": "...", "timestamp": "..."}

event: heartbeat
data: {}
```

### 4. Log Tailing Implementation

**Strategy**: Polling-based file tailing
- Poll stdout/stderr files every 100ms
- Track last read position (file offset)
- Detect file rotation/truncation
- Send new lines as SSE events

```go
type Tailer struct {
    filePath string
    offset   int64
    ticker   *time.Ticker
    events   chan SSEEvent
}

func (t *Tailer) Start() {
    // Poll file every 100ms
    // Read new content from offset
    // Send as SSE events
}
```

### 5. Run Discovery

**Problem**: Clients need to discover new runs as they're created
**Solution**: Poll runs directory every 1 second

```go
type RunDiscovery struct {
    runsDir     string
    knownRuns   map[string]bool
    ticker      *time.Ticker
    newRunChan  chan string
}

func (d *RunDiscovery) Poll() {
    // List runs directory
    // Compare with knownRuns
    // Notify on new runs
}
```

### 6. Concurrent Streaming

Support multiple clients streaming different runs:
- Each client gets dedicated goroutine
- Each run gets dedicated tailer
- Ref-counted tailers (start/stop based on client count)
- Proper cleanup on client disconnect

```go
type StreamManager struct {
    tailers map[string]*Tailer
    clients map[string][]*SSEClient
    mu      sync.RWMutex
}
```

### 7. Client Reconnection

Support `Last-Event-ID` header for resume:
- Client sends last received log line number
- Server resends from that position
- Prevents missing logs on reconnect

### 8. Configuration
Add to config.yaml:
```yaml
api:
  sse:
    poll_interval_ms: 100
    discovery_interval_ms: 1000
    heartbeat_interval_s: 30
    max_clients_per_run: 10
```

### 9. Tests Required
Location: `test/integration/sse_test.go`
- TestSSEStreaming
- TestLogTailing
- TestRunDiscovery
- TestMultipleClients
- TestClientReconnect
- TestHeartbeat

## Implementation Steps

1. **Research Phase** (15 minutes)
   - Study Go SSE libraries (standard vs third-party)
   - Research file tailing patterns
   - Document approach in MESSAGE-BUS.md

2. **Implementation Phase** (60 minutes)
   - Implement Tailer for log file polling
   - Implement SSE handler with event formatting
   - Implement RunDiscovery polling
   - Implement StreamManager for concurrent clients
   - Add reconnection support

3. **Testing Phase** (30 minutes)
   - Write integration tests
   - Test with multiple concurrent clients
   - Test reconnection scenarios
   - Test with curl (manual verification)

4. **IntelliJ Checks** (15 minutes)
   - Run all inspections
   - Fix any warnings
   - Ensure >80% test coverage

## Success Criteria
- Real-time log streaming working
- Multiple clients supported
- Reconnection working
- All tests passing
- No goroutine leaks

## Output
Log to MESSAGE-BUS.md:
- FACT: SSE streaming implemented
- FACT: Log tailing working
- FACT: Run discovery implemented
EOF
}

create_prompt_ui_frontend() {
    cat > "$PROMPTS_DIR/ui-frontend.md" <<'EOF'
# Task: Implement Monitoring UI

**Task ID**: ui-frontend
**Phase**: API and Frontend
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: api-rest, api-sse

## Objective
Build React-based web UI for monitoring task execution and viewing logs.

## Specifications
Read: docs/specifications/subsystem-ui-frontend.md

## Required Implementation

### 1. Technology Stack Decision

Research and choose:
- **Framework**: React vs Vue vs Svelte
- **Language**: TypeScript (recommended)
- **Build Tool**: Vite vs Create React App
- **UI Library**: Tailwind CSS vs Material-UI vs Chakra UI
- **SSE Client**: Native EventSource vs library

**Recommendation**: React + TypeScript + Vite + Tailwind CSS (modern, fast, popular)

### 2. Project Structure
Location: `frontend/`
```
frontend/
├── src/
│   ├── components/
│   │   ├── TaskList.tsx
│   │   ├── RunDetail.tsx
│   │   ├── LogViewer.tsx
│   │   ├── MessageBus.tsx
│   │   └── RunTree.tsx
│   ├── hooks/
│   │   ├── useSSE.ts
│   │   ├── useAPI.ts
│   │   └── useWebSocket.ts (future)
│   ├── api/
│   │   └── client.ts
│   ├── types/
│   │   └── index.ts
│   ├── App.tsx
│   └── main.tsx
├── package.json
├── tsconfig.json
├── vite.config.ts
└── tailwind.config.js
```

### 3. Core Components

#### TaskList Component
- Display all tasks and runs
- Filter by status (running, completed, failed)
- Sort by start time
- Click to view details

#### RunDetail Component
- Show run metadata (run_id, status, times, exit_code)
- Display run tree (parent-child relationships)
- Link to logs

#### LogViewer Component
- Real-time log streaming via SSE
- Auto-scroll to bottom
- Toggle between stdout/stderr
- Search/filter logs
- ANSI color support (terminal colors)

#### MessageBus Component
- Display all message bus messages
- Filter by agent/run
- Collapsible sections
- Auto-refresh

#### RunTree Component
- Visualize parent-child run relationships
- Expand/collapse nodes
- Color by status
- Click to navigate

### 4. SSE Integration

Custom hook for SSE:
```typescript
function useSSE(url: string, onMessage: (event: MessageEvent) => void) {
  useEffect(() => {
    const eventSource = new EventSource(url);

    eventSource.addEventListener('log', onMessage);
    eventSource.addEventListener('status', onMessage);
    eventSource.addEventListener('message', onMessage);

    eventSource.onerror = (err) => {
      console.error('SSE error:', err);
      // Reconnect logic
    };

    return () => eventSource.close();
  }, [url]);
}
```

### 5. API Client

```typescript
class APIClient {
  private baseURL: string;

  async getTasks(): Promise<Task[]> { ... }
  async getRuns(): Promise<Run[]> { ... }
  async getRunInfo(runId: string): Promise<RunInfo> { ... }
  async getMessages(): Promise<Message[]> { ... }
  async stopRun(runId: string): Promise<void> { ... }
}
```

### 6. Features

**Must Have**:
- Task list view
- Run detail view
- Real-time log streaming
- Message bus viewer
- Status indicators

**Nice to Have**:
- Search/filter logs
- Run tree visualization
- Dark mode toggle
- Export logs
- Run comparison

### 7. Styling
- Responsive design (desktop + mobile)
- Dark theme by default
- Color-coded status (green=success, red=error, yellow=running)
- Monospace font for logs
- Clean, minimal UI

### 8. Tests Required
Location: `frontend/tests/`

**Component Tests** (Vitest + React Testing Library):
- TaskList.test.tsx
- RunDetail.test.tsx
- LogViewer.test.tsx

**E2E Tests** (Playwright):
- test/e2e/ui_test.go (using Playwright MCP)
- Test full user flow: view tasks → click run → see logs

### 9. Development Setup
```bash
cd frontend
npm create vite@latest . -- --template react-ts
npm install
npm install -D tailwindcss postcss autoprefixer
npm install @tanstack/react-query  # for data fetching
npm run dev  # start dev server
```

### 10. Production Build
```bash
npm run build  # outputs to frontend/dist
# Serve via Go HTTP server or nginx
```

## Implementation Steps

1. **Research Phase** (20 minutes)
   - Compare React vs Vue vs Svelte
   - Choose UI library (Tailwind recommended)
   - Document decisions in MESSAGE-BUS.md

2. **Setup Phase** (15 minutes)
   - Create Vite + React + TypeScript project
   - Configure Tailwind CSS
   - Set up project structure

3. **Implementation Phase** (90 minutes)
   - Build TaskList component
   - Build RunDetail component
   - Build LogViewer with SSE
   - Build MessageBus component
   - Wire up API client

4. **Styling Phase** (30 minutes)
   - Apply Tailwind styles
   - Make responsive
   - Add dark theme

5. **Testing Phase** (30 minutes)
   - Write component tests
   - Write E2E test with Playwright
   - Manual browser testing

6. **IntelliJ Checks** (15 minutes)
   - Run linter (ESLint)
   - Check TypeScript errors
   - Verify build

## Success Criteria
- All components rendering
- SSE streaming working in browser
- Logs displaying in real-time
- UI responsive and styled
- All tests passing

## Output
Log to MESSAGE-BUS.md:
- DECISION: Technology stack choices and rationale
- FACT: Frontend implemented
- FACT: SSE streaming working in browser
- FACT: All components tested
EOF
}

#############################################################################
# CREATE ALL PROMPTS
#############################################################################

create_all_prompts() {
    log "PROGRESS: Creating all task prompts..."

    # Bootstrap prompts
    create_prompt_bootstrap_01
    create_prompt_bootstrap_02
    create_prompt_bootstrap_03
    create_prompt_bootstrap_04

    # Infrastructure prompts
    create_prompt_infra_storage
    create_prompt_infra_config
    create_prompt_infra_messagebus

    # Agent system prompts
    create_prompt_agent_protocol
    create_prompt_agent_claude
    create_prompt_agent_codex
    create_prompt_agent_gemini
    create_prompt_agent_perplexity
    create_prompt_agent_xai

    # Runner orchestration prompts
    create_prompt_runner_process
    create_prompt_runner_ralph
    create_prompt_runner_orchestration

    # API and frontend prompts
    create_prompt_api_rest
    create_prompt_api_sse
    create_prompt_ui_frontend

    # TODO: Add remaining prompts for:
    # - Test suites (5 prompts)
    # - Documentation (3 prompts)

    log "FACT: Task prompts created in $PROMPTS_DIR/"
}

#############################################################################
# AGENT EXECUTION
#############################################################################

run_agent_task() {
    local task_id="$1"
    local agent_type="$2"  # claude, codex, gemini
    local prompt_file="$3"
    local cwd="${4:-$PROJECT_ROOT}"

    log "PROGRESS: Starting task $task_id with $agent_type agent"

    # Run agent via run-agent.sh
    "${PROJECT_ROOT}/run-agent.sh" "$agent_type" "$cwd" "$prompt_file" &
    local pid=$!

    echo "$pid" > "${RUNS_DIR}/${task_id}.pid"

    log "FACT: Task $task_id started (PID: $pid)"
}

wait_for_tasks() {
    local -a task_ids=("$@")
    local timeout="${STAGE_TIMEOUT}"
    local start_time=$(date +%s)

    log "PROGRESS: Waiting for ${#task_ids[@]} tasks to complete (timeout: ${timeout}s)..."

    while true; do
        local all_done=true
        local elapsed=$(($(date +%s) - start_time))

        if [ "$elapsed" -gt "$timeout" ]; then
            log_error "TIMEOUT: Stage exceeded ${timeout}s"
            return 1
        fi

        for task_id in "${task_ids[@]}"; do
            local pid_file="${RUNS_DIR}/${task_id}.pid"
            if [ -f "$pid_file" ]; then
                local pid=$(cat "$pid_file")
                if ps -p "$pid" > /dev/null 2>&1; then
                    all_done=false
                fi
            fi
        done

        if $all_done; then
            break
        fi

        sleep 5
    done

    log_success "All tasks in stage completed"
}

check_task_success() {
    local task_id="$1"

    # Find the run directory for this task by searching prompt.md files for task ID
    local run_dir=$(find "$RUNS_DIR" -type f -name "prompt.md" -exec grep -l "^\\*\\*Task ID\\*\\*: $task_id" {} \; 2>/dev/null | head -1 | xargs dirname 2>/dev/null)

    if [ -z "$run_dir" ]; then
        log_error "Cannot find run directory for task $task_id"
        return 1
    fi

    # Check exit code in cwd.txt
    if grep -q "EXIT_CODE=0" "$run_dir/cwd.txt" 2>/dev/null; then
        log_success "Task $task_id completed successfully"
        return 0
    else
        log_error "Task $task_id failed"
        cat "$run_dir/agent-stderr.txt" | tail -20 >> "$ISSUES_FILE"
        return 1
    fi
}

#############################################################################
# STAGE EXECUTION
#############################################################################

run_stage_0_bootstrap() {
    log "=========================================="
    log "STAGE 0: BOOTSTRAP"
    log "=========================================="

    # Run all 4 bootstrap tasks in parallel
    run_agent_task "bootstrap-01" "codex" "$PROMPTS_DIR/bootstrap-01-structure.md"
    run_agent_task "bootstrap-02" "claude" "$PROMPTS_DIR/bootstrap-02-documentation.md"
    run_agent_task "bootstrap-03" "codex" "$PROMPTS_DIR/bootstrap-03-tooling.md"
    run_agent_task "bootstrap-04" "claude" "$PROMPTS_DIR/bootstrap-04-arch-review.md"

    # Wait for all to complete
    wait_for_tasks "bootstrap-01" "bootstrap-02" "bootstrap-03" "bootstrap-04"

    # Check success
    for task in bootstrap-01 bootstrap-02 bootstrap-03 bootstrap-04; do
        if ! check_task_success "$task"; then
            log_error "Stage 0 failed at task $task"
            return 1
        fi
    done

    log_success "STAGE 0 COMPLETE: Bootstrap successful"
}

run_stage_1_infrastructure() {
    log "=========================================="
    log "STAGE 1: CORE INFRASTRUCTURE"
    log "=========================================="

    # Run storage and config in parallel
    run_agent_task "infra-storage" "codex" "$PROMPTS_DIR/infra-storage.md"
    run_agent_task "infra-config" "codex" "$PROMPTS_DIR/infra-config.md"

    wait_for_tasks "infra-storage" "infra-config"

    # Check success before message bus (depends on storage)
    if ! check_task_success "infra-storage"; then
        log_error "Stage 1 failed: storage"
        return 1
    fi

    # Run message bus (depends on storage)
    run_agent_task "infra-messagebus" "codex" "$PROMPTS_DIR/infra-messagebus.md"
    wait_for_tasks "infra-messagebus"

    if ! check_task_success "infra-messagebus"; then
        log_error "Stage 1 failed: messagebus"
        return 1
    fi

    log_success "STAGE 1 COMPLETE: Core infrastructure implemented"
}

run_stage_2_agents() {
    log "=========================================="
    log "STAGE 2: AGENT SYSTEM"
    log "=========================================="

    # First: agent-protocol (must complete before backends)
    log "Phase 2a: Agent Protocol Interface"
    run_agent_task "agent-protocol" "codex" "$PROMPTS_DIR/agent-protocol.md"
    wait_for_tasks "agent-protocol"

    if ! check_task_success "agent-protocol"; then
        log_error "Stage 2 failed: agent-protocol"
        return 1
    fi

    # Then: All 5 agent backends in parallel
    log "Phase 2b: Agent Backend Implementations (5 parallel)"
    run_agent_task "agent-claude" "codex" "$PROMPTS_DIR/agent-claude.md"
    run_agent_task "agent-codex" "codex" "$PROMPTS_DIR/agent-codex.md"
    run_agent_task "agent-gemini" "codex" "$PROMPTS_DIR/agent-gemini.md"
    run_agent_task "agent-perplexity" "codex" "$PROMPTS_DIR/agent-perplexity.md"
    run_agent_task "agent-xai" "codex" "$PROMPTS_DIR/agent-xai.md"

    wait_for_tasks "agent-claude" "agent-codex" "agent-gemini" "agent-perplexity" "agent-xai"

    # Check all backends succeeded
    for task in agent-claude agent-codex agent-gemini agent-perplexity agent-xai; do
        if ! check_task_success "$task"; then
            log_error "Stage 2 failed at task $task"
            return 1
        fi
    done

    log_success "STAGE 2 COMPLETE: Agent system implemented"
}

run_stage_3_runner() {
    log "=========================================="
    log "STAGE 3: RUNNER ORCHESTRATION"
    log "=========================================="

    # Sequential execution (dependencies)
    log "Step 3.1: Process Management"
    run_agent_task "runner-process" "codex" "$PROMPTS_DIR/runner-process.md"
    wait_for_tasks "runner-process"

    if ! check_task_success "runner-process"; then
        log_error "Stage 3 failed: runner-process"
        return 1
    fi

    log "Step 3.2: Ralph Loop"
    run_agent_task "runner-ralph" "codex" "$PROMPTS_DIR/runner-ralph.md"
    wait_for_tasks "runner-ralph"

    if ! check_task_success "runner-ralph"; then
        log_error "Stage 3 failed: runner-ralph"
        return 1
    fi

    log "Step 3.3: Run Orchestration"
    run_agent_task "runner-orchestration" "codex" "$PROMPTS_DIR/runner-orchestration.md"
    wait_for_tasks "runner-orchestration"

    if ! check_task_success "runner-orchestration"; then
        log_error "Stage 3 failed: runner-orchestration"
        return 1
    fi

    log_success "STAGE 3 COMPLETE: Runner orchestration implemented"
}

run_stage_4_api() {
    log "=========================================="
    log "STAGE 4: API AND FRONTEND"
    log "=========================================="

    # All 3 tasks in parallel (no hard dependencies)
    log "Starting API and UI components in parallel..."
    run_agent_task "api-rest" "codex" "$PROMPTS_DIR/api-rest.md"
    run_agent_task "api-sse" "codex" "$PROMPTS_DIR/api-sse.md"
    run_agent_task "ui-frontend" "codex" "$PROMPTS_DIR/ui-frontend.md"

    log "PROGRESS: Waiting for 3 tasks to complete (timeout: ${STAGE_TIMEOUT}s)..."
    wait_for_tasks "api-rest" "api-sse" "ui-frontend"

    # Check each task
    local failed=0
    if ! check_task_success "api-rest"; then
        log_error "Task api-rest failed"
        failed=1
    fi

    if ! check_task_success "api-sse"; then
        log_error "Task api-sse failed"
        failed=1
    fi

    if ! check_task_success "ui-frontend"; then
        log_error "Task ui-frontend failed"
        failed=1
    fi

    if [ $failed -eq 1 ]; then
        log_error "Stage 4 failed: One or more tasks failed"
        return 1
    fi

    log_success "STAGE 4 COMPLETE: API and frontend implemented"
}

#############################################################################
# MAIN EXECUTION
#############################################################################

main() {
    log "======================================================================"
    log "CONDUCTOR LOOP - PARALLEL IMPLEMENTATION ORCHESTRATION"
    log "======================================================================"
    log "Project Root: $PROJECT_ROOT"
    log "Message Bus: $MESSAGE_BUS"
    log "Max Parallel: $MAX_PARALLEL_AGENTS agents"
    log "======================================================================"

    # Create all prompts
    create_all_prompts

    # Execute stages
    if ! run_stage_0_bootstrap; then
        log_error "FATAL: Stage 0 (Bootstrap) failed"
        exit 1
    fi

    if ! run_stage_1_infrastructure; then
        log_error "FATAL: Stage 1 (Infrastructure) failed"
        exit 1
    fi

    if ! run_stage_2_agents; then
        log_error "FATAL: Stage 2 (Agent System) failed"
        exit 1
    fi

    if ! run_stage_3_runner; then
        log_error "FATAL: Stage 3 (Runner Orchestration) failed"
        exit 1
    fi

    if ! run_stage_4_api; then
        log_error "FATAL: Stage 4 (API and Frontend) failed"
        exit 1
    fi

    # TODO: Add remaining stages:
    # run_stage_5_testing
    # run_stage_6_documentation

    log "======================================================================"
    log_success "IMPLEMENTATION COMPLETE"
    log "======================================================================"
    log "Review MESSAGE-BUS.md for full trace"
    log "Review ISSUES.md for any blockers"
    log "Next: Run acceptance tests"
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
