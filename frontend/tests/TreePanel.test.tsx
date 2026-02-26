import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

let mockProjects: Array<{ id: string; task_count: number; last_activity: string; project_root?: string }> = []
let mockTasks: Array<{
  id: string
  status: string
  last_activity: string
  queue_position?: number
  thread_parent?: {
    project_id: string
    task_id: string
    run_id: string
    message_id: string
    message_type?: string
  }
}> = []
let mockRuns: Array<{
  id: string
  task_id: string
  agent: string
  status: string
  exit_code: number
  start_time: string
  end_time?: string
  previous_run_id?: string
  parent_run_id?: string
}> = []

const mockMutateAsync = vi.fn()
const mockCreateProjectMutateAsync = vi.fn()
const mockFlatRunsRefetch = vi.fn()

vi.mock('../src/hooks/useAPI', () => ({
  useProjects: () => ({ data: mockProjects }),
  useTasks: () => ({ data: mockTasks }),
  useProjectRunsFlat: () => ({ data: mockRuns, refetch: mockFlatRunsRefetch }),
  useHomeDirs: () => ({ data: { dirs: [] } }),
  useCreateProject: () => ({ mutateAsync: mockCreateProjectMutateAsync, isPending: false }),
  useStartTask: () => ({ mutateAsync: mockMutateAsync, isPending: false }),
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

import { TreePanel } from '../src/components/TreePanel'

describe('TreePanel', () => {
  beforeEach(() => {
    mockProjects = []
    mockTasks = []
    mockRuns = []
    mockMutateAsync.mockReset()
    mockCreateProjectMutateAsync.mockReset()
    mockFlatRunsRefetch.mockReset()
    mockFlatRunsRefetch.mockResolvedValue(undefined)
    mockMutateAsync.mockResolvedValue({ task_id: 'task-test', status: 'created', run_id: 'run-1' })
    mockCreateProjectMutateAsync.mockResolvedValue({ id: 'conductor-loop' })
  })

  it('shows restart chain as visible tree and duration badge on task row', () => {
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 1,
        last_activity: '2026-02-21T21:00:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-20260221-210000-ui-density',
        status: 'running',
        last_activity: '2026-02-21T21:03:15Z',
      },
    ]
    mockRuns = [
      {
        id: 'run-001',
        task_id: 'task-20260221-210000-ui-density',
        agent: 'codex',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-21T21:00:00Z',
        end_time: '2026-02-21T21:01:00Z',
      },
      {
        id: 'run-002',
        task_id: 'task-20260221-210000-ui-density',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-21T21:01:00Z',
        end_time: '2026-02-21T21:03:15Z',
        previous_run_id: 'run-001',
      },
    ]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId="task-20260221-210000-ui-density"
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    // Restart chain forms a visible tree: run-002 is a root node, run-001 is its child.
    // No restart:N badge (restartCount is no longer set for chains shown as a tree).
    expect(screen.queryByText('restart:1')).not.toBeInTheDocument()
    // dur:2m15s appears on both the task row (latestRun) and the run-002 node row.
    expect(screen.getAllByText('dur:2m15s').length).toBeGreaterThan(0)
    expect(screen.getByText(/run-002/)).toBeInTheDocument()
  })

  it('keeps level-3 subtask hierarchy visible alongside nested runs', () => {
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 3,
        last_activity: '2026-02-22T18:07:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-20260222-180000-root-review',
        status: 'running',
        last_activity: '2026-02-22T18:05:00Z',
      },
      {
        id: 'task-20260222-180100-child-audit',
        status: 'running',
        last_activity: '2026-02-22T18:06:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-20260222-180000-root-review',
          run_id: 'run-root',
          message_id: 'MSG-root',
        },
      },
      {
        id: 'task-20260222-180200-grandchild-check',
        status: 'running',
        last_activity: '2026-02-22T18:07:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-20260222-180100-child-audit',
          run_id: 'run-child',
          message_id: 'MSG-child',
        },
      },
    ]
    mockRuns = [
      {
        id: 'run-root',
        task_id: 'task-20260222-180000-root-review',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T18:00:00Z',
      },
      {
        id: 'run-child',
        task_id: 'task-20260222-180100-child-audit',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T18:02:00Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-grandchild',
        task_id: 'task-20260222-180200-grandchild-check',
        agent: 'gemini',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T18:04:00Z',
        parent_run_id: 'run-child',
      },
    ]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    // Tasks are nested via thread_parent; each has a single run inlined into
    // the task row (no separate run nodes since there is no restart chain).
    expect(screen.getByText('root-review')).toBeInTheDocument()
    expect(screen.getByText('child-audit')).toBeInTheDocument()
    expect(screen.getByText('grandchild-check')).toBeInTheDocument()
  })

  it('keeps ancestor bridge run rows when selected deep task uses restarted parent run', async () => {
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 3,
        last_activity: '2026-02-22T21:09:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-root',
        status: 'completed',
        last_activity: '2026-02-22T21:05:00Z',
      },
      {
        id: 'task-child',
        status: 'completed',
        last_activity: '2026-02-22T21:07:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-root',
          run_id: 'run-root',
          message_id: 'MSG-root',
        },
      },
      {
        id: 'task-grandchild',
        status: 'failed',
        last_activity: '2026-02-22T21:09:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-child',
          run_id: 'run-child-linked',
          message_id: 'MSG-child',
        },
      },
    ]
    mockRuns = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
        end_time: '2026-02-22T21:01:00Z',
      },
      {
        id: 'run-child-linked',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:02:00Z',
        end_time: '2026-02-22T21:03:00Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-child-restarted',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:04:00Z',
        end_time: '2026-02-22T21:05:00Z',
        previous_run_id: 'run-child-linked',
      },
      {
        id: 'run-grandchild-selected',
        task_id: 'task-grandchild',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T21:06:00Z',
        end_time: '2026-02-22T21:07:00Z',
        parent_run_id: 'run-child-restarted',
      },
    ]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId="task-grandchild"
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    await waitFor(() => {
      // task-root has a single run (run-root) inlined into its task row — run-root is not a
      // separate tree node. Verify that task-root itself (ancestor) remains visible.
      expect(screen.getByText('task-root')).toBeInTheDocument()
      expect(screen.getByText(/run-child-rest/)).toBeInTheDocument()
      expect(screen.getByText('task-grandchild')).toBeInTheDocument()
    })

    const childRow = screen.getByText('task-child').closest('button')
    const grandchildRow = screen.getByText('task-grandchild').closest('button')
    expect(childRow).toHaveStyle({ paddingLeft: '24px' })
    expect(grandchildRow).toHaveStyle({ paddingLeft: '34px' })
  })

  it('keeps ancestor bridge run rows when selected deep task uses detached parent run', async () => {
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 3,
        last_activity: '2026-02-22T21:09:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-root',
        status: 'completed',
        last_activity: '2026-02-22T21:05:00Z',
      },
      {
        id: 'task-child',
        status: 'completed',
        last_activity: '2026-02-22T21:07:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-root',
          run_id: 'run-root',
          message_id: 'MSG-root',
        },
      },
      {
        id: 'task-grandchild',
        status: 'failed',
        last_activity: '2026-02-22T21:09:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-child',
          run_id: 'run-child-linked',
          message_id: 'MSG-child',
        },
      },
    ]
    // Simulates active_only + selected_task filter payload where backend keeps
    // ancestor bridge runs even though the selected parent run is detached.
    mockRuns = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:00:00Z',
        end_time: '2026-02-22T21:01:00Z',
      },
      {
        id: 'run-child-linked',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:02:00Z',
        end_time: '2026-02-22T21:03:00Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-child-detached',
        task_id: 'task-child',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T21:04:00Z',
        end_time: '2026-02-22T21:05:00Z',
      },
      {
        id: 'run-grandchild-selected',
        task_id: 'task-grandchild',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T21:06:00Z',
        end_time: '2026-02-22T21:07:00Z',
        parent_run_id: 'run-child-detached',
      },
    ]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId="task-grandchild"
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    await waitFor(() => {
      // task-root has a single run (run-root) inlined into its task row.
      // Verify the ancestor task itself is visible, not its individual run node.
      expect(screen.getByText('task-root')).toBeInTheDocument()
      expect(screen.getByText(/run-child-det/)).toBeInTheDocument()
      expect(screen.getByText('task-grandchild')).toBeInTheDocument()
    })

    const childRow = screen.getByText('task-child').closest('button')
    const grandchildRow = screen.getByText('task-grandchild').closest('button')
    expect(childRow).toHaveStyle({ paddingLeft: '24px' })
    expect(grandchildRow).toHaveStyle({ paddingLeft: '34px' })
  })

  it('preserves task nesting for active-only runs/flat payloads from backend', async () => {
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 3,
        last_activity: '2026-02-22T19:34:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-root',
        status: 'completed',
        last_activity: '2026-02-22T19:31:00Z',
      },
      {
        id: 'task-selected',
        status: 'completed',
        last_activity: '2026-02-22T19:33:30Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-root',
          run_id: 'run-root',
          message_id: 'MSG-root',
        },
      },
      {
        id: 'task-grandchild',
        status: 'completed',
        last_activity: '2026-02-22T19:32:30Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-selected',
          run_id: 'run-selected-old',
          message_id: 'MSG-selected',
        },
      },
    ]
    mockRuns = [
      {
        id: 'run-root',
        task_id: 'task-root',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:30:00Z',
        end_time: '2026-02-22T19:30:30Z',
      },
      {
        id: 'run-selected-old',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:31:00Z',
        end_time: '2026-02-22T19:31:30Z',
        parent_run_id: 'run-root',
      },
      {
        id: 'run-grandchild',
        task_id: 'task-grandchild',
        agent: 'gemini',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:32:00Z',
        end_time: '2026-02-22T19:32:30Z',
        parent_run_id: 'run-selected-old',
      },
      {
        id: 'run-selected-new',
        task_id: 'task-selected',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T19:33:00Z',
        end_time: '2026-02-22T19:33:30Z',
        previous_run_id: 'run-selected-old',
      },
    ]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId="task-selected"
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    await waitFor(() => {
      expect(screen.getByText('task-root')).toBeInTheDocument()
      expect(screen.getByText('task-selected')).toBeInTheDocument()
      expect(screen.getByText('task-grandchild')).toBeInTheDocument()
    })

    const selectedTaskRow = screen.getByText('task-selected').closest('button')
    const grandchildTaskRow = screen.getByText('task-grandchild').closest('button')
    expect(selectedTaskRow).toHaveStyle({ paddingLeft: '24px' })
    expect(grandchildTaskRow).toHaveStyle({ paddingLeft: '34px' })
  })

  it('keeps terminal parent branches visible when descendants are still active', async () => {
    const user = userEvent.setup()
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 2,
        last_activity: '2026-02-22T21:02:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-root-terminal',
        status: 'completed',
        last_activity: '2026-02-22T21:00:00Z',
      },
      {
        id: 'task-child-active',
        status: 'running',
        last_activity: '2026-02-22T21:02:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-root-terminal',
          run_id: 'run-root-terminal',
          message_id: 'MSG-root-terminal',
        },
      },
    ]
    mockRuns = [
      {
        id: 'run-root-terminal',
        task_id: 'task-root-terminal',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T20:58:00Z',
        end_time: '2026-02-22T21:00:00Z',
      },
      {
        id: 'run-child-active',
        task_id: 'task-child-active',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:01:00Z',
        parent_run_id: 'run-root-terminal',
      },
    ]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    expect(screen.getByText('task-root-terminal')).toBeInTheDocument()
    expect(screen.queryByTestId('tree-terminal-summary-toggle')).not.toBeInTheDocument()

    expect(screen.queryByText('task-child-active')).not.toBeInTheDocument()
    await user.click(screen.getByLabelText('Expand'))
    expect(screen.getByText('task-child-active')).toBeInTheDocument()
  })

  it('renders threaded task hierarchy when runs lack parent_run_id links', () => {
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 3,
        last_activity: '2026-02-22T21:02:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-thread-root',
        status: 'running',
        last_activity: '2026-02-22T21:00:00Z',
      },
      {
        id: 'task-thread-child',
        status: 'running',
        last_activity: '2026-02-22T21:01:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-root',
          run_id: 'run-thread-root',
          message_id: 'MSG-thread-root',
        },
      },
      {
        id: 'task-thread-grandchild',
        status: 'running',
        last_activity: '2026-02-22T21:02:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-child',
          run_id: 'run-thread-child',
          message_id: 'MSG-thread-child',
        },
      },
    ]
    mockRuns = [
      {
        id: 'run-thread-root',
        task_id: 'task-thread-root',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:58:00Z',
      },
      {
        id: 'run-thread-child',
        task_id: 'task-thread-child',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:59:00Z',
      },
      {
        id: 'run-thread-grandchild',
        task_id: 'task-thread-grandchild',
        agent: 'gemini',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
      },
    ]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    expect(screen.getByText('task-thread-root')).toBeInTheDocument()
    expect(screen.getByText('task-thread-child')).toBeInTheDocument()
    expect(screen.getByText('task-thread-grandchild')).toBeInTheDocument()
    expect(screen.queryByText('run-thread-root')).not.toBeInTheDocument()
    expect(screen.queryByText('run-thread-child')).not.toBeInTheDocument()
    expect(screen.queryByText('run-thread-grandchild')).not.toBeInTheDocument()
  })

  it('keeps threaded task nesting when run parent edges point to a conflicting task', () => {
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 3,
        last_activity: '2026-02-22T21:02:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-thread-root',
        status: 'running',
        last_activity: '2026-02-22T21:00:00Z',
      },
      {
        id: 'task-thread-child',
        status: 'running',
        last_activity: '2026-02-22T21:01:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-root',
          run_id: 'run-thread-root',
          message_id: 'MSG-thread-root',
        },
      },
      {
        id: 'task-noise-parent',
        status: 'running',
        last_activity: '2026-02-22T21:02:00Z',
      },
    ]
    mockRuns = [
      {
        id: 'run-thread-root',
        task_id: 'task-thread-root',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:58:00Z',
      },
      {
        id: 'run-noise-parent',
        task_id: 'task-noise-parent',
        agent: 'claude',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:59:00Z',
      },
      {
        id: 'run-thread-child',
        task_id: 'task-thread-child',
        agent: 'gemini',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T21:00:00Z',
        parent_run_id: 'run-noise-parent',
      },
    ]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    const rootRow = screen.getByText('task-thread-root').closest('button')
    const childRow = screen.getByText('task-thread-child').closest('button')
    expect(rootRow).toBeInTheDocument()
    expect(childRow).toBeInTheDocument()
    expect(rootRow).toHaveStyle({ paddingLeft: '14px' })
    expect(childRow).toHaveStyle({ paddingLeft: '24px' })
  })

  it('renders threaded queued tasks nested before child runs are created', async () => {
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 3,
        last_activity: '2026-02-22T21:02:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-thread-root',
        status: 'running',
        last_activity: '2026-02-22T21:00:00Z',
      },
      {
        id: 'task-thread-child',
        status: 'queued',
        last_activity: '2026-02-22T21:01:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-root',
          run_id: 'run-thread-root',
          message_id: 'MSG-thread-root',
        },
      },
      {
        id: 'task-thread-grandchild',
        status: 'queued',
        last_activity: '2026-02-22T21:02:00Z',
        thread_parent: {
          project_id: 'conductor-loop',
          task_id: 'task-thread-child',
          run_id: 'run-thread-child',
          message_id: 'MSG-thread-child',
        },
      },
    ]
    mockRuns = [
      {
        id: 'run-thread-root',
        task_id: 'task-thread-root',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T20:58:00Z',
      },
    ]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId="task-thread-grandchild"
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    await waitFor(() => {
      expect(screen.getByText('task-thread-root')).toBeInTheDocument()
      expect(screen.getByText('task-thread-child')).toBeInTheDocument()
      expect(screen.getByText('task-thread-grandchild')).toBeInTheDocument()
    })

    const childTaskRow = screen.getByText('task-thread-child').closest('button')
    const grandchildTaskRow = screen.getByText('task-thread-grandchild').closest('button')
    expect(childTaskRow).toHaveStyle({ paddingLeft: '24px' })
    expect(grandchildTaskRow).toHaveStyle({ paddingLeft: '34px' })
  })

  it('collapses completed and failed tasks under a summary row by default', () => {
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 3,
        last_activity: '2026-02-22T17:20:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-20260222-170000-active-flow',
        status: 'running',
        last_activity: '2026-02-22T17:02:00Z',
      },
      {
        id: 'task-20260222-170100-done-flow',
        status: 'completed',
        last_activity: '2026-02-22T17:10:00Z',
      },
      {
        id: 'task-20260222-170200-broken-flow',
        status: 'failed',
        last_activity: '2026-02-22T17:20:00Z',
      },
    ]
    mockRuns = [
      {
        id: 'run-active',
        task_id: 'task-20260222-170000-active-flow',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T17:00:00Z',
      },
      {
        id: 'run-completed',
        task_id: 'task-20260222-170100-done-flow',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T17:05:00Z',
        end_time: '2026-02-22T17:10:00Z',
      },
      {
        id: 'run-failed',
        task_id: 'task-20260222-170200-broken-flow',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T17:15:00Z',
        end_time: '2026-02-22T17:20:00Z',
      },
    ]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    const summaryToggle = screen.getByTestId('tree-terminal-summary-toggle')
    const summaryText = screen.getByText('... and 2 more tasks (1 completed, 1 failed)')
    const activeTask = screen.getByText('active-flow')
    expect(summaryToggle).toHaveAttribute('aria-expanded', 'false')
    expect(summaryText).toBeInTheDocument()
    expect(activeTask).toBeInTheDocument()
    expect((activeTask.compareDocumentPosition(summaryText) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0).toBe(true)
    expect(screen.queryByText('done-flow')).not.toBeInTheDocument()
    expect(screen.queryByText('broken-flow')).not.toBeInTheDocument()
  })

  it('keeps terminal tasks visible when all project tasks are completed or failed', () => {
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 2,
        last_activity: '2026-02-22T17:20:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-20260222-170100-done-flow',
        status: 'completed',
        last_activity: '2026-02-22T17:10:00Z',
      },
      {
        id: 'task-20260222-170200-broken-flow',
        status: 'failed',
        last_activity: '2026-02-22T17:20:00Z',
      },
    ]
    mockRuns = [
      {
        id: 'run-completed',
        task_id: 'task-20260222-170100-done-flow',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T17:05:00Z',
        end_time: '2026-02-22T17:10:00Z',
      },
      {
        id: 'run-failed',
        task_id: 'task-20260222-170200-broken-flow',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T17:15:00Z',
        end_time: '2026-02-22T17:20:00Z',
      },
    ]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    const summaryToggle = screen.getByTestId('tree-terminal-summary-toggle')
    expect(summaryToggle).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('...done-flow')).toBeInTheDocument()
    expect(screen.getByText('...broken-flow')).toBeInTheDocument()
  })

  it('toggles terminal task visibility and keeps state across live task updates', async () => {
    const user = userEvent.setup()
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 3,
        last_activity: '2026-02-22T17:20:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-20260222-170000-active-flow',
        status: 'running',
        last_activity: '2026-02-22T17:02:00Z',
      },
      {
        id: 'task-20260222-170100-done-flow',
        status: 'completed',
        last_activity: '2026-02-22T17:10:00Z',
      },
      {
        id: 'task-20260222-170200-broken-flow',
        status: 'failed',
        last_activity: '2026-02-22T17:20:00Z',
      },
    ]
    mockRuns = [
      {
        id: 'run-active',
        task_id: 'task-20260222-170000-active-flow',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T17:00:00Z',
      },
      {
        id: 'run-completed',
        task_id: 'task-20260222-170100-done-flow',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T17:05:00Z',
        end_time: '2026-02-22T17:10:00Z',
      },
      {
        id: 'run-failed',
        task_id: 'task-20260222-170200-broken-flow',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T17:15:00Z',
        end_time: '2026-02-22T17:20:00Z',
      },
    ]

    const view = render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    const summaryToggle = screen.getByTestId('tree-terminal-summary-toggle')
    expect(summaryToggle).toHaveAttribute('aria-expanded', 'false')

    await user.click(summaryToggle)
    expect(summaryToggle).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('...done-flow')).toBeInTheDocument()
    expect(screen.getByText('...broken-flow')).toBeInTheDocument()

    mockTasks = [
      ...mockTasks,
      {
        id: 'task-20260222-170300-later-done',
        status: 'completed',
        last_activity: '2026-02-22T17:30:00Z',
      },
    ]
    mockRuns = [
      ...mockRuns,
      {
        id: 'run-completed-2',
        task_id: 'task-20260222-170300-later-done',
        agent: 'codex',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T17:25:00Z',
        end_time: '2026-02-22T17:30:00Z',
      },
    ]

    view.rerender(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    const summaryAfterUpdate = screen.getByTestId('tree-terminal-summary-toggle')
    expect(summaryAfterUpdate).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('... and 3 more tasks (2 completed, 1 failed)')).toBeInTheDocument()
    expect(screen.getByText('...later-done')).toBeInTheDocument()

    summaryAfterUpdate.focus()
    await user.keyboard('{Enter}')
    expect(summaryAfterUpdate).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('done-flow')).not.toBeInTheDocument()
    expect(screen.queryByText('broken-flow')).not.toBeInTheDocument()
    expect(screen.queryByText('later-done')).not.toBeInTheDocument()
  })

  it('keeps selected collapsed task under summary section and shows full ID on hover', async () => {
    const user = userEvent.setup()
    const onSelectTask = vi.fn()
    mockProjects = [
      {
        id: 'conductor-loop',
        task_count: 3,
        last_activity: '2026-02-22T17:20:00Z',
      },
    ]
    mockTasks = [
      {
        id: 'task-20260222-170000-active-flow',
        status: 'running',
        last_activity: '2026-02-22T17:02:00Z',
      },
      {
        id: 'task-20260222-170100-done-flow',
        status: 'completed',
        last_activity: '2026-02-22T17:10:00Z',
      },
      {
        id: 'task-20260222-170200-broken-flow',
        status: 'failed',
        last_activity: '2026-02-22T17:20:00Z',
      },
    ]
    mockRuns = [
      {
        id: 'run-active',
        task_id: 'task-20260222-170000-active-flow',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T17:00:00Z',
      },
      {
        id: 'run-completed',
        task_id: 'task-20260222-170100-done-flow',
        agent: 'claude',
        status: 'completed',
        exit_code: 0,
        start_time: '2026-02-22T17:05:00Z',
        end_time: '2026-02-22T17:10:00Z',
      },
      {
        id: 'run-failed',
        task_id: 'task-20260222-170200-broken-flow',
        agent: 'gemini',
        status: 'failed',
        exit_code: 1,
        start_time: '2026-02-22T17:15:00Z',
        end_time: '2026-02-22T17:20:00Z',
      },
    ]

    const view = render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={onSelectTask}
        onSelectRun={() => undefined}
      />
    )

    const summaryToggle = screen.getByTestId('tree-terminal-summary-toggle')
    await user.click(summaryToggle)
    const collapsedDoneLabel = screen.getByText('...done-flow')
    expect(collapsedDoneLabel).toHaveAttribute('title', 'task-20260222-170100-done-flow')

    await user.click(collapsedDoneLabel.closest('button')!)
    expect(onSelectTask).toHaveBeenCalledWith('conductor-loop', 'task-20260222-170100-done-flow')

    view.rerender(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId="task-20260222-170100-done-flow"
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={onSelectTask}
        onSelectRun={() => undefined}
      />
    )

    const summaryAfterSelect = screen.getByTestId('tree-terminal-summary-toggle')
    const activeTaskRow = screen.getByText('active-flow').closest('button')!
    const collapsedTaskRow = screen.getByText('...done-flow').closest('button')!
    expect((activeTaskRow.compareDocumentPosition(summaryAfterSelect) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0).toBe(true)
    expect((summaryAfterSelect.compareDocumentPosition(collapsedTaskRow) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0).toBe(true)
  })

  it('shows success feedback and focuses the created task', async () => {
    const user = userEvent.setup()
    const onSelectTask = vi.fn()
    mockProjects = [{ id: 'conductor-loop', task_count: 0, last_activity: '2026-02-22T17:00:00Z' }]

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={onSelectTask}
        onSelectRun={() => undefined}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Create new task' }))
    await user.type(screen.getByPlaceholderText('Describe what the agent should do...'), 'Make submit visible now')
    await user.click(screen.getByRole('button', { name: 'Create Task' }))

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledTimes(1)
    })
    await waitFor(() => {
      expect(onSelectTask).toHaveBeenCalledWith('conductor-loop', 'task-test')
    })
    expect(screen.getByTestId('create-task-feedback')).toHaveTextContent(
      'Task task-test created and focused in the tree.'
    )
    expect(screen.getByText('task-test')).toBeInTheDocument()
  })

  it('shows failure feedback and keeps form data when submit fails', async () => {
    const user = userEvent.setup()
    const onSelectTask = vi.fn()
    mockProjects = [{ id: 'conductor-loop', task_count: 0, last_activity: '2026-02-22T17:00:00Z' }]
    mockMutateAsync.mockRejectedValueOnce(new Error('api 500: boom'))

    render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={onSelectTask}
        onSelectRun={() => undefined}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Create new task' }))
    const promptInput = screen.getByPlaceholderText('Describe what the agent should do...')
    await user.type(promptInput, 'Do not lose this draft')
    await user.click(screen.getByRole('button', { name: 'Create Task' }))

    expect(await screen.findByText('api 500: boom')).toBeInTheDocument()
    expect(screen.getByTestId('create-task-feedback')).toHaveTextContent('Task creation failed: api 500: boom')
    expect(promptInput).toHaveValue('Do not lose this draft')
    expect(onSelectTask).not.toHaveBeenCalled()
  })

  it('auto-expands collapsed project rows when selected task is hidden', async () => {
    const user = userEvent.setup()
    mockProjects = [{ id: 'conductor-loop', task_count: 1, last_activity: '2026-02-22T17:00:00Z' }]
    mockTasks = [{ id: 'task-auto-expand-target', status: 'running', last_activity: '2026-02-22T17:01:00Z' }]
    mockRuns = [
      {
        id: 'run-auto-expand',
        task_id: 'task-auto-expand-target',
        agent: 'codex',
        status: 'running',
        exit_code: -1,
        start_time: '2026-02-22T17:00:00Z',
      },
    ]

    const view = render(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    await user.click(screen.getAllByLabelText('Collapse')[0])
    expect(screen.queryByText('task-auto-expand-target')).not.toBeInTheDocument()

    view.rerender(
      <TreePanel
        projectId="conductor-loop"
        selectedProjectId="conductor-loop"
        selectedTaskId="task-auto-expand-target"
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    await waitFor(() => {
      expect(screen.getByText('task-auto-expand-target')).toBeInTheDocument()
    })
  })

  it('creates a new project and selects it immediately', async () => {
    const user = userEvent.setup()
    const onSelectProject = vi.fn()
    mockProjects = [{ id: 'existing-project', task_count: 0, last_activity: '2026-02-22T17:00:00Z' }]
    mockCreateProjectMutateAsync.mockResolvedValueOnce({ id: 'new-project' })

    render(
      <TreePanel
        projectId="existing-project"
        selectedProjectId="existing-project"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={onSelectProject}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Create new project' }))
    await user.type(screen.getByLabelText(/Project ID \/ Name/i), 'new-project')
    await user.type(screen.getByLabelText(/Home \/ Work Folder/i), '/tmp/new-project')
    await user.click(screen.getByRole('button', { name: 'Create Project' }))

    await waitFor(() => {
      expect(mockCreateProjectMutateAsync).toHaveBeenCalledWith({
        project_id: 'new-project',
        project_root: '/tmp/new-project',
      })
    })
    expect(onSelectProject).toHaveBeenCalledWith('new-project')
    expect(screen.getByTestId('create-task-feedback')).toHaveTextContent(
      'Project new-project created and selected.'
    )
  })

  it('blocks duplicate project IDs before submit', async () => {
    const user = userEvent.setup()
    mockProjects = [{ id: 'existing-project', task_count: 0, last_activity: '2026-02-22T17:00:00Z' }]

    render(
      <TreePanel
        projectId="existing-project"
        selectedProjectId="existing-project"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Create new project' }))
    await user.type(screen.getByLabelText(/Project ID \/ Name/i), 'existing-project')
    await user.type(screen.getByLabelText(/Home \/ Work Folder/i), '/tmp/new-project')
    await user.click(screen.getByRole('button', { name: 'Create Project' }))

    expect(await screen.findByText('Project "existing-project" already exists')).toBeInTheDocument()
    expect(mockCreateProjectMutateAsync).not.toHaveBeenCalled()
  })

  it('validates project home folder path before submit', async () => {
    const user = userEvent.setup()
    mockProjects = [{ id: 'existing-project', task_count: 0, last_activity: '2026-02-22T17:00:00Z' }]

    render(
      <TreePanel
        projectId="existing-project"
        selectedProjectId="existing-project"
        selectedTaskId={undefined}
        selectedRunId={undefined}
        onSelectProject={() => undefined}
        onSelectTask={() => undefined}
        onSelectRun={() => undefined}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Create new project' }))
    await user.type(screen.getByLabelText(/Project ID \/ Name/i), 'fresh-project')
    await user.type(screen.getByLabelText(/Home \/ Work Folder/i), 'relative/path')
    await user.click(screen.getByRole('button', { name: 'Create Project' }))

    expect(await screen.findByText('Folder path must be absolute (or use ~/...)')).toBeInTheDocument()
    expect(mockCreateProjectMutateAsync).not.toHaveBeenCalled()
  })
})
