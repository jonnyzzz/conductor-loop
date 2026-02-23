import { describe, expect, it } from 'vitest'
import type { BusMessage, FlatRunItem, TaskSummary } from '../src/types'
import { runFileRefetchIntervalFor, runsFlatRefetchIntervalFor, scopedRunsForTree, stabilizeFlatRuns } from '../src/hooks/useAPI'
import { LOG_REFRESH_DELAY_MS, STATUS_REFRESH_DELAY_MS } from '../src/hooks/useLiveRunRefresh'
import { upsertMessageBatch, upsertMessageByID } from '../src/utils/messageStore'
import {
  buildSelectionPathNodeIDs,
  buildTree,
  selectTreeRuns,
  type TreeNode,
} from '../src/utils/treeBuilder'

function legacyBuildTree(projectId: string, tasks: TaskSummary[], flatRuns: FlatRunItem[]) {
  const runMap = new Map<string, FlatRunItem>()
  for (const run of flatRuns) {
    runMap.set(run.id, run)
  }

  const runIdsWithChildren = new Set<string>()
  for (const run of flatRuns) {
    if (run.parent_run_id && runMap.has(run.parent_run_id)) {
      runIdsWithChildren.add(run.parent_run_id)
    }
  }

  const taskRunMap = new Map<string, FlatRunItem[]>()
  for (const run of flatRuns) {
    const arr = taskRunMap.get(run.task_id) ?? []
    arr.push(run)
    taskRunMap.set(run.task_id, arr)
  }

  const taskSummaryMap = new Map<string, TaskSummary>()
  for (const task of tasks) {
    taskSummaryMap.set(task.id, task)
  }

  const taskIdSet = new Set<string>()
  for (const task of tasks) {
    taskIdSet.add(task.id)
  }
  for (const run of flatRuns) {
    taskIdSet.add(run.task_id)
  }

  const placedRunIds = new Set<string>()

  function buildRunNode(run: FlatRunItem): any {
    placedRunIds.add(run.id)
    const children: any[] = []
    for (const r of flatRuns) {
      if (r.parent_run_id === run.id && !placedRunIds.has(r.id)) {
        children.push(buildRunNode(r))
      }
    }
    return {
      id: run.id,
      status: run.status,
      children,
    }
  }

  function deriveTaskStatus(runs: FlatRunItem[]): string {
    if (runs.some((r) => r.status === 'running')) return 'running'
    if (runs.length === 0) return 'unknown'
    return runs[runs.length - 1].status
  }

  function buildTaskNode(taskId: string): any {
    const runs = taskRunMap.get(taskId) ?? []
    const summary = taskSummaryMap.get(taskId)
    const taskStatus = summary?.status ?? deriveTaskStatus(runs)
    const runIds = new Set(runs.map((run) => run.id))
    const inChain = new Set<string>()
    for (const run of runs) {
      if (run.previous_run_id && runIds.has(run.previous_run_id)) {
        inChain.add(run.previous_run_id)
      }
    }
    const rootRuns = runs.filter((run) => {
      if (placedRunIds.has(run.id)) return false
      if (run.parent_run_id && runMap.has(run.parent_run_id)) return false
      if (inChain.has(run.id) && !runIdsWithChildren.has(run.id)) return false
      return true
    })
    const children = rootRuns.map(buildRunNode)
    return {
      id: taskId,
      status: taskStatus,
      children,
    }
  }

  const sortedTaskIds = Array.from(taskIdSet).sort((a, b) => {
    const runsA = taskRunMap.get(a) ?? []
    const runsB = taskRunMap.get(b) ?? []
    const latestA = runsA.reduce((acc, r) => {
      const t = r.end_time ?? r.start_time
      return t > acc ? t : acc
    }, '')
    const latestB = runsB.reduce((acc, r) => {
      const t = r.end_time ?? r.start_time
      return t > acc ? t : acc
    }, '')
    return latestB.localeCompare(latestA)
  })

  return sortedTaskIds.map(buildTaskNode)
}

