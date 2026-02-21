import { useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import clsx from 'clsx'
import type { BusMessage } from '../types'
import { useSSE } from '../hooks/useSSE'
import {
  usePostProjectMessage,
  usePostTaskMessage,
  useProjectMessages,
  useTaskMessages,
} from '../hooks/useAPI'

const MESSAGE_TYPES = ['USER', 'QUESTION', 'ANSWER', 'INFO', 'FACT', 'PROGRESS', 'DECISION', 'ERROR']

function parseTimestamp(timestamp: string): number {
  const parsed = Date.parse(timestamp)
  return Number.isNaN(parsed) ? 0 : parsed
}

function mergeMessagesByID(existing: BusMessage[], incoming: BusMessage[]): BusMessage[] {
  const byID = new Map<string, BusMessage>()
  const merged = [...existing, ...incoming]
  for (const message of merged) {
    const current = byID.get(message.msg_id)
    if (!current) {
      byID.set(message.msg_id, message)
      continue
    }
    if (parseTimestamp(message.timestamp) >= parseTimestamp(current.timestamp)) {
      byID.set(message.msg_id, message)
    }
  }
  return [...byID.values()].sort((a, b) => {
    const byTime = parseTimestamp(b.timestamp) - parseTimestamp(a.timestamp)
    if (byTime !== 0) {
      return byTime
    }
    return b.msg_id.localeCompare(a.msg_id)
  })
}

function renderMessageBody(text: string) {
  const trimmed = text.trim()
  if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
    try {
      const parsed: unknown = JSON.parse(trimmed)
      return <pre className="bus-message-json">{JSON.stringify(parsed, null, 2)}</pre>
    } catch {
      // not valid JSON, fall through
    }
  }
  return <span>{text}</span>
}

