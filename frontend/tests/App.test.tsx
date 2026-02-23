import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { TaskDetail } from '../src/types'

const project = {
  id: 'conductor-loop',
  task_count: 2,
  last_activity: '2026-02-22T17:31:55Z',
}

const tasksByID: Record<string, TaskDetail> = {
  'task-1': {
    id: 'task-1',
    name: 'task-1',
    project_id: 'conductor-loop',
    status: 'running',
    last_activity: '2026-02-22T17:31:55Z',
    created_at: '2026-02-22T17:00:00Z',
    done: false,
    state: 'Working',
    runs: [
      {
        id: 'run-1',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T17:30:00Z',
        end_time: '',
        files: [{ name: 'output.md', label: 'Output' }],
      },
    ],
  },
  'task-2': {
    id: 'task-2',
    name: 'task-2',
    project_id: 'conductor-loop',
    status: 'running',
    last_activity: '2026-02-22T17:32:55Z',
    created_at: '2026-02-22T17:01:00Z',
    done: false,
    state: 'Working',
    runs: [
      {
        id: 'run-2',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T17:31:00Z',
        end_time: '',
        files: [{ name: 'output.md', label: 'Output' }],
      },
    ],
  },
}

const mockedState = vi.hoisted(() => ({
  projectsRefetch: vi.fn(),
  stopRunMutate: vi.fn(),
  resumeTaskMutate: vi.fn(),
  useTaskArgs: [] as Array<[string | undefined, string | undefined]>,
  useRunInfoArgs: [] as Array<[string | undefined, string | undefined, string | undefined]>,
  useTaskFileArgs: [] as Array<[string | undefined, string | undefined, string | undefined]>,
  useRunFileArgs: [] as Array<[string | undefined, string | undefined, string | undefined, string | undefined, number | undefined]>,
}))

vi.mock('../src/hooks/useAPI', () => ({
  useProjects: () => ({
    data: [project],
    refetch: mockedState.projectsRefetch,
  }),
  useTask: (_projectID?: string, taskID?: string) => ({
    data: (() => {
      mockedState.useTaskArgs.push([_projectID, taskID])
      return taskID ? tasksByID[taskID] : undefined
    })(),
  }),
  useRunInfo: (_projectID?: string, taskID?: string, runID?: string) => ({
    data: (() => {
      mockedState.useRunInfoArgs.push([_projectID, taskID, runID])
      return runID
        ? {
          version: 1,
          run_id: runID,
          project_id: _projectID ?? 'conductor-loop',
          task_id: taskID ?? 'task-1',
          parent_run_id: '',
          previous_run_id: '',
          agent: 'codex',
          pid: 123,
          pgid: 123,
          start_time: '2026-02-22T17:30:00Z',
          end_time: '2026-02-22T17:31:00Z',
          exit_code: 0,
          cwd: '/tmp',
        }
        : undefined
    })(),
  }),
  useTaskFile: (_projectID?: string, _taskID?: string, name?: string) => ({
    data: (() => {
      mockedState.useTaskFileArgs.push([_projectID, _taskID, name])
      return {
        name: 'TASK.md',
        content: 'task state',
        modified: '2026-02-22T17:31:55Z',
      }
    })(),
  }),
  useRunFile: (_projectID?: string, _taskID?: string, runID?: string, name?: string, tail?: number) => ({
    data: (() => {
      mockedState.useRunFileArgs.push([_projectID, _taskID, runID, name, tail])
      return runID
        ? {
          name: name ?? 'output.md',
          content: `${runID} content`,
          modified: '2026-02-22T17:31:55Z',
        }
        : undefined
    })(),
  }),
  useStopRun: () => ({ mutate: mockedState.stopRunMutate }),
  useResumeTask: () => ({ mutate: mockedState.resumeTaskMutate }),
}))

vi.mock('../src/components/TreePanel', () => ({
  TreePanel: ({ onSelectTask }: { onSelectTask: (projectID: string, taskID: string) => void }) => (
    <div data-testid="tree-panel">
      <button type="button" onClick={() => onSelectTask('conductor-loop', 'task-1')}>
        Select task 1
      </button>
      <button type="button" onClick={() => onSelectTask('conductor-loop', 'task-2')}>
        Select task 2
      </button>
    </div>
  ),
}))

