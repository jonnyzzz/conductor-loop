# Contributing to Conductor Loop

Thank you for your interest in contributing to Conductor Loop! This guide will help you get started with contributing to our agent orchestration framework.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How to Contribute](#how-to-contribute)
- [Development Setup](#development-setup)
- [Running Tests](#running-tests)
- [Code Style](#code-style)
- [Commit Message Format](#commit-message-format)
- [Pull Request Process](#pull-request-process)
- [Review Guidelines](#review-guidelines)
- [Documentation Requirements](#documentation-requirements)
- [Testing Requirements](#testing-requirements)

## Code of Conduct

### Our Pledge

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone, regardless of age, body size, visible or invisible disability, ethnicity, sex characteristics, gender identity and expression, level of experience, education, socio-economic status, nationality, personal appearance, race, religion, or sexual identity and orientation.

### Our Standards

Examples of behavior that contributes to a positive environment:

- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

Examples of unacceptable behavior:

- The use of sexualized language or imagery and unwelcome sexual attention or advances
- Trolling, insulting/derogatory comments, and personal or political attacks
- Public or private harassment
- Publishing others' private information without explicit permission
- Other conduct which could reasonably be considered inappropriate in a professional setting

### Enforcement

Instances of abusive, harassing, or otherwise unacceptable behavior may be reported by contacting the project team. All complaints will be reviewed and investigated promptly and fairly.

## How to Contribute

There are many ways to contribute to Conductor Loop:

### Reporting Bugs

If you find a bug, please open an issue on GitHub with:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Your environment (Go version, OS, etc.)
- Relevant logs or error messages

**Before submitting**, please check if the issue already exists.

### Suggesting Enhancements

We welcome enhancement suggestions! Please open an issue with:

- A clear, descriptive title
- Detailed description of the proposed feature
- Use cases and motivation
- Possible implementation approach (optional)
- Any relevant examples or mockups

### Contributing Code

1. **Fork the repository** on GitHub
2. **Clone your fork** locally
3. **Create a branch** for your changes
4. **Make your changes** with tests
5. **Run tests** and linters
6. **Commit your changes** following our commit format
7. **Push to your fork**
8. **Open a Pull Request**

### Contributing Documentation

Documentation improvements are always welcome! This includes:

- Fixing typos or unclear wording
- Adding examples
- Improving API documentation
- Writing tutorials or guides

### Participating in Discussions

Join our [GitHub Discussions](https://github.com/jonnyzzz/conductor-loop/discussions) to:

- Ask questions
- Share ideas
- Help other users
- Discuss future direction

## Development Setup

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go**: Version 1.24 or higher ([installation guide](https://go.dev/doc/install))
- **Git**: Version 2.30 or higher
- **Make**: For running build tasks
- **Docker**: Version 20.10+ (optional, for Docker tests)
- **Node.js**: Version 18+ (optional, for frontend development)

### Initial Setup

1. **Fork and Clone the Repository**

   ```bash
   # Fork the repository on GitHub, then clone your fork
   git clone https://github.com/YOUR-USERNAME/conductor-loop.git
   cd conductor-loop
   ```

2. **Add Upstream Remote**

   ```bash
   git remote add upstream https://github.com/jonnyzzz/conductor-loop.git
   git fetch upstream
   ```

3. **Install Go Dependencies**

   ```bash
   go mod download
   ```

4. **Install Development Tools**

   ```bash
   # Install golangci-lint for linting
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

   # Install delve for debugging (optional)
   go install github.com/go-delve/delve/cmd/dlv@latest
   ```

5. **Verify Setup**

   ```bash
   # Build the project
   make build

   # Run tests
   make test
   ```

### Project Structure

Understanding the project structure will help you navigate the codebase:

```
conductor-loop/
├── cmd/                    # Command-line applications
│   ├── conductor/          # Main server binary
│   └── run-agent/          # Agent runner binary
├── internal/               # Private application code
│   ├── agent/              # Agent implementations
│   ├── api/                # REST API and SSE
│   ├── config/             # Configuration management
│   ├── messagebus/         # Message bus system
│   ├── runner/             # Task orchestration
│   └── storage/            # Storage layer
├── test/                   # Test files
│   ├── unit/               # Unit tests
│   ├── integration/        # Integration tests
│   ├── docker/             # Docker tests
│   ├── performance/        # Benchmark tests
│   └── acceptance/         # End-to-end tests
├── docs/                   # Documentation
│   ├── user/               # User-facing documentation
│   ├── dev/                # Developer documentation
│   └── specifications/     # Technical specifications
├── web/                    # Frontend (plain HTML/CSS/JS)
└── scripts/                # Build and utility scripts
```

### Building the Project

```bash
# Build all binaries
make build

# Build specific binary
go build -o bin/conductor ./cmd/conductor
go build -o bin/run-agent ./cmd/run-agent

# Build with optimizations
go build -ldflags="-s -w" -o bin/conductor ./cmd/conductor
```

### Running Locally

1. **Create a Configuration File**

   ```bash
   cat > config.yaml <<EOF
   agents:
     codex:
       type: codex
       token_file: ./tokens/codex.token
       timeout: 300

   defaults:
     agent: codex

   storage:
     runs_dir: ./runs

   api:
     host: localhost
     port: 14355
   EOF
   ```

2. **Start the Server**

   ```bash
   ./bin/conductor --config config.yaml --root $(pwd)
   ```

3. **Access the Web UI**

   Open http://localhost:14355 in your browser.

## Running Tests

We maintain a comprehensive test suite across multiple levels.

### Quick Test Commands

```bash
# Run all tests
make test

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./internal/storage/

# Run a specific test
go test -run TestMessageBusPost ./internal/messagebus/

# Run tests with race detector (important for concurrency)
go test -race ./...
```

### Test Types

#### Unit Tests

Unit tests are located alongside the code they test:

```bash
# Run all unit tests
go test ./internal/...

# Run unit tests in test/unit/
go test ./test/unit/...
```

#### Integration Tests

Integration tests verify interactions between components:

```bash
# Run integration tests
go test ./test/integration/...

# Run with verbose output
go test -v ./test/integration/...
```

#### Docker Tests

Docker tests verify containerized deployment:

```bash
# Run Docker tests (requires Docker)
go test ./test/docker/...
```

#### Performance Tests

Benchmark tests measure performance:

```bash
# Run benchmarks
go test -bench=. ./test/performance/

# With memory statistics
go test -bench=. -benchmem ./test/performance/
```

#### Acceptance Tests

End-to-end tests verify complete workflows:

```bash
# Run acceptance tests
go test ./test/acceptance/...
```

### Coverage

We aim for at least 80% test coverage:

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# Check if coverage meets threshold
go test -cover ./... | grep -E 'coverage: [0-9]+' | \
  awk '{if ($2+0 < 80) exit 1}'
```

### Race Detection

Always run tests with the race detector when working with concurrent code:

```bash
# Run with race detector
go test -race ./...

# For a specific package
go test -race ./internal/messagebus/
```

## Code Style

We follow standard Go conventions and use automated tools to enforce consistency.

### Go Conventions

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Follow [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- Keep functions small and focused (prefer <50 lines)
- Use meaningful variable names
- Avoid premature optimization

### Formatting

All code must be formatted with `gofmt`:

```bash
# Format all files
go fmt ./...

# Check if files are formatted (returns non-zero if not)
test -z "$(gofmt -l .)"
```

Our CI will reject PRs with unformatted code.

### Linting

We use `golangci-lint` to enforce code quality:

```bash
# Run all linters
golangci-lint run

# Run with auto-fixes
golangci-lint run --fix

# Run specific linter
golangci-lint run --disable-all --enable=errcheck
```

Install golangci-lint:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Naming Conventions

- **Packages**: Short, lowercase, single-word names (e.g., `runner`, `storage`, `messagebus`)
- **Interfaces**: Nouns or noun phrases (e.g., `Agent`, `Storage`, `MessageWriter`)
- **Functions**: Verbs or verb phrases (e.g., `StartAgent`, `WriteMessage`, `LoadConfig`)
- **Variables**: CamelCase for exported, camelCase for unexported
- **Constants**: CamelCase for exported, camelCase for unexported

### Error Handling

Always check and handle errors appropriately:

```go
// Good: Check and wrap errors with context
data, err := os.ReadFile(path)
if err != nil {
    return fmt.Errorf("reading config file %s: %w", path, err)
}

// Bad: Ignoring errors
data, _ := os.ReadFile(path)  // Never do this
```

### Comments

- Add package comments for all packages
- Document all exported functions, types, and constants
- Use complete sentences
- Keep comments current with code changes

```go
// Package storage provides persistent storage for task runs.
// It implements a hierarchical file-based storage system.
package storage

// LoadRun loads a run's metadata from disk.
// It returns an error if the run does not exist or cannot be read.
func LoadRun(runID string) (*Run, error) {
    // Implementation...
}
```

### Import Organization

Organize imports into three groups:

```go
import (
    // Standard library
    "context"
    "fmt"
    "os"

    // Third-party packages
    "github.com/pkg/errors"
    "gopkg.in/yaml.v3"

    // Project packages
    "github.com/jonnyzzz/conductor-loop/internal/runner"
    "github.com/jonnyzzz/conductor-loop/internal/storage"
)
```

## Commit Message Format

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type

Must be one of:

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Code style changes (formatting, missing semicolons, etc.)
- **refactor**: Code changes that neither fix a bug nor add a feature
- **perf**: Performance improvements
- **test**: Adding or updating tests
- **chore**: Maintenance tasks (dependencies, tooling, etc.)
- **ci**: CI/CD changes

### Scope

Use subsystem names:

- **agent**: Agent protocol and implementations
- **runner**: Task orchestration and Ralph Loop
- **storage**: Storage layer
- **messagebus**: Message bus system
- **config**: Configuration management
- **api**: REST API and SSE
- **ui**: Frontend UI
- **test**: Test infrastructure
- **ci**: CI/CD configuration

### Subject

- Use imperative mood ("add" not "added" or "adds")
- Don't capitalize the first letter
- No period at the end
- Maximum 50 characters

### Body (Optional)

- Use imperative mood
- Explain what and why, not how
- Wrap at 72 characters
- Separate from subject with blank line

### Footer (Optional)

- Reference issues: `Refs: #123`
- Breaking changes: `BREAKING CHANGE: description`
- Close issues: `Closes: #123`

### Examples

```
feat(agent): add Claude backend with stdio handling

Implement the Claude agent backend using stdio-based communication.
Adds support for streaming responses and proper signal handling.

Refs: #42
```

```
fix(storage): prevent race condition in atomic writes

Use file locking to prevent concurrent writes to the same file.
Adds integration test to verify fix.

Fixes: #67
```

```
docs(api): update REST API reference

Add examples for SSE streaming endpoints and clarify
authentication requirements.
```

```
test(runner): add benchmark for task spawning

Measure performance impact of spawning 100 concurrent tasks.
Baseline: ~500ms for 100 tasks.
```

## Pull Request Process

### Before Opening a PR

Ensure your changes meet these requirements:

- [ ] All tests pass: `make test`
- [ ] Race detector passes: `go test -race ./...`
- [ ] Linter passes: `make lint`
- [ ] Code is formatted: `go fmt ./...`
- [ ] Coverage is maintained or improved
- [ ] Commit messages follow the format
- [ ] Documentation is updated
- [ ] Branch is rebased on latest main

### Opening a PR

1. **Push Your Branch**

   ```bash
   git push origin feature/my-feature
   ```

2. **Create Pull Request**

   Go to GitHub and create a pull request from your branch to `main`.

3. **Fill Out PR Template**

   Provide:
   - Clear description of changes
   - Motivation and context
   - Link to related issues
   - Testing performed
   - Screenshots (if UI changes)

### PR Title Format

Use the same format as commit messages:

```
feat(storage): add atomic write support
```

### PR Description Template

```markdown
## Summary
Brief description of changes

## Motivation
Why is this change needed?

## Changes
- Change 1
- Change 2
- Change 3

## Testing
How was this tested?

## Related Issues
Refs: #123
Closes: #456

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Changelog updated (if applicable)
- [ ] Breaking changes noted
```

### During Review

- Be responsive to feedback
- Make requested changes promptly
- Ask questions if feedback is unclear
- Keep discussions focused and professional
- Push fixes as new commits (don't force-push during review)

### CI Checks

Your PR must pass all CI checks:

- **Build**: Code compiles successfully
- **Tests**: All tests pass
- **Lint**: Code passes linting
- **Coverage**: Coverage meets threshold

If CI fails, fix the issues and push new commits.

### Merging

Once approved and CI passes:

1. Squash commits if requested by reviewers
2. Rebase on latest main if needed
3. Maintainer will merge the PR

## Review Guidelines

### For Contributors

When your PR is under review:

- **Be patient**: Reviews may take time
- **Be open**: Accept feedback gracefully
- **Be thorough**: Address all comments
- **Be responsive**: Reply to questions promptly

### For Reviewers

When reviewing PRs:

- **Be kind**: Assume good intentions
- **Be constructive**: Provide actionable feedback
- **Be thorough**: Check code, tests, and docs
- **Be timely**: Review within 2-3 business days

### What to Look For

Reviewers should check:

- **Correctness**: Does the code work as intended?
- **Tests**: Are there adequate tests?
- **Performance**: Any performance concerns?
- **Security**: Any security implications?
- **Maintainability**: Is the code clear and maintainable?
- **Documentation**: Is documentation updated?
- **Style**: Does it follow our conventions?

### Review Process

1. **Initial Review**: Check overall approach and design
2. **Detailed Review**: Review code line-by-line
3. **Testing Review**: Verify tests are adequate
4. **Final Review**: Check that feedback is addressed

### Approval Requirements

PRs require:

- At least 1 approval from a maintainer
- All CI checks passing
- No unresolved conversations

## Documentation Requirements

All changes must include appropriate documentation updates.

### When to Update Documentation

Update documentation when you:

- Add a new feature
- Change existing behavior
- Add/change configuration options
- Add/change API endpoints
- Fix a bug that affects user behavior

### What to Document

- **User Documentation** (`docs/user/`): How to use the feature
- **Developer Documentation** (`docs/dev/`): How the feature works internally
- **API Documentation**: OpenAPI specs or inline comments
- **Code Comments**: Complex logic or non-obvious behavior
- **Examples**: Usage examples in `docs/examples/`

### Documentation Standards

- Write in clear, simple English
- Use present tense
- Use active voice
- Include code examples
- Keep examples up-to-date
- Format with Markdown

### Example Structure

```markdown
# Feature Name

## Overview
Brief description of what the feature does.

## Usage
How to use the feature with examples.

## Configuration
Configuration options and their defaults.

## Examples
Real-world usage examples.

## Troubleshooting
Common issues and solutions.

## See Also
Links to related documentation.
```

## Testing Requirements

All code contributions must include appropriate tests.

### Test Coverage

- Minimum 80% code coverage for new code
- Maintain or improve overall coverage
- Cover both happy path and error cases
- Test edge cases and boundary conditions

### When to Add Tests

Add tests when you:

- Add a new feature
- Fix a bug
- Refactor existing code
- Change API behavior

### Test Structure

Use table-driven tests:

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "hello",
            want:    "HELLO",
            wantErr: false,
        },
        {
            name:    "empty input",
            input:   "",
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Transform(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Transform() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Transform() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Types Required

Depending on your change:

- **Unit tests**: Always required
- **Integration tests**: For interactions between components
- **Benchmark tests**: For performance-critical code
- **Example tests**: For public API functions

### Concurrency Testing

For concurrent code, always include:

- Race detector tests: `go test -race`
- Stress tests: Run tests multiple times
- Deadlock detection: Verify no goroutine leaks

### Test Best Practices

- Keep tests simple and focused
- Use descriptive test names
- Test one thing per test
- Avoid test interdependencies
- Clean up resources (files, goroutines, etc.)
- Use `t.Helper()` for test helper functions

## Getting Help

If you need help:

- Read the [Developer Guide](development-setup.md)
- Check [Architecture Documentation](architecture.md)
- Review [Specifications](../specifications/)
- Ask in [GitHub Discussions](https://github.com/jonnyzzz/conductor-loop/discussions)
- Open an issue for bugs

## Recognition

Contributors are recognized in several ways:

- Listed in the repository contributors
- Mentioned in release notes for significant contributions
- Invited to become maintainers after consistent quality contributions

## License

By contributing to Conductor Loop, you agree that your contributions will be
licensed under the Apache License 2.0 (see [LICENSE](../../LICENSE) and
[NOTICE](../../NOTICE)).

---

Thank you for contributing to Conductor Loop! Your efforts help make this project better for everyone.