function legacyMergeMessagesByID(existing: BusMessage[], incoming: BusMessage[]): BusMessage[] {
  const byID = new Map<string, BusMessage>()
  const merged = [...existing, ...incoming]
  for (const message of merged) {
    const current = byID.get(message.msg_id)
    if (!current) {
      byID.set(message.msg_id, message)
      continue
    }
    const nextTs = Date.parse(message.timestamp)
    const curTs = Date.parse(current.timestamp)
    if ((Number.isNaN(nextTs) ? 0 : nextTs) >= (Number.isNaN(curTs) ? 0 : curTs)) {
      byID.set(message.msg_id, message)
    }
  }
  return [...byID.values()].sort((a, b) => {
    const byTime = (Date.parse(b.timestamp) || 0) - (Date.parse(a.timestamp) || 0)
    if (byTime !== 0) {
      return byTime
    }
    return b.msg_id.localeCompare(a.msg_id)
  })
}

function buildBenchmarkData(taskCount: number, runsPerTask: number): { tasks: TaskSummary[]; runs: FlatRunItem[] } {
  const tasks: TaskSummary[] = []
  const runs: FlatRunItem[] = []

  for (let taskIndex = 0; taskIndex < taskCount; taskIndex += 1) {
    const taskId = `task-20260222-210000-${taskIndex.toString().padStart(4, '0')}`
    tasks.push({
      id: taskId,
      status: 'running',
      last_activity: '2026-02-22T21:00:00Z',
    })

    let previousRunID = ''
    for (let runIndex = 0; runIndex < runsPerTask; runIndex += 1) {
      const runID = `run-${taskIndex.toString().padStart(4, '0')}-${runIndex.toString().padStart(4, '0')}`
      const parentRunID = runIndex > 0 && runIndex % 5 === 0
        ? `run-${taskIndex.toString().padStart(4, '0')}-${(runIndex - 5).toString().padStart(4, '0')}`
        : ''
      runs.push({
        id: runID,
        task_id: taskId,
        agent: runIndex % 2 === 0 ? 'codex' : 'claude',
        status: runIndex === runsPerTask - 1 ? 'running' : 'completed',
        exit_code: runIndex === runsPerTask - 1 ? -1 : 0,
        start_time: `2026-02-22T21:${String(Math.floor(runIndex / 60)).padStart(2, '0')}:${String(runIndex % 60).padStart(2, '0')}Z`,
        end_time: runIndex === runsPerTask - 1
          ? undefined
          : `2026-02-22T21:${String(Math.floor(runIndex / 60)).padStart(2, '0')}:${String((runIndex % 60) + 1).padStart(2, '0')}Z`,
        previous_run_id: previousRunID || undefined,
        parent_run_id: parentRunID || undefined,
      })
      previousRunID = runID
    }
  }
  return { tasks, runs }
}

function buildSkewedBenchmarkData(
  taskCount: number,
  runsPerTask: number,
  activeTaskCount: number
): { tasks: TaskSummary[]; runs: FlatRunItem[] } {
  const tasks: TaskSummary[] = []
  const runs: FlatRunItem[] = []

  for (let taskIndex = 0; taskIndex < taskCount; taskIndex += 1) {
    const taskId = `task-skew-${taskIndex.toString().padStart(4, '0')}`
    const active = taskIndex < activeTaskCount
    tasks.push({
      id: taskId,
      status: active ? 'running' : 'completed',
      last_activity: `2026-02-22T22:${String(Math.floor(taskIndex / 60)).padStart(2, '0')}:${String(taskIndex % 60).padStart(2, '0')}Z`,
    })

    let previousRunID = ''
    for (let runIndex = 0; runIndex < runsPerTask; runIndex += 1) {
      const runID = `run-skew-${taskIndex.toString().padStart(4, '0')}-${runIndex.toString().padStart(4, '0')}`
      const isLastRun = runIndex === runsPerTask - 1
      const runStatus = active && isLastRun ? 'running' : 'completed'
      runs.push({
        id: runID,
        task_id: taskId,
        agent: runIndex % 2 === 0 ? 'codex' : 'claude',
        status: runStatus,
        exit_code: runStatus === 'running' ? -1 : 0,
        start_time: `2026-02-22T21:${String(Math.floor(runIndex / 60)).padStart(2, '0')}:${String(runIndex % 60).padStart(2, '0')}Z`,
        end_time: runStatus === 'running'
          ? undefined
          : `2026-02-22T21:${String(Math.floor(runIndex / 60)).padStart(2, '0')}:${String((runIndex % 60) + 1).padStart(2, '0')}Z`,
        previous_run_id: previousRunID || undefined,
      })
      previousRunID = runID
    }
  }

  return { tasks, runs }
}

