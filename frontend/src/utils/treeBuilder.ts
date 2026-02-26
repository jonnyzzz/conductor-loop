import type { FlatRunItem, TaskSummary } from '../types'

export interface TreeNode {
  type: 'project' | 'task' | 'run'
  id: string
  label: string
  status: string
  queuePosition?: number
  agent?: string
  taskId?: string
  projectId?: string
  startTime?: string
  endTime?: string
  latestRunId?: string
  latestRunAgent?: string
  latestRunStatus?: string
  latestRunTime?: string
  latestRunStartTime?: string
  latestRunEndTime?: string
  children: TreeNode[]
  restartCount?: number
  parentRunId?: string
  previousRunId?: string
  inlineLatestRun?: boolean
  dependsOn?: string[]
  blockedBy?: string[]
}

function isActiveTaskStatus(status: string): boolean {
  return status === 'running' || status === 'queued' || status === 'blocked'
}

function isActiveRunStatus(status: string): boolean {
  return status === 'running' || status === 'queued'
}

// selectTreeRuns trims terminal-task history from the high-frequency tree refresh path.
// We keep runs for active tasks, the selected task, and the parent/child chain needed
// to preserve threaded task nesting.
export function selectTreeRuns(
  tasks: TaskSummary[],
  flatRuns: FlatRunItem[],
  selectedTaskId: string | undefined
): FlatRunItem[] {
  if (!flatRuns || flatRuns.length === 0) {
    return flatRuns
  }

  const includedTaskIDs = new Set<string>()
  for (const task of tasks) {
    if (selectedTaskId && task.id === selectedTaskId) {
      includedTaskIDs.add(task.id)
      continue
    }
    if (isActiveTaskStatus(task.status)) {
      includedTaskIDs.add(task.id)
    }
  }
  for (const run of flatRuns) {
    if (isActiveRunStatus(run.status)) {
      includedTaskIDs.add(run.task_id)
    }
  }
  if (includedTaskIDs.size > 0) {
    const threadParentTaskByTask = new Map<string, string>()
    for (const task of tasks) {
      const threadParent = task.thread_parent
      if (!threadParent) {
        continue
      }
      if (task.project_id && threadParent.project_id !== task.project_id) {
        continue
      }
      const parentTaskID = threadParent.task_id
      if (!parentTaskID || parentTaskID === task.id) {
        continue
      }
      threadParentTaskByTask.set(task.id, parentTaskID)
    }
    let changed = true
    while (changed) {
      changed = false
      for (const [taskID, parentTaskID] of threadParentTaskByTask.entries()) {
        const childIncluded = includedTaskIDs.has(taskID)
        const parentIncluded = includedTaskIDs.has(parentTaskID)
        if (childIncluded && !parentIncluded) {
          includedTaskIDs.add(parentTaskID)
          changed = true
        }
        if (!childIncluded && parentIncluded) {
          includedTaskIDs.add(taskID)
          changed = true
        }
      }
    }
  }

  if (includedTaskIDs.size === 0) {
    // Keep full history when no active/selected task seeds exist.
    // This preserves parent_run_id chains used for task nesting.
    return flatRuns
  }

  const runByID = new Map<string, FlatRunItem>()
  const childrenByParent = new Map<string, FlatRunItem[]>()
  for (const run of flatRuns) {
    runByID.set(run.id, run)
    if (!run.parent_run_id) {
      continue
    }
    const children = childrenByParent.get(run.parent_run_id)
    if (children) {
      children.push(run)
    } else {
      childrenByParent.set(run.parent_run_id, [run])
    }
  }

  const includeRunIDs = new Set<string>()
  for (const run of flatRuns) {
    if (includedTaskIDs.has(run.task_id)) {
      includeRunIDs.add(run.id)
    }
  }
  seedLatestCrossTaskParentAnchorRuns(flatRuns, runByID, includeRunIDs)

  const seedRunIDs = Array.from(includeRunIDs)
  const descendantSeedRunIDs = [...seedRunIDs]
  const descendantSeedRunSet = new Set(descendantSeedRunIDs)
  const lineageQueue = [...seedRunIDs]
  const lineageQueued = new Set(lineageQueue)

  for (let index = 0; index < lineageQueue.length; index += 1) {
    const runID = lineageQueue[index]
    const run = runByID.get(runID)
    if (!run) {
      continue
    }

    let parentCursor: FlatRunItem | undefined = run
    let parentGuard = 0
    while (parentCursor?.parent_run_id) {
      const parentRunID = parentCursor.parent_run_id
      includeRunIDs.add(parentRunID)
      if (!lineageQueued.has(parentRunID)) {
        lineageQueued.add(parentRunID)
        lineageQueue.push(parentRunID)
      }
      const next = runByID.get(parentRunID)
      parentGuard += 1
      if (!next || parentGuard > flatRuns.length) {
        break
      }
      parentCursor = next
    }

    let previousCursor: FlatRunItem | undefined = run
    let previousGuard = 0
    while (previousCursor?.previous_run_id) {
      const previousRunID = previousCursor.previous_run_id
      includeRunIDs.add(previousRunID)
      if (!descendantSeedRunSet.has(previousRunID)) {
        descendantSeedRunSet.add(previousRunID)
        descendantSeedRunIDs.push(previousRunID)
      }
      if (!lineageQueued.has(previousRunID)) {
        lineageQueued.add(previousRunID)
        lineageQueue.push(previousRunID)
      }
      const next = runByID.get(previousRunID)
      previousGuard += 1
      if (!next || previousGuard > flatRuns.length) {
        break
      }
      previousCursor = next
    }
  }

  const descendantQueue = [...descendantSeedRunIDs]
  for (let index = 0; index < descendantQueue.length; index += 1) {
    const runID = descendantQueue[index]
    const children = childrenByParent.get(runID) ?? []
    for (const child of children) {
      if (includeRunIDs.has(child.id)) {
        continue
      }
      includeRunIDs.add(child.id)
      descendantQueue.push(child.id)
    }
  }

  if (includeRunIDs.size === flatRuns.length) {
    return flatRuns
  }
  return flatRuns.filter((run) => includeRunIDs.has(run.id))
}

