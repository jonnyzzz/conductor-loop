import { render, screen } from '@testing-library/react'
import { RunDetail } from '../src/components/RunDetail'
import type { FileContent, RunInfo, TaskDetail } from '../src/types'

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

  it('renders metadata and file content', () => {
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

    expect(screen.getByText('Run detail')).toBeInTheDocument()
    expect(screen.getAllByText('run-1').length).toBeGreaterThan(0)
    expect(screen.getByText('Hello output')).toBeInTheDocument()
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
})
