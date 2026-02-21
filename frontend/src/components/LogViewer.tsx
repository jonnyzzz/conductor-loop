import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import clsx from 'clsx'
import AnsiToHtml from 'ansi-to-html'
import type { LogEvent } from '../types'
import { useSSE } from '../hooks/useSSE'

const ansi = new AnsiToHtml({
  fg: '#d7e1ef',
  bg: 'transparent',
  newline: false,
  escapeXML: true,
})

type StreamFilter = 'all' | 'stdout' | 'stderr'

interface LogLine extends LogEvent {
  id: string
  dedupeKey: string
}

const EMPTY_INITIAL_LINES: LogEvent[] = []

function formatLine(line: LogLine) {
  const prefix = `[${line.run_id}:${line.stream}] `
  return ansi.toHtml(prefix + line.line)
}

function formatAgo(ms: number): string {
  const s = Math.floor(ms / 1000)
  if (s < 60) return `${s}s ago`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  return `${Math.floor(m / 60)}h ago`
}

function normalizeInitialLines(initialLines: LogEvent[], maxLines: number): LogLine[] {
  const start = Math.max(0, initialLines.length - maxLines)
  const normalized: LogLine[] = []
  for (let idx = start; idx < initialLines.length; idx += 1) {
    const line = initialLines[idx]
    const dedupeKey = `initial|${idx}|${line.run_id}|${line.stream}|${line.line}`
    normalized.push({
      ...line,
      id: dedupeKey,
      dedupeKey,
    })
  }
  return normalized
}

function buildLogDedupeKey(line: LogEvent, cursorID?: string): string {
  const tsCandidate =
    (line as { timestamp?: string }).timestamp ??
    (line as { ts?: string }).ts ??
    ''
  const cursor = cursorID?.trim()
  if (cursor) {
    return `log|${line.run_id}|${line.stream}|${cursor}`
  }
  if (tsCandidate) {
    return `log|${line.run_id}|${line.stream}|${tsCandidate}|${line.line}`
  }
  return `log|${line.run_id}|${line.stream}|${line.line}|${Date.now()}`
}

