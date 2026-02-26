import { describe, it, expect } from 'vitest'
import { buildTree, selectTreeRuns } from '../src/utils/treeBuilder'
import type { FlatRunItem, TaskSummary } from '../src/types'

function makeTask(id: string, lastActivity: string, status = 'completed'): TaskSummary {
  return {
    id,
    project_id: 'proj',
    status: status as TaskSummary['status'],
    last_activity: lastActivity,
  }
}

function makeRun(id: string, taskId: string, opts: Partial<FlatRunItem> = {}): FlatRunItem {
  return {
    id,
    task_id: taskId,
    agent: 'claude',
    status: 'completed',
    exit_code: 0,
    start_time: '2026-02-26T10:00:00Z',
    ...opts,
  }
}

// ────────────────────────────────────────────────────────────────────────────
// Stable sort: task order must not change when a different run subset is loaded
// ────────────────────────────────────────────────────────────────────────────

describe('buildTree – stable task sort', () => {
  const taskA = makeTask('task-a', '2026-02-26T09:00:00Z')
  const taskB = makeTask('task-b', '2026-02-26T10:00:00Z')
  const tasks = [taskA, taskB]

  it('sorts tasks by last_activity descending (newer first)', () => {
    const tree = buildTree('proj', tasks, [])
    expect(tree.children.map((n) => n.id)).toEqual(['task-b', 'task-a'])
  })

  it('does not reorder when a run for the older task has end_time equal to newer task last_activity', () => {
    // run end_time == taskB.last_activity — must NOT promote task-a above task-b
    const runA = makeRun('run-a1', 'task-a', {
      start_time: '2026-02-26T09:50:00Z',
      end_time: '2026-02-26T10:00:00Z',
    })
    const tree = buildTree('proj', tasks, [runA])
    expect(tree.children.map((n) => n.id)).toEqual(['task-b', 'task-a'])
  })

  it('does not reorder when a run for the older task has end_time NEWER than newer task last_activity', () => {
    // Even if run end_time > taskB.last_activity, task-a's sort key is pinned to
    // task.last_activity so order must stay stable.
    const runA = makeRun('run-a2', 'task-a', {
      start_time: '2026-02-26T09:50:00Z',
      end_time: '2026-02-26T11:00:00Z',
    })
    const tree = buildTree('proj', tasks, [runA])
    expect(tree.children.map((n) => n.id)).toEqual(['task-b', 'task-a'])
  })

  it('order is identical across multiple buildTree calls with different run subsets', () => {
    // Simulates clicking between tasks: each click causes selectTreeRuns to
    // return a different subset.  Order must be identical in all cases.
    const runA = makeRun('run-a3', 'task-a', { end_time: '2026-02-26T11:00:00Z' })
    const runB = makeRun('run-b1', 'task-b', { end_time: '2026-02-26T10:00:00Z' })

    const tree1 = buildTree('proj', tasks, [runB])          // only task-b selected
    const tree2 = buildTree('proj', tasks, [runA, runB])    // task-a selected

    expect(tree1.children.map((n) => n.id)).toEqual(['task-b', 'task-a'])
    expect(tree2.children.map((n) => n.id)).toEqual(['task-b', 'task-a'])
  })
})

// ────────────────────────────────────────────────────────────────────────────
// Stable tiebreaker: tasks with equal last_activity sort by id (deterministic)
// ────────────────────────────────────────────────────────────────────────────

describe('buildTree – stable tiebreaker', () => {
  it('tasks with equal last_activity produce the same order regardless of input order', () => {
    const sameTime = '2026-02-26T10:00:00Z'
    const t1 = makeTask('task-20260226-100000-aaa', sameTime)
    const t2 = makeTask('task-20260226-100000-bbb', sameTime)
    const t3 = makeTask('task-20260226-100000-ccc', sameTime)

    const order1 = buildTree('proj', [t1, t2, t3], []).children.map((n) => n.id)
    const order2 = buildTree('proj', [t3, t1, t2], []).children.map((n) => n.id)
    const order3 = buildTree('proj', [t2, t3, t1], []).children.map((n) => n.id)

    expect(order1).toEqual(order2)
    expect(order2).toEqual(order3)
  })
})