vi.mock('../src/components/RunDetail', () => ({
  RunDetail: ({ task }: { task?: TaskDetail }) => (
    <div data-testid="run-detail">run-detail:{task?.id ?? 'none'}</div>
  ),
}))

vi.mock('../src/components/MessageBus', () => ({
  MessageBus: ({ title, scope, headerActions }: { title: string; scope: string; headerActions: unknown }) => (
    <div data-testid="message-bus">
      <div>{title}</div>
      <div>{scope}</div>
      <div>{headerActions}</div>
    </div>
  ),
}))

vi.mock('../src/components/LogViewer', () => ({
  LogViewer: ({ streamUrl }: { streamUrl?: string }) => (
    <div data-testid="live-logs">{streamUrl ?? 'no-stream'}</div>
  ),
}))

vi.mock('../src/hooks/useLiveRunRefresh', () => ({
  useLiveRunRefresh: () => ({ state: 'open', errorCount: 0 }),
}))

vi.mock('@jetbrains/ring-ui-built/components/button/button', () => ({
  default: ({ children, inline: _inline, ...props }: any) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
}))

import { App } from '../src/App'

describe('App', () => {
  beforeEach(() => {
    mockedState.projectsRefetch.mockReset()
    mockedState.stopRunMutate.mockReset()
    mockedState.resumeTaskMutate.mockReset()
    mockedState.useTaskArgs = []
    mockedState.useRunInfoArgs = []
    mockedState.useTaskFileArgs = []
    mockedState.useRunFileArgs = []
  })

  it('renders dedicated task section tabs and no standalone logs column', async () => {
    const user = userEvent.setup()
    const view = render(<App />)

    expect(screen.getByRole('tab', { name: 'Task details' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Message bus' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Live logs' })).toBeInTheDocument()
    expect(screen.getByTestId('run-detail')).toBeInTheDocument()
    expect(view.container.querySelector('.app-panel-logs')).toBeNull()

    await user.click(screen.getByRole('tab', { name: 'Live logs' }))
    expect(screen.getByTestId('live-logs')).toBeInTheDocument()
    expect(screen.queryByTestId('run-detail')).not.toBeInTheDocument()
  })

  it('resets to Task details when selecting a different task', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('button', { name: 'Select task 1' }))
    await user.click(screen.getByRole('tab', { name: 'Live logs' }))
    expect(screen.getByTestId('live-logs')).toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: 'Message bus' }))
    expect(screen.getByTestId('message-bus')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Select task 2' }))
    await waitFor(() => {
      expect(screen.getByTestId('run-detail')).toHaveTextContent('task-2')
    })
    expect(screen.queryByTestId('message-bus')).not.toBeInTheDocument()
    expect(screen.queryByTestId('live-logs')).not.toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Task details' })).toHaveAttribute('aria-selected', 'true')
  })

  it('disables detail queries when viewing non-details sections', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('button', { name: 'Select task 1' }))
    await waitFor(() => {
      expect(screen.getByTestId('run-detail')).toHaveTextContent('task-1')
    })

    expect(mockedState.useTaskArgs.at(-1)).toEqual(['conductor-loop', 'task-1'])
    expect(mockedState.useRunFileArgs.at(-1)?.[0]).toBe('conductor-loop')

    await user.click(screen.getByRole('tab', { name: 'Message bus' }))
    await waitFor(() => {
      expect(screen.getByTestId('message-bus')).toBeInTheDocument()
    })

    expect(mockedState.useTaskArgs.at(-1)).toEqual([undefined, undefined])
    expect(mockedState.useRunInfoArgs.at(-1)).toEqual([undefined, undefined, undefined])
    expect(mockedState.useTaskFileArgs.at(-1)).toEqual([undefined, undefined, undefined])
    expect(mockedState.useRunFileArgs.at(-1)?.slice(0, 4)).toEqual([undefined, undefined, undefined, undefined])
  })
})
