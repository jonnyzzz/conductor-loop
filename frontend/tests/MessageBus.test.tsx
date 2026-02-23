import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { BusMessage } from '../src/types'

const mockedState = vi.hoisted(() => ({
  projectMessages: [] as BusMessage[],
  taskMessages: [] as BusMessage[],
  projectError: null as Error | null,
  taskError: null as Error | null,
  postTaskMutateAsync: vi.fn(async () => ({ msg_id: 'task-msg' })),
  postProjectMutateAsync: vi.fn(async () => ({ msg_id: 'project-msg' })),
  startTaskMutateAsync: vi.fn(async () => ({ task_id: 'thread-task', status: 'started', run_id: 'run-1' })),
}))

vi.mock('../src/hooks/useAPI', () => ({
  useProjectMessages: () => ({
    data: mockedState.projectMessages,
    isFetching: false,
    error: mockedState.projectError,
  }),
  useTaskMessages: () => ({
    data: mockedState.taskMessages,
    isFetching: false,
    error: mockedState.taskError,
  }),
  usePostTaskMessage: () => ({
    mutateAsync: mockedState.postTaskMutateAsync,
    isPending: false,
  }),
  usePostProjectMessage: () => ({
    mutateAsync: mockedState.postProjectMutateAsync,
    isPending: false,
  }),
  useStartTask: () => ({
    mutateAsync: mockedState.startTaskMutateAsync,
    isPending: false,
  }),
}))

vi.mock('../src/hooks/useSSE', () => ({
  useSSE: () => ({ state: 'open', errorCount: 0 }),
}))

type MockRingButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  children?: ReactNode
  inline?: boolean
  primary?: boolean
}

vi.mock('@jetbrains/ring-ui-built/components/button/button', () => ({
  default: ({ children, inline, primary, ...props }: MockRingButtonProps) => {
    void inline
    void primary
    return (
      <button type="button" {...props}>
        {children}
      </button>
    )
  },
}))

vi.mock('@jetbrains/ring-ui-built/components/dialog/dialog', () => ({
  default: ({ children, show }: { children: ReactNode; show: boolean }) => (show ? <div>{children}</div> : null),
}))

import { MessageBus } from '../src/components/MessageBus'