function seedLatestCrossTaskParentAnchorRuns(
  flatRuns: FlatRunItem[],
  runByID: Map<string, FlatRunItem>,
  includeRunIDs: Set<string>
): void {
  const hasIncludedParentByTask = new Set<string>()
  const anchorByTask = new Map<string, FlatRunItem>()

  for (const run of flatRuns) {
    if (!run.parent_run_id) {
      continue
    }
    const parentRun = runByID.get(run.parent_run_id)
    if (!parentRun || parentRun.task_id === run.task_id) {
      continue
    }
    if (includeRunIDs.has(run.id)) {
      hasIncludedParentByTask.add(run.task_id)
    }
    const current = anchorByTask.get(run.task_id)
    if (!current || isLaterFlatRun(run, current)) {
      anchorByTask.set(run.task_id, run)
    }
  }

  for (const [taskID, anchor] of anchorByTask.entries()) {
    if (hasIncludedParentByTask.has(taskID)) {
      continue
    }
    includeRunIDs.add(anchor.id)
  }
}

function isLaterFlatRun(candidate: FlatRunItem, current: FlatRunItem): boolean {
  const candidateTime = flatRunActivityTime(candidate)
  const currentTime = flatRunActivityTime(current)
  if (candidateTime > currentTime) {
    return true
  }
  if (candidateTime < currentTime) {
    return false
  }
  if (candidate.start_time > current.start_time) {
    return true
  }
  if (candidate.start_time < current.start_time) {
    return false
  }
  return candidate.id > current.id
}

function flatRunActivityTime(run: FlatRunItem): string {
  if (run.end_time && run.end_time > run.start_time) {
    return run.end_time
  }
  return run.start_time
}

function isSelectionTarget(
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
  if (
    selectedRunId &&
    node.type === 'task' &&
    node.inlineLatestRun &&
    node.latestRunId === selectedRunId
  ) {
    return true
  }
  return false
}

