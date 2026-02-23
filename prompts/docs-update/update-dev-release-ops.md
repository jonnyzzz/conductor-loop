# Docs Update: Dev Docs — Release, Security, Testing, Contributing, Dev Setup, Self-Update

You are a documentation update agent. Update docs to match FACTS (facts take priority over existing content).

## Files to update (overwrite each file in-place)

1. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/release-checklist.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/security-review-2026-02-23.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/testing.md`
4. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/contributing.md`
5. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/development-setup.md`
6. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/self-update.md`
7. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/feature-requests-project-goal-manual-workflows.md`
8. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/documentation-site.md`

## Facts sources (read ALL of these first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-issues-decisions.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-suggested-tasks.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runs-conductor.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
```

## Verify against source

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Go version requirement
cat go.mod | head -5

# CI/CD
ls .github/workflows/ 2>/dev/null | head -10
cat .github/workflows/*.yml 2>/dev/null | grep -E "go-version:|GO_VERSION" | head -5

# Security: what was fixed in the 2026-02-23 security review
git log --oneline | grep -iE "security|fix|cve|auth|token" | head -10
git show ab5ea6e --stat 2>/dev/null | head -20

# Test structure
ls internal/ | head -20
find . -name "*_test.go" | grep -v vendor | wc -l

# self-update: install script
cat install.sh | head -60
cat run-agent.cmd | head -40
grep -rn "self.update\|selfUpdate\|update.*binary\|VERSION" cmd/ internal/ --include="*.go" | head -15
```

## Rules

- **Facts override docs** — if a fact contradicts a doc, update the doc to match the fact
- Key fixes:
  - Go version: update to actual version from go.mod
  - Release checklist: verify against what was actually done in the v1 release tasks
  - Security review: update with actual remediated findings (ab5ea6e commit: GHA SHA pins, Docker base, CI permissions scoped, linter pinned, inline token docs removed)
  - Testing: update test count/coverage if stale
  - Development-setup: update Go version requirement
  - Self-update: verify install.sh and run-agent.cmd paths and behavior
- For security-review-2026-02-23.md: this is a dated document — preserve history but add remediation notes
- Do not rewrite from scratch — targeted corrections only

## Output

Overwrite each file in-place with corrections applied.
Write a summary to `output.md` listing what changed in each file.
