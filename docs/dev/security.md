# Security

## Token Leak Prevention

conductor-loop handles API tokens for multiple agent backends (Anthropic, Google,
OpenAI/Codex, xAI, Perplexity). Accidental token commits must be prevented.

### Pre-commit and Pre-push Hooks

Two git hooks are installed to block accidental secret commits:

- `.git/hooks/pre-commit` — runs `gitleaks protect --staged` before each commit
- `.git/hooks/pre-push` — runs `gitleaks protect` before each push

#### Installing the Hooks

After cloning, install the hooks:

```bash
# Install gitleaks
brew install gitleaks    # macOS
# or: go install github.com/gitleaks/gitleaks/v8@latest

# Install the hooks (from repo root)
bash scripts/install-hooks.sh
```

Or install manually:

```bash
# pre-commit hook
cat > .git/hooks/pre-commit <<'EOF'
#!/usr/bin/env bash
if command -v gitleaks >/dev/null 2>&1; then
  gitleaks protect --staged || exit 1
fi
EOF
chmod +x .git/hooks/pre-commit

# pre-push hook
cat > .git/hooks/pre-push <<'EOF'
#!/usr/bin/env bash
if command -v gitleaks >/dev/null 2>&1; then
  gitleaks protect || exit 1
fi
EOF
chmod +x .git/hooks/pre-push
```

#### Verifying Hook Installation

```bash
ls -la .git/hooks/pre-commit .git/hooks/pre-push
```

Both files should be executable (`-rwxr-xr-x`).

#### Smoke Test

To verify the hook blocks a commit with a fake secret:

```bash
echo 'ANTHROPIC_API_KEY=sk-ant-testFAKEFAKE1234567890' > /tmp/test-secret-DO-NOT-COMMIT.txt
cp /tmp/test-secret-DO-NOT-COMMIT.txt .
git add test-secret-DO-NOT-COMMIT.txt
git commit -m "test: should be blocked" && echo "FAIL: hook did not block" || echo "PASS: hook blocked commit"
git restore --staged test-secret-DO-NOT-COMMIT.txt
rm test-secret-DO-NOT-COMMIT.txt
```

### Historical Token Scan

To scan the full git history for accidentally committed tokens:

```bash
# Install gitleaks if needed
brew install gitleaks

# Scan all commits
cd /path/to/conductor-loop
gitleaks detect --source . --log-opts="HEAD" \
  --report-format json --report-path /tmp/gitleaks-report.json
echo "Exit code: $?"

# Review findings
cat /tmp/gitleaks-report.json | python3 -m json.tool | head -100
```

If any real tokens are found:
1. Document the finding in `docs/security/token-leak-findings.md`
2. Assess severity (environment, whether tokens were rotated)
3. Propose remediation via BFG Repo Cleaner or `git filter-branch`
4. **Do NOT execute history rewrite without explicit sign-off**

### Environment Sanitization

The `internal/runner/env_sanitize.go` module ensures each agent subprocess
only receives its specific API key, not all keys in the environment. See
`prompts/tasks/env-sanitization.md` for details.

### Known High-Risk Paths

Files that should never contain real tokens:
- `config.local.yaml`, `config.docker.yaml`, `config.yaml`
- `runs/` directory (agent output artifacts)
- `.env`, `*.env`, `*.secret` files

These patterns are excluded from git via `.gitignore`.
