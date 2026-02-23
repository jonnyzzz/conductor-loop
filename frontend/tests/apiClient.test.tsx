import { afterEach, describe, expect, it, vi } from 'vitest'
import { APIClient } from '../src/api/client'

function mockJSONResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? 'OK' : 'ERROR',
    json: async () => body,
    text: async () => JSON.stringify(body),
  } as Response
}

describe('APIClient.getTasks', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('fetches all paginated task pages to preserve full tree ancestry', async () => {
    const fetchSpy = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        mockJSONResponse({
          items: [
            {
              id: 'task-child',
              status: 'running',
              last_activity: '2026-02-22T20:01:00Z',
              depends_on: null,
              blocked_by: null,
            },
          ],
          total: 2,
          limit: 500,
          offset: 0,
          has_more: true,
        })
      )
      .mockResolvedValueOnce(
        mockJSONResponse({
          items: [
            {
              id: 'task-parent',
              status: 'completed',
              last_activity: '2026-02-22T19:00:00Z',
              depends_on: ['task-bootstrap'],
              blocked_by: [],
            },
          ],
          total: 2,
          limit: 500,
          offset: 1,
          has_more: false,
        })
      )

    const client = new APIClient('')
    const tasks = await client.getTasks('conductor-loop')

    expect(tasks.map((task) => task.id)).toEqual(['task-child', 'task-parent'])
    expect(fetchSpy).toHaveBeenCalledTimes(2)
    expect(fetchSpy.mock.calls[0][0]).toBe('/api/projects/conductor-loop/tasks?limit=500&offset=0')
    expect(fetchSpy.mock.calls[1][0]).toBe('/api/projects/conductor-loop/tasks?limit=500&offset=1')
  })

  it('stops pagination when a page is empty even if has_more is true', async () => {
    const fetchSpy = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        mockJSONResponse({
          items: [
            {
              id: 'task-only',
              status: 'running',
              last_activity: '2026-02-22T20:01:00Z',
            },
          ],
          total: 2,
          limit: 500,
          offset: 0,
          has_more: true,
        })
      )
      .mockResolvedValueOnce(
        mockJSONResponse({
          items: [],
          total: 2,
          limit: 500,
          offset: 1,
          has_more: true,
        })
      )

    const client = new APIClient('')
    const tasks = await client.getTasks('conductor-loop')

    expect(tasks.map((task) => task.id)).toEqual(['task-only'])
    expect(fetchSpy).toHaveBeenCalledTimes(2)
  })
})

describe('APIClient.getProjectRunsFlat', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('requests active-only runs, selected-task context, and selected-task limit', async () => {
    const fetchSpy = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        mockJSONResponse({
          runs: [],
        })
      )

    const client = new APIClient('')
    await client.getProjectRunsFlat('conductor-loop', {
      activeOnly: true,
      selectedTaskId: 'task-20260222-214200-ui-latency',
      selectedTaskLimit: 1,
    })

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    expect(fetchSpy.mock.calls[0][0]).toBe(
      '/api/projects/conductor-loop/runs/flat?active_only=1&selected_task_id=task-20260222-214200-ui-latency&selected_task_limit=1'
    )
  })

  it('omits query parameters when no filtering options are provided', async () => {
    const fetchSpy = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        mockJSONResponse({
          runs: [],
        })
      )

    const client = new APIClient('')
    await client.getProjectRunsFlat('conductor-loop')

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    expect(fetchSpy.mock.calls[0][0]).toBe('/api/projects/conductor-loop/runs/flat')
  })
})

describe('APIClient run normalization', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('reuses run objects when files are already normalized', async () => {
    const payload = {
      items: [
        {
          id: 'run-1',
          agent: 'codex',
          status: 'running',
          exit_code: -1,
          start_time: '2026-02-22T21:00:00Z',
          files: [
            { name: 'agent-stdout.txt', label: 'stdout' },
          ],
        },
      ],
      total: 1,
      limit: 100,
      offset: 0,
      has_more: false,
    }
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(mockJSONResponse(payload))

    const client = new APIClient('')
    const runs = await client.getRuns('conductor-loop', 'task-1')

    expect(runs).toHaveLength(1)
    expect(runs[0]).toBe(payload.items[0])
  })

  it('normalizes nullable files while keeping run fields intact', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      mockJSONResponse({
        items: [
          {
            id: 'run-1',
            agent: 'codex',
            status: 'completed',
            exit_code: 0,
            start_time: '2026-02-22T21:00:00Z',
            end_time: '2026-02-22T21:00:30Z',
            files: null,
          },
        ],
        total: 1,
        limit: 100,
        offset: 0,
        has_more: false,
      })
    )

    const client = new APIClient('')
    const runs = await client.getRuns('conductor-loop', 'task-1')

    expect(runs[0]).toMatchObject({
      id: 'run-1',
      status: 'completed',
      exit_code: 0,
    })
    expect(runs[0].files).toBeUndefined()
  })
})

describe('APIClient bus message listing', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('requests message deltas with since cursor when provided', async () => {
    const fetchSpy = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        mockJSONResponse({
          messages: [],
        })
      )

    const client = new APIClient('')
    await client.getProjectMessages('conductor-loop', { since: 'MSG-123' })

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    expect(fetchSpy.mock.calls[0][0]).toBe('/api/projects/conductor-loop/messages?limit=500&since=MSG-123')
  })

  it('falls back to bounded snapshot fetch when since cursor is not found', async () => {
    const fetchSpy = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        mockJSONResponse(
          {
            error: { message: 'message id not found' },
          },
          404
        )
      )
      .mockResolvedValueOnce(
        mockJSONResponse({
          messages: [],
        })
      )

    const client = new APIClient('')
    await client.getTaskMessages('conductor-loop', 'task-1', { since: 'MSG-stale' })

    expect(fetchSpy).toHaveBeenCalledTimes(2)
    expect(fetchSpy.mock.calls[0][0]).toBe('/api/projects/conductor-loop/tasks/task-1/messages?limit=500&since=MSG-stale')
    expect(fetchSpy.mock.calls[1][0]).toBe('/api/projects/conductor-loop/tasks/task-1/messages?limit=500')
  })
})
