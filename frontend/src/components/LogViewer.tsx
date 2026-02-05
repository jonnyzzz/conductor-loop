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
}

function formatLine(line: LogLine) {
  const prefix = `[${line.run_id}:${line.stream}] `
  return ansi.toHtml(prefix + line.line)
}

export function LogViewer({
  streamUrl,
  initialLines = [],
  maxLines = 5000,
}: {
  streamUrl?: string
  initialLines?: LogLine[]
  maxLines?: number
}) {
  const [lines, setLines] = useState<LogLine[]>(initialLines)
  const [streamFilter, setStreamFilter] = useState<StreamFilter>('all')
  const [search, setSearch] = useState('')
  const [runFilter, setRunFilter] = useState('')
  const [autoScroll, setAutoScroll] = useState(true)
  const containerRef = useRef<HTMLDivElement | null>(null)

  const pushLine = useCallback(
    (line: LogEvent) => {
      setLines((prev) => {
        const next = [...prev, { ...line, id: `${Date.now()}-${Math.random()}` }]
        if (next.length > maxLines) {
          return next.slice(next.length - maxLines)
        }
        return next
      })
    },
    [maxLines]
  )

  const sseHandlers = useMemo(
    () => ({
      log: (event: MessageEvent) => {
        try {
          const payload = JSON.parse(event.data) as LogEvent
          pushLine(payload)
        } catch {
          // Ignore malformed events.
        }
      },
      run_start: (event: MessageEvent) => {
        try {
          const payload = JSON.parse(event.data) as { run_id: string; agent: string; start_time: string }
          pushLine({ run_id: payload.run_id, stream: 'stdout', line: `RUN START (${payload.agent}) ${payload.start_time}` })
        } catch {
          // Ignore malformed events.
        }
      },
      run_end: (event: MessageEvent) => {
        try {
          const payload = JSON.parse(event.data) as { run_id: string; exit_code: number; end_time: string }
          pushLine({ run_id: payload.run_id, stream: 'stderr', line: `RUN END (exit ${payload.exit_code}) ${payload.end_time}` })
        } catch {
          // Ignore malformed events.
        }
      },
    }),
    [pushLine]
  )

  useSSE(streamUrl, {
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
          <div className="panel-subtitle">Streaming via SSE</div>
        </div>
        <div className="panel-actions">
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
        {filteredLines.length === 0 && <div className="empty-state">No log lines yet.</div>}
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
