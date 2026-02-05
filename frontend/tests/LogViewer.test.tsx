import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LogViewer } from '../src/components/LogViewer'

describe('LogViewer', () => {
  it('filters by stream', async () => {
    const user = userEvent.setup()
    render(
      <LogViewer
        streamUrl={undefined}
        initialLines={[
          { id: '1', run_id: 'run-1', stream: 'stdout', line: 'hello stdout' },
          { id: '2', run_id: 'run-1', stream: 'stderr', line: 'hello stderr' },
        ]}
      />
    )

    expect(screen.getByText(/hello stdout/i)).toBeInTheDocument()
    expect(screen.getByText(/hello stderr/i)).toBeInTheDocument()

    const stderrButtons = screen.getAllByRole('button', { name: /stderr/i })
    await user.click(stderrButtons[0])

    expect(screen.queryByText(/hello stdout/i)).not.toBeInTheDocument()
    expect(screen.getByText(/hello stderr/i)).toBeInTheDocument()
  })
})