// ────────────────────────────────────────────────────────────────────────────
// Run tree: previous_run_id restart chains form a nested tree
// ────────────────────────────────────────────────────────────────────────────

describe('buildTree – run tree structure', () => {
  const task = makeTask('task-a', '2026-02-26T10:00:00Z', 'running')

  it('inlines single run into task node when there is no restart history', () => {
    const run = makeRun('run-1', 'task-a', { status: 'running' })
    const tree = buildTree('proj', [task], [run])
    const taskNode = tree.children[0]
    expect(taskNode.inlineLatestRun).toBe(true)
    expect(taskNode.children).toHaveLength(0)
  })

  it('shows restart chain as nested tree: newest run at root, older run as child', () => {
    const runOld = makeRun('run-1', 'task-a', {
      start_time: '2026-02-26T09:00:00Z',
      end_time: '2026-02-26T09:30:00Z',
      status: 'failed',
    })
    const runNew = makeRun('run-2', 'task-a', {
      start_time: '2026-02-26T09:31:00Z',
      status: 'running',
      previous_run_id: 'run-1',
    })
    const tree = buildTree('proj', [task], [runOld, runNew])
    const taskNode = tree.children[0]

    expect(taskNode.children).toHaveLength(1)
    const rootRunNode = taskNode.children[0]
    expect(rootRunNode.id).toBe('run-2')
    expect(rootRunNode.children).toHaveLength(1)
    expect(rootRunNode.children[0].id).toBe('run-1')
  })

  it('superseded run does NOT appear as a root-level run node', () => {
    const runOld = makeRun('run-1', 'task-a', { status: 'failed' })
    const runNew = makeRun('run-2', 'task-a', {
      status: 'running',
      previous_run_id: 'run-1',
    })
    const tree = buildTree('proj', [task], [runOld, runNew])
    const taskNode = tree.children[0]

    const topRunIds = taskNode.children.map((n) => n.id)
    expect(topRunIds).not.toContain('run-1')
    expect(topRunIds).toContain('run-2')
  })

  it('three-run restart chain forms a 3-level nesting', () => {
    const run1 = makeRun('run-1', 'task-a', { status: 'failed' })
    const run2 = makeRun('run-2', 'task-a', { status: 'failed', previous_run_id: 'run-1' })
    const run3 = makeRun('run-3', 'task-a', { status: 'running', previous_run_id: 'run-2' })
    const tree = buildTree('proj', [task], [run1, run2, run3])
    const taskNode = tree.children[0]

    expect(taskNode.children).toHaveLength(1)
    const n3 = taskNode.children[0]
    expect(n3.id).toBe('run-3')
    expect(n3.children).toHaveLength(1)
    const n2 = n3.children[0]
    expect(n2.id).toBe('run-2')
    expect(n2.children).toHaveLength(1)
    expect(n2.children[0].id).toBe('run-1')
  })
})

// ────────────────────────────────────────────────────────────────────────────
// selectTreeRuns
// ────────────────────────────────────────────────────────────────────────────

describe('selectTreeRuns', () => {
  it('includes all runs for the selected task', () => {
    const task = makeTask('task-a', '2026-02-26T10:00:00Z', 'completed')
    const runs = [makeRun('run-1', 'task-a'), makeRun('run-2', 'task-a')]
    const result = selectTreeRuns([task], runs, 'task-a')
    expect(result.map((r) => r.id).sort()).toEqual(['run-1', 'run-2'])
  })

  it('includes runs for active tasks even when not selected', () => {
    const taskA = makeTask('task-a', '2026-02-26T09:00:00Z', 'completed')
    const taskB = makeTask('task-b', '2026-02-26T10:00:00Z', 'running')
    const runA = makeRun('run-a', 'task-a')
    const runB = makeRun('run-b', 'task-b', { status: 'running' })
    const result = selectTreeRuns([taskA, taskB], [runA, runB], undefined)
    expect(result.map((r) => r.id)).toContain('run-b')
  })
})
