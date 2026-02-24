# Research Task: Review and Update SUGGESTED-TASKS.md

You are a research agent. Your goal is to review the existing SUGGESTED-TASKS.md,
cross-reference against all completed run-agent tasks, docs/dev/todos.md, docs/dev/issues.md,
and the new FACTS-runs-*.md files to produce an updated, accurate task list.

## Output

OVERWRITE: `/Users/jonnyzzz/Work/conductor-loop/docs/SUGGESTED-TASKS.md`

## Step 1: Read existing SUGGESTED-TASKS.md

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/SUGGESTED-TASKS.md
```

## Step 2: Read FACTS-runs files for recent task outcomes

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runs-conductor.md 2>/dev/null || echo "not yet generated"
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runs-jonnyzzz.md 2>/dev/null || echo "not yet generated"
```

## Step 3: Check which suggested tasks are now done

For each task listed in SUGGESTED-TASKS.md, check if there's a completed run-agent task:

```bash
ROOT=/Users/jonnyzzz/run-agent/conductor-loop

# Check for specific tasks by their IDs
for task_id in \
  "task-20260223-155200-monitor-process-cap-limit" \
  "task-20260223-155210-monitor-stop-respawn-race" \
  "task-20260223-155220-blocked-dependency-deadlock-recovery" \
  "task-20260223-155230-run-status-finish-criteria" \
  "task-20260223-155240-runinfo-missing-noise-hardening" \
  "task-20260223-155250-webserver-uptime-autorecover" \
  "task-20260222-214200-ui-latency-regression-investigation" \
  "task-20260222-214100-ui-task-tree-nesting-regression-research" \
  "task-20260223-071900-ui-agent-output-regression-tdd-claude-codex-review" \
  "task-20260222-100000-ci-fix" \
  "task-20260222-100100-shell-wrap" \
  "task-20260222-100200-shell-setup" \
  "task-20260222-100300-native-watch" \
  "task-20260222-100400-native-status" \
  "task-20260222-101350-release-shellscripts" \
  "task-20260222-101520-release-finalize" \
  "task-20260222-103300-integration-json-release" \
  "task-20260222-104100-release-rc-publish" \
  "task-20260222-111500-ci-gha-green" \
  "task-20260222-111510-startup-scripts" \
  "task-20260222-111520-release-v1" \
  "task-20260222-111530-devrig-latest-release-flow" \
  "task-20260222-111540-hugo-docs-docker" \
  "task-20260222-111550-unified-run-agent-cmd" \
  "task-20260222-111600-license-apache20-audit" \
  "task-20260222-111610-internal-paths-audit" \
  "task-20260222-111620-startup-url-visibility" \
  "task-20260222-174700-fix-api-root-escape" \
  "task-20260222-174701-add-api-traversal-regression-tests" \
  "task-20260222-174702-add-installer-integrity-verification" \
  "task-20260222-181500-security-review-multi-agent-rlm" \
  "task-20260223-071700-agent-diversification-claude-gemini" \
  "task-20260223-071800-security-audit-followup-action-plan" \
  "task-20260223-103400-serve-cpu-hotspot-sse-stream-all" \
  "task-20260222-102100-goal-decompose-cli" \
  "task-20260222-102110-job-batch-cli" \
  "task-20260222-102120-workflow-runner-cli" \
  "task-20260222-102130-output-synthesize-cli" \
  "task-20260222-102140-review-quorum-cli" \
  "task-20260222-102150-iteration-loop-cli" \
  "task-20260222-173000-task-complete-fact-propagation-agent" \
  "task-20260223-072800-cli-monitor-loop-simplification"; do
  dir="$ROOT/$task_id"
  if [ -d "$dir" ]; then
    status="EXISTS"
    [ -f "$dir/DONE" ] && status="DONE"
    echo "$task_id: $status"
  else
    echo "$task_id: NOT_FOUND"
  fi
done
```

## Step 4: Read recent task outputs for new items to add

```bash
ROOT=/Users/jonnyzzz/run-agent/conductor-loop

# These recent tasks may have generated new open items
for task in \
  "task-20260222-201500-today-tasks-full-audit" \
  "task-20260222-202600-followup-missing-runinfo-recovery" \
  "task-20260222-202610-followup-restart-exhausted-status-normalization" \
  "task-20260222-202620-followup-blocked-rlm-backlog-completion" \
  "task-20260222-202630-followup-unstarted-security-fixes-execution" \
  "task-20260222-202640-followup-legacy-artifact-backfill" \
  "task-20260222-202650-followup-goal-decompose-cli-retry" \
  "task-20260222-213000-hot-update-while-running" \
  "task-20260222-193500-running-tasks-stale-status-review" \
  "task-20260222-192500-unified-bootstrap-script-design" \
  "task-20260222-191500-root-limited-parallelism-planner" \
  "task-20260222-190500-bus-post-env-context-defaults" \
  "task-20260222-185200-docs-two-working-scenarios" \
  "task-20260222-184500-system-logging-coverage-review" \
  "task-20260222-183800-ui-subtask-hierarchy-level3-debug" \
  "task-20260222-181912-loss-repro-a"; do
  dir="$ROOT/$task"
  if [ -d "$dir" ]; then
    latest=$(ls -t "$dir/runs/" 2>/dev/null | head -1)
    if [ -n "$latest" ] && [ -f "$dir/runs/$latest/output.md" ]; then
      echo "=== $task ==="
      tail -40 "$dir/runs/$latest/output.md"
      echo ""
    fi
  fi
done
```

## Step 5: Read docs/dev/todos.md and docs/dev/issues.md in full

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/dev/todos.md
echo "==="
cat /Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md | grep -E "^###|Status:|Severity:|ISSUE-" | head -60
```

## Step 6: Rewrite SUGGESTED-TASKS.md

Produce an updated SUGGESTED-TASKS.md that:
1. **Removes** tasks that have a DONE run in `/Users/jonnyzzz/run-agent/conductor-loop/`
2. **Updates** status of in-progress tasks
3. **Adds** new tasks found in recent runs that aren't yet listed
4. **Preserves** all still-open items with their IDs and priorities
5. **Adds** a new section at the bottom: `## Newly Discovered (2026-02-23)` for items found in this scan
6. Maintains the same format: P0/P1/P1-security/P2/Architecture/Swarm-Design/DevRig-Release/Operational sections

Keep the file focused — only open/not-yet-completed items.
