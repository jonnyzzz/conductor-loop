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
