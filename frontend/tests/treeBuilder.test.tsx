import { describe, expect, it } from 'vitest'
import { buildSelectionPathNodeIDs, buildTree, selectTreeRuns } from '../src/utils/treeBuilder'
import type { FlatRunItem, TaskSummary } from '../src/types'

describe('treeBuilder', () => {
  it('filters tree runs to active or selected tasks and keeps required parent chain', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-root-terminal',
        status: 'completed',
        last_activity: '2026-02-22T21:00:00Z',
      },
      {
        id: 'task-child-active',
        status: 'running',
        last_activity: '2026-02-22T21:10:00Z',
      },
      {
        id: 'task-other-terminal',
        status: 'failed',
        last_activity: '2026-02-22T20:40:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root-terminal',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
        end_time: '2026-02-22T21:01:00Z',
      },
      {
        id: 'run-child',
        task_id: 'task-child-active',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:05:00Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-ignored',
        task_id: 'task-other-terminal',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T20:30:00Z',
        end_time: '2026-02-22T20:31:00Z',
      },
    ]

    const filtered = selectTreeRuns(tasks, runs, undefined)
    expect(filtered.map((run) => run.id)).toEqual(['run-root', 'run-child'])
  })

  it('keeps descendant runs for active branches so nested tasks stay attached', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-root-active',
        status: 'running',
        last_activity: '2026-02-22T21:20:00Z',
      },
      {
        id: 'task-child-terminal',
        status: 'completed',
        last_activity: '2026-02-22T21:19:00Z',
      },
      {
        id: 'task-grand-terminal',
        status: 'failed',
        last_activity: '2026-02-22T21:18:00Z',
      },
      {
        id: 'task-unrelated-terminal',
        status: 'completed',
        last_activity: '2026-02-22T21:17:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root-active',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:10:00Z',
      },
      {
        id: 'run-child',
        task_id: 'task-child-terminal',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:11:00Z',
        end_time: '2026-02-22T21:12:00Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-grand',
        task_id: 'task-grand-terminal',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T21:13:00Z',
        end_time: '2026-02-22T21:14:00Z',
        parent_run_id: 'run-child',
      },
      {
        id: 'run-unrelated',
        task_id: 'task-unrelated-terminal',
        agent: 'xai',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:15:00Z',
        end_time: '2026-02-22T21:16:00Z',
      },
    ]

    const filtered = selectTreeRuns(tasks, runs, undefined)
    expect(filtered.map((run) => run.id)).toEqual(['run-root', 'run-child', 'run-grand'])
  })

  it('keeps threaded child runs when only the parent task is active', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-thread-root',
        status: 'running',
        last_activity: '2026-02-22T21:20:00Z',
      },
      {
        id: 'task-thread-child',
        status: 'queued',
        last_activity: '2026-02-22T21:19:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-root',
          run_id: 'run-root',
          message_id: 'MSG-thread-root',
        },
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-thread-root',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:10:00Z',
      },
      {
        id: 'run-child-terminal',
        task_id: 'task-thread-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:11:00Z',
        end_time: '2026-02-22T21:12:00Z',
      },
    ]

    const filtered = selectTreeRuns(tasks, runs, undefined)
    expect(filtered.map((run) => run.id)).toEqual(['run-root', 'run-child-terminal'])
  })

  it('keeps threaded parent runs when only child task is active and run parent edge is missing', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-thread-root',
        status: 'completed',
        last_activity: '2026-02-22T21:18:00Z',
      },
      {
        id: 'task-thread-child',
        status: 'running',
        last_activity: '2026-02-22T21:20:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-root',
          run_id: 'run-thread-root',
          message_id: 'MSG-thread-root',
        },
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-thread-root',
        task_id: 'task-thread-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:10:00Z',
        end_time: '2026-02-22T21:12:00Z',
      },
      {
        id: 'run-thread-child-active',
        task_id: 'task-thread-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:19:00Z',
      },
    ]

    const filtered = selectTreeRuns(tasks, runs, undefined)
    expect(filtered.map((run) => run.id)).toEqual(['run-thread-root', 'run-thread-child-active'])

    const tree = buildTree('conductor-loop', tasks, filtered)
    const rootTask = tree.children.find((node) => node.type === 'task' && node.id === 'task-thread-root')
    expect(rootTask).toBeDefined()
    const childTask = rootTask?.children.find((node) => node.type === 'task' && node.id === 'task-thread-child')
    expect(childTask).toBeDefined()
  })

  it('keeps terminal hierarchy anchors when an unrelated task is active', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-root-terminal',
        status: 'completed',
        last_activity: '2026-02-22T21:00:00Z',
      },
      {
        id: 'task-child-terminal',
        status: 'completed',
        last_activity: '2026-02-22T21:01:00Z',
      },
      {
        id: 'task-grand-terminal',
        status: 'failed',
        last_activity: '2026-02-22T21:02:00Z',
      },
      {
        id: 'task-active-unrelated',
        status: 'running',
        last_activity: '2026-02-22T21:03:00Z',
      },
      {
        id: 'task-terminal-unrelated',
        status: 'completed',
        last_activity: '2026-02-22T21:04:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root-terminal',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
        end_time: '2026-02-22T21:00:30Z',
      },
      {
        id: 'run-child',
        task_id: 'task-child-terminal',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:01:00Z',
        end_time: '2026-02-22T21:01:30Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-grand',
        task_id: 'task-grand-terminal',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T21:02:00Z',
        end_time: '2026-02-22T21:02:30Z',
        parent_run_id: 'run-child',
      },
      {
        id: 'run-active',
        task_id: 'task-active-unrelated',
        agent: 'xai',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:03:00Z',
      },
      {
        id: 'run-terminal-unrelated',
        task_id: 'task-terminal-unrelated',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:04:00Z',
        end_time: '2026-02-22T21:04:30Z',
      },
    ]

    const filtered = selectTreeRuns(tasks, runs, undefined)
    expect(filtered.map((run) => run.id)).toEqual(['run-root', 'run-child', 'run-grand', 'run-active'])

    const tree = buildTree('conductor-loop', tasks, filtered)
    const rootTask = tree.children.find((node) => node.type === 'task' && node.id === 'task-root-terminal')
    expect(rootTask).toBeDefined()

    const childTask = rootTask?.children.find((node) => node.type === 'task' && node.id === 'task-child-terminal')
    expect(childTask).toBeDefined()

    const grandTask = childTask?.children.find((node) => node.type === 'task' && node.id === 'task-grand-terminal')
    expect(grandTask).toBeDefined()
  })

  it('keeps full run history when every task is terminal and no task is selected', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-selected-terminal',
        status: 'completed',
        last_activity: '2026-02-22T21:00:00Z',
      },
      {
        id: 'task-other-terminal',
        status: 'failed',
        last_activity: '2026-02-22T20:40:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-selected',
        task_id: 'task-selected-terminal',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:55:00Z',
        end_time: '2026-02-22T21:00:00Z',
      },
      {
        id: 'run-ignored',
        task_id: 'task-other-terminal',
        agent: 'claude',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T20:35:00Z',
        end_time: '2026-02-22T20:40:00Z',
      },
    ]

    expect(selectTreeRuns(tasks, runs, 'task-selected-terminal').map((run) => run.id)).toEqual(['run-selected'])
    expect(selectTreeRuns(tasks, runs, undefined).map((run) => run.id)).toEqual([
      'run-selected',
      'run-ignored',
    ])
  })

  it('preserves task nesting when an older run carries the cross-task parent link', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-root',
        status: 'completed',
        last_activity: '2026-02-22T21:05:00Z',
      },
      {
        id: 'task-child',
        status: 'failed',
        last_activity: '2026-02-22T21:07:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
        end_time: '2026-02-22T21:01:00Z',
      },
      {
        id: 'run-child-linked',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:02:00Z',
        end_time: '2026-02-22T21:03:00Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-child-latest',
        task_id: 'task-child',
        agent: 'claude',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T21:04:00Z',
        end_time: '2026-02-22T21:05:00Z',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    expect(tree.children).toHaveLength(1)

    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-root')

    const nestedChildTask = rootTask.children.find((node) => node.id === 'task-child')
    expect(nestedChildTask).toBeDefined()
    expect(nestedChildTask?.type).toBe('task')
  })

  it('keeps ancestor bridge runs for selected deep task after parent restart', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-root',
        status: 'completed',
        last_activity: '2026-02-22T21:05:00Z',
      },
      {
        id: 'task-child',
        status: 'completed',
        last_activity: '2026-02-22T21:07:00Z',
      },
      {
        id: 'task-grandchild',
        status: 'failed',
        last_activity: '2026-02-22T21:09:00Z',
      },
    ]

    // Simulates backend runs/flat payload for:
    // active_only=1&selected_task_id=task-grandchild&selected_task_limit=1
    // where the parent task restarted and only an older run keeps root linkage.
    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
        end_time: '2026-02-22T21:01:00Z',
      },
      {
        id: 'run-child-linked',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:02:00Z',
        end_time: '2026-02-22T21:03:00Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-child-restarted',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:04:00Z',
        end_time: '2026-02-22T21:05:00Z',
        previous_run_id: 'run-child-linked',
      },
      {
        id: 'run-grandchild-selected',
        task_id: 'task-grandchild',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T21:06:00Z',
        end_time: '2026-02-22T21:07:00Z',
        parent_run_id: 'run-child-restarted',
      },
    ]

    const filtered = selectTreeRuns(tasks, runs, 'task-grandchild')
    expect(filtered.map((run) => run.id)).toEqual([
      'run-root',
      'run-child-linked',
      'run-child-restarted',
      'run-grandchild-selected',
    ])

    const tree = buildTree('conductor-loop', tasks, filtered)
    expect(tree.children).toHaveLength(1)

    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-root')

    const childTask = rootTask.children.find((node) => node.type === 'task' && node.id === 'task-child')
    expect(childTask).toBeDefined()

    const grandchildTask = childTask?.children.find(
      (node) => node.type === 'task' && node.id === 'task-grandchild'
    )
    expect(grandchildTask).toBeDefined()
  })

  it('keeps ancestor bridge runs when selected deep task uses detached parent run', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-root',
        status: 'completed',
        last_activity: '2026-02-22T21:05:00Z',
      },
      {
        id: 'task-child',
        status: 'completed',
        last_activity: '2026-02-22T21:07:00Z',
      },
      {
        id: 'task-grandchild',
        status: 'failed',
        last_activity: '2026-02-22T21:09:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
        end_time: '2026-02-22T21:01:00Z',
      },
      {
        id: 'run-child-linked',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:02:00Z',
        end_time: '2026-02-22T21:03:00Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-child-detached',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:04:00Z',
        end_time: '2026-02-22T21:05:00Z',
      },
      {
        id: 'run-grandchild-selected',
        task_id: 'task-grandchild',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T21:06:00Z',
        end_time: '2026-02-22T21:07:00Z',
        parent_run_id: 'run-child-detached',
      },
    ]

    const filtered = selectTreeRuns(tasks, runs, 'task-grandchild')
    expect(filtered.map((run) => run.id)).toEqual([
      'run-root',
      'run-child-linked',
      'run-child-detached',
      'run-grandchild-selected',
    ])

    const tree = buildTree('conductor-loop', tasks, filtered)
    expect(tree.children).toHaveLength(1)

    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-root')

    const childTask = rootTask.children.find((node) => node.type === 'task' && node.id === 'task-child')
    expect(childTask).toBeDefined()

    const grandchildTask = childTask?.children.find(
      (node) => node.type === 'task' && node.id === 'task-grandchild'
    )
    expect(grandchildTask).toBeDefined()
  })

  it('inlines task rows when there is a single visible run', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-20260221-210000-ui-density',
        status: 'running',
        last_activity: '2026-02-21T21:03:15Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-002',
        task_id: 'task-20260221-210000-ui-density',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-21T21:01:00Z',
        end_time: '2026-02-21T21:03:15Z',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    const taskNode = tree.children[0]
    expect(taskNode.inlineLatestRun).toBe(true)
    expect(taskNode.children).toHaveLength(0)
    expect(taskNode.latestRunAgent).toBe('codex')
    expect(taskNode.latestRunStartTime).toBe('2026-02-21T21:01:00Z')
    expect(taskNode.latestRunEndTime).toBe('2026-02-21T21:03:15Z')
  })

  it('keeps run rows visible when a run has child runs', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-20260221-220000-review',
        status: 'running',
        last_activity: '2026-02-21T22:10:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-20260221-220000-review',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-21T22:00:00Z',
      },
      {
        id: 'run-child',
        task_id: 'task-20260221-220000-review',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-21T22:02:00Z',
        parent_run_id: 'run-root',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    const taskNode = tree.children[0]
    expect(taskNode.inlineLatestRun).toBeUndefined()
    expect(taskNode.children).toHaveLength(1)
    expect(taskNode.children[0].children).toHaveLength(1)
  })

  it('tracks restart chain metadata for clear restart badges', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-20260221-230000-restart',
        status: 'failed',
        last_activity: '2026-02-21T23:06:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-1',
        task_id: 'task-20260221-230000-restart',
        agent: 'codex',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-21T23:00:00Z',
        end_time: '2026-02-21T23:01:00Z',
      },
      {
        id: 'run-2',
        task_id: 'task-20260221-230000-restart',
        agent: 'codex',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-21T23:02:00Z',
        end_time: '2026-02-21T23:03:00Z',
        previous_run_id: 'run-1',
      },
      {
        id: 'run-3',
        task_id: 'task-20260221-230000-restart',
        agent: 'codex',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-21T23:04:00Z',
        end_time: '2026-02-21T23:06:00Z',
        previous_run_id: 'run-2',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    const taskNode = tree.children[0]
    expect(taskNode.restartCount).toBe(2)
    expect(taskNode.inlineLatestRun).toBe(true)
    expect(taskNode.latestRunId).toBe('run-3')
  })

  it('keeps level-3 subtask runs nested without detached sibling task rows', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-20260222-180000-root-review',
        status: 'running',
        last_activity: '2026-02-22T18:05:00Z',
      },
      {
        id: 'task-20260222-180100-child-audit',
        status: 'running',
        last_activity: '2026-02-22T18:06:00Z',
      },
      {
        id: 'task-20260222-180200-grandchild-check',
        status: 'running',
        last_activity: '2026-02-22T18:07:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-20260222-180000-root-review',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T18:00:00Z',
      },
      {
        id: 'run-child',
        task_id: 'task-20260222-180100-child-audit',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T18:02:00Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-grandchild',
        task_id: 'task-20260222-180200-grandchild-check',
        agent: 'gemini',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T18:04:00Z',
        parent_run_id: 'run-child',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    expect(tree.children).toHaveLength(1)

    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-20260222-180000-root-review')

    const rootRun = rootTask.children.find((node) => node.id === 'run-root')
    const childTaskNode = rootTask.children.find((node) => node.id === 'task-20260222-180100-child-audit')
    expect(childTaskNode).toBeDefined()
    expect(childTaskNode?.type).toBe('task')
    const grandchildTaskNode = childTaskNode?.children.find(
      (node) => node.id === 'task-20260222-180200-grandchild-check'
    )
    expect(grandchildTaskNode).toBeDefined()

    expect(rootRun).toBeDefined()
    expect(rootRun?.id).toBe('run-root')
    expect(rootRun?.children).toHaveLength(1)

    const childRun = rootRun?.children[0]
    expect(childRun?.id).toBe('run-child')
    expect(childRun?.children).toHaveLength(1)
    expect(childRun?.children[0].id).toBe('run-grandchild')
  })

  it('preserves level-3 task hierarchy for threaded chains', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-20260222-200000-root',
        status: 'running',
        last_activity: '2026-02-22T20:05:00Z',
      },
      {
        id: 'task-20260222-200100-child',
        status: 'running',
        last_activity: '2026-02-22T20:06:00Z',
      },
      {
        id: 'task-20260222-200200-grandchild',
        status: 'running',
        last_activity: '2026-02-22T20:07:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root-hierarchy',
        task_id: 'task-20260222-200000-root',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:00:00Z',
      },
      {
        id: 'run-child-hierarchy',
        task_id: 'task-20260222-200100-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:02:00Z',
        parent_run_id: 'run-root-hierarchy',
      },
      {
        id: 'run-grandchild-hierarchy',
        task_id: 'task-20260222-200200-grandchild',
        agent: 'gemini',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:04:00Z',
        parent_run_id: 'run-child-hierarchy',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    expect(tree.children).toHaveLength(1)
    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-20260222-200000-root')

    const childTask = rootTask.children.find(
      (node) => node.type === 'task' && node.id === 'task-20260222-200100-child'
    )
    expect(childTask).toBeDefined()

    const grandchildTask = childTask?.children.find(
      (node) => node.type === 'task' && node.id === 'task-20260222-200200-grandchild'
    )
    expect(grandchildTask).toBeDefined()
  })

  it('nests threaded tasks even when runs have no parent_run_id linkage', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-thread-root',
        status: 'running',
        last_activity: '2026-02-22T21:00:00Z',
      },
      {
        id: 'task-thread-child',
        status: 'running',
        last_activity: '2026-02-22T21:01:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-root',
          run_id: 'run-thread-root',
          message_id: 'MSG-thread-root',
        },
      },
      {
        id: 'task-thread-grandchild',
        status: 'running',
        last_activity: '2026-02-22T21:02:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-child',
          run_id: 'run-thread-child',
          message_id: 'MSG-thread-child',
        },
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-thread-root',
        task_id: 'task-thread-root',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:58:00Z',
      },
      {
        id: 'run-thread-child',
        task_id: 'task-thread-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:59:00Z',
      },
      {
        id: 'run-thread-grandchild',
        task_id: 'task-thread-grandchild',
        agent: 'gemini',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    expect(tree.children).toHaveLength(1)

    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-thread-root')
    const childTask = rootTask.children.find((node) => node.type === 'task' && node.id === 'task-thread-child')
    expect(childTask).toBeDefined()

    const grandchildTask = childTask?.children.find(
      (node) => node.type === 'task' && node.id === 'task-thread-grandchild'
    )
    expect(grandchildTask).toBeDefined()
  })

  it('keeps threaded hierarchy when parent task summary is absent from scoped payload', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-thread-child',
        status: 'running',
        last_activity: '2026-02-22T21:01:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-root',
          run_id: 'run-thread-root',
          message_id: 'MSG-thread-root',
        },
      },
      {
        id: 'task-thread-grandchild',
        status: 'running',
        last_activity: '2026-02-22T21:02:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-child',
          run_id: 'run-thread-child',
          message_id: 'MSG-thread-child',
        },
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-thread-child',
        task_id: 'task-thread-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:59:00Z',
      },
      {
        id: 'run-thread-grandchild',
        task_id: 'task-thread-grandchild',
        agent: 'gemini',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    expect(tree.children).toHaveLength(1)

    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-thread-root')

    const childTask = rootTask.children.find((node) => node.type === 'task' && node.id === 'task-thread-child')
    expect(childTask).toBeDefined()
    const grandchildTask = childTask?.children.find(
      (node) => node.type === 'task' && node.id === 'task-thread-grandchild'
    )
    expect(grandchildTask).toBeDefined()
  })

  it('prefers thread_parent hierarchy when run parent edges point to a conflicting task', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-thread-root',
        status: 'running',
        last_activity: '2026-02-22T21:00:00Z',
      },
      {
        id: 'task-thread-child',
        status: 'running',
        last_activity: '2026-02-22T21:01:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-root',
          run_id: 'run-thread-root',
          message_id: 'MSG-thread-root',
        },
      },
      {
        id: 'task-noise-parent',
        status: 'running',
        last_activity: '2026-02-22T21:02:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-thread-root',
        task_id: 'task-thread-root',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:58:00Z',
      },
      {
        id: 'run-noise-parent',
        task_id: 'task-noise-parent',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:59:00Z',
      },
      {
        id: 'run-thread-child',
        task_id: 'task-thread-child',
        agent: 'gemini',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
        parent_run_id: 'run-noise-parent',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    const rootTask = tree.children.find((node) => node.type === 'task' && node.id === 'task-thread-root')
    expect(rootTask).toBeDefined()

    const nestedChild = rootTask?.children.find((node) => node.type === 'task' && node.id === 'task-thread-child')
    expect(nestedChild).toBeDefined()
  })

  it('nests threaded queued tasks when only parent task has an active run', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-thread-root',
        status: 'running',
        last_activity: '2026-02-22T21:00:00Z',
      },
      {
        id: 'task-thread-child',
        status: 'queued',
        last_activity: '2026-02-22T21:01:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-root',
          run_id: 'run-thread-root',
          message_id: 'MSG-thread-root',
        },
      },
      {
        id: 'task-thread-grandchild',
        status: 'queued',
        last_activity: '2026-02-22T21:02:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-child',
          run_id: 'run-thread-child',
          message_id: 'MSG-thread-child',
        },
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-thread-root',
        task_id: 'task-thread-root',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:58:00Z',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    expect(tree.children).toHaveLength(1)

    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-thread-root')

    const childTask = rootTask.children.find((node) => node.type === 'task' && node.id === 'task-thread-child')
    expect(childTask).toBeDefined()

    const grandchildTask = childTask?.children.find(
      (node) => node.type === 'task' && node.id === 'task-thread-grandchild'
    )
    expect(grandchildTask).toBeDefined()
  })

  it('keeps child runs nested when parent task restarts from previous run', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-20260222-190000-root',
        status: 'running',
        last_activity: '2026-02-22T19:05:00Z',
      },
      {
        id: 'task-20260222-190100-child',
        status: 'running',
        last_activity: '2026-02-22T19:03:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root-1',
        task_id: 'task-20260222-190000-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:00:00Z',
        end_time: '2026-02-22T19:01:00Z',
      },
      {
        id: 'run-child-1',
        task_id: 'task-20260222-190100-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T19:02:00Z',
        parent_run_id: 'run-root-1',
      },
      {
        id: 'run-root-2',
        task_id: 'task-20260222-190000-root',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T19:04:00Z',
        previous_run_id: 'run-root-1',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    expect(tree.children).toHaveLength(1)

    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-20260222-190000-root')
    expect(rootTask.children.some((node) => node.id === 'task-20260222-190100-child')).toBe(true)

    const oldRootRun = rootTask.children.find((node) => node.id === 'run-root-1')
    const latestRootRun = rootTask.children.find((node) => node.id === 'run-root-2')
    expect(oldRootRun).toBeDefined()
    expect(latestRootRun).toBeDefined()
    expect(oldRootRun?.children).toHaveLength(1)
    expect(oldRootRun?.children[0].id).toBe('run-child-1')
  })

  it('keeps child task nested when its latest run only links through previous_run_id', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-20260222-191500-root',
        status: 'completed',
        last_activity: '2026-02-22T19:15:00Z',
      },
      {
        id: 'task-20260222-191600-child',
        status: 'running',
        last_activity: '2026-02-22T19:17:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root-1',
        task_id: 'task-20260222-191500-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:10:00Z',
        end_time: '2026-02-22T19:11:00Z',
      },
      {
        id: 'run-child-1',
        task_id: 'task-20260222-191600-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:12:00Z',
        end_time: '2026-02-22T19:13:00Z',
        parent_run_id: 'run-root-1',
      },
      {
        id: 'run-child-2',
        task_id: 'task-20260222-191600-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T19:16:00Z',
        previous_run_id: 'run-child-1',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    expect(tree.children).toHaveLength(1)

    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-20260222-191500-root')

    const childTask = rootTask.children.find(
      (node) => node.type === 'task' && node.id === 'task-20260222-191600-child'
    )
    expect(childTask).toBeDefined()
    expect(childTask?.latestRunId).toBe('run-child-2')
  })

  it('keeps selected-task cross-task anchors when latest selected run links only within same task', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-root',
        status: 'completed',
        last_activity: '2026-02-22T19:30:00Z',
      },
      {
        id: 'task-selected',
        status: 'completed',
        last_activity: '2026-02-22T19:33:00Z',
      },
      {
        id: 'task-grandchild',
        status: 'completed',
        last_activity: '2026-02-22T19:32:00Z',
      },
    ]

    // Simulates backend runs/flat payload for:
    // active_only=1&selected_task_id=task-selected&selected_task_limit=1
    // where latest selected run links only to previous run in the same task,
    // and the older selected run carries the cross-task parent anchor.
    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:30:00Z',
        end_time: '2026-02-22T19:30:30Z',
      },
      {
        id: 'run-selected-old',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:31:00Z',
        end_time: '2026-02-22T19:31:30Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-grandchild',
        task_id: 'task-grandchild',
        agent: 'gemini',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:32:00Z',
        end_time: '2026-02-22T19:32:30Z',
        parent_run_id: 'run-selected-old',
      },
      {
        id: 'run-selected-new',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:33:00Z',
        end_time: '2026-02-22T19:33:30Z',
        previous_run_id: 'run-selected-old',
      },
    ]

    const filtered = selectTreeRuns(tasks, runs, 'task-selected')
    expect(filtered.map((run) => run.id)).toEqual([
      'run-root',
      'run-selected-old',
      'run-grandchild',
      'run-selected-new',
    ])

    const tree = buildTree('conductor-loop', tasks, filtered)
    expect(tree.children).toHaveLength(1)

    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-root')

    const selectedTask = rootTask.children.find(
      (node) => node.type === 'task' && node.id === 'task-selected'
    )
    expect(selectedTask).toBeDefined()

    const grandchildTask = selectedTask?.children.find(
      (node) => node.type === 'task' && node.id === 'task-grandchild'
    )
    expect(grandchildTask).toBeDefined()
  })

  it('builds level-3 nesting from backend-filtered runs payload without extra selection', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-root',
        status: 'completed',
        last_activity: '2026-02-22T19:30:00Z',
      },
      {
        id: 'task-selected',
        status: 'completed',
        last_activity: '2026-02-22T19:33:00Z',
      },
      {
        id: 'task-grandchild',
        status: 'completed',
        last_activity: '2026-02-22T19:32:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:30:00Z',
        end_time: '2026-02-22T19:30:30Z',
      },
      {
        id: 'run-selected-old',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:31:00Z',
        end_time: '2026-02-22T19:31:30Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-grandchild',
        task_id: 'task-grandchild',
        agent: 'gemini',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:32:00Z',
        end_time: '2026-02-22T19:32:30Z',
        parent_run_id: 'run-selected-old',
      },
      {
        id: 'run-selected-new',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:33:00Z',
        end_time: '2026-02-22T19:33:30Z',
        previous_run_id: 'run-selected-old',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    expect(tree.children).toHaveLength(1)

    const rootTask = tree.children[0]
    expect(rootTask.id).toBe('task-root')

    const selectedTask = rootTask.children.find(
      (node) => node.type === 'task' && node.id === 'task-selected'
    )
    expect(selectedTask).toBeDefined()

    const grandchildTask = selectedTask?.children.find(
      (node) => node.type === 'task' && node.id === 'task-grandchild'
    )
    expect(grandchildTask).toBeDefined()
  })

  it('builds selection paths once for subtree expansion checks', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-root',
        status: 'running',
        last_activity: '2026-02-22T19:05:00Z',
      },
      {
        id: 'task-child',
        status: 'running',
        last_activity: '2026-02-22T19:03:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T19:00:00Z',
      },
      {
        id: 'run-child',
        task_id: 'task-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T19:01:00Z',
        parent_run_id: 'run-root',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    const path = buildSelectionPathNodeIDs(tree, 'task-root', 'run-child')
    expect(path.has('conductor-loop')).toBe(true)
    expect(path.has('task-root')).toBe(true)
    expect(path.has('run-root')).toBe(true)
    expect(path.has('run-child')).toBe(true)
  })

  it('marks inline latest task rows as selected path targets', () => {
    const tasks: TaskSummary[] = [
      {
        id: 'task-inline',
        status: 'running',
        last_activity: '2026-02-22T19:05:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-inline',
        task_id: 'task-inline',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T19:00:00Z',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)
    const path = buildSelectionPathNodeIDs(tree, 'task-inline', 'run-inline')
    expect(path.has('conductor-loop')).toBe(true)
    expect(path.has('task-inline')).toBe(true)
    expect(path.has('run-inline')).toBe(false)
  })

  it('nests tasks correctly when backend-scoped payload contains only the ancestry-consistent anchor', () => {
    // Regression scenario: backend selected-task scoped payloads used to include unrelated
    // cross-task anchor runs (by wall-clock proximity) instead of the ancestry-consistent one.
    // This caused the frontend to attach task-child under the wrong parent task.
    // Fix: backend now only includes ancestry-consistent anchors in scoped payloads.
    // This test verifies that given the CORRECT scoped payload (ancestry-consistent anchor
    // only, no unrelated branch), buildTree nests task-child under task-root-a.
    //
    // Payload shape mirrors what /api/projects/{p}/runs/flat?active_only=1&selected_task_id=task-grandchild&selected_task_limit=1
    // returns after the backend fix: only the ancestry chain is included, not the unrelated
    // run-child-b (which had parent=run-root-b and was newer than run-child-a).
    const tasks: TaskSummary[] = [
      {
        id: 'task-root-a',
        status: 'completed',
        last_activity: '2026-02-22T19:01:00Z',
      },
      {
        id: 'task-root-b',
        status: 'completed',
        last_activity: '2026-02-22T19:03:00Z',
      },
      {
        id: 'task-child',
        status: 'completed',
        last_activity: '2026-02-22T19:05:00Z',
      },
      {
        id: 'task-grandchild',
        status: 'failed',
        last_activity: '2026-02-22T19:06:00Z',
      },
    ]

    // Scoped payload: only ancestry-consistent chain included.
    // run-root-b and run-child-b (which had parent=run-root-b) are absent.
    const runs: FlatRunItem[] = [
      {
        id: 'run-root-a',
        task_id: 'task-root-a',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:00:00Z',
        end_time: '2026-02-22T19:01:00Z',
      },
      {
        id: 'run-child-a',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:01:30Z',
        end_time: '2026-02-22T19:02:30Z',
        parent_run_id: 'run-root-a',
      },
      {
        id: 'run-child-selected',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:04:00Z',
        end_time: '2026-02-22T19:05:00Z',
        previous_run_id: 'run-child-a',
      },
      {
        id: 'run-grandchild-selected',
        task_id: 'task-grandchild',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T19:05:30Z',
        end_time: '2026-02-22T19:06:00Z',
        parent_run_id: 'run-child-selected',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)

    // task-root-a should be a direct child of the project root.
    const rootA = tree.children.find((n) => n.type === 'task' && n.id === 'task-root-a')
    expect(rootA).toBeDefined()

    // task-child should be nested under task-root-a (via run-child-a → run-root-a cross-task link).
    const childUnderRootA = rootA?.children.find((n) => n.type === 'task' && n.id === 'task-child')
    expect(childUnderRootA).toBeDefined()

    // task-grandchild should be nested under task-child.
    const grandchild = childUnderRootA?.children.find((n) => n.type === 'task' && n.id === 'task-grandchild')
    expect(grandchild).toBeDefined()

    // task-root-b may appear in the tree (it's in the tasks list) but task-child must NOT
    // be nested under it — task-child has no cross-task link to task-root-b in this payload.
    const rootB = tree.children.find((n) => n.type === 'task' && n.id === 'task-root-b')
    if (rootB) {
      const childUnderRootB = rootB.children.find((n) => n.type === 'task' && n.id === 'task-child')
      expect(childUnderRootB).toBeUndefined()
    }

    // task-child must NOT appear as a top-level task (it should be nested under task-root-a).
    const childAtRoot = tree.children.find((n) => n.type === 'task' && n.id === 'task-child')
    expect(childAtRoot).toBeUndefined()
  })

  it('uses latest cross-task run link when payload includes conflicting run parent edges', () => {
    // Documents the known frontend behavior: when the flat-runs payload contains two
    // cross-task runs for the same task (run-child-a → run-root-a and run-child-b → run-root-b,
    // with run-child-b being newer), buildTree attaches task-child under whichever parent had
    // the LATER cross-task run. This is why the backend anchor-selection fix in
    // seedSelectedTaskParentAnchorRunInfos (handlers_projects.go) is critical: the frontend
    // cannot independently distinguish "correct" from "unrelated" anchors in the payload.
    const tasks: TaskSummary[] = [
      {
        id: 'task-root-a',
        status: 'completed',
        last_activity: '2026-02-22T19:01:00Z',
      },
      {
        id: 'task-root-b',
        status: 'completed',
        last_activity: '2026-02-22T19:03:00Z',
      },
      {
        id: 'task-child',
        status: 'completed',
        last_activity: '2026-02-22T19:05:00Z',
      },
    ]

    const runs: FlatRunItem[] = [
      {
        id: 'run-root-a',
        task_id: 'task-root-a',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:00:00Z',
        end_time: '2026-02-22T19:01:00Z',
      },
      {
        id: 'run-child-a',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:01:30Z',
        end_time: '2026-02-22T19:02:30Z',
        parent_run_id: 'run-root-a',
      },
      {
        id: 'run-root-b',
        task_id: 'task-root-b',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:02:30Z',
        end_time: '2026-02-22T19:03:00Z',
      },
      // run-child-b is newer than run-child-a and has a cross-task link to run-root-b.
      // When both are present in the payload, buildTree attaches task-child under task-root-b
      // because run-child-b has a later end_time. This documents the regression vector.
      {
        id: 'run-child-b',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:03:30Z',
        end_time: '2026-02-22T19:04:00Z',
        parent_run_id: 'run-root-b',
      },
    ]

    const tree = buildTree('conductor-loop', tasks, runs)

    // With conflicting anchors in the payload, the frontend uses the LATEST cross-task link.
    // task-child ends up under task-root-b (run-child-b is newer than run-child-a).
    const rootB = tree.children.find((n) => n.type === 'task' && n.id === 'task-root-b')
    expect(rootB).toBeDefined()
    const childUnderRootB = rootB?.children.find((n) => n.type === 'task' && n.id === 'task-child')
    expect(childUnderRootB).toBeDefined()

    // task-child does NOT appear under task-root-a in this conflicting scenario.
    const rootA = tree.children.find((n) => n.type === 'task' && n.id === 'task-root-a')
    const childUnderRootA = rootA?.children.find((n) => n.type === 'task' && n.id === 'task-child')
    expect(childUnderRootA).toBeUndefined()
  })

})