function capCompletedHistoryForRunDetail(
  runs: FlatRunItem[],
  selectedRunId: string | undefined,
  completedLimit: number
): FlatRunItem[] {
  const nonCompletedRuns = runs.filter((run) => run.status !== 'completed')
  const completedRuns = runs.filter((run) => run.status === 'completed')

  const completedNewestFirst = [...completedRuns].sort((a, b) => {
    const aTime = Date.parse(a.end_time ?? a.start_time) || 0
    const bTime = Date.parse(b.end_time ?? b.start_time) || 0
    if (aTime !== bTime) {
      return bTime - aTime
    }
    return b.id.localeCompare(a.id)
  })

  const selectedCompletedRun = selectedRunId
    ? completedRuns.find((run) => run.id === selectedRunId)
    : undefined
  const visibleCompleted = completedNewestFirst.slice(0, completedLimit)

  if (
    selectedCompletedRun &&
    !visibleCompleted.some((run) => run.id === selectedCompletedRun.id)
  ) {
    visibleCompleted.push(selectedCompletedRun)
  }

  return [...nonCompletedRuns, ...visibleCompleted]
}

function buildMessages(total: number, startSecond = 0): BusMessage[] {
  const out: BusMessage[] = []
  for (let i = 0; i < total; i += 1) {
    out.push({
      msg_id: `MSG-${String(i).padStart(6, '0')}`,
      timestamp: `2026-02-22T21:${String(Math.floor((startSecond + i) / 60)).padStart(2, '0')}:${String((startSecond + i) % 60).padStart(2, '0')}Z`,
      type: 'PROGRESS',
      body: `message ${i}`,
    })
  }
  return out.reverse()
}

function buildSelectedTaskSnapshots(total: number): FlatRunItem[][] {
  const snapshots: FlatRunItem[][] = []
  for (let i = 0; i < total; i += 1) {
    const runID = `run-selected-${String(i).padStart(5, '0')}`
    const previousRunID = i > 0 ? `run-selected-${String(i - 1).padStart(5, '0')}` : undefined
    snapshots.push([
      {
        id: runID,
        task_id: 'task-selected',
        agent: i % 2 === 0 ? 'codex' : 'claude',
        status: 'running',
        exit_code: -1,
        start_time: `2026-02-22T21:${String(Math.floor(i / 60)).padStart(2, '0')}:${String(i % 60).padStart(2, '0')}Z`,
        previous_run_id: previousRunID,
      },
    ])
  }
  return snapshots
}

function resolveMessageSource(message: BusMessage): string {
  const meta = message.meta ?? {}
  const explicitSource = meta.source ?? meta.author ?? meta.agent ?? meta.sender ?? meta.origin
  if (explicitSource) {
    return explicitSource
  }
  if (message.run_id) {
    return 'run'
  }
  if (message.task_id) {
    return 'task'
  }
  if (message.project_id) {
    return 'project'
  }
  return 'unknown'
}