export function MessageBus({
  streamUrl,
  title = 'Message bus',
  projectId,
  taskId,
  scope = 'task',
  headerActions,
}: {
  streamUrl?: string
  title?: string
  projectId?: string
  taskId?: string
  scope?: 'project' | 'task'
  headerActions?: ReactNode
}) {
  const [messages, setMessages] = useState<BusMessage[]>([])
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [filter, setFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')

  // Compose form state
  const [composeType, setComposeType] = useState('USER')
  const [composeText, setComposeText] = useState('')
  const [postStatus, setPostStatus] = useState<'idle' | 'success' | 'error'>('idle')
  const [postError, setPostError] = useState('')

  const projectMessagesQuery = useProjectMessages(scope === 'project' ? projectId : undefined)
  const taskMessagesQuery = useTaskMessages(
    scope === 'task' ? projectId : undefined,
    scope === 'task' ? taskId : undefined
  )
  const postTaskMessage = usePostTaskMessage(projectId, taskId)
  const postProjectMessage = usePostProjectMessage(projectId)

  useEffect(() => {
    setMessages([])
    setExpanded(new Set())
    setFilter('')
    setTypeFilter('')
    setComposeText('')
    setPostStatus('idle')
    setPostError('')
  }, [scope, projectId, taskId, streamUrl])

  const hydratedMessages = scope === 'task' ? taskMessagesQuery.data : projectMessagesQuery.data

  useEffect(() => {
    if (!hydratedMessages) {
      return
    }
    setMessages((prev) => mergeMessagesByID(prev, hydratedMessages))
  }, [hydratedMessages])

  const sseHandlers = useMemo(
    () => ({
      message: (event: MessageEvent) => {
        try {
          const payload = JSON.parse(event.data) as BusMessage
          setMessages((prev) => mergeMessagesByID(prev, [payload]))
        } catch {
          // Ignore malformed events.
        }
      },
    }),
    []
  )

  const stream = useSSE(streamUrl, {
    events: sseHandlers,
  })

  const activeQuery = scope === 'task' ? taskMessagesQuery : projectMessagesQuery
  const loadError = activeQuery.error instanceof Error ? activeQuery.error.message : ''
  const isHydrating = activeQuery.isFetching && messages.length === 0

  const filteredMessages = useMemo(() => {
    const query = filter.trim().toLowerCase()
    const typeQuery = typeFilter.trim().toLowerCase()
    return messages.filter((msg) => {
      if (typeQuery && msg.type.toLowerCase() !== typeQuery) {
        return false
      }
      if (query && !msg.body.toLowerCase().includes(query) && !msg.msg_id.toLowerCase().includes(query)) {
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

  const handlePost = async () => {
    const text = composeText.trim()
    const hasContext = Boolean(projectId && (scope === 'project' || taskId))
    if (!text || !hasContext || !projectId) return
    setPostStatus('idle')
    setPostError('')
    try {
      if (scope === 'task' && taskId) {
        await postTaskMessage.mutateAsync({ type: composeType, body: text })
      } else {
        await postProjectMessage.mutateAsync({ type: composeType, body: text })
      }
      setComposeText('')
      setPostStatus('success')
      setTimeout(() => setPostStatus('idle'), 2000)
    } catch (err) {
      setPostStatus('error')
      setPostError(err instanceof Error ? err.message : 'Failed to post message')
    }
  }

  const canCompose = Boolean(projectId && (scope === 'project' || taskId))
  const canSubmit = Boolean(canCompose && composeText.trim())
  const composePlaceholder = !projectId
    ? 'Select a project to post'
    : (scope === 'task' && !taskId)
      ? 'Select a task to post'
      : 'Message body…'

  return (
    <div className="panel panel-scroll">
      <div className="panel-header">
        <div>
          <div className="panel-title">{title}</div>
          <div className="panel-subtitle bus-stream-state">
            {!streamUrl && 'Stream unavailable for current scope'}
            {streamUrl && stream.state === 'connecting' && 'Connecting to stream…'}
            {streamUrl && stream.state === 'open' && 'Live updates via SSE'}
            {streamUrl && stream.state === 'reconnecting' && `Reconnecting… (${stream.errorCount})`}
            {streamUrl && stream.state === 'error' && `Stream error (${stream.errorCount})`}
          </div>
        </div>
        <div className="panel-actions bus-panel-actions">
          {headerActions}
          <Button inline onClick={() => setMessages([])}>Clear</Button>
        </div>
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
      <div className="panel-section panel-section-tight bus-list">
        {filteredMessages.length === 0 && (
          <div className="empty-state">
            {!canCompose && scope === 'task' && 'Select a project and task to view task messages.'}
            {!canCompose && scope === 'project' && 'Select a project to view project messages.'}
            {canCompose && isHydrating && 'Loading existing messages…'}
            {canCompose && !isHydrating && loadError && `Failed to load messages: ${loadError}`}
            {canCompose && !isHydrating && !loadError && 'No messages yet.'}
          </div>
        )}
        {filteredMessages.map((msg) => {
          const isOpen = expanded.has(msg.msg_id)
          return (
            <div key={msg.msg_id} className={clsx('bus-message', isOpen && 'bus-message-open')}>
              <button type="button" className="bus-message-header" onClick={() => toggleExpanded(msg.msg_id)}>
                <span className={clsx('status-dot', `status-${msg.type.toLowerCase()}`)} />
                <span className="bus-message-title">{msg.type}</span>
                <span className="bus-message-meta">{new Date(msg.timestamp).toLocaleTimeString()}</span>
                <span className="bus-message-id">{msg.msg_id}</span>
              </button>
              {isOpen && (
                <div className="bus-message-body">
                  <div className="bus-message-text">{renderMessageBody(msg.body)}</div>
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
      <div className="panel-divider" />
      <div className="panel-section bus-compose">
        <div className="bus-compose-header">Post message</div>
        <div className="bus-compose-row">
          <select
            className="input bus-compose-type"
            value={composeType}
            onChange={(e) => setComposeType(e.target.value)}
            aria-label="Message type"
          >
            {MESSAGE_TYPES.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
          <Button
            inline
            onClick={handlePost}
            disabled={!canSubmit || postTaskMessage.isPending || postProjectMessage.isPending}
            aria-label="Post message"
          >
            {postTaskMessage.isPending || postProjectMessage.isPending ? 'Posting…' : 'Post'}
          </Button>
        </div>
        <textarea
          className="input bus-compose-textarea"
          placeholder={composePlaceholder}
          value={composeText}
          onChange={(e) => setComposeText(e.target.value)}
          rows={2}
          disabled={!canCompose}
          aria-label="Message body"
        />
        {postStatus === 'success' && (
          <div className="bus-compose-feedback bus-compose-success">Posted successfully</div>
        )}
        {postStatus === 'error' && (
          <div className="bus-compose-feedback bus-compose-error">{postError}</div>
        )}
      </div>
    </div>
  )
}
