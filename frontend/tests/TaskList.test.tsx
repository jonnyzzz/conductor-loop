import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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

    await user.click(screen.getByTestId('task-filter-running'))
    expect(screen.getByTestId('task-item-task-running')).toBeInTheDocument()
    expect(screen.queryByTestId('task-item-task-completed')).not.toBeInTheDocument()
  })
})