function legacyFilterMessages(messages: BusMessage[], filter: string, typeFilter: string): BusMessage[] {
  const query = filter.trim().toLowerCase()
  const normalizedType = typeFilter.trim().toLowerCase()
  return messages.filter((msg) => {
    if (normalizedType && msg.type.toLowerCase() !== normalizedType) {
      return false
    }
    if (!query) {
      return true
    }
    const haystack = [
      msg.body,
      msg.msg_id,
      msg.type,
      resolveMessageSource(msg),
      msg.project_id ?? msg.project ?? '',
      msg.task_id ?? msg.task ?? '',
      msg.run_id ?? '',
    ]
      .join('\n')
      .toLowerCase()
    return haystack.includes(query)
  })
}

function optimizedFilterMessages(messages: BusMessage[], filter: string, typeFilter: string): BusMessage[] {
  const query = filter.trim().toLowerCase()
  const normalizedType = typeFilter.trim().toLowerCase()
  if (query === '' && normalizedType === '') {
    return messages
  }
  return legacyFilterMessages(messages, query, normalizedType)
}

function avgDurationMs(iterations: number, run: () => void): number {
  let total = 0
  for (let i = 0; i < iterations; i += 1) {
    const started = performance.now()
    run()
    total += (performance.now() - started)
  }
  return total / iterations
}

function legacyTreeContainsSelection(
  node: TreeNode,
  selectedTaskId: string | undefined,
  selectedRunId: string | undefined
): boolean {
  if (selectedRunId && node.type === 'run' && node.id === selectedRunId) {
    return true
  }
  if (selectedTaskId && node.type === 'task' && node.id === selectedTaskId) {
    return true
  }
  if (selectedRunId && node.type === 'task' && node.inlineLatestRun && node.latestRunId === selectedRunId) {
    return true
  }
  return node.children.some((child) => legacyTreeContainsSelection(child, selectedTaskId, selectedRunId))
}

function legacySelectionChecks(root: TreeNode, selectedTaskId: string | undefined, selectedRunId: string | undefined): number {
  let selectedCount = 0
  const stack: TreeNode[] = [root]
  while (stack.length > 0) {
    const node = stack.pop()!
    if (legacyTreeContainsSelection(node, selectedTaskId, selectedRunId)) {
      selectedCount += 1
    }
    for (const child of node.children) {
      stack.push(child)
    }
  }
  return selectedCount
}

function optimizedSelectionChecks(root: TreeNode, selectedTaskId: string | undefined, selectedRunId: string | undefined): number {
  const selectedPath = buildSelectionPathNodeIDs(root, selectedTaskId, selectedRunId)
  let selectedCount = 0
  const stack: TreeNode[] = [root]
  while (stack.length > 0) {
    const node = stack.pop()!
    if (selectedPath.has(node.id)) {
      selectedCount += 1
    }
    for (const child of node.children) {
      stack.push(child)
    }
  }
  return selectedCount
}

function buildSelectionBenchmarkTree(depth: number): { tree: TreeNode; selectedTaskId: string; selectedRunId: string } {
  const taskId = 'task-selection-root'
  const tasks: TaskSummary[] = [
    {
      id: taskId,
      status: 'running',
      last_activity: '2026-02-22T21:00:00Z',
    },
  ]
  const runs: FlatRunItem[] = []
  let selectedRunId = ''
  let parentRunId = ''
  for (let runIndex = 0; runIndex < depth; runIndex += 1) {
    const runId = `run-selection-${runIndex.toString().padStart(5, '0')}`
    runs.push({
      id: runId,
      task_id: taskId,
      agent: runIndex % 2 === 0 ? 'codex' : 'claude',
      status: runIndex === depth - 1 ? 'running' : 'completed',
      exit_code: runIndex === depth - 1 ? -1 : 0,
      start_time: `2026-02-22T21:${String(Math.floor(runIndex / 60)).padStart(2, '0')}:${String(runIndex % 60).padStart(2, '0')}Z`,
      end_time: runIndex === depth - 1
        ? undefined
        : `2026-02-22T21:${String(Math.floor(runIndex / 60)).padStart(2, '0')}:${String((runIndex % 60) + 1).padStart(2, '0')}Z`,
      parent_run_id: parentRunId || undefined,
    })
    parentRunId = runId
    selectedRunId = runId
  }
  return {
    tree: buildTree('bench-project', tasks, runs),
    selectedTaskId: taskId,
    selectedRunId,
  }
}

