import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent, ReactNode } from 'react'
import Button from '@jetbrains/ring-ui-built/components/button/button'
import Dialog from '@jetbrains/ring-ui-built/components/dialog/dialog'
import clsx from 'clsx'
import type { TaskStartRequest } from '../api/client'
import type { BusMessage } from '../types'
import { useSSE } from '../hooks/useSSE'
import { DEFAULT_MAX_MESSAGES, mergeMessagesByID, parseMessageTimestamp, upsertMessageBatch } from '../utils/messageStore'
import {
  usePostProjectMessage,
  usePostTaskMessage,
  useProjectMessages,
  useStartTask,
  useTaskMessages,
} from '../hooks/useAPI'

const MESSAGE_TYPES = ['PROGRESS', 'FACT', 'DECISION', 'ERROR', 'QUESTION']
const THREAD_PARENT_SOURCE_TYPES = new Set(['QUESTION', 'FACT'])
const BODY_PREVIEW_CHAR_LIMIT = 1800
const BODY_PREVIEW_LINE_LIMIT = 18
const JSON_PRETTY_PRINT_CHAR_LIMIT = 10000
const SSE_BATCH_WINDOW_MS = 120
const UNFILTERED_RENDER_LIMIT = 120

const THREAD_META_PARENT_PROJECT_ID = 'thread_parent_project_id'
const THREAD_META_PARENT_TASK_ID = 'thread_parent_task_id'
const THREAD_META_PARENT_RUN_ID = 'thread_parent_run_id'
const THREAD_META_PARENT_MESSAGE_ID = 'thread_parent_message_id'
const THREAD_META_CHILD_PROJECT_ID = 'thread_child_project_id'
const THREAD_META_CHILD_TASK_ID = 'thread_child_task_id'
const THREAD_META_CHILD_MESSAGE_ID = 'thread_child_message_id'

type BusMessageWithAliases = BusMessage & {
  project_id?: string
  task_id?: string
  issue_id?: string
  meta?: Record<string, string>
}

type ThreadParentRef = {
  projectID: string
  taskID: string
  runID: string
  messageID: string
}

type ThreadChildRef = {
  projectID: string
  taskID: string
  messageID?: string
}

type ThreadDraft = {
  parent: ThreadParentRef
  taskID: string
  agentType: string
  prompt: string
  projectRoot: string
}

function parseTimestamp(timestamp: string): number {
  return parseMessageTimestamp(timestamp)
}

function formatMessageTimestamp(timestamp: string): string {
  const parsed = parseTimestamp(timestamp)
  if (!parsed) {
    return timestamp
  }
  return new Date(parsed).toLocaleString()
}

function resolveProjectID(msg: BusMessageWithAliases): string {
  return msg.project_id ?? msg.project ?? ''
}

function resolveTaskID(msg: BusMessageWithAliases): string {
  return msg.task_id ?? msg.task ?? ''
}

function resolveRunID(msg: BusMessageWithAliases): string {
  return msg.run_id ?? ''
}

function resolveMessageSource(msg: BusMessageWithAliases): string {
  const meta = msg.meta ?? {}
  const explicitSource = meta.source ?? meta.author ?? meta.agent ?? meta.sender ?? meta.origin
  if (explicitSource) {
    return explicitSource
  }
  if (resolveRunID(msg)) {
    return 'run'
  }
  if (resolveTaskID(msg)) {
    return 'task'
  }
  if (resolveProjectID(msg)) {
    return 'project'
  }
  return 'unknown'
}

function isLongBody(text: string): boolean {
  if (text.length > BODY_PREVIEW_CHAR_LIMIT) {
    return true
  }
  let lines = 1
  for (const ch of text) {
    if (ch === '\n') {
      lines += 1
      if (lines > BODY_PREVIEW_LINE_LIMIT) {
        return true
      }
    }
  }
  return false
}

function truncateBody(text: string): string {
  const limitedText = text.slice(0, BODY_PREVIEW_CHAR_LIMIT)
  const lines = limitedText.split('\n')
  if (lines.length > BODY_PREVIEW_LINE_LIMIT) {
    return lines.slice(0, BODY_PREVIEW_LINE_LIMIT).join('\n').trimEnd()
  }
  return limitedText.trimEnd()
}

