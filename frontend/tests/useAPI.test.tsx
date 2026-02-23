import { describe, expect, it } from 'vitest'
import type { FlatRunItem, ProjectStats } from '../src/types'
import {
  messageFallbackRefetchIntervalFor,
  MESSAGE_FALLBACK_REFETCH_MS,
  mergeFlatRunsForTree,
  runFileRefetchIntervalFor,
  runsFlatRefetchIntervalFor,
  runsFlatScopedQueryKey,
  scopedRunsForTree,
  stabilizeFlatRuns,
  stabilizeProjectStats,
} from '../src/hooks/useAPI'

describe('runsFlatRefetchIntervalFor', () => {
  it('returns fast fallback interval when no runs are available yet', () => {
    expect(runsFlatRefetchIntervalFor(undefined)).toBe(1500)
    expect(runsFlatRefetchIntervalFor([])).toBe(1500)
  })

  it('uses tighter polling while active runs are present', () => {
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
    expect(runsFlatRefetchIntervalFor(activeRuns)).toBe(800)
  })

  it('backs off when all runs are terminal', () => {
    const idleRuns: FlatRunItem[] = [
      {
        id: 'run-1',
        task_id: 'task-1',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
      },
    ]
    expect(runsFlatRefetchIntervalFor(idleRuns)).toBe(2500)
  })

  it('uses stream-synchronized cadence when SSE is healthy', () => {
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
    const idleRuns: FlatRunItem[] = [
      {
        id: 'run-2',
        task_id: 'task-2',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:00:00Z',
      },
    ]

    expect(runsFlatRefetchIntervalFor(undefined, 'open')).toBe(1000)
    expect(runsFlatRefetchIntervalFor(activeRuns, 'open')).toBe(810)
    expect(runsFlatRefetchIntervalFor(idleRuns, 'open')).toBe(3000)
  })

  it('keeps fast fallback polling when SSE is degraded', () => {
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
    expect(runsFlatRefetchIntervalFor(activeRuns, 'reconnecting')).toBe(800)
  })

  it('keeps idle fallback polling bounded when SSE is degraded', () => {
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
    expect(runsFlatRefetchIntervalFor(idleRuns, 'reconnecting')).toBe(2500)
  })

  it('keys runs-flat query by selected task context to refetch immediately on selection change', () => {
    expect(runsFlatScopedQueryKey('proj-1', undefined, undefined)).toEqual(['runs-flat', 'proj-1', '', 0])
    expect(runsFlatScopedQueryKey('proj-1', 'task-1', 1)).toEqual(['runs-flat', 'proj-1', 'task-1', 1])
    expect(runsFlatScopedQueryKey('proj-1', 'task-2', 1)).toEqual(['runs-flat', 'proj-1', 'task-2', 1])
  })

  it('reuses previous runs-flat slice when polled payload is unchanged', () => {
    const previous: FlatRunItem[] = [
      {
        id: 'run-1',
        task_id: 'task-1',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
        parent_run_id: 'run-0',
      },
      {
        id: 'run-0',
        task_id: 'task-1',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:58:00Z',
        end_time: '2026-02-22T20:59:00Z',
      },
    ]
    const incoming = previous.map((run) => ({ ...run }))
    expect(stabilizeFlatRuns(previous, incoming)).toBe(previous)
  })

  it('uses incoming runs-flat slice when any run field changed', () => {
    const previous: FlatRunItem[] = [
      {
        id: 'run-1',
        task_id: 'task-1',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
      },
    ]
    const incoming: FlatRunItem[] = [
      {
        ...previous[0],
        status: 'completed',
        end_time: '2026-02-22T21:01:00Z',
      },
    ]
    expect(stabilizeFlatRuns(previous, incoming)).toBe(incoming)
  })

  it('keeps existing ancestor runs when scoped payload only has selected-task slice', () => {
    const previousProjectRuns: FlatRunItem[] = [
      {
        id: 'run-parent',
        task_id: 'task-parent',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:58:00Z',
      },
      {
        id: 'run-child',
        task_id: 'task-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
        parent_run_id: 'run-parent',
      },
    ]
    const incomingScopedRuns: FlatRunItem[] = [
      {
        id: 'run-child',
        task_id: 'task-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
        parent_run_id: 'run-parent',
      },
    ]

    expect(mergeFlatRunsForTree(previousProjectRuns, incomingScopedRuns)).toBe(previousProjectRuns)
  })

  it('upserts scoped run updates into cached project runs without dropping ancestry', () => {
    const previousProjectRuns: FlatRunItem[] = [
      {
        id: 'run-parent',
        task_id: 'task-parent',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:58:00Z',
      },
      {
        id: 'run-child',
        task_id: 'task-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
        parent_run_id: 'run-parent',
      },
    ]
    const incomingScopedRuns: FlatRunItem[] = [
      {
        id: 'run-child',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
        end_time: '2026-02-22T21:01:00Z',
        parent_run_id: 'run-parent',
      },
      {
        id: 'run-grandchild',
        task_id: 'task-grandchild',
        agent: 'gemini',
        status: 'queued',
        exit_code: -1,
        start_time: '2026-02-22T21:02:00Z',
        parent_run_id: 'run-child',
      },
    ]

    const merged = mergeFlatRunsForTree(previousProjectRuns, incomingScopedRuns)
    expect(merged).toEqual([
      previousProjectRuns[0],
      {
        id: 'run-child',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
        end_time: '2026-02-22T21:01:00Z',
        parent_run_id: 'run-parent',
      },
      {
        id: 'run-grandchild',
        task_id: 'task-grandchild',
        agent: 'gemini',
        status: 'queued',
        exit_code: -1,
        start_time: '2026-02-22T21:02:00Z',
        parent_run_id: 'run-child',
      },
    ])
  })

  it('keeps root-tree scoped payload bounded when no task is selected', () => {
    const previousScopedRuns: FlatRunItem[] = [
      {
        id: 'run-parent-old',
        task_id: 'task-parent',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:50:00Z',
      },
      {
        id: 'run-selected-old',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:55:00Z',
      },
    ]
    const incomingScopedRuns: FlatRunItem[] = [
      {
        id: 'run-selected-new',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
      },
    ]

    const result = scopedRunsForTree(previousScopedRuns, incomingScopedRuns, undefined)
    expect(result).toBe(incomingScopedRuns)
  })

  it('retains selected-task ancestry continuity within scoped cache', () => {
    const previousScopedRuns: FlatRunItem[] = [
      {
        id: 'run-parent',
        task_id: 'task-parent',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:58:00Z',
      },
      {
        id: 'run-child',
        task_id: 'task-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
        parent_run_id: 'run-parent',
      },
    ]
    const incomingScopedRuns: FlatRunItem[] = [
      {
        id: 'run-child',
        task_id: 'task-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
        parent_run_id: 'run-parent',
      },
    ]

    const result = scopedRunsForTree(previousScopedRuns, incomingScopedRuns, 'task-child')
    expect(result).toBe(previousScopedRuns)
  })

  it('bounds selected-task limited scoped cache across long restart chains', () => {
    let scopedRuns: FlatRunItem[] | undefined

    for (let i = 0; i < 120; i += 1) {
      const runID = `run-selected-${String(i).padStart(4, '0')}`
      const previousRunID = i > 0
        ? `run-selected-${String(i - 1).padStart(4, '0')}`
        : undefined
      const incoming: FlatRunItem[] = [
        {
          id: runID,
          task_id: 'task-selected',
          agent: 'codex',
          status: 'running',
          exit_code: -1,
          start_time: `2026-02-22T21:${String(Math.floor(i / 60)).padStart(2, '0')}:${String(i % 60).padStart(2, '0')}Z`,
          previous_run_id: previousRunID,
        },
      ]

      scopedRuns = scopedRunsForTree(scopedRuns, incoming, 'task-selected', 1)
      expect(scopedRuns.some((run) => run.id === runID)).toBe(true)
      expect(scopedRuns.length).toBeLessThanOrEqual(2)
    }
  })

  it('rehydrates missing parent ancestry in selected-task limited mode without reviving unrelated history', () => {
    const previousScopedRuns: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:58:00Z',
      },
      {
        id: 'run-selected-old',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:59:00Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-unrelated',
        task_id: 'task-unrelated',
        agent: 'gemini',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
      },
    ]

    const incomingScopedRuns: FlatRunItem[] = [
      {
        id: 'run-selected-new',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:01:00Z',
        parent_run_id: 'run-root',
        previous_run_id: 'run-selected-old',
      },
    ]

    const result = scopedRunsForTree(previousScopedRuns, incomingScopedRuns, 'task-selected', 1)
    const ids = result.map((run) => run.id)
    expect(ids).toEqual(['run-root', 'run-selected-old', 'run-selected-new'])
    expect(ids).not.toContain('run-unrelated')
  })

  it('uses project cache ancestry when selected-task cache starts empty', () => {
    const previousProjectRuns: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:58:00Z',
      },
      {
        id: 'run-selected-old',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:59:00Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-unrelated',
        task_id: 'task-unrelated',
        agent: 'gemini',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
      },
    ]

    const incomingScopedRuns: FlatRunItem[] = [
      {
        id: 'run-selected-new',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:01:00Z',
        parent_run_id: 'run-root',
        previous_run_id: 'run-selected-old',
      },
    ]

    const result = scopedRunsForTree(
      undefined,
      incomingScopedRuns,
      'task-selected',
      1,
      previousProjectRuns
    )
    const ids = result.map((run) => run.id)
    expect(ids).toEqual(['run-root', 'run-selected-old', 'run-selected-new'])
    expect(ids).not.toContain('run-unrelated')
  })

  it('prefers richer project ancestry when scoped cache has stale duplicate run metadata', () => {
    const previousScopedRuns: FlatRunItem[] = [
      {
        id: 'run-selected-old',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:59:00Z',
      },
    ]

    const previousProjectRuns: FlatRunItem[] = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:58:00Z',
      },
      {
        id: 'run-selected-old',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:59:00Z',
        parent_run_id: 'run-root',
      },
    ]

    const incomingScopedRuns: FlatRunItem[] = [
      {
        id: 'run-selected-new',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:01:00Z',
        previous_run_id: 'run-selected-old',
      },
    ]

    const result = scopedRunsForTree(
      previousScopedRuns,
      incomingScopedRuns,
      'task-selected',
      1,
      previousProjectRuns
    )
    const ids = result.map((run) => run.id)
    expect(ids).toEqual(['run-root', 'run-selected-old', 'run-selected-new'])
  })

  it('reuses previous project stats when payload is unchanged', () => {
    const previous: ProjectStats = {
      project_id: 'proj-1',
      total_tasks: 4,
      total_runs: 12,
      running_runs: 1,
      completed_runs: 10,
      failed_runs: 1,
      crashed_runs: 0,
      message_bus_files: 2,
      message_bus_total_bytes: 64000,
    }
    const incoming: ProjectStats = { ...previous }
    expect(stabilizeProjectStats(previous, incoming)).toBe(previous)
  })

  it('uses incoming project stats when counters changed', () => {
    const previous: ProjectStats = {
      project_id: 'proj-1',
      total_tasks: 4,
      total_runs: 12,
      running_runs: 1,
      completed_runs: 10,
      failed_runs: 1,
      crashed_runs: 0,
      message_bus_files: 2,
      message_bus_total_bytes: 64000,
    }
    const incoming: ProjectStats = {
      ...previous,
      running_runs: 0,
      completed_runs: 11,
    }
    expect(stabilizeProjectStats(previous, incoming)).toBe(incoming)
  })
})