describe('ui performance regressions', () => {
  it('buildTree outperforms legacy O(n^2) traversal on large run sets', () => {
    const { tasks, runs } = buildBenchmarkData(80, 30)

    // Warm-up
    buildTree('bench-project', tasks, runs)
    legacyBuildTree('bench-project', tasks, runs)

    const optimizedMs = avgDurationMs(3, () => {
      buildTree('bench-project', tasks, runs)
    })
    const legacyMs = avgDurationMs(3, () => {
      legacyBuildTree('bench-project', tasks, runs)
    })

    // Keep a visible metric in test output for regression tracking.
    // eslint-disable-next-line no-console
    console.info(`ui-perf treeBuilder optimized_ms=${optimizedMs.toFixed(2)} legacy_ms=${legacyMs.toFixed(2)}`)

    expect(optimizedMs).toBeLessThan(legacyMs)
  })

  it('incremental message upsert beats legacy full merge/sort for streaming updates', () => {
    const initial = buildMessages(1500)
    const incoming = buildMessages(200, 1800)

    const optimizedMs = avgDurationMs(2, () => {
      let state = initial
      for (const item of incoming) {
        state = upsertMessageByID(state, item, 5000)
      }
    })

    const legacyMs = avgDurationMs(2, () => {
      let state = initial
      for (const item of incoming) {
        state = legacyMergeMessagesByID(state, [item])
      }
    })

    // eslint-disable-next-line no-console
    console.info(`ui-perf messageStore optimized_ms=${optimizedMs.toFixed(2)} legacy_ms=${legacyMs.toFixed(2)}`)

    expect(optimizedMs).toBeLessThan(legacyMs)
  }, 15000)

  it('batched message upsert beats per-message insertion for SSE bursts', () => {
    const initial = buildMessages(1800)
    const incoming = buildMessages(280, 2200)

    const optimizedMs = avgDurationMs(3, () => {
      upsertMessageBatch(initial, incoming, 5000)
    })

    const legacyMs = avgDurationMs(3, () => {
      let state = initial
      for (const item of incoming) {
        state = upsertMessageByID(state, item, 5000)
      }
    })

    // eslint-disable-next-line no-console
    console.info(`ui-perf messageBatch optimized_ms=${optimizedMs.toFixed(2)} legacy_ms=${legacyMs.toFixed(2)}`)

    expect(optimizedMs).toBeLessThan(legacyMs)
  })

  it('duplicate SSE batches stay as no-op references to avoid rerenders', () => {
    const initial = buildMessages(1400)
    const duplicateBatch = initial.slice(0, 180).map((message) => ({ ...message }))

    let optimizedState = initial
    let optimizedStateChanges = 0
    for (let i = 0; i < 120; i += 1) {
      const next = upsertMessageBatch(optimizedState, duplicateBatch, 5000)
      if (next !== optimizedState) {
        optimizedStateChanges += 1
      }
      optimizedState = next
    }

    let legacyState = initial
    let legacyStateChanges = 0
    for (let i = 0; i < 120; i += 1) {
      const next = legacyMergeMessagesByID(legacyState, duplicateBatch)
      if (next !== legacyState) {
        legacyStateChanges += 1
      }
      legacyState = next
    }

    // eslint-disable-next-line no-console
    console.info(`ui-perf messageBatchNoop optimized_state_changes=${optimizedStateChanges} legacy_state_changes=${legacyStateChanges}`)

    expect(optimizedStateChanges).toBe(0)
    expect(legacyStateChanges).toBeGreaterThan(0)
  })

  it('selection path precomputation avoids per-node recursive subtree scans', () => {
    const { tree, selectedTaskId, selectedRunId } = buildSelectionBenchmarkTree(1200)

    const optimizedMs = avgDurationMs(3, () => {
      optimizedSelectionChecks(tree, selectedTaskId, selectedRunId)
    })
    const legacyMs = avgDurationMs(3, () => {
      legacySelectionChecks(tree, selectedTaskId, selectedRunId)
    })

    // eslint-disable-next-line no-console
    console.info(`ui-perf treeSelection optimized_ms=${optimizedMs.toFixed(2)} legacy_ms=${legacyMs.toFixed(2)}`)

    expect(optimizedMs).toBeLessThan(legacyMs)
  })

  it('keeps root-tree rebuilds scoped instead of rehydrating full run history', () => {
    const { tasks, runs } = buildSkewedBenchmarkData(260, 24, 8)
    const incomingScopedRuns = runs.filter((run) => run.status === 'running')

    const optimizedTreeRuns = scopedRunsForTree(runs, incomingScopedRuns, undefined)
    const legacyTreeRuns = runs

    const optimizedMs = avgDurationMs(3, () => {
      buildTree('bench-project', tasks, optimizedTreeRuns)
    })
    const legacyMs = avgDurationMs(3, () => {
      buildTree('bench-project', tasks, legacyTreeRuns)
    })

    // eslint-disable-next-line no-console
    console.info(
      `ui-perf runsFlatScopedTree optimized_ms=${optimizedMs.toFixed(2)} legacy_ms=${legacyMs.toFixed(2)} optimized_runs=${optimizedTreeRuns.length} legacy_runs=${legacyTreeRuns.length}`
    )

    expect(optimizedTreeRuns.length).toBeLessThan(legacyTreeRuns.length)
    expect(optimizedMs).toBeLessThan(legacyMs)
  })

  it('run filtering cuts tree rebuild cost for mostly-terminal projects', () => {
    const { tasks, runs } = buildSkewedBenchmarkData(240, 24, 8)
    const selectedTaskId = tasks[0]?.id
    const filteredRuns = selectTreeRuns(tasks, runs, selectedTaskId)

    const optimizedMs = avgDurationMs(3, () => {
      buildTree('bench-project', tasks, filteredRuns)
    })
    const legacyMs = avgDurationMs(3, () => {
      buildTree('bench-project', tasks, runs)
    })

    // eslint-disable-next-line no-console
    console.info(
      `ui-perf treeRunFilter optimized_ms=${optimizedMs.toFixed(2)} legacy_ms=${legacyMs.toFixed(2)} filtered_runs=${filteredRuns.length} total_runs=${runs.length}`
    )

    expect(filteredRuns.length).toBeLessThan(runs.length)
    expect(optimizedMs).toBeLessThan(legacyMs)
  })

  it('completed-history capping cuts selected-task tree rebuild cost', () => {
    const { tasks, runs } = buildBenchmarkData(1, 2200)
    const selectedRunId = runs.find((run) => run.status === 'completed')?.id
    const cappedRuns = capCompletedHistoryForRunDetail(runs, selectedRunId, 200)

    const optimizedMs = avgDurationMs(5, () => {
      buildTree('bench-project', tasks, cappedRuns)
    })
    const legacyMs = avgDurationMs(5, () => {
      buildTree('bench-project', tasks, runs)
    })

    // eslint-disable-next-line no-console
    console.info(
      `ui-perf runDetailCompletedCap optimized_ms=${optimizedMs.toFixed(2)} legacy_ms=${legacyMs.toFixed(2)} rendered_runs=${cappedRuns.length} total_runs=${runs.length}`
    )

    expect(cappedRuns.length).toBeLessThan(runs.length)
    expect(optimizedMs).toBeLessThan(legacyMs)
  })

  it('selected-task limited scoped cache prevents restart-history accumulation', () => {
    const snapshots = buildSelectedTaskSnapshots(1400)
    const tasks: TaskSummary[] = [
      {
        id: 'task-selected',
        status: 'running',
        last_activity: '2026-02-22T21:00:00Z',
      },
    ]

    let optimizedFinalRuns: FlatRunItem[] = []
    const optimizedMs = avgDurationMs(3, () => {
      let state: FlatRunItem[] | undefined
      for (const snapshot of snapshots) {
        state = scopedRunsForTree(state, snapshot, 'task-selected', 1)
      }
      optimizedFinalRuns = state ?? []
      buildTree('bench-project', tasks, optimizedFinalRuns)
    })

    let legacyFinalRuns: FlatRunItem[] = []
    const legacyMs = avgDurationMs(3, () => {
      let state: FlatRunItem[] | undefined
      for (const snapshot of snapshots) {
        state = scopedRunsForTree(state, snapshot, 'task-selected')
      }
      legacyFinalRuns = state ?? []
      buildTree('bench-project', tasks, legacyFinalRuns)
    })

    // eslint-disable-next-line no-console
    console.info(
      `ui-perf selectedTaskScopedBounded optimized_ms=${optimizedMs.toFixed(2)} legacy_ms=${legacyMs.toFixed(2)} optimized_runs=${optimizedFinalRuns.length} legacy_runs=${legacyFinalRuns.length}`
    )

    expect(optimizedFinalRuns.length).toBeLessThan(legacyFinalRuns.length)
    expect(optimizedFinalRuns.length).toBeLessThanOrEqual(2)
    expect(legacyFinalRuns.length).toBe(snapshots.length)
    expect(optimizedMs).toBeLessThan(legacyMs)
  })

  it('adaptive runs-flat cadence reduces polling volume while SSE is healthy', () => {
    const activeRuns: FlatRunItem[] = [
      {
        id: 'run-1',
        task_id: 'task-1',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
      },
    ]
    const previousRegressionCadenceMs = 2500
    const legacyPerMinute = 60000 / 800
    const optimizedCadenceMs = runsFlatRefetchIntervalFor(activeRuns, 'open')
    const optimizedPerMinute = 60000 / optimizedCadenceMs

    // eslint-disable-next-line no-console
    console.info(
      `ui-perf runsFlatPolling legacy_per_min=${legacyPerMinute.toFixed(1)} optimized_per_min=${optimizedPerMinute.toFixed(1)} optimized_cadence_ms=${optimizedCadenceMs}`
    )

    expect(optimizedCadenceMs).toBeLessThan(previousRegressionCadenceMs)
    expect(optimizedPerMinute).toBeLessThan(legacyPerMinute)
  })

  it('caps idle refresh staleness window when SSE is degraded', () => {
    const idleRuns: FlatRunItem[] = [
      {
        id: 'run-idle',
        task_id: 'task-idle',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
      },
    ]
    const previousIdleFallbackMs = 10000
    const previousOpenIdleFallbackMs = 12000
    const degradedIdleFallbackMs = runsFlatRefetchIntervalFor(idleRuns, 'reconnecting')
    const openIdleFallbackMs = runsFlatRefetchIntervalFor(idleRuns, 'open')
    const degradedReductionPct = ((previousIdleFallbackMs - degradedIdleFallbackMs) / previousIdleFallbackMs) * 100
    const openReductionPct = ((previousOpenIdleFallbackMs - openIdleFallbackMs) / previousOpenIdleFallbackMs) * 100

    // eslint-disable-next-line no-console
    console.info(
      `ui-perf runsFlatIdleFallback degraded_ms=${degradedIdleFallbackMs} open_ms=${openIdleFallbackMs} degraded_reduction_pct=${degradedReductionPct.toFixed(1)} open_reduction_pct=${openReductionPct.toFixed(1)}`
    )

    expect(degradedIdleFallbackMs).toBeLessThan(previousIdleFallbackMs)
    expect(openIdleFallbackMs).toBeLessThan(previousOpenIdleFallbackMs)
  })

  it('adds active-run file fallback polling when stream events pause', () => {
    const previousNoFallbackPolling = false
    const activeRunFallbackMs = runFileRefetchIntervalFor('running')
    const queuedRunFallbackMs = runFileRefetchIntervalFor('queued')

    // eslint-disable-next-line no-console
    console.info(
      `ui-perf runFileFallback active_ms=${activeRunFallbackMs} queued_ms=${queuedRunFallbackMs}`
    )

    expect(previousNoFallbackPolling).toBe(false)
    expect(activeRunFallbackMs).toBe(2500)
    expect(queuedRunFallbackMs).toBe(2500)
    expect(runFileRefetchIntervalFor('completed')).toBe(false)
  })

  it('stabilizes unchanged runs-flat payloads to avoid no-op tree rebuilds', () => {
    const { tasks, runs } = buildBenchmarkData(70, 22)
    const duplicateSnapshots = Array.from({ length: 80 }, () => runs.map((run) => ({ ...run })))

    let optimizedTreeBuilds = 0
    const optimizedMs = avgDurationMs(2, () => {
      let state = runs
      for (const snapshot of duplicateSnapshots) {
        const next = stabilizeFlatRuns(state, snapshot)
        if (next !== state) {
          buildTree('bench-project', tasks, next)
          optimizedTreeBuilds += 1
        }
        state = next
      }
    })

    let legacyTreeBuilds = 0
    const legacyMs = avgDurationMs(2, () => {
      let state = runs
      for (const snapshot of duplicateSnapshots) {
        const next = snapshot
        if (next !== state) {
          buildTree('bench-project', tasks, next)
          legacyTreeBuilds += 1
        }
        state = next
      }
    })

    // eslint-disable-next-line no-console
    console.info(
      `ui-perf runsFlatNoopRerender optimized_ms=${optimizedMs.toFixed(2)} legacy_ms=${legacyMs.toFixed(2)} optimized_tree_builds=${optimizedTreeBuilds} legacy_tree_builds=${legacyTreeBuilds}`
    )

    expect(optimizedTreeBuilds).toBe(0)
    expect(legacyTreeBuilds).toBeGreaterThan(0)
    expect(optimizedMs).toBeLessThan(legacyMs)
  })

  it('keeps status debounce short and disables log-triggered file invalidation debounce', () => {
    const previousStatusDebounceMs = 150

    // eslint-disable-next-line no-console
    console.info(
      `ui-perf liveRefreshDebounce status_ms=${STATUS_REFRESH_DELAY_MS} log_ms=${LOG_REFRESH_DELAY_MS}`
    )

    expect(STATUS_REFRESH_DELAY_MS).toBeLessThan(previousStatusDebounceMs)
    expect(LOG_REFRESH_DELAY_MS).toBe(0)
  })

  it('eliminates log-driven run-file request churn while stream is open', () => {
    const previousLogDrivenRefetchMs = 180
    const previousPerMinute = 60000 / previousLogDrivenRefetchMs
    const fallbackInterval = runFileRefetchIntervalFor('running', 'error')
    const fallbackPerMinute = fallbackInterval ? (60000 / fallbackInterval) : 0
    const streamOpenInterval = runFileRefetchIntervalFor('running', 'open')
    const streamOpenPerMinute = streamOpenInterval ? (60000 / streamOpenInterval) : 0

    // eslint-disable-next-line no-console
    console.info(
      `ui-perf runFileRequestChurn legacy_per_min=${previousPerMinute.toFixed(1)} fallback_per_min=${fallbackPerMinute.toFixed(1)} stream_open_per_min=${streamOpenPerMinute.toFixed(1)}`
    )

    expect(streamOpenPerMinute).toBe(0)
    expect(fallbackPerMinute).toBeLessThan(previousPerMinute)
  })

  it('skips full message scans for unfiltered message-bus view', () => {
    const messages = buildMessages(1800)

    const optimizedMs = avgDurationMs(6, () => {
      optimizedFilterMessages(messages, '', '')
    })
    const legacyMs = avgDurationMs(6, () => {
      legacyFilterMessages(messages, '', '')
    })

    // eslint-disable-next-line no-console
    console.info(`ui-perf messageFilterShortCircuit optimized_ms=${optimizedMs.toFixed(4)} legacy_ms=${legacyMs.toFixed(4)}`)

    expect(optimizedMs).toBeLessThan(legacyMs)
  })
})