export function buildSelectionPathNodeIDs(
  root: TreeNode,
  selectedTaskId: string | undefined,
  selectedRunId: string | undefined
): Set<string> {
  const path = new Set<string>()

  function visit(node: TreeNode): boolean {
    let matches = isSelectionTarget(node, selectedTaskId, selectedRunId)
    for (const child of node.children) {
      if (visit(child)) {
        matches = true
      }
    }
    if (matches) {
      path.add(node.id)
    }
    return matches
  }

  visit(root)
  return path
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
  const runMap = new Map<string, FlatRunItem>()
  const taskRunMap = new Map<string, FlatRunItem[]>()
  const taskSummaryMap = new Map<string, TaskSummary>()
  const taskIdSet = new Set<string>()
  const latestActivityByTask = new Map<string, string>()

  for (const task of tasks) {
    taskSummaryMap.set(task.id, task)
    taskIdSet.add(task.id)
    // Seed from the stable server-computed last_activity so the sort key is
    // independent of which run subset is currently visible.  Without this,
    // clicking a task changes the filtered run slice → latestActivityByTask
    // jumps from '' to a real timestamp → the whole list reorders.
    if (task.last_activity) {
      latestActivityByTask.set(task.id, task.last_activity)
    }
  }

  for (const run of flatRuns) {
    runMap.set(run.id, run)
    taskIdSet.add(run.task_id)

    const runs = taskRunMap.get(run.task_id)
    if (runs) {
      runs.push(run)
    } else {
      taskRunMap.set(run.task_id, [run])
    }

    const runTime = run.end_time ?? run.start_time
    const latest = latestActivityByTask.get(run.task_id)
    if (!latest || runTime > latest) {
      latestActivityByTask.set(run.task_id, runTime)
    }
  }

  const childRunsByParent = new Map<string, FlatRunItem[]>()
  const parentTaskByTask = new Map<string, string>()
  const parentTaskLinkTime = new Map<string, string>()
  for (const run of flatRuns) {
    if (!run.parent_run_id || !runMap.has(run.parent_run_id)) {
      continue
    }
    const parentRun = runMap.get(run.parent_run_id)
    if (parentRun && parentRun.task_id !== run.task_id) {
      const runTime = run.end_time ?? run.start_time
      const existingLinkTime = parentTaskLinkTime.get(run.task_id)
      if (!existingLinkTime || runTime > existingLinkTime) {
        parentTaskByTask.set(run.task_id, parentRun.task_id)
        parentTaskLinkTime.set(run.task_id, runTime)
      }
    }
    const children = childRunsByParent.get(run.parent_run_id)
    if (children) {
      children.push(run)
    } else {
      childRunsByParent.set(run.parent_run_id, [run])
    }
  }

  // Task-level thread metadata is the canonical task hierarchy source.
  // Use it when present so scoped/partial run slices cannot override expected
  // parentage with stale or missing parent_run_id edges.
  for (const task of tasks) {
    const threadParent = task.thread_parent
    if (!threadParent || threadParent.project_id !== projectId) {
      continue
    }
    const parentTaskID = threadParent.task_id
    if (!parentTaskID || parentTaskID === task.id) {
      continue
    }
    const hasParentSummary = taskSummaryMap.has(parentTaskID)
    const hasParentRuns = taskRunMap.has(parentTaskID)
    taskIdSet.add(parentTaskID)
    if (!hasParentSummary && !hasParentRuns && task.last_activity) {
      latestActivityByTask.set(parentTaskID, task.last_activity)
    }
    parentTaskByTask.set(task.id, parentTaskID)
    if (task.last_activity) {
      parentTaskLinkTime.set(task.id, task.last_activity)
    }
  }

  const runIDsWithChildren = new Set(childRunsByParent.keys())

  const placedRunIds = new Set<string>()

  function buildRunNode(run: FlatRunItem): TreeNode {
    placedRunIds.add(run.id)

    const children: TreeNode[] = []
    for (const child of childRunsByParent.get(run.id) ?? []) {
      if (!placedRunIds.has(child.id)) {
        children.push(buildRunNode(child))
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
      endTime: run.end_time,
      children,
      parentRunId: run.parent_run_id,
      previousRunId: run.previous_run_id,
    }
  }

  function buildTaskNode(taskId: string): TreeNode {
    const runs = taskRunMap.get(taskId) ?? []
    const summary = taskSummaryMap.get(taskId)
    const taskStatus = deriveTaskStatus(runs, summary?.status)

    const runIds = new Set<string>()
    let latestRun: FlatRunItem | undefined
    for (const run of runs) {
      runIds.add(run.id)
      if (!latestRun) {
        latestRun = run
        continue
      }
      const runTime = run.end_time ?? run.start_time
      const latestTime = latestRun.end_time ?? latestRun.start_time
      if (runTime > latestTime) {
        latestRun = run
      }
    }

    const supersededRunIDs = new Set<string>()
    let restartCount = 0
    for (const run of runs) {
      if (run.previous_run_id && runIds.has(run.previous_run_id)) {
        supersededRunIDs.add(run.previous_run_id)
        restartCount += 1
      }
    }

    const rootRuns: FlatRunItem[] = []
    for (const run of runs) {
      if (placedRunIds.has(run.id)) {
        continue
      }
      if (run.parent_run_id && runMap.has(run.parent_run_id)) {
        continue
      }
      if (supersededRunIDs.has(run.id) && !runIDsWithChildren.has(run.id)) {
        continue
      }
      rootRuns.push(run)
    }

    const children = rootRuns.map(buildRunNode)
    const shouldInlineLatestRun =
      children.length === 1 &&
      children[0].children.length === 0 &&
      (
        runs.length === 1 ||
        (latestRun !== undefined && children[0].id === latestRun.id)
      )
    const visibleChildren = shouldInlineLatestRun ? [] : children

    const shortId = taskId.replace(/^task-\d{8}-\d{6}-/, '')
    return {
      type: 'task',
      id: taskId,
      label: shortId,
      status: taskStatus,
      queuePosition: summary?.queue_position,
      taskId,
      projectId,
      latestRunId: latestRun?.id,
      latestRunAgent: latestRun?.agent,
      latestRunStatus: latestRun?.status,
      latestRunTime: latestRun?.start_time,
      latestRunStartTime: latestRun?.start_time,
      latestRunEndTime: latestRun?.end_time,
      children: visibleChildren,
      restartCount: restartCount > 0 ? restartCount : undefined,
      inlineLatestRun: shouldInlineLatestRun ? true : undefined,
      dependsOn: summary?.depends_on,
      blockedBy: summary?.blocked_by,
    }
  }

  const sortedTaskIds = Array.from(taskIdSet).sort((a, b) => {
    const latestA = latestActivityByTask.get(a) ?? ''
    const latestB = latestActivityByTask.get(b) ?? ''
    return latestB.localeCompare(latestA)
  })

  const taskNodes = sortedTaskIds
    .map(buildTaskNode)
    .filter((taskNode) => {
      const runs = taskRunMap.get(taskNode.id) ?? []
      if (runs.length === 0) {
        return true
      }
      if (
        taskNode.children.length > 0 ||
        taskNode.inlineLatestRun ||
        parentTaskByTask.has(taskNode.id)
      ) {
        return true
      }
      return !runs.every((run) => placedRunIds.has(run.id))
    })

  function wouldCreateTaskCycle(childTaskID: string, parentTaskID: string): boolean {
    let cursor = parentTaskID
    while (cursor !== '') {
      if (cursor === childTaskID) {
        return true
      }
      cursor = parentTaskByTask.get(cursor) ?? ''
    }
    return false
  }

  const taskNodeByID = new Map<string, TreeNode>()
  for (const taskNode of taskNodes) {
    taskNodeByID.set(taskNode.id, taskNode)
  }

  const nestedTaskIDs = new Set<string>()
  for (const taskNode of taskNodes) {
    const parentTaskID = parentTaskByTask.get(taskNode.id)
    if (!parentTaskID || parentTaskID === taskNode.id) {
      continue
    }
    if (wouldCreateTaskCycle(taskNode.id, parentTaskID)) {
      continue
    }
    const parentTaskNode = taskNodeByID.get(parentTaskID)
    if (!parentTaskNode) {
      continue
    }
    parentTaskNode.children.push(taskNode)
    nestedTaskIDs.add(taskNode.id)
  }

  const rootTaskNodes = taskNodes.filter((taskNode) => !nestedTaskIDs.has(taskNode.id))

  return {
    type: 'project',
    id: projectId,
    label: projectId,
    status: rootTaskNodes.some((task) => task.status === 'running')
      ? 'running'
      : rootTaskNodes.some((task) => task.status === 'queued')
        ? 'queued'
        : 'idle',
    projectId,
    children: rootTaskNodes,
  }
}

function deriveTaskStatus(runs: FlatRunItem[], summaryStatus?: string): string {
  if (runs.some((run) => run.status === 'running')) return 'running'
  if (runs.some((run) => run.status === 'queued')) return 'queued'
  if (runs.length === 0) return summaryStatus ?? 'unknown'

  let latestRun = runs[0]
  let latestTime = latestRun.end_time ?? latestRun.start_time
  for (const run of runs) {
    const runTime = run.end_time ?? run.start_time
    if (runTime > latestTime) {
      latestRun = run
      latestTime = runTime
    }
  }
  return latestRun.status || summaryStatus || 'unknown'
}
