import React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { FlatRunItem, RunInfo, TaskDetail } from '../src/types'

const mockedSSE = vi.hoisted(() => ({
  handlers: undefined as Record<string, (event: MessageEvent) => void> | undefined,
}))

vi.mock('../src/hooks/useSSE', () => ({
  useSSE: (_url?: string, options?: { events?: Record<string, (event: MessageEvent) => void> }) => {
    mockedSSE.handlers = options?.events
    return { state: _url ? 'open' : 'disabled', errorCount: 0 }
  },
}))

import {
  LOG_REFRESH_DELAY_MS,
  STATUS_REFRESH_DELAY_MS,
  useLiveRunRefresh,
} from '../src/hooks/useLiveRunRefresh'

function HookHarness({
  projectId,
  taskId,
  runId,
}: {
  projectId?: string
  taskId?: string
  runId?: string
}) {
  useLiveRunRefresh({ projectId, taskId, runId })
  return null
}

describe('useLiveRunRefresh', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockedSSE.handlers = undefined
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('invalidates project queries when a status event references an unknown run', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(['runs-flat', 'proj-1'], [{ id: 'run-1' }])

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      mockedSSE.handlers?.status(new MessageEvent('status', {
        data: JSON.stringify({ run_id: 'run-2', task_id: 'task-2', status: 'completed' }),
      }))
      vi.advanceTimersByTime(STATUS_REFRESH_DELAY_MS + 50)
    })

    const keys = invalidateSpy.mock.calls
      .map((call) => call[0]?.queryKey)
      .filter(Boolean)
      .map((queryKey) => JSON.stringify(queryKey))

    expect(keys).toContain(JSON.stringify(['tasks', 'proj-1']))
    expect(keys).toContain(JSON.stringify(['runs-flat', 'proj-1']))
    expect(keys).toContain(JSON.stringify(['task', 'proj-1', 'task-1']))
    expect(keys).not.toContain(JSON.stringify(['run', 'proj-1', 'task-1', 'run-1']))
    expect(keys).not.toContain(JSON.stringify(['projects']))
  })

  it('patches selected run status in cache immediately before invalidation', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const flatRun: FlatRunItem = {
      id: 'run-1',
      task_id: 'task-1',
      agent: 'codex',
      status: 'running',
      exit_code: -1,
      start_time: '2026-02-22T21:00:00Z',
      end_time: undefined,
    }
    queryClient.setQueryData(['runs-flat', 'proj-1'], [flatRun])

    const taskDetail: TaskDetail = {
      id: 'task-1',
      project_id: 'proj-1',
      status: 'running',
      last_activity: '2026-02-22T21:00:00Z',
      created_at: '2026-02-22T21:00:00Z',
      done: false,
      state: 'active',
      runs: [{
        id: 'run-1',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
      }],
    }
    queryClient.setQueryData(['task', 'proj-1', 'task-1'], taskDetail)

    const runInfo: RunInfo = {
      version: 1,
      run_id: 'run-1',
      project_id: 'proj-1',
      task_id: 'task-1',
      parent_run_id: '',
      previous_run_id: '',
      agent: 'codex',
      pid: 123,
      pgid: 123,
      start_time: '2026-02-22T21:00:00Z',
      end_time: '0001-01-01T00:00:00Z',
      exit_code: -1,
      cwd: '/tmp',
    }
    queryClient.setQueryData(['run', 'proj-1', 'task-1', 'run-1'], runInfo)

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      mockedSSE.handlers?.status(new MessageEvent('status', {
        data: JSON.stringify({
          run_id: 'run-1',
          project_id: 'proj-1',
          task_id: 'task-1',
          status: 'completed',
          exit_code: 0,
        }),
      }))
    })

    const patchedFlatRuns = queryClient.getQueryData<FlatRunItem[]>(['runs-flat', 'proj-1']) ?? []
    expect(patchedFlatRuns[0]?.status).toBe('completed')
    expect(patchedFlatRuns[0]?.exit_code).toBe(0)

    const patchedTask = queryClient.getQueryData<TaskDetail>(['task', 'proj-1', 'task-1'])
    expect(patchedTask?.runs[0]?.status).toBe('completed')
    expect(patchedTask?.runs[0]?.exit_code).toBe(0)

    const patchedRunInfo = queryClient.getQueryData<RunInfo>(['run', 'proj-1', 'task-1', 'run-1'])
    expect(patchedRunInfo?.exit_code).toBe(0)
    expect(patchedRunInfo?.end_time).not.toBe('0001-01-01T00:00:00Z')

    expect(invalidateSpy).not.toHaveBeenCalled()
    act(() => {
      vi.advanceTimersByTime(STATUS_REFRESH_DELAY_MS + 50)
    })
    expect(invalidateSpy).toHaveBeenCalled()
    const keys = invalidateSpy.mock.calls
      .map((call) => call[0]?.queryKey)
      .filter(Boolean)
      .map((queryKey) => JSON.stringify(queryKey))
    expect(keys).toContain(JSON.stringify(['task', 'proj-1', 'task-1']))
    expect(keys).toContain(JSON.stringify(['run', 'proj-1', 'task-1', 'run-1']))
    expect(keys).not.toContain(JSON.stringify(['tasks', 'proj-1']))
    expect(keys).not.toContain(JSON.stringify(['runs-flat', 'proj-1']))
  })

  it('avoids full project invalidation for known runs even when task_id is omitted', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(['runs-flat', 'proj-1'], [{ id: 'run-1', task_id: 'task-1' }])

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      mockedSSE.handlers?.status(new MessageEvent('status', {
        data: JSON.stringify({ run_id: 'run-1', project_id: 'proj-1', status: 'running' }),
      }))
      vi.advanceTimersByTime(STATUS_REFRESH_DELAY_MS + 50)
    })

    const keys = invalidateSpy.mock.calls
      .map((call) => call[0]?.queryKey)
      .filter(Boolean)
      .map((queryKey) => JSON.stringify(queryKey))

    expect(keys).toContain(JSON.stringify(['task', 'proj-1', 'task-1']))
    expect(keys).toContain(JSON.stringify(['run', 'proj-1', 'task-1', 'run-1']))
    expect(keys).not.toContain(JSON.stringify(['tasks', 'proj-1']))
    expect(keys).not.toContain(JSON.stringify(['runs-flat', 'proj-1']))
  })

  it('ignores events from runs outside the selected project cache', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(['runs-flat', 'proj-1'], [{ id: 'run-1' }])

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      mockedSSE.handlers?.status(new MessageEvent('status', {
        data: JSON.stringify({ run_id: 'run-external', project_id: 'proj-external', status: 'running' }),
      }))
      vi.advanceTimersByTime(STATUS_REFRESH_DELAY_MS + 200)
    })

    expect(invalidateSpy).not.toHaveBeenCalled()
  })

  it('does not invalidate for known unrelated runs in the same project', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(['runs-flat', 'proj-1'], [
      { id: 'run-1', task_id: 'task-1' },
      { id: 'run-2', task_id: 'task-2' },
    ])

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      mockedSSE.handlers?.status(new MessageEvent('status', {
        data: JSON.stringify({ run_id: 'run-2', project_id: 'proj-1', task_id: 'task-2', status: 'running' }),
      }))
      vi.advanceTimersByTime(STATUS_REFRESH_DELAY_MS + 100)
    })

    expect(invalidateSpy).not.toHaveBeenCalled()
  })

  it('refreshes quickly for newly discovered runs in the selected project', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(['runs-flat', 'proj-1'], [{ id: 'run-1' }])

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      mockedSSE.handlers?.status(new MessageEvent('status', {
        data: JSON.stringify({ run_id: 'run-new', project_id: 'proj-1', task_id: 'task-1', status: 'running' }),
      }))
      vi.advanceTimersByTime(STATUS_REFRESH_DELAY_MS + 50)
    })

    expect(invalidateSpy).toHaveBeenCalled()
    const keys = invalidateSpy.mock.calls
      .map((call) => call[0]?.queryKey)
      .filter(Boolean)
      .map((queryKey) => JSON.stringify(queryKey))
    expect(keys).toContain(JSON.stringify(['runs-flat', 'proj-1']))
  })

  it('skips task-list invalidation when unknown run belongs to an existing task', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(['runs-flat', 'proj-1'], [{ id: 'run-1', task_id: 'task-1' }])
    queryClient.setQueryData(['tasks', 'proj-1'], [
      {
        id: 'task-1',
        status: 'running',
        last_activity: '2026-02-22T21:00:00Z',
      },
    ])

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      mockedSSE.handlers?.status(new MessageEvent('status', {
        data: JSON.stringify({
          run_id: 'run-new',
          project_id: 'proj-1',
          task_id: 'task-1',
          status: 'running',
        }),
      }))
      vi.advanceTimersByTime(STATUS_REFRESH_DELAY_MS + 50)
    })

    const keys = invalidateSpy.mock.calls
      .map((call) => call[0]?.queryKey)
      .filter(Boolean)
      .map((queryKey) => JSON.stringify(queryKey))
    expect(keys).toContain(JSON.stringify(['runs-flat', 'proj-1']))
    expect(keys).toContain(JSON.stringify(['task', 'proj-1', 'task-1']))
    expect(keys).not.toContain(JSON.stringify(['tasks', 'proj-1']))
  })

  it('coalesces bursty status events into a single invalidation cycle', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(['runs-flat', 'proj-1'], [{ id: 'run-1', task_id: 'task-1' }])

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      for (let i = 0; i < 4; i += 1) {
        mockedSSE.handlers?.status(new MessageEvent('status', {
          data: JSON.stringify({
            run_id: 'run-1',
            project_id: 'proj-1',
            task_id: 'task-1',
            status: i % 2 === 0 ? 'running' : 'completed',
            exit_code: i % 2 === 0 ? -1 : 0,
          }),
        }))
      }
      vi.advanceTimersByTime(Math.max(0, STATUS_REFRESH_DELAY_MS - 40))
    })
    expect(invalidateSpy).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(80)
    })
    const keys = invalidateSpy.mock.calls
      .map((call) => call[0]?.queryKey)
      .filter(Boolean)
      .map((queryKey) => JSON.stringify(queryKey))
    expect(keys.filter((key) => key === JSON.stringify(['task', 'proj-1', 'task-1']))).toHaveLength(1)
    expect(keys.filter((key) => key === JSON.stringify(['run', 'proj-1', 'task-1', 'run-1']))).toHaveLength(1)
    expect(keys).not.toContain(JSON.stringify(['tasks', 'proj-1']))
    expect(keys).not.toContain(JSON.stringify(['runs-flat', 'proj-1']))
  })

  it('coalesces bursty log events into a single invalidation cycle', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(['runs-flat', 'proj-1'], [{ id: 'run-1' }])

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      for (let i = 0; i < 4; i += 1) {
        mockedSSE.handlers?.log(new MessageEvent('log', {
          data: JSON.stringify({ run_id: 'run-1', line: `line-${i}` }),
        }))
      }
      vi.advanceTimersByTime(Math.max(0, LOG_REFRESH_DELAY_MS - 50))
    })
    expect(invalidateSpy).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(100)
    })
    expect(invalidateSpy).toHaveBeenCalled()
    const firstFlushKeys = invalidateSpy.mock.calls
      .map((call) => call[0]?.queryKey)
      .filter(Boolean)
      .map((queryKey) => JSON.stringify(queryKey))
    expect(firstFlushKeys).toContain(JSON.stringify(['run-file', 'proj-1', 'task-1', 'run-1']))
    expect(firstFlushKeys).not.toContain(JSON.stringify(['runs-flat', 'proj-1']))

    const callsAfterFirstFlush = invalidateSpy.mock.calls.length
    act(() => {
      mockedSSE.handlers?.log(new MessageEvent('log', {
        data: JSON.stringify({ run_id: 'run-1', line: 'late-line' }),
      }))
      vi.advanceTimersByTime(LOG_REFRESH_DELAY_MS + 100)
    })
    expect(invalidateSpy.mock.calls.length).toBeGreaterThan(callsAfterFirstFlush)
  })

  it('refreshes project/task caches when a log arrives for an unknown run in the selected project', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(['runs-flat', 'proj-1'], [{ id: 'run-1', task_id: 'task-1' }])

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      mockedSSE.handlers?.log(new MessageEvent('log', {
        data: JSON.stringify({
          run_id: 'run-new',
          project_id: 'proj-1',
          task_id: 'task-2',
          line: 'new run log',
        }),
      }))
      vi.advanceTimersByTime(STATUS_REFRESH_DELAY_MS + 100)
    })

    const keys = invalidateSpy.mock.calls
      .map((call) => call[0]?.queryKey)
      .filter(Boolean)
      .map((queryKey) => JSON.stringify(queryKey))
    expect(keys).toContain(JSON.stringify(['tasks', 'proj-1']))
    expect(keys).toContain(JSON.stringify(['runs-flat', 'proj-1']))
    expect(keys).toContain(JSON.stringify(['task', 'proj-1', 'task-1']))
    expect(keys).not.toContain(JSON.stringify(['run-file', 'proj-1', 'task-1', 'run-1']))
  })

  it('queues unknown-run refresh even while runs-flat fetch is already in flight', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(['runs-flat', 'proj-1'], [{ id: 'run-1', task_id: 'task-1' }])

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      mockedSSE.handlers?.log(new MessageEvent('log', {
        data: JSON.stringify({
          run_id: 'run-new',
          project_id: 'proj-1',
          task_id: 'task-2',
          line: 'new run log',
        }),
      }))
      vi.advanceTimersByTime(STATUS_REFRESH_DELAY_MS + 100)
    })

    const keys = invalidateSpy.mock.calls
      .map((call) => call[0]?.queryKey)
      .filter(Boolean)
      .map((queryKey) => JSON.stringify(queryKey))
    expect(keys).toContain(JSON.stringify(['tasks', 'proj-1']))
    expect(keys).toContain(JSON.stringify(['runs-flat', 'proj-1']))
    expect(keys).toContain(JSON.stringify(['task', 'proj-1', 'task-1']))
  })

  it('ignores malformed log events missing run identifiers', () => {
    const queryClient = new QueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(['runs-flat', 'proj-1'], [{ id: 'run-1', task_id: 'task-1' }])

    render(
      <QueryClientProvider client={queryClient}>
        <HookHarness projectId="proj-1" taskId="task-1" runId="run-1" />
      </QueryClientProvider>
    )

    act(() => {
      mockedSSE.handlers?.log(new MessageEvent('log', {
        data: JSON.stringify({ project_id: 'proj-1', task_id: 'task-1', line: 'missing-run-id' }),
      }))
      vi.advanceTimersByTime(LOG_REFRESH_DELAY_MS + 100)
    })

    expect(invalidateSpy).not.toHaveBeenCalled()
  })
})
