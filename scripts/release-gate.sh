#!/usr/bin/env bash
# scripts/release-gate.sh — Release readiness gate for conductor-loop
#
# Runs all release checks in sequence. Each check prints [PASS] or [FAIL] <reason>.
# Exit code is 0 only if ALL checks pass.
#
# Usage: ./scripts/release-gate.sh [--skip-ci] [--skip-install]

set -uo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

# ── flags ────────────────────────────────────────────────────────────────────
skip_ci=0
skip_install=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-ci)       skip_ci=1; shift ;;
    --skip-install)  skip_install=1; shift ;;
    -h|--help)
      echo "Usage: $0 [--skip-ci] [--skip-install]"
      echo "  --skip-ci       Skip GitHub CI status check (offline use)"
      echo "  --skip-install  Skip install path smoke test"
      exit 0 ;;
    *) echo "unknown argument: $1"; exit 1 ;;
  esac
done

# ── helpers ──────────────────────────────────────────────────────────────────
PASS_COUNT=0
FAIL_COUNT=0

pass() { echo "[PASS] $*"; ((PASS_COUNT++)) || true; }
fail() { echo "[FAIL] $*"; ((FAIL_COUNT++)) || true; }
skip_check() { echo "[SKIP] $*"; }

has_cmd() { command -v "$1" >/dev/null 2>&1; }

# ── checks ───────────────────────────────────────────────────────────────────

# 1. CI green
if [[ "$skip_ci" -eq 1 ]]; then
  skip_check "CI status (--skip-ci)"
elif has_cmd gh; then
  ci_result="$(gh run list --branch main --status completed --limit 5 --json conclusion,workflowName 2>/dev/null || echo "error")"
  if [[ "$ci_result" == "error" ]]; then
    fail "GitHub CI: could not query CI status (gh run list failed)"
  elif echo "$ci_result" | grep -q '"conclusion":"failure"'; then
    fail "GitHub CI: one or more recent workflow runs on main have failed"
  elif [[ "$ci_result" == "[]" ]]; then
    fail "GitHub CI: no completed workflow runs found on main"
  else
    pass "GitHub CI: recent runs on main completed successfully"
  fi
else
  skip_check "GitHub CI: gh CLI not installed (install gh to enable)"
fi

# 2. Binary build
if go build -o /tmp/release-gate-conductor ./cmd/run-agent/ 2>&1; then
  pass "Binary build: go build ./cmd/run-agent/ succeeded"
  rm -f /tmp/release-gate-conductor
else
  fail "Binary build: go build ./cmd/run-agent/ failed"
fi

# 3. Unit tests
test_output="$(go test ./... 2>&1)"
test_exit=$?
if [[ $test_exit -eq 0 ]]; then
  pass "Unit tests: go test ./... passed"
else
  fail "Unit tests: go test ./... failed — $(echo "$test_output" | grep FAIL | head -3)"
fi

# 4. Port consistency
go build -o bin/run-agent ./cmd/run-agent/ 2>/dev/null || true
if [[ -x "bin/run-agent" ]]; then
  port_line="$(./bin/run-agent serve --help 2>&1 | grep -i 'port' | head -3)"
  if echo "$port_line" | grep -q "14355"; then
    pass "Port consistency: run-agent serve reports default port 14355"
  else
    fail "Port consistency: run-agent serve does not mention 14355 in --help — got: $port_line"
  fi
else
  fail "Port consistency: bin/run-agent not found after build"
fi

# 5. Startup scripts
if [[ -x "scripts/smoke-startup-scripts.sh" ]]; then
  if bash scripts/smoke-startup-scripts.sh 2>&1; then
    pass "Startup scripts: smoke-startup-scripts.sh passed"
  else
    fail "Startup scripts: smoke-startup-scripts.sh failed"
  fi
else
  skip_check "Startup scripts: scripts/smoke-startup-scripts.sh not found or not executable"
fi

# 6. Install path
if [[ "$skip_install" -eq 1 ]]; then
  skip_check "Install path (--skip-install)"
elif [[ -x "scripts/smoke-install-release.sh" ]]; then
  if bash scripts/smoke-install-release.sh 2>&1; then
    pass "Install path: smoke-install-release.sh passed"
  else
    fail "Install path: smoke-install-release.sh failed (no published release yet? use --skip-install)"
  fi
else
  skip_check "Install path: scripts/smoke-install-release.sh not found (skipping)"
fi

# 7. CLI surface check
if [[ -x "bin/run-agent" ]]; then
  help_output="$(./bin/run-agent --help 2>&1)"
  expected_cmds=("serve" "task" "job" "bus" "list" "iterate" "review" "output")
  missing=()
  for cmd in "${expected_cmds[@]}"; do
    if ! echo "$help_output" | grep -q "^  $cmd "; then
      missing+=("$cmd")
    fi
  done
  if [[ ${#missing[@]} -eq 0 ]]; then
    pass "CLI surface: all expected top-level commands present"
  else
    fail "CLI surface: missing commands: ${missing[*]}"
  fi

  # Check synthesize and review quorum
  if ./bin/run-agent output synthesize --help >/dev/null 2>&1; then
    pass "CLI surface: run-agent output synthesize --help works"
  else
    fail "CLI surface: run-agent output synthesize --help failed"
  fi
  if ./bin/run-agent review quorum --help >/dev/null 2>&1; then
    pass "CLI surface: run-agent review quorum --help works"
  else
    fail "CLI surface: run-agent review quorum --help failed"
  fi
  if ./bin/run-agent iterate --help >/dev/null 2>&1; then
    pass "CLI surface: run-agent iterate --help works"
  else
    fail "CLI surface: run-agent iterate --help failed"
  fi
else
  fail "CLI surface: bin/run-agent not found"
fi

# 8. Security baseline
if has_cmd gitleaks; then
  gitleaks_output="$(gitleaks protect --staged 2>&1)" && gitleaks_exit=0 || gitleaks_exit=$?
  if [[ $gitleaks_exit -eq 0 ]]; then
    pass "Security: gitleaks protect --staged found no secrets"
  else
    fail "Security: gitleaks protect found potential secrets — review output: $gitleaks_output"
  fi
else
  skip_check "Security: gitleaks not installed (brew install gitleaks to enable)"
fi

# 9. Clean working tree
git_status="$(git status --porcelain 2>&1)"
if [[ -z "$git_status" ]]; then
  pass "Clean tree: working tree is clean"
else
  fail "Clean tree: working tree has uncommitted changes — $(echo "$git_status" | head -5)"
fi

# ── summary ──────────────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════"
echo " Release Gate Summary"
echo "═══════════════════════════════════════"
echo " PASS: $PASS_COUNT  FAIL: $FAIL_COUNT"
echo "═══════════════════════════════════════"

if [[ $FAIL_COUNT -gt 0 ]]; then
  echo "RESULT: NOT READY — $FAIL_COUNT check(s) failed"
  exit 1
else
  echo "RESULT: READY FOR RELEASE"
  exit 0
fi
