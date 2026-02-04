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

    # TODO: Add remaining prompts for:
    # - infra-config
    # - All agent backends (6 prompts)
    # - Runner components (3 prompts)
    # - API components (3 prompts)
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

    # TODO: Add remaining stages:
    # run_stage_2_agents
    # run_stage_3_runner
    # run_stage_4_api
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