export function LogViewer({
  streamUrl,
  initialLines = EMPTY_INITIAL_LINES,
  maxLines = 5000,
}: {
  streamUrl?: string
  initialLines?: LogEvent[]
  maxLines?: number
}) {
  const [lines, setLines] = useState<LogLine[]>(() => normalizeInitialLines(initialLines, maxLines))
  const [streamFilter, setStreamFilter] = useState<StreamFilter>('all')
  const [search, setSearch] = useState('')
  const [runFilter, setRunFilter] = useState('')
  const [autoScroll, setAutoScroll] = useState(true)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const [lastLogTime, setLastLogTime] = useState<number | null>(null)
  const [now, setNow] = useState(() => Date.now())
  const seenKeysRef = useRef<Set<string>>(new Set())

  const resetLogState = useCallback(() => {
    const normalized = normalizeInitialLines(initialLines, maxLines)
    setLines(normalized)
    seenKeysRef.current = new Set(normalized.map((line) => line.dedupeKey))
    setLastLogTime(normalized.length > 0 ? Date.now() : null)
    setSearch('')
    setRunFilter('')
    setStreamFilter('all')
    setAutoScroll(true)
  }, [initialLines, maxLines])

  useEffect(() => {
    resetLogState()
  }, [streamUrl, resetLogState])

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 5000)
    return () => clearInterval(timer)
  }, [])

  const heartbeatStatus = useMemo(() => {
    if (lastLogTime === null) return 'none'
    const age = now - lastLogTime
    if (age < 30_000) return 'recent'
    if (age < 120_000) return 'stale'
    return 'silent'
  }, [lastLogTime, now])

  const pushLine = useCallback(
    (line: LogEvent, dedupeKey: string) => {
      if (seenKeysRef.current.has(dedupeKey)) {
        return
      }
      seenKeysRef.current.add(dedupeKey)
      setLines((prev) => {
        const next = [...prev, { ...line, id: dedupeKey, dedupeKey }]
        if (next.length > maxLines) {
          const dropCount = next.length - maxLines
          const dropped = next.slice(0, dropCount)
          for (const item of dropped) {
            seenKeysRef.current.delete(item.dedupeKey)
          }
          return next.slice(dropCount)
        }
        return next
      })
      setLastLogTime(Date.now())
    },
    [maxLines]
  )

  const pushSystemLine = useCallback(
    (line: LogEvent, marker: string) => {
      pushLine(line, `meta|${marker}`)
    },
    [pushLine]
  )

  const sseHandlers = useMemo(
    () => ({
      log: (event: MessageEvent) => {
        try {
          const payload = JSON.parse(event.data) as LogEvent
          pushLine(payload, buildLogDedupeKey(payload, event.lastEventId))
        } catch {
          // Ignore malformed events.
        }
      },
      run_start: (event: MessageEvent) => {
        try {
          const payload = JSON.parse(event.data) as { run_id: string; agent: string; start_time: string }
          pushSystemLine(
            { run_id: payload.run_id, stream: 'stdout', line: `RUN START (${payload.agent}) ${payload.start_time}` },
            `run_start|${payload.run_id}|${payload.start_time}`
          )
        } catch {
          // Ignore malformed events.
        }
      },
      run_end: (event: MessageEvent) => {
        try {
          const payload = JSON.parse(event.data) as { run_id: string; exit_code: number; end_time: string }
          pushSystemLine(
            { run_id: payload.run_id, stream: 'stderr', line: `RUN END (exit ${payload.exit_code}) ${payload.end_time}` },
            `run_end|${payload.run_id}|${payload.end_time}|${payload.exit_code}`
          )
        } catch {
          // Ignore malformed events.
        }
      },
    }),
    [pushLine, pushSystemLine]
  )

  const stream = useSSE(streamUrl, {
    events: sseHandlers,
  })

  const filteredLines = useMemo(() => {
    const query = search.trim().toLowerCase()
    return lines.filter((line) => {
      if (streamFilter !== 'all' && line.stream !== streamFilter) {
        return false
      }
      if (runFilter && !line.run_id.includes(runFilter)) {
        return false
      }
      if (query && !line.line.toLowerCase().includes(query)) {
        return false
      }
      return true
    })
  }, [lines, streamFilter, search, runFilter])

  const handleScroll = () => {
    const container = containerRef.current
    if (!container) {
      return
    }
    const threshold = 48
    const atBottom = container.scrollHeight - container.scrollTop - container.clientHeight < threshold
    setAutoScroll(atBottom)
  }

  useEffect(() => {
    const container = containerRef.current
    if (!container || !autoScroll) {
      return
    }
    container.scrollTop = container.scrollHeight
  }, [filteredLines, autoScroll])

  const exportLogs = () => {
    const payload = filteredLines.map((line) => `[${line.run_id}:${line.stream}] ${line.line}`).join('\n')
    const blob = new Blob([payload], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `conductor-logs-${Date.now()}.txt`
    link.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="panel panel-logs">
      <div className="panel-header">
        <div>
          <div className="panel-title">Live logs</div>
          <div className="panel-subtitle">
            {!streamUrl && 'Select a project/task to stream logs'}
            {streamUrl && stream.state === 'connecting' && 'Connecting to stream…'}
            {streamUrl && stream.state === 'open' && 'Streaming via SSE'}
            {streamUrl && stream.state === 'reconnecting' && `Reconnecting… (${stream.errorCount})`}
            {streamUrl && stream.state === 'error' && `Stream error (${stream.errorCount})`}
          </div>
        </div>
        <div className="panel-actions">
          {heartbeatStatus !== 'none' && (
            <span className={clsx('heartbeat-badge', `heartbeat-${heartbeatStatus}`)}>
              Last output: {formatAgo(now - (lastLogTime ?? 0))}
            </span>
          )}
          <Button inline onClick={() => setLines([])}>Clear</Button>
          <Button inline onClick={exportLogs}>Export</Button>
        </div>
      </div>
      <div className="panel-section log-controls">
        <div className="filter-group">
          {(['all', 'stdout', 'stderr'] as StreamFilter[]).map((filter) => (
            <Button
              key={filter}
              inline
              className={clsx('filter-button', streamFilter === filter && 'filter-button-active')}
              onClick={() => setStreamFilter(filter)}
            >
              {filter}
            </Button>
          ))}
        </div>
        <input
          className="input"
          placeholder="Filter by run id"
          value={runFilter}
          onChange={(event) => setRunFilter(event.target.value)}
        />
        <input
          className="input"
          placeholder="Search logs"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />
        <div className={clsx('auto-scroll', autoScroll ? 'auto-scroll-on' : 'auto-scroll-off')}>
          Auto-scroll {autoScroll ? 'on' : 'off'}
        </div>
      </div>
      <div className="panel-section panel-section-tight log-stream" ref={containerRef} onScroll={handleScroll}>
        {filteredLines.length === 0 && (
          <div className="empty-state">
            {!streamUrl && 'Select a project and task to view logs.'}
            {streamUrl && lines.length === 0 && 'No log lines yet.'}
            {streamUrl && lines.length > 0 && 'No lines match current filters.'}
          </div>
        )}
        {filteredLines.map((line) => (
          <div
            key={line.id}
            className={clsx('log-line', line.stream === 'stderr' && 'log-line-error')}
            dangerouslySetInnerHTML={{ __html: formatLine(line) }}
          />
        ))}
      </div>
    </div>
  )
}
