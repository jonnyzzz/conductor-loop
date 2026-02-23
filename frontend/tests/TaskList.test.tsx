import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

vi.mock('../src/hooks/useAPI', () => ({
  useStartTask: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useProjectStats: () => ({ data: undefined, isLoading: false, isError: false }),
}))

vi.mock('@jetbrains/ring-ui-built/components/button/button', () => ({
  default: ({ children, ...props }: any) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
}))

vi.mock('@jetbrains/ring-ui-built/components/dialog/dialog', () => ({
  default: ({ children, show }: any) => (show ? <div>{children}</div> : null),
}))

import { TaskList } from '../src/components/TaskList'
import type { Project, TaskSummary } from '../src/types'

describe('TaskList', () => {
  const projects: Project[] = [
    { id: 'swarm', last_activity: '2026-02-04T17:31:55Z', task_count: 2 },
  ]

  const tasks: TaskSummary[] = [
    {
      id: 'task-running',
      name: 'running',
      status: 'running',
      last_activity: '2026-02-04T17:31:55Z',
      run_count: 1,
    },
    {
      id: 'task-completed',
      name: 'completed',
      status: 'completed',
      last_activity: '2026-02-03T12:01:00Z',
      run_count: 2,
    },
    {
      id: 'task-queued',
      name: 'queued',
      status: 'queued',
      queue_position: 2,
      last_activity: '2026-02-02T12:01:00Z',
      run_count: 0,
    },
  ]

  it('filters tasks by status', async () => {
    const user = userEvent.setup()
    render(
      <TaskList
        projects={projects}
        tasks={tasks}
        selectedProjectId="swarm"
        selectedTaskId="task-running"
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
      />
    )

    expect(screen.getByTestId('task-item-task-running')).toBeInTheDocument()
    expect(screen.getByTestId('task-item-task-completed')).toBeInTheDocument()
    expect(screen.getByTestId('task-item-task-queued')).toBeInTheDocument()

    await user.click(screen.getByTestId('task-filter-running'))
    expect(screen.getByTestId('task-item-task-running')).toBeInTheDocument()
    expect(screen.queryByTestId('task-item-task-completed')).not.toBeInTheDocument()
    expect(screen.queryByTestId('task-item-task-queued')).not.toBeInTheDocument()
  })

  it('shows queue position and supports queued filter', async () => {
    const user = userEvent.setup()
    render(
      <TaskList
        projects={projects}
        tasks={tasks}
        selectedProjectId="swarm"
        selectedTaskId="task-queued"
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
      />
    )

    expect(screen.getByText(/queued .*queue #2/i)).toBeInTheDocument()
    await user.click(screen.getByTestId('task-filter-queued'))
    expect(screen.queryByTestId('task-item-task-running')).not.toBeInTheDocument()
    expect(screen.queryByTestId('task-item-task-completed')).not.toBeInTheDocument()
    expect(screen.getByTestId('task-item-task-queued')).toBeInTheDocument()
  })
})
