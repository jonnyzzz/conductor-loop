#!/usr/bin/env bash
# scripts/install-hooks.sh — Install git hooks for secret leak prevention
#
# Installs pre-commit and pre-push hooks that run gitleaks to block accidental
# secret commits. Safe to run multiple times (idempotent).
#
# Usage: bash scripts/install-hooks.sh

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
hooks_dir="$repo_root/.git/hooks"

log()  { printf '[install-hooks] %s\n' "$*"; }
fail() { printf '[install-hooks] error: %s\n' "$*" >&2; exit 1; }

[[ -d "$repo_root/.git" ]] || fail "not a git repository: $repo_root"
mkdir -p "$hooks_dir"

install_hook() {
  local name="$1" body="$2"
  local path="$hooks_dir/$name"
  if [[ -f "$path" ]] && ! grep -q "gitleaks" "$path" 2>/dev/null; then
    log "WARNING: $name hook already exists and does not mention gitleaks — skipping"
    return
  fi
  printf '%s\n' "$body" > "$path"
  chmod +x "$path"
  log "installed $path"
}

PRE_COMMIT_BODY='#!/usr/bin/env bash
# Prevent accidental secret commits. Installed by scripts/install-hooks.sh
if command -v gitleaks >/dev/null 2>&1; then
  gitleaks protect --staged || exit 1
fi'

PRE_PUSH_BODY='#!/usr/bin/env bash
# Prevent accidental secret pushes. Installed by scripts/install-hooks.sh
if command -v gitleaks >/dev/null 2>&1; then
  gitleaks protect || exit 1
fi'

install_hook "pre-commit" "$PRE_COMMIT_BODY"
install_hook "pre-push"   "$PRE_PUSH_BODY"

log "hooks installed. Verify with: ls -la .git/hooks/pre-commit .git/hooks/pre-push"

# Check if gitleaks is available
if ! command -v gitleaks >/dev/null 2>&1; then
  log "WARNING: gitleaks is not installed. Hooks will be no-ops until it is."
  log "Install: brew install gitleaks  (macOS)"
  log "         go install github.com/gitleaks/gitleaks/v8@latest  (any platform)"
fi
