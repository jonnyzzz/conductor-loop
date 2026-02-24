:<<"::CMDLITERAL"
@ECHO OFF
GOTO :CMDSCRIPT
::CMDLITERAL

#!/usr/bin/env bash
set -euo pipefail

# Single-file launcher: shell block for Unix environments, CMD block below for Windows.

fail() {
  printf 'run-agent.cmd: %s\n' "$*" >&2
  exit 1
}

resolve_script_dir() {
  local src="${BASH_SOURCE[0]}"
  while [ -h "$src" ]; do
    local dir
    dir="$(cd -P "$(dirname "$src")" && pwd)"
    src="$(readlink "$src")"
    case "$src" in
      /*) ;;
      *) src="$dir/$src" ;;
    esac
  done
  cd -P "$(dirname "$src")" && pwd
}

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'linux\n' ;;
    Darwin) printf 'darwin\n' ;;
    *) fail "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64\n' ;;
    arm64|aarch64) printf 'arm64\n' ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac
}

resolve_run_agent_binary() {
  local script_dir="$1"
  local os arch
  os="$(detect_os)"
  arch="$(detect_arch)"

  local looked=()

  if [ -n "${RUN_AGENT_BIN:-}" ]; then
    looked+=("$RUN_AGENT_BIN")
    if [ -f "$RUN_AGENT_BIN" ] && [ -x "$RUN_AGENT_BIN" ]; then
      printf '%s\n' "$RUN_AGENT_BIN"
      return 0
    fi
  fi

  # ~/.run-agent/binaries/_latest/run-agent — maintained by deploy_locally.sh / install scripts.
  local home_latest="${HOME}/.run-agent/binaries/_latest/run-agent"
  looked+=("$home_latest")
  if [ -f "$home_latest" ] && [ -x "$home_latest" ]; then
    printf '%s\n' "$home_latest"
    return 0
  fi

  local sibling_bin="$script_dir/run-agent"
  looked+=("$sibling_bin")
  if [ -f "$sibling_bin" ] && [ -x "$sibling_bin" ]; then
    printf '%s\n' "$sibling_bin"
    return 0
  fi

  local dist_bin="$script_dir/dist/run-agent-${os}-${arch}"
  looked+=("$dist_bin")
  if [ -f "$dist_bin" ] && [ -x "$dist_bin" ]; then
    printf '%s\n' "$dist_bin"
    return 0
  fi

  if [ "${RUN_AGENT_CMD_DISABLE_PATH:-0}" != "1" ]; then
    if command -v run-agent >/dev/null 2>&1; then
      command -v run-agent
      return 0
    fi
  fi

  local looked_msg
  looked_msg="$(IFS=', '; printf '%s' "${looked[*]}")"
  fail "run-agent binary not found (checked: ${looked_msg}; PATH fallback disabled: ${RUN_AGENT_CMD_DISABLE_PATH:-0})"
}

SCRIPT_DIR="$(resolve_script_dir)"
RUN_AGENT_CMD_DIR="$SCRIPT_DIR"
RUN_AGENT_CMD_PATH="$SCRIPT_DIR/$(basename "$0")"
export RUN_AGENT_CMD_DIR RUN_AGENT_CMD_PATH

if [ -z "${RUN_AGENT_LAUNCHER:-}" ]; then
  export RUN_AGENT_LAUNCHER="run-agent.cmd"
fi

BIN_PATH="$(resolve_run_agent_binary "$SCRIPT_DIR")"
exec "$BIN_PATH" "$@"
exit $?

:CMDSCRIPT

@echo off
setlocal

set RUN_AGENT_CMD_PATH=%~f0
set RUN_AGENT_CMD_DIR=%~dp0

:: Workaround for PowerShell Core environments with incompatible PSModulePath.
set PSModulePath=

set POWERSHELL=%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe
set POWERSHELL_COMMAND= ^
Set-StrictMode -Version 3.0; ^
$ErrorActionPreference = 'Stop'; ^
function Resolve-Arch ^
{ ^
  $arch = $env:PROCESSOR_ARCHITEW6432; ^
  if (-not $arch) { $arch = $env:PROCESSOR_ARCHITECTURE; }; ^
  switch (($arch + '').ToUpperInvariant()) ^
  { ^
    'AMD64' { return 'amd64'; }; ^
    'ARM64' { return 'arm64'; }; ^
    default { throw ('unsupported Windows architecture: ' + $arch); }; ^
  }; ^
}; ^
function Resolve-RunAgentBinary ^
{ ^
  $scriptDir = [System.IO.Path]::GetFullPath($env:RUN_AGENT_CMD_DIR); ^
  $arch = Resolve-Arch; ^
  $looked = New-Object System.Collections.Generic.List[string]; ^
 ^
  if ($env:RUN_AGENT_BIN) ^
  { ^
    $looked.Add($env:RUN_AGENT_BIN); ^
    if (Test-Path -LiteralPath $env:RUN_AGENT_BIN -PathType Leaf) ^
    { ^
      return [System.IO.Path]::GetFullPath($env:RUN_AGENT_BIN); ^
    }; ^
  }; ^
 ^
  $homeLatest = [System.IO.Path]::GetFullPath((Join-Path $env:USERPROFILE '.run-agent\binaries\_latest\run-agent.exe')); ^
  $looked.Add($homeLatest); ^
  if (Test-Path -LiteralPath $homeLatest -PathType Leaf) ^
  { ^
    return $homeLatest; ^
  }; ^
 ^
  $sibling = [System.IO.Path]::GetFullPath((Join-Path $scriptDir 'run-agent.exe')); ^
  $looked.Add($sibling); ^
  if (Test-Path -LiteralPath $sibling -PathType Leaf) ^
  { ^
    return $sibling; ^
  }; ^
 ^
  $dist = [System.IO.Path]::GetFullPath((Join-Path $scriptDir ('dist\\run-agent-windows-' + $arch + '.exe'))); ^
  $looked.Add($dist); ^
  if (Test-Path -LiteralPath $dist -PathType Leaf) ^
  { ^
    return $dist; ^
  }; ^
 ^
  if ($env:RUN_AGENT_CMD_DISABLE_PATH -ne '1') ^
  { ^
    $cmd = Get-Command run-agent.exe -ErrorAction SilentlyContinue; ^
    if (-not $cmd) ^
    { ^
      $cmd = Get-Command run-agent -ErrorAction SilentlyContinue; ^
    }; ^
    if ($cmd) ^
    { ^
      return $cmd.Source; ^
    }; ^
  }; ^
 ^
  throw ('run-agent binary not found (checked: ' + ($looked -join ', ') + '; PATH fallback disabled: ' + ($env:RUN_AGENT_CMD_DISABLE_PATH + '') + ')'); ^
}; ^
 ^
if (-not $env:RUN_AGENT_LAUNCHER) ^
{ ^
  $env:RUN_AGENT_LAUNCHER = 'run-agent.cmd'; ^
}; ^
 ^
$binary = Resolve-RunAgentBinary; ^
& $binary @args; ^
exit $LASTEXITCODE;

"%POWERSHELL%" -NoLogo -NoProfile -Command "%POWERSHELL_COMMAND%" -- %*
exit /B %ERRORLEVEL%

endlocal
