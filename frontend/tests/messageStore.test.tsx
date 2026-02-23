import { describe, expect, it, vi } from 'vitest'
import type { BusMessage } from '../src/types'
import { mergeMessagesByID, upsertMessageBatch, upsertMessageByID } from '../src/utils/messageStore'

function msg(id: string, ts: string, body = id): BusMessage {
  return {
    msg_id: id,
    timestamp: ts,
    type: 'FACT',
    body,
  }
}

describe('messageStore', () => {
  it('merges by msg_id and keeps newest payload first', () => {
    const base = [
      msg('M-1', '2026-02-22T18:00:00Z', 'older'),
      msg('M-2', '2026-02-22T18:00:01Z', 'stable'),
    ]
    const merged = mergeMessagesByID(base, [
      msg('M-1', '2026-02-22T18:00:02Z', 'newer'),
      msg('M-3', '2026-02-22T18:00:03Z', 'fresh'),
    ])

    expect(merged.map((item) => item.msg_id)).toEqual(['M-3', 'M-1', 'M-2'])
    expect(merged.find((item) => item.msg_id === 'M-1')?.body).toBe('newer')
  })

  it('upserts a single streaming message without reordering unrelated entries', () => {
    const existing = [
      msg('M-3', '2026-02-22T18:00:03Z'),
      msg('M-2', '2026-02-22T18:00:02Z'),
      msg('M-1', '2026-02-22T18:00:01Z'),
    ]

    const withNewest = upsertMessageByID(existing, msg('M-4', '2026-02-22T18:00:04Z'))
    expect(withNewest.map((item) => item.msg_id)).toEqual(['M-4', 'M-3', 'M-2', 'M-1'])

    const updatedExisting = upsertMessageByID(withNewest, msg('M-2', '2026-02-22T18:00:05Z', 'updated'))
    expect(updatedExisting.map((item) => item.msg_id)).toEqual(['M-2', 'M-4', 'M-3', 'M-1'])
    expect(updatedExisting.find((item) => item.msg_id === 'M-2')?.body).toBe('updated')
  })

  it('ignores older duplicate updates and enforces max size', () => {
    const existing = [
      msg('M-3', '2026-02-22T18:00:03Z'),
      msg('M-2', '2026-02-22T18:00:02Z'),
      msg('M-1', '2026-02-22T18:00:01Z'),
    ]

    const ignored = upsertMessageByID(existing, msg('M-2', '2026-02-22T18:00:00Z', 'stale'))
    expect(ignored).toEqual(existing)

    const trimmed = upsertMessageByID(existing, msg('M-4', '2026-02-22T18:00:04Z'), 3)
    expect(trimmed.map((item) => item.msg_id)).toEqual(['M-4', 'M-3', 'M-2'])
  })

  it('applies streaming batches with id deduplication and recency sort', () => {
    const existing = [
      msg('M-3', '2026-02-22T18:00:03Z'),
      msg('M-2', '2026-02-22T18:00:02Z'),
      msg('M-1', '2026-02-22T18:00:01Z'),
    ]

    const next = upsertMessageBatch(existing, [
      msg('M-4', '2026-02-22T18:00:04Z'),
      msg('M-2', '2026-02-22T18:00:05Z', 'fresh replacement'),
      msg('M-5', '2026-02-22T18:00:00Z'),
    ], 4)

    expect(next.map((item) => item.msg_id)).toEqual(['M-2', 'M-4', 'M-3', 'M-1'])
    expect(next.find((item) => item.msg_id === 'M-2')?.body).toBe('fresh replacement')
  })

  it('uses linear fast path for recency-ordered incoming batches', () => {
    const existing = [
      msg('M-6', '2026-02-22T18:00:06Z', 'base-6'),
      msg('M-4', '2026-02-22T18:00:04Z', 'base-4'),
      msg('M-2', '2026-02-22T18:00:02Z', 'base-2'),
    ]

    const sortSpy = vi.spyOn(Array.prototype, 'sort')
    try {
      const merged = mergeMessagesByID(existing, [
        msg('M-7', '2026-02-22T18:00:07Z', 'fresh-7'),
        msg('M-4', '2026-02-22T18:00:04Z', 'incoming-4-first'),
        msg('M-4', '2026-02-22T18:00:04Z', 'incoming-4-final'),
        msg('M-3', '2026-02-22T18:00:03Z', 'fresh-3'),
        msg('M-1', '2026-02-22T18:00:01Z', 'fresh-1'),
      ])

      expect(sortSpy).not.toHaveBeenCalled()
      expect(merged.map((item) => item.msg_id)).toEqual(['M-7', 'M-6', 'M-4', 'M-3', 'M-2', 'M-1'])
      expect(merged.find((item) => item.msg_id === 'M-4')?.body).toBe('incoming-4-final')
    } finally {
      sortSpy.mockRestore()
    }
  })

  it('falls back to sort merge when incoming order is not recency-sorted', () => {
    const existing = [
      msg('M-3', '2026-02-22T18:00:03Z'),
      msg('M-2', '2026-02-22T18:00:02Z'),
      msg('M-1', '2026-02-22T18:00:01Z'),
    ]

    const sortSpy = vi.spyOn(Array.prototype, 'sort')
    try {
      const merged = mergeMessagesByID(existing, [
        msg('M-0', '2026-02-22T18:00:00Z'),
        msg('M-4', '2026-02-22T18:00:04Z'),
      ])

      expect(sortSpy).toHaveBeenCalled()
      expect(merged.map((item) => item.msg_id)).toEqual(['M-4', 'M-3', 'M-2', 'M-1', 'M-0'])
    } finally {
      sortSpy.mockRestore()
    }
  })

  it('keeps reference stable for duplicate no-op batches', () => {
    const existing = [
      msg('M-3', '2026-02-22T18:00:03Z'),
      msg('M-2', '2026-02-22T18:00:02Z'),
      msg('M-1', '2026-02-22T18:00:01Z'),
    ]

    const noOp = upsertMessageBatch(existing, [
      msg('M-3', '2026-02-22T18:00:03Z'),
      msg('M-2', '2026-02-22T18:00:02Z'),
    ], 10)

    expect(noOp).toBe(existing)
  })

  it('replaces equal-timestamp entries when payload fields differ', () => {
    const existing = [
      msg('M-1', '2026-02-22T18:00:01Z'),
    ]
    const enriched = upsertMessageByID(existing, {
      ...msg('M-1', '2026-02-22T18:00:01Z'),
      task_id: 'task-1',
      project_id: 'proj-1',
    })

    expect(enriched).not.toBe(existing)
    expect(enriched[0]?.task_id).toBe('task-1')
    expect(enriched[0]?.project_id).toBe('proj-1')
  })
})
