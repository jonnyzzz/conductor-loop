# Task: Repository Token Leak Audit

## Context

Task ID: `task-20260223-155350-repo-history-token-leak-audit`
Priority: P1 — Security / Release / Delivery

The conductor-loop project handles API tokens for multiple agent backends (Anthropic, Google, OpenAI/Codex, xAI, Perplexity). The git history spans 200+ commits across a period of rapid iteration and multi-agent development. Prior security work (commit `ab5ea6e`, task `task-20260223-071800-security-audit-followup-action-plan`) fixed five open findings including removing inline token docs and scoping CI permissions. However, no full scan of git history for accidentally committed secrets has been performed.

Known risk surface:
- `config.local.yaml` and `config.docker.yaml` are present in the repo root and may have been committed with real tokens at any point
- `runs/` directory contains agent stdout/stderr — agents may have echoed tokens
- `.env`-style files or test fixtures may contain real credentials
- The `internal/agent/` subdirectories (anthropic, gemini, xai, openai, opencode) each interface directly with API tokens

The repo also lacks pre-commit and pre-push hooks to block future accidental commits of secrets.

## Requirements

- Run a full git history token scan using `truffleHog` or `gitleaks` (prefer `gitleaks` — faster, more configurable)
- Scan ALL commits in git history, not just the working tree
- Target the conductor-loop repo at `/Users/jonnyzzz/Work/conductor-loop`
- Produce a findings report: file path, commit SHA, secret type, line content (redacted to first 6 + last 2 chars if a real token is found)
- If any real tokens are found in history: document the finding, assess severity, and propose a remediation plan (git filter-branch or BFG Repo Cleaner) — do NOT execute remediation without explicit user confirmation
- Check the following high-risk paths explicitly:
  - `config.local.yaml`, `config.docker.yaml`, `config.yaml`
  - `runs/` directory (agent output artifacts)
  - `.env`, `*.env`, `*.secret` files
  - Any file matching `*token*`, `*key*`, `*credential*`, `*secret*`
- Install pre-commit hook at `.git/hooks/pre-commit` that runs `gitleaks protect --staged` before each commit
- Install pre-push hook at `.git/hooks/pre-push` that runs `gitleaks protect` before each push
- Both hooks must exit non-zero (blocking the operation) if secrets are detected
- Document the hook installation in `docs/dev/security.md` (create if absent) with instructions for developers to verify hook installation after cloning

## Acceptance Criteria

- `gitleaks` (or equivalent) scan completes with zero findings, OR all findings are documented in `docs/security/token-leak-findings.md` with severity and status
- Pre-commit hook installed at `.git/hooks/pre-commit` — `ls -la .git/hooks/pre-commit` shows executable file
- Pre-push hook installed at `.git/hooks/pre-push` — `ls -la .git/hooks/pre-push` shows executable file
- Hook smoke test passes: create a temp file with a fake secret pattern (`ANTHROPIC_API_KEY=sk-ant-FAKEFAKEFAKE`), `git add` it, `git commit` is blocked with a non-zero exit code
- Remove the temp file and verify normal commits pass through
- `docs/dev/security.md` contains a "Token Leak Prevention" section with hook setup instructions

## Verification

```bash
# 1. Install gitleaks if not present
brew install gitleaks  # macOS
# or: go install github.com/zricethezav/gitleaks/v8@latest

# 2. Scan full git history
cd /Users/jonnyzzz/Work/conductor-loop
gitleaks detect --source . --log-opts="HEAD" --report-format json --report-path /tmp/gitleaks-report.json
echo "Exit code: $?"

# 3. Review report
cat /tmp/gitleaks-report.json | python3 -m json.tool | head -100

# 4. Verify pre-commit hook blocks secrets
echo 'ANTHROPIC_API_KEY=sk-ant-testFAKEFAKE1234567890' > /tmp/test-secret.txt
cp /tmp/test-secret.txt /Users/jonnyzzz/Work/conductor-loop/test-secret-DO-NOT-COMMIT.txt
git add test-secret-DO-NOT-COMMIT.txt
git commit -m "test: should be blocked" && echo "FAIL: hook did not block" || echo "PASS: hook blocked commit"
git restore --staged test-secret-DO-NOT-COMMIT.txt
rm test-secret-DO-NOT-COMMIT.txt

# 5. Verify hooks are installed and executable
ls -la .git/hooks/pre-commit .git/hooks/pre-push
```
