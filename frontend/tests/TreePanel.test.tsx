import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

let mockProjects: Array<{ id: string; task_count: number; last_activity: string }> = []
let mockTasks: Array<{ id: string; status: string; last_activity: string }> = []
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

vi.mock('../src/hooks/useAPI', () => ({
  useProjects: () => ({ data: mockProjects }),
  useTasks: () => ({ data: mockTasks }),
  useProjectRunsFlat: () => ({ data: mockRuns }),
  useHomeDirs: () => ({ data: { dirs: [] } }),
  useStartTask: () => ({ mutateAsync: vi.fn(), isPending: false }),
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
  it('shows explicit restart and duration badges on merged single-run task rows', () => {
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

    expect(screen.getByText('restart:1')).toBeInTheDocument()
    expect(screen.getByText('dur:2m15s')).toBeInTheDocument()
    expect(screen.queryByText(/run-002/)).not.toBeInTheDocument()
  })
})
