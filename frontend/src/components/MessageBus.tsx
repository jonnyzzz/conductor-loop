import { useMemo, useState } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import clsx from 'clsx'
import type { BusMessage } from '../types'
import { useSSE } from '../hooks/useSSE'

export function MessageBus({
  streamUrl,
  title = 'Message bus',
}: {
  streamUrl?: string
  title?: string
}) {
  const [messages, setMessages] = useState<BusMessage[]>([])
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [filter, setFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')

  const sseHandlers = useMemo(
    () => ({
      message: (event: MessageEvent) => {
        try {
          const payload = JSON.parse(event.data) as BusMessage
          setMessages((prev) => {
            if (prev.some((msg) => msg.msg_id === payload.msg_id)) {
              return prev
            }
            return [payload, ...prev]
          })
        } catch {
          // Ignore malformed events.
        }
      },
    }),
    []
  )

  useSSE(streamUrl, {
    events: sseHandlers,
  })

  const filteredMessages = useMemo(() => {
    const query = filter.trim().toLowerCase()
    const typeQuery = typeFilter.trim().toLowerCase()
    return messages.filter((msg) => {
      if (typeQuery && msg.type.toLowerCase() !== typeQuery) {
        return false
      }
      if (query && !msg.message.toLowerCase().includes(query) && !msg.msg_id.toLowerCase().includes(query)) {
        return false
      }
      return true
    })
  }, [messages, filter, typeFilter])

  const toggleExpanded = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  return (
    <div className="panel panel-scroll">
      <div className="panel-header">
        <div>
          <div className="panel-title">{title}</div>
          <div className="panel-subtitle">Auto-refresh via SSE</div>
        </div>
        <Button inline onClick={() => setMessages([])}>Clear</Button>
      </div>
      <div className="panel-section message-controls">
        <input
          className="input"
          placeholder="Filter by text or id"
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
        />
        <input
          className="input"
          placeholder="Type (FACT, ERROR…)"
          value={typeFilter}
          onChange={(event) => setTypeFilter(event.target.value)}
        />
      </div>
      <div className="panel-section panel-section-tight">
        {filteredMessages.length === 0 && <div className="empty-state">No messages yet.</div>}
        {filteredMessages.map((msg) => {
          const isOpen = expanded.has(msg.msg_id)
          return (
            <div key={msg.msg_id} className={clsx('bus-message', isOpen && 'bus-message-open')}>
              <button type="button" className="bus-message-header" onClick={() => toggleExpanded(msg.msg_id)}>
                <span className={clsx('status-dot', `status-${msg.type.toLowerCase()}`)} />
                <span className="bus-message-title">{msg.type}</span>
                <span className="bus-message-meta">{new Date(msg.ts).toLocaleTimeString()}</span>
                <span className="bus-message-id">{msg.msg_id}</span>
              </button>
              {isOpen && (
                <div className="bus-message-body">
                  <div className="bus-message-text">{msg.message}</div>
                  {msg.parents && msg.parents.length > 0 && (
                    <div className="bus-message-parents">
                      Parents: {msg.parents.map((parent) => (typeof parent === 'string' ? parent : parent.msg_id)).join(', ')}
                    </div>
                  )}
                  {msg.attachment_path && (
                    <div className="bus-message-attachment">Attachment: {msg.attachment_path}</div>
                  )}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
