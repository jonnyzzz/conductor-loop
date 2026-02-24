# Technical Debt Map Agent

You are a Research/Planning Agent. Your goal is to map the technical debt and structural inconsistencies in the project.

## Inputs
- `docs/facts/FACTS-issues-decisions.md` (Historical decisions and known issues)
- `docs/facts/FACTS-architecture.md` (Architecture state)
- `cmd/conductor/` (Source code)
- `internal/` (Source code)

## Tasks
1.  **Deferred Issues**: Review `FACTS-issues-decisions.md`. What deferred issues (marked PARTIALLY RESOLVED or RESOLVED with "deferred" notes) need resolution?
2.  **Binary Port Mismatch**: Verify the binary port mismatch status. Does `conductor` binary help message match the actual default port in code? (Check `cmd/conductor/main.go` vs `internal/api/server.go` vs `bin/conductor --help`).
3.  **Spec vs Code Drift**: Identify areas where `docs/specifications/` (if any, or architecture facts) diverge from `internal/` implementation.
4.  **Test Coverage Gaps**: Identify major subsystems in `internal/` that lack corresponding tests in `test/` or `*_test.go` files.

## Output
Create a new file: `docs/roadmap/technical-debt.md`
Format: Markdown.
Sections:
- **Deferred Maintenance**: Issues explicitly deferred that are now due.
- **Inconsistencies**: Binary/Source mismatches (like ports), Spec/Code drift.
- **Coverage Gaps**: Critical areas with low/no testing.
- **Refactoring Needs**: Areas identified as "monolithic" or "complex" in facts.

## Constraints
- Do not modify existing files.
- Only create `docs/roadmap/technical-debt.md`.
- Read files using absolute paths or relative to project root.
- Use `grep_search` to check ports and code structures.

## Final Action
Commit the new file:
`git add docs/roadmap/technical-debt.md`
`git commit -m "docs(roadmap): add technical debt map"`
