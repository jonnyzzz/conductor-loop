import type { FlatRunItem, TaskSummary } from '../types'

export interface TreeNode {
  type: 'project' | 'task' | 'run'
  id: string
  label: string
  status: string
  agent?: string
  taskId?: string
  projectId?: string
  startTime?: string
  latestRunId?: string
  latestRunAgent?: string
  latestRunStatus?: string
  latestRunTime?: string
  children: TreeNode[]
  restartCount?: number
  parentRunId?: string
  inlineLatestRun?: boolean
}

/**
 * Builds a project tree from a flat list of runs and tasks.
 *
 * Tree structure:
 *   project
 *     task-A
 *       run-1 (root run, parent_run_id == "")
 *         run-3 (child spawned by run-1, parent_run_id == "run-1")
 *       run-2 (another root run for same task)
 *     task-B
 *       ...
 *
 * previous_run_id chains (Ralph Loop restarts) are collapsed into the task node:
 * the task shows a restartCount badge. The latest run in a restart chain is
 * shown as the representative run node.
 *
 * parent_run_id cross-task links: runs whose parent_run_id points to a run in
 * a different task are attached as children of that parent run node.
 */
export function buildTree(
  projectId: string,
  tasks: TaskSummary[],
  flatRuns: FlatRunItem[]
): TreeNode {
  // Build a run map for O(1) lookup.
  const runMap = new Map<string, FlatRunItem>()
  for (const run of flatRuns) {
    runMap.set(run.id, run)
  }

  // Build a per-task run index: taskId -> runs[]
  const taskRunMap = new Map<string, FlatRunItem[]>()
  for (const run of flatRuns) {
    const arr = taskRunMap.get(run.task_id) ?? []
    arr.push(run)
    taskRunMap.set(run.task_id, arr)
  }

  // For each task, compute the task status (running > failed > completed)
  // using the task summary data if available, else derive from runs.
  const taskStatusMap = new Map<string, string>()
  for (const task of tasks) {
    taskStatusMap.set(task.id, task.status)
  }

  // Collect all task IDs (from both tasks list and runs).
  const taskIdSet = new Set<string>()
  for (const task of tasks) {
    taskIdSet.add(task.id)
  }
  for (const run of flatRuns) {
    taskIdSet.add(run.task_id)
  }

  // Build run nodes recursively.
  // We'll track which runs have been placed already to avoid duplicates.
  const placedRunIds = new Set<string>()

  function buildRunNode(run: FlatRunItem): TreeNode {
    placedRunIds.add(run.id)
    // Find direct children (runs that name this run as parent_run_id).
    const children: TreeNode[] = []
    for (const r of flatRuns) {
      if (r.parent_run_id === run.id && !placedRunIds.has(r.id)) {
        children.push(buildRunNode(r))
      }
    }
    const shortId = run.id.length > 20 ? run.id.slice(0, 20) + '…' : run.id
    return {
      type: 'run',
      id: run.id,
      label: `[${run.agent}] ${shortId}`,
      status: run.status,
      agent: run.agent,
      taskId: run.task_id,
      projectId,
      startTime: run.start_time,
      children,
      parentRunId: run.parent_run_id,
    }
  }

  // Build task nodes.
  function buildTaskNode(taskId: string): TreeNode {
    const runs = taskRunMap.get(taskId) ?? []
    const taskStatus = taskStatusMap.get(taskId) ?? deriveTaskStatus(runs)
    const runIds = new Set(runs.map((run) => run.id))

    // Identify restart chains (previous_run_id links).
    // A run is in a chain if it has a previous_run_id that points to another run in the same task.
    // The "head" of a chain is the run with no previous_run_id (or previous_run_id not found).
    // The "tail" is the latest run in the chain.
    const inChain = new Set<string>()
    for (const run of runs) {
      if (run.previous_run_id && runIds.has(run.previous_run_id)) {
        inChain.add(run.previous_run_id) // the previous run is "superseded"
      }
    }

    // Root runs: no parent_run_id pointing to a run in the global run map,
    // and not superseded in a restart chain.
    // A run is a root run in this task if parent_run_id is empty (or points outside the project).
    const rootRuns = runs.filter(run => {
      if (placedRunIds.has(run.id)) return false
      if (run.parent_run_id && runMap.has(run.parent_run_id)) return false // has a parent elsewhere
      if (inChain.has(run.id)) return false // superseded by a newer restart
      return true
    })

    // Compute restart count: how many previous_run_id chains exist for this task.
    // restartCount = number of runs that have a valid previous_run_id within this task.
    const restartCount = runs.filter(
      (r) => r.previous_run_id && runIds.has(r.previous_run_id)
    ).length

    const children: TreeNode[] = rootRuns.map(buildRunNode)
    const latestRun = runs.reduce<FlatRunItem | undefined>((acc, run) => {
      if (!acc) return run
      const currentTime = run.end_time ?? run.start_time
      const accTime = acc.end_time ?? acc.start_time
      return currentTime > accTime ? run : acc
    }, undefined)
    const shouldInlineLatestRun =
      latestRun !== undefined &&
      children.length === 1 &&
      children[0].id === latestRun.id &&
      children[0].children.length === 0
    const visibleChildren = shouldInlineLatestRun ? [] : children

    const shortId = taskId.replace(/^task-\d{8}-\d{6}-/, '')
    return {
      type: 'task',
      id: taskId,
      label: shortId,
      status: taskStatus,
      taskId,
      projectId,
      latestRunId: latestRun?.id,
      latestRunAgent: latestRun?.agent,
      latestRunStatus: latestRun?.status,
      latestRunTime: latestRun?.start_time,
      children: visibleChildren,
      restartCount: restartCount > 0 ? restartCount : undefined,
      inlineLatestRun: shouldInlineLatestRun ? true : undefined,
    }
  }

  // Sort task IDs by last activity (most recent first).
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

  const taskNodes = sortedTaskIds.map(buildTaskNode)

  return {
    type: 'project',
    id: projectId,
    label: projectId,
    status: taskNodes.some(t => t.status === 'running') ? 'running' : 'idle',
    projectId,
    children: taskNodes,
  }
}

function deriveTaskStatus(runs: FlatRunItem[]): string {
  if (runs.some(r => r.status === 'running')) return 'running'
  if (runs.length === 0) return 'unknown'
  const last = runs[runs.length - 1]
  return last.status
}