describe('runFileRefetchIntervalFor', () => {
  it('enables fallback polling for active runs', () => {
    expect(runFileRefetchIntervalFor('running')).toBe(2500)
    expect(runFileRefetchIntervalFor('queued')).toBe(2500)
  })

  it('disables fallback polling while run-file stream is healthy', () => {
    expect(runFileRefetchIntervalFor('running', 'connecting')).toBe(false)
    expect(runFileRefetchIntervalFor('running', 'open')).toBe(false)
    expect(runFileRefetchIntervalFor('queued', 'open')).toBe(false)
  })

  it('restores fallback polling when run-file stream is degraded', () => {
    expect(runFileRefetchIntervalFor('running', 'reconnecting')).toBe(2500)
    expect(runFileRefetchIntervalFor('running', 'error')).toBe(2500)
    expect(runFileRefetchIntervalFor('running', 'disabled')).toBe(2500)
  })

  it('disables fallback polling for terminal runs', () => {
    expect(runFileRefetchIntervalFor('completed')).toBe(false)
    expect(runFileRefetchIntervalFor('failed')).toBe(false)
    expect(runFileRefetchIntervalFor(undefined)).toBe(false)
  })
})

describe('messageFallbackRefetchIntervalFor', () => {
  it('disables fallback polling when stream is healthy', () => {
    expect(messageFallbackRefetchIntervalFor('open')).toBe(false)
    expect(messageFallbackRefetchIntervalFor('connecting')).toBe(false)
  })

  it('uses bounded polling when stream is degraded', () => {
    expect(messageFallbackRefetchIntervalFor('reconnecting')).toBe(MESSAGE_FALLBACK_REFETCH_MS)
    expect(messageFallbackRefetchIntervalFor('error')).toBe(MESSAGE_FALLBACK_REFETCH_MS)
    expect(messageFallbackRefetchIntervalFor('disabled')).toBe(MESSAGE_FALLBACK_REFETCH_MS)
  })

  it('keeps legacy default behavior when stream state is unknown', () => {
    expect(messageFallbackRefetchIntervalFor(undefined)).toBe(false)
  })
})