function renderMessageBody(text: string) {
  const trimmed = text.trim()
  const maybeJSON = trimmed.startsWith('{') || trimmed.startsWith('[')
  if (maybeJSON && trimmed.length <= JSON_PRETTY_PRINT_CHAR_LIMIT) {
    try {
      const parsed: unknown = JSON.parse(trimmed)
      return <pre className="bus-message-json">{JSON.stringify(parsed, null, 2)}</pre>
    } catch {
      // not valid JSON, fall through
    }
  }
  return <pre className="bus-message-plain">{text}</pre>
}

function parseParentIDs(msg: BusMessage): string[] {
  if (!msg.parents || msg.parents.length === 0) {
    return []
  }
  return msg.parents
    .map((parent) => (typeof parent === 'string' ? parent : parent.msg_id))
    .filter((value): value is string => Boolean(value))
}

function extractThreadParentFromMeta(msg: BusMessageWithAliases): ThreadParentRef | null {
  const meta = msg.meta ?? {}
  const projectID = (meta[THREAD_META_PARENT_PROJECT_ID] ?? '').trim()
  const taskID = (meta[THREAD_META_PARENT_TASK_ID] ?? '').trim()
  const runID = (meta[THREAD_META_PARENT_RUN_ID] ?? '').trim()
  const messageID = (meta[THREAD_META_PARENT_MESSAGE_ID] ?? '').trim()
  if (!projectID || !taskID || !runID || !messageID) {
    return null
  }
  return { projectID, taskID, runID, messageID }
}

function extractThreadChildFromMeta(msg: BusMessageWithAliases): ThreadChildRef | null {
  const meta = msg.meta ?? {}
  const projectID = (meta[THREAD_META_CHILD_PROJECT_ID] ?? '').trim()
  const taskID = (meta[THREAD_META_CHILD_TASK_ID] ?? '').trim()
  const messageID = (meta[THREAD_META_CHILD_MESSAGE_ID] ?? '').trim()
  if (!projectID || !taskID) {
    return null
  }
  return { projectID, taskID, messageID: messageID || undefined }
}

function canStartThreadedAnswer(scope: 'project' | 'task', msg: BusMessageWithAliases): boolean {
  if (scope !== 'task') {
    return false
  }
  if (!THREAD_PARENT_SOURCE_TYPES.has(msg.type)) {
    return false
  }
  return Boolean(resolveProjectID(msg) && resolveTaskID(msg) && resolveRunID(msg))
}

function generateThreadTaskID(): string {
  const now = new Date()
  const date = now.toISOString().slice(0, 10).replace(/-/g, '')
  const time = now.toTimeString().slice(0, 8).replace(/:/g, '')
  const rand = Math.random().toString(36).slice(2, 7)
  return `task-${date}-${time}-${rand}`
}

function buildDefaultThreadPrompt(message: BusMessageWithAliases): string {
  const sourceTask = resolveTaskID(message)
  const header = `Answer ${message.type} ${message.msg_id} from ${sourceTask}.`
  return `${header}\n\nSource message:\n${message.body}\n`
}

