#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'HELP'
Usage: scripts/docs.sh <command>

Commands:
  serve   Run Hugo development server in Docker (http://localhost:1313)
  build   Build static docs site in Docker (output: website/public)
  verify  Verify generated artifacts and key internal links
HELP
}

fail() {
  printf 'docs.sh error: %s\n' "$*" >&2
  exit 1
}

if ! command -v docker >/dev/null 2>&1; then
  fail "docker is required"
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
website_dir="$repo_root/website"
compose_file="$website_dir/docker-compose.yml"

if [[ ! -f "$compose_file" ]]; then
  fail "missing compose file: $compose_file"
fi

command_name="${1:-}"
shift || true

docker_uid="${DOCKER_UID:-$(id -u)}"
docker_gid="${DOCKER_GID:-$(id -g)}"

run_compose() {
  (
    cd "$website_dir"
    DOCKER_UID="$docker_uid" DOCKER_GID="$docker_gid" docker compose -f "$compose_file" "$@"
  )
}

verify_output() {
  public_dir="$website_dir/public"

  required_files=(
    "$public_dir/index.html"
    "$public_dir/docs/index.html"
    "$public_dir/docs/getting-started/index.html"
    "$public_dir/docs/docker-builds/index.html"
    "$public_dir/docs/architecture/index.html"
    "$public_dir/docs/message-bus/index.html"
  )

  for path in "${required_files[@]}"; do
    [[ -f "$path" ]] || fail "missing generated artifact: $path"
  done

  # Verify key internal links resolved into generated HTML.
  rg -q 'href=/docs/getting-started/' "$public_dir/index.html" || fail "missing link /docs/getting-started/ in home page"
  rg -q 'href=/docs/docker-builds/' "$public_dir/docs/getting-started/index.html" || fail "missing link /docs/docker-builds/ in getting-started page"
  rg -q 'href=/docs/architecture/' "$public_dir/docs/getting-started/index.html" || fail "missing link /docs/architecture/ in getting-started page"
  rg -q 'href=/docs/message-bus/' "$public_dir/docs/getting-started/index.html" || fail "missing link /docs/message-bus/ in getting-started page"

  printf 'Verification complete: generated docs and key internal links are valid.\n'
}

case "$command_name" in
  serve)
    run_compose up --remove-orphans hugo-serve "$@"
    ;;
  build)
    run_compose run --rm hugo-build "$@"
    ;;
  verify)
    verify_output
    ;;
  *)
    usage
    exit 1
    ;;
esac
