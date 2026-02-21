import { describe, expect, it } from 'vitest'
import { buildTree } from '../src/utils/treeBuilder'
import type { FlatRunItem, TaskSummary } from '../src/types'

describe('treeBuilder', () => {
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
})