export function MessageBus({
  streamUrl,
  title = 'Message bus',
  projectId,
  taskId,
  scope = 'task',
  headerActions,
  focusedMessageId,
  onNavigateToTask,
  onNavigateToMessage,
}: {
  streamUrl?: string
  title?: string
  projectId?: string
  taskId?: string
  scope?: 'project' | 'task'
  headerActions?: ReactNode
  focusedMessageId?: string
  onNavigateToTask?: (projectID: string, taskID: string) => void
  onNavigateToMessage?: (projectID: string, taskID: string, messageID: string) => void
}) {
  const [messages, setMessages] = useState<BusMessage[]>([])
  const [expandedBodies, setExpandedBodies] = useState<Set<string>>(new Set())
  const [filter, setFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const pendingSSEMessagesRef = useRef<BusMessage[]>([])
  const pendingSSEFlushTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Compose form state
  const [composeType, setComposeType] = useState(MESSAGE_TYPES[0])
  const [composeText, setComposeText] = useState('')
  const [postStatus, setPostStatus] = useState<'idle' | 'success' | 'error'>('idle')
  const [postError, setPostError] = useState('')

  // Threaded answer task state
  const [threadDraft, setThreadDraft] = useState<ThreadDraft | null>(null)
  const [threadError, setThreadError] = useState('')
  const [threadDialogOpen, setThreadDialogOpen] = useState(false)

  const projectMessagesQuery = useProjectMessages(scope === 'project' ? projectId : undefined)
  const taskMessagesQuery = useTaskMessages(
    scope === 'task' ? projectId : undefined,
    scope === 'task' ? taskId : undefined
  )
  const postTaskMessage = usePostTaskMessage(projectId, taskId)
  const postProjectMessage = usePostProjectMessage(projectId)
  const startTaskMutation = useStartTask(projectId)

  const clearPendingSSEMessages = useCallback(() => {
    pendingSSEMessagesRef.current = []
    if (pendingSSEFlushTimerRef.current !== null) {
      clearTimeout(pendingSSEFlushTimerRef.current)
      pendingSSEFlushTimerRef.current = null
    }
  }, [])

  const flushPendingSSEMessages = useCallback(() => {
    const pending = pendingSSEMessagesRef.current
    if (pending.length === 0) {
      pendingSSEFlushTimerRef.current = null
      return
    }
    pendingSSEMessagesRef.current = []
    pendingSSEFlushTimerRef.current = null
    setMessages((prev) => upsertMessageBatch(prev, pending, DEFAULT_MAX_MESSAGES))
  }, [])

  const queueSSEMessage = useCallback((message: BusMessage) => {
    pendingSSEMessagesRef.current.push(message)
    if (pendingSSEFlushTimerRef.current !== null) {
      return
    }
    pendingSSEFlushTimerRef.current = setTimeout(flushPendingSSEMessages, SSE_BATCH_WINDOW_MS)
  }, [flushPendingSSEMessages])

  useEffect(() => {
    clearPendingSSEMessages()
    setMessages([])
    setExpandedBodies(new Set())
    setFilter('')
    setTypeFilter('')
    setComposeText('')
    setPostStatus('idle')
    setPostError('')
    setThreadDraft(null)
    setThreadError('')
    setThreadDialogOpen(false)
  }, [clearPendingSSEMessages, scope, projectId, taskId, streamUrl])

  useEffect(() => {
    return () => {
      clearPendingSSEMessages()
    }
  }, [clearPendingSSEMessages])

  useEffect(() => {
    if (!focusedMessageId) {
      return
    }
    setFilter(focusedMessageId)
  }, [focusedMessageId])

  const hydratedMessages = scope === 'task' ? taskMessagesQuery.data : projectMessagesQuery.data

  useEffect(() => {
    if (!hydratedMessages) {
      return
    }
    clearPendingSSEMessages()
    setMessages((prev) => mergeMessagesByID(prev, hydratedMessages, DEFAULT_MAX_MESSAGES))
  }, [clearPendingSSEMessages, hydratedMessages])

  const sseHandlers = useMemo(
    () => ({
      message: (event: MessageEvent) => {
        try {
          const payload = JSON.parse(event.data) as BusMessage
          queueSSEMessage(payload)
        } catch {
          // Ignore malformed events.
        }
      },
    }),
    [queueSSEMessage]
  )

  const stream = useSSE(streamUrl, {
    events: sseHandlers,
  })

  const activeQuery = scope === 'task' ? taskMessagesQuery : projectMessagesQuery
  const loadError = activeQuery.error instanceof Error ? activeQuery.error.message : ''
  const isHydrating = activeQuery.isFetching && messages.length === 0

  const normalizedFilter = filter.trim().toLowerCase()
  const normalizedTypeFilter = typeFilter.trim().toLowerCase()
  const isUnfilteredView = normalizedFilter === '' && normalizedTypeFilter === ''

  const filteredMessages = useMemo(() => {
    if (isUnfilteredView) {
      return messages
    }

    return messages.filter((msg) => {
      if (normalizedTypeFilter && msg.type.toLowerCase() !== normalizedTypeFilter) {
        return false
      }
      if (!normalizedFilter) {
        return true
      }
      const enrichedMsg = msg as BusMessageWithAliases
      const haystack = [
        msg.body,
        msg.msg_id,
        msg.type,
        resolveMessageSource(enrichedMsg),
        resolveProjectID(enrichedMsg),
        resolveTaskID(enrichedMsg),
        resolveRunID(enrichedMsg),
      ]
        .join('\n')
        .toLowerCase()
      return haystack.includes(normalizedFilter)
    })
  }, [isUnfilteredView, messages, normalizedFilter, normalizedTypeFilter])

  const visibleMessages = useMemo(() => {
    if (!isUnfilteredView || filteredMessages.length <= UNFILTERED_RENDER_LIMIT) {
      return filteredMessages
    }
    return filteredMessages.slice(0, UNFILTERED_RENDER_LIMIT)
  }, [filteredMessages, isUnfilteredView])
  const hiddenMessagesCount = filteredMessages.length - visibleMessages.length

  const threadedChildrenByParentMessageID = useMemo(() => {
    const threadScanMessages = isUnfilteredView ? visibleMessages : filteredMessages
    const byParent = new Map<string, ThreadChildRef[]>()
    for (const msg of threadScanMessages) {
      const enriched = msg as BusMessageWithAliases
      const child = extractThreadChildFromMeta(enriched)
      if (!child) {
        continue
      }
      const parentIDs = parseParentIDs(msg)
      for (const parentID of parentIDs) {
        const current = byParent.get(parentID) ?? []
        const exists = current.some((entry) => (
          entry.projectID === child.projectID &&
          entry.taskID === child.taskID &&
          entry.messageID === child.messageID
        ))
        if (!exists) {
          current.push(child)
          byParent.set(parentID, current)
        }
      }
    }
    return byParent
  }, [filteredMessages, isUnfilteredView, visibleMessages])

  const toggleBodyExpansion = (id: string) => {
    setExpandedBodies((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const bodyDisplayByID = useMemo(() => {
    const byID = new Map<string, { text: string; long: boolean; truncated: boolean; rendered: ReactNode }>()
    for (const msg of visibleMessages) {
      const body = msg.body ?? ''
      const long = isLongBody(body)
      if (!long) {
        byID.set(msg.msg_id, {
          text: body,
          long: false,
          truncated: false,
          rendered: renderMessageBody(body),
        })
        continue
      }
      const expanded = expandedBodies.has(msg.msg_id)
      const text = expanded ? body : truncateBody(body)
      byID.set(msg.msg_id, {
        text,
        long: true,
        truncated: !expanded,
        rendered: renderMessageBody(text),
      })
    }
    return byID
  }, [expandedBodies, visibleMessages])

  const handlePost = async () => {
    const text = composeText.trim()
    const hasContext = Boolean(projectId && (scope === 'project' || taskId))
    if (!text || !hasContext || !projectId) {
      return
    }
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

  const openThreadDialog = (msg: BusMessageWithAliases) => {
    const parentProjectID = resolveProjectID(msg)
    const parentTaskID = resolveTaskID(msg)
    const parentRunID = resolveRunID(msg)
    const parentMessageID = msg.msg_id
    if (!parentProjectID || !parentTaskID || !parentRunID || !parentMessageID) {
      return
    }
    setThreadDraft({
      parent: {
        projectID: parentProjectID,
        taskID: parentTaskID,
        runID: parentRunID,
        messageID: parentMessageID,
      },
      taskID: generateThreadTaskID(),
      agentType: 'claude',
      prompt: buildDefaultThreadPrompt(msg),
      projectRoot: '',
    })
    setThreadError('')
    setThreadDialogOpen(true)
  }

  const closeThreadDialog = () => {
    setThreadDialogOpen(false)
    setThreadDraft(null)
    setThreadError('')
  }

  const handleThreadSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setThreadError('')
    if (!projectId || !threadDraft) {
      setThreadError('Select a project before creating a threaded task')
      return
    }
    const payload: TaskStartRequest = {
      task_id: threadDraft.taskID.trim(),
      prompt: threadDraft.prompt,
      project_root: threadDraft.projectRoot,
      attach_mode: 'create',
      agent_type: threadDraft.agentType.trim() || 'claude',
      thread_parent: {
        project_id: threadDraft.parent.projectID,
        task_id: threadDraft.parent.taskID,
        run_id: threadDraft.parent.runID,
        message_id: threadDraft.parent.messageID,
      },
      thread_message_type: 'USER_REQUEST',
    }
    try {
      await startTaskMutation.mutateAsync(payload)
      closeThreadDialog()
      onNavigateToTask?.(projectId, payload.task_id)
    } catch (err) {
      setThreadError(err instanceof Error ? err.message : 'Failed to create threaded task')
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
    <div className="panel bus-panel">
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
          <Button
            inline
            onClick={() => {
              clearPendingSSEMessages()
              setMessages([])
              setExpandedBodies(new Set())
            }}
          >
            Clear
          </Button>
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
      <div className="panel-section panel-section-tight bus-list" role="list" aria-label="Message list">
        {filteredMessages.length === 0 && (
          <div className="empty-state">
            {!canCompose && scope === 'task' && 'Select a project and task to view task messages.'}
            {!canCompose && scope === 'project' && 'Select a project to view project messages.'}
            {canCompose && isHydrating && 'Loading existing messages…'}
            {canCompose && !isHydrating && loadError && `Failed to load messages: ${loadError}`}
            {canCompose && !isHydrating && !loadError && 'No messages yet.'}
          </div>
        )}
        {hiddenMessagesCount > 0 && (
          <div className="bus-render-window-hint" role="status">
            Showing latest {visibleMessages.length} of {filteredMessages.length} messages. Add a filter to inspect older entries.
          </div>
        )}
        {visibleMessages.map((msg) => {
          const enrichedMsg = msg as BusMessageWithAliases
          const projectID = resolveProjectID(enrichedMsg)
          const taskID = resolveTaskID(enrichedMsg)
          const runID = resolveRunID(enrichedMsg)
          const source = resolveMessageSource(enrichedMsg)
          const issueID = enrichedMsg.issue_id ?? ''
          const fallbackBody = msg.body ?? ''
          const bodyInfo = bodyDisplayByID.get(msg.msg_id) ?? {
            text: fallbackBody,
            long: false,
            truncated: false,
            rendered: renderMessageBody(fallbackBody),
          }
          const bodyID = `bus-message-body-${msg.msg_id}`
          const parentRef = extractThreadParentFromMeta(enrichedMsg)
          const childRef = extractThreadChildFromMeta(enrichedMsg)
          const answerLinks = threadedChildrenByParentMessageID.get(msg.msg_id) ?? []
          const canThreadAnswer = canStartThreadedAnswer(scope, enrichedMsg)

          return (
            <article
              key={msg.msg_id}
              className={clsx('bus-message', focusedMessageId === msg.msg_id && 'bus-message-focused')}
              role="listitem"
              aria-labelledby={`bus-message-title-${msg.msg_id}`}
            >
              <div className="bus-message-header">
                <span className={clsx('status-dot', `status-${msg.type.toLowerCase()}`)} />
                <span id={`bus-message-title-${msg.msg_id}`} className="bus-message-title">{msg.type}</span>
                <time className="bus-message-meta" dateTime={msg.timestamp}>{formatMessageTimestamp(msg.timestamp)}</time>
                <span className="bus-message-source">source: {source}</span>
                <span className="bus-message-id">{msg.msg_id}</span>
              </div>
              <div className="bus-message-context">
                {projectID && (
                  <span className="bus-message-context-item">
                    <span className="bus-message-context-label">project</span>
                    {projectID}
                  </span>
                )}
                {taskID && (
                  <span className="bus-message-context-item">
                    <span className="bus-message-context-label">task</span>
                    {taskID}
                  </span>
                )}
                {runID && (
                  <span className="bus-message-context-item">
                    <span className="bus-message-context-label">run</span>
                    {runID}
                  </span>
                )}
                {issueID && (
                  <span className="bus-message-context-item">
                    <span className="bus-message-context-label">issue</span>
                    {issueID}
                  </span>
                )}
              </div>
              <div className="bus-message-body" id={bodyID}>
                <div className="bus-message-text">{bodyInfo.rendered}</div>
                {bodyInfo.long && (
                  <button
                    type="button"
                    className="bus-message-expand"
                    onClick={() => toggleBodyExpansion(msg.msg_id)}
                    aria-expanded={!bodyInfo.truncated}
                    aria-controls={bodyID}
                  >
                    {bodyInfo.truncated ? 'Show more' : 'Show less'}
                  </button>
                )}
                {msg.parents && msg.parents.length > 0 && (
                  <div className="bus-message-parents">
                    Parents: {parseParentIDs(msg).join(', ')}
                  </div>
                )}
                {msg.attachment_path && (
                  <div className="bus-message-attachment">Attachment: {msg.attachment_path}</div>
                )}
                <div className="bus-message-links">
                  {parentRef && onNavigateToMessage && (
                    <button
                      type="button"
                      className="bus-link-button"
                      onClick={() => onNavigateToMessage(parentRef.projectID, parentRef.taskID, parentRef.messageID)}
                    >
                      Open source message
                    </button>
                  )}
                  {childRef && onNavigateToTask && (
                    <button
                      type="button"
                      className="bus-link-button"
                      onClick={() => onNavigateToTask(childRef.projectID, childRef.taskID)}
                    >
                      Open child task
                    </button>
                  )}
                  {answerLinks.map((child) => (
                    <button
                      key={`${msg.msg_id}-${child.projectID}-${child.taskID}`}
                      type="button"
                      className="bus-link-button"
                      onClick={() => onNavigateToTask?.(child.projectID, child.taskID)}
                    >
                      Open answer task: {child.taskID}
                    </button>
                  ))}
                  {canThreadAnswer && (
                    <button
                      type="button"
                      className="bus-link-button"
                      onClick={() => openThreadDialog(enrichedMsg)}
                    >
                      Answer in new task
                    </button>
                  )}
                </div>
              </div>
            </article>
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
            {MESSAGE_TYPES.map((typeName) => (
              <option key={typeName} value={typeName}>{typeName}</option>
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

      <Dialog
        show={threadDialogOpen}
        label="Create threaded answer task"
        showCloseButton
        onCloseAttempt={closeThreadDialog}
      >
        <div className="dialog-content dialog-content-wide">
          <div className="dialog-title">New threaded answer task</div>
          {threadDraft && (
            <form onSubmit={handleThreadSubmit}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <span className="form-label">Parent message</span>
                  <input
                    className="input"
                    value={`${threadDraft.parent.projectID}/${threadDraft.parent.taskID}/${threadDraft.parent.messageID}`}
                    readOnly
                  />
                </label>

                <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <span className="form-label">Thread message type</span>
                  <input className="input" value="USER_REQUEST" readOnly />
                </label>

                <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <span className="form-label">Task ID</span>
                  <input
                    className="input"
                    value={threadDraft.taskID}
                    onChange={(e) => setThreadDraft((prev) => (
                      prev ? { ...prev, taskID: e.target.value } : prev
                    ))}
                    required
                  />
                </label>

                <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <span className="form-label">Agent</span>
                  <select
                    className="input"
                    value={threadDraft.agentType}
                    onChange={(e) => setThreadDraft((prev) => (
                      prev ? { ...prev, agentType: e.target.value } : prev
                    ))}
                  >
                    <option value="claude">claude</option>
                    <option value="codex">codex</option>
                    <option value="gemini">gemini</option>
                  </select>
                </label>

                <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <span className="form-label">Prompt</span>
                  <textarea
                    className="input new-task-prompt"
                    value={threadDraft.prompt}
                    onChange={(e) => setThreadDraft((prev) => (
                      prev ? { ...prev, prompt: e.target.value } : prev
                    ))}
                    rows={8}
                    required
                  />
                </label>

                <label style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                  <span className="form-label">Project Home Directory</span>
                  <input
                    className="input"
                    value={threadDraft.projectRoot}
                    onChange={(e) => setThreadDraft((prev) => (
                      prev ? { ...prev, projectRoot: e.target.value } : prev
                    ))}
                    placeholder="~/Work/my-project  or  /absolute/path/to/project"
                  />
                </label>

                {threadError && (
                  <div className="form-error">{threadError}</div>
                )}

                <div className="dialog-actions">
                  <Button
                    inline
                    onClick={closeThreadDialog}
                    type="button"
                    disabled={startTaskMutation.isPending}
                  >
                    Cancel
                  </Button>
                  <Button
                    inline
                    primary
                    type="submit"
                    disabled={startTaskMutation.isPending}
                  >
                    {startTaskMutation.isPending ? 'Creating…' : 'Create Threaded Task'}
                  </Button>
                </div>
              </div>
            </form>
          )}
        </div>
      </Dialog>
    </div>
  )
}
