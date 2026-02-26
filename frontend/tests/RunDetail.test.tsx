import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi, describe, it, expect } from 'vitest'
import { RunDetail } from '../src/components/RunDetail'
import type { FileContent, RunInfo, TaskDetail } from '../src/types'

async function clickTab(user: ReturnType<typeof userEvent.setup>, name: string) {
  await user.click(screen.getByRole('tab', { name }))
}

describe('RunDetail', () => {
  const task: TaskDetail = {
    id: 'task-1',
    name: 'task-1',
    project_id: 'swarm',
    status: 'running',
    last_activity: '2026-02-04T17:31:55Z',
    created_at: '2026-02-04T17:00:00Z',
    done: false,
    state: 'Working',
    runs: [
      {
        id: 'run-1',
        agent: 'codex',
        status: 'running',
        exit_code: 0,
        start_time: '2026-02-04T17:30:42Z',
        end_time: '2026-02-04T17:31:55Z',
      },
    ],
  }

  const runInfo: RunInfo = {
    version: 1,
    run_id: 'run-1',
    project_id: 'swarm',
    task_id: 'task-1',
    parent_run_id: '',
    previous_run_id: '',
    agent: 'codex',
    pid: 123,
    pgid: 123,
    start_time: '2026-02-04T17:30:42Z',
    end_time: '2026-02-04T17:31:55Z',
    exit_code: 0,
    cwd: '/tmp',
  }

  const fileContent: FileContent = {
    name: 'output.md',
    content: 'Hello output',
    modified: '2026-02-04T17:31:55Z',
  }

  it('renders metadata and file content', async () => {
    const user = userEvent.setup()
    render(
      <RunDetail
        task={task}
        runInfo={runInfo}
        selectedRunId="run-1"
        onSelectRun={() => undefined}
        fileName="output.md"
        onSelectFile={() => undefined}
        fileContent={fileContent}
        taskState="Task state"
      />
    )

    // Header and overview are always visible
    expect(screen.getByText('Run detail')).toBeInTheDocument()
    expect(screen.getAllByText('run-1').length).toBeGreaterThan(0)

    // Files tab is the default: file content visible
    expect(screen.getByText('Hello output')).toBeInTheDocument()

    // Task state and full metadata are in the "Run metadata" tab
    await clickTab(user, 'Run metadata')
    expect(screen.getByText('Task state')).toBeInTheDocument()
  })

  it('renders compact empty state when no task is selected', () => {
    render(
      <RunDetail
        task={undefined}
        runInfo={undefined}
        selectedRunId={undefined}
        onSelectRun={() => undefined}
        fileName="output.md"
        onSelectFile={() => undefined}
        fileContent={undefined}
      />
    )

    expect(screen.getByText('Select a task')).toBeInTheDocument()
    expect(screen.getByText('Select a task from the tree to inspect metadata and run files.')).toBeInTheDocument()
    expect(screen.queryByText('Run files')).not.toBeInTheDocument()
  })

  it('shows stop agent for running runs and calls handler', async () => {
    const user = userEvent.setup()
    const onStopRun = vi.fn()
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    render(
      <RunDetail
        task={task}
        runInfo={runInfo}
        selectedRunId="run-1"
        onSelectRun={() => undefined}
        fileName="output.md"
        onSelectFile={() => undefined}
        fileContent={fileContent}
        onStopRun={onStopRun}
      />
    )

    // Stop agent button is in the Run metadata tab
    await clickTab(user, 'Run metadata')
    await user.click(screen.getByRole('button', { name: 'Stop agent' }))
    expect(onStopRun).toHaveBeenCalledWith('run-1')
    confirmSpy.mockRestore()
  })

  it('does not render delete actions in run detail', () => {
    const completedTask: TaskDetail = {
      ...task,
      status: 'completed',
      done: true,
      runs: [
        {
          ...task.runs[0],
          status: 'completed',
          end_time: '2026-02-04T17:31:55Z',
        },
      ],
    }
    const completedRunInfo: RunInfo = {
      ...runInfo,
      exit_code: 0,
      end_time: '2026-02-04T17:31:55Z',
    }

    render(
      <RunDetail
        task={completedTask}
        runInfo={completedRunInfo}
        selectedRunId="run-1"
        onSelectRun={() => undefined}
        fileName="output.md"
        onSelectFile={() => undefined}
        fileContent={fileContent}
        onResumeTask={() => undefined}
      />
    )

    expect(screen.queryByRole('button', { name: /delete/i })).not.toBeInTheDocument()
    expect(screen.queryByText('Delete run')).not.toBeInTheDocument()
    expect(screen.queryByText('Delete task')).not.toBeInTheDocument()
  })

  it('handles tasks with null runs without crashing', async () => {
    const user = userEvent.setup()
    const taskWithNullRuns: TaskDetail = {
      ...task,
      status: 'unknown',
      runs: null as unknown as TaskDetail['runs'],
    }

    render(
      <RunDetail
        task={taskWithNullRuns}
        runInfo={undefined}
        selectedRunId={undefined}
        onSelectRun={() => undefined}
        fileName="output.md"
        onSelectFile={() => undefined}
        fileContent={fileContent}
      />
    )

    // "No runs yet" is in the task overview (always visible)
    expect(screen.getByText('No runs yet')).toBeInTheDocument()

    // "No running or failed runs..." message is in the Run tree tab
    await clickTab(user, 'Run tree')
    expect(screen.getByText(/No running or failed runs\. Expand completed runs to inspect history\./)).toBeInTheDocument()
  })

  it('caps completed run rendering and progressively loads older history', async () => {
    const user = userEvent.setup()
    const completedRuns = Array.from({ length: 260 }, (_, i) => {
      const sec = String(i % 60).padStart(2, '0')
      const min = String(Math.floor(i / 60)).padStart(2, '0')
      return {
        id: `run-completed-${String(i).padStart(3, '0')}`,
        agent: 'codex',
        status: 'completed' as const,
        exit_code: 0,
        start_time: `2026-02-04T17:${min}:${sec}Z`,
        end_time: `2026-02-04T17:${min}:${sec}Z`,
      }
    })
    const heavyTask: TaskDetail = {
      ...task,
      runs: [
        {
          id: 'run-live',
          agent: 'codex',
          status: 'running',
          exit_code: -1,
          start_time: '2026-02-04T18:00:00Z',
        },
        ...completedRuns,
      ],
    }

    render(
      <RunDetail
        task={heavyTask}
        runInfo={undefined}
        selectedRunId="run-live"
        onSelectRun={() => undefined}
        fileName="output.md"
        onSelectFile={() => undefined}
        fileContent={fileContent}
      />
    )

    // Run tree controls are in the Run tree tab
    await clickTab(user, 'Run tree')
    await user.click(screen.getByRole('button', { name: '... 260 completed' }))

    expect(screen.getByText('Showing latest 200 of 260 completed runs.')).toBeInTheDocument()
    expect(screen.queryByText('run-completed-000')).not.toBeInTheDocument()
    expect(screen.getByText('run-completed-259')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Load older completed (+60)' }))
    expect(screen.getByText('Showing 260 completed runs.')).toBeInTheDocument()
    expect(screen.getByText('run-completed-000')).toBeInTheDocument()
  })

  it('keeps selected completed run visible even when history is capped', async () => {
    const completedRuns = Array.from({ length: 230 }, (_, i) => {
      const sec = String(i % 60).padStart(2, '0')
      const min = String(Math.floor(i / 60)).padStart(2, '0')
      return {
        id: `run-archive-${String(i).padStart(3, '0')}`,
        agent: 'claude',
        status: 'completed' as const,
        exit_code: 0,
        start_time: `2026-02-04T17:${min}:${sec}Z`,
        end_time: `2026-02-04T17:${min}:${sec}Z`,
      }
    })
    const archiveTask: TaskDetail = {
      ...task,
      status: 'completed',
      runs: completedRuns,
    }

    const user = userEvent.setup()
    render(
      <RunDetail
        task={archiveTask}
        runInfo={undefined}
        selectedRunId="run-archive-000"
        onSelectRun={() => undefined}
        fileName="output.md"
        onSelectFile={() => undefined}
        fileContent={fileContent}
      />
    )

    // Run tree content is in the Run tree tab
    await clickTab(user, 'Run tree')
    expect(screen.getByText('run-archive-000')).toBeInTheDocument()
    expect(screen.getByText('Showing latest 201 of 230 completed runs.')).toBeInTheDocument()
  })
})