describe('MessageBus', () => {
  beforeEach(() => {
    mockedState.projectMessages = []
    mockedState.taskMessages = []
    mockedState.projectError = null
    mockedState.taskError = null
    mockedState.postTaskMutateAsync.mockReset()
    mockedState.postProjectMutateAsync.mockReset()
    mockedState.startTaskMutateAsync.mockReset()
  })

  it('renders message content and hierarchy without per-item expand click', () => {
    mockedState.taskMessages = [
      {
        msg_id: 'MSG-1',
        timestamp: '2026-02-22T17:21:01Z',
        type: 'FACT',
        body: 'Inline body visible immediately',
        project_id: 'conductor-loop',
        task_id: 'task-1',
        run_id: 'run-1',
        meta: { source: 'codex' },
      },
    ]

    render(
      <MessageBus
        title="Task message bus"
        projectId="conductor-loop"
        taskId="task-1"
        scope="task"
      />
    )

    expect(screen.getByText('Inline body visible immediately')).toBeInTheDocument()
    expect(screen.getByText('FACT', { selector: '.bus-message-title' })).toBeInTheDocument()
    expect(screen.getByText('source: codex')).toBeInTheDocument()
    expect(screen.getByText('conductor-loop')).toBeInTheDocument()
    expect(screen.getByText('task-1')).toBeInTheDocument()
    expect(screen.getByText('run-1')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /show more/i })).not.toBeInTheDocument()
  })

  it('truncates only very long bodies and supports optional expand/collapse', async () => {
    const user = userEvent.setup()
    const longBody = `${'line of detail\n'.repeat(220)}UNIQUE_END_MARKER`
    mockedState.taskMessages = [
      {
        msg_id: 'MSG-LONG',
        timestamp: '2026-02-22T17:30:00Z',
        type: 'PROGRESS',
        body: longBody,
        project_id: 'conductor-loop',
        task_id: 'task-1',
      },
    ]

    render(
      <MessageBus
        title="Task message bus"
        projectId="conductor-loop"
        taskId="task-1"
        scope="task"
      />
    )

    const messageItem = screen.getByRole('listitem')
    expect(messageItem).not.toHaveTextContent('UNIQUE_END_MARKER')
    const showMore = screen.getByRole('button', { name: 'Show more' })
    expect(showMore).toHaveAttribute('aria-expanded', 'false')

    await user.click(showMore)
    expect(messageItem).toHaveTextContent('UNIQUE_END_MARKER')
    const showLess = screen.getByRole('button', { name: 'Show less' })
    expect(showLess).toHaveAttribute('aria-expanded', 'true')

    await user.click(showLess)
    expect(messageItem).not.toHaveTextContent('UNIQUE_END_MARKER')
  })

  it('navigates from child threaded message to source message', async () => {
    const user = userEvent.setup()
    const navigateToMessage = vi.fn()
    mockedState.taskMessages = [
      {
        msg_id: 'MSG-CHILD',
        timestamp: '2026-02-22T17:31:00Z',
        type: 'USER_REQUEST',
        body: 'Answer request',
        project_id: 'conductor-loop',
        task_id: 'task-child',
        run_id: 'run-child',
        meta: {
          thread_parent_project_id: 'conductor-loop',
          thread_parent_task_id: 'task-parent',
          thread_parent_run_id: 'run-parent',
          thread_parent_message_id: 'MSG-PARENT',
        },
      },
    ]

    render(
      <MessageBus
        title="Task message bus"
        projectId="conductor-loop"
        taskId="task-child"
        scope="task"
        onNavigateToMessage={navigateToMessage}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Open source message' }))
    expect(navigateToMessage).toHaveBeenCalledWith('conductor-loop', 'task-parent', 'MSG-PARENT')
  })

  it('shows answer-task navigation on source messages via threaded linkage metadata', async () => {
    const user = userEvent.setup()
    const navigateToTask = vi.fn()
    mockedState.taskMessages = [
      {
        msg_id: 'MSG-PARENT',
        timestamp: '2026-02-22T17:31:00Z',
        type: 'QUESTION',
        body: 'How should we proceed?',
        project_id: 'conductor-loop',
        task_id: 'task-parent',
        run_id: 'run-parent',
      },
      {
        msg_id: 'MSG-LINK',
        timestamp: '2026-02-22T17:32:00Z',
        type: 'USER_REQUEST',
        body: 'threaded user request opened child task conductor-loop/task-child',
        project_id: 'conductor-loop',
        task_id: 'task-parent',
        run_id: 'run-parent',
        parents: ['MSG-PARENT'],
        meta: {
          thread_child_project_id: 'conductor-loop',
          thread_child_task_id: 'task-child',
          thread_child_message_id: 'MSG-CHILD',
        },
      },
    ]

    render(
      <MessageBus
        title="Task message bus"
        projectId="conductor-loop"
        taskId="task-parent"
        scope="task"
        onNavigateToTask={navigateToTask}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Open answer task: task-child' }))
    expect(navigateToTask).toHaveBeenCalledWith('conductor-loop', 'task-child')
  })

  it('limits unfiltered rendering to the latest window and keeps older messages discoverable via filter', async () => {
    const user = userEvent.setup()
    mockedState.taskMessages = Array.from({ length: 320 }, (_, i) => ({
      msg_id: `MSG-${String(i).padStart(3, '0')}`,
      timestamp: new Date(Date.UTC(2026, 1, 22, 17, 0, i)).toISOString(),
      type: 'PROGRESS',
      body: `message-${String(i).padStart(3, '0')}`,
      project_id: 'conductor-loop',
      task_id: 'task-1',
      run_id: 'run-1',
    }))

    render(
      <MessageBus
        title="Task message bus"
        projectId="conductor-loop"
        taskId="task-1"
        scope="task"
      />
    )

    expect(screen.getByText('Showing latest 120 of 320 messages. Add a filter to inspect older entries.')).toBeInTheDocument()
    expect(screen.queryByText('message-000')).not.toBeInTheDocument()
    expect(screen.getAllByRole('listitem')).toHaveLength(120)

    await user.type(screen.getByPlaceholderText('Filter by text or id'), 'message-000')
    expect(screen.getByText('message-000')).toBeInTheDocument()
    expect(screen.queryByText('Showing latest 120 of 320 messages. Add a filter to inspect older entries.')).not.toBeInTheDocument()
  })
})
