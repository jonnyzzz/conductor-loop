import clsx from 'clsx'
import type { RunSummary } from '../types'

interface RunNode {
  run: RunSummary
  children: RunNode[]
}

interface RestartHint {
  state: string
  title: string
  detail: string
}

function buildTree(runs: RunSummary[]): RunNode[] {
  const map = new Map<string, RunNode>()
  const roots: RunNode[] = []

  runs.forEach((run) => {
    map.set(run.id, { run, children: [] })
  })

  runs.forEach((run) => {
    const node = map.get(run.id)
    if (!node) {
      return
    }
    if (run.parent_run_id && map.has(run.parent_run_id)) {
      map.get(run.parent_run_id)?.children.push(node)
    } else {
      roots.push(node)
    }
  })

  const sortNodes = (nodes: RunNode[]) => {
    nodes.sort((a, b) => a.run.start_time.localeCompare(b.run.start_time))
    nodes.forEach((child) => sortNodes(child.children))
  }
  sortNodes(roots)

  return roots
}

export function RunTree({
  runs,
  selectedRunId,
  onSelect,
  restartHint,
}: {
  runs: RunSummary[]
  selectedRunId?: string
  onSelect: (runId: string) => void
  restartHint?: RestartHint | null
}) {
  const tree = buildTree(runs)

  const renderNode = (node: RunNode, depth: number) => (
    <div key={node.run.id} className="run-tree-node" style={{ paddingLeft: `${depth * 16}px` }}>
      <button
        type="button"
        onClick={() => onSelect(node.run.id)}
        className={clsx('run-tree-button', selectedRunId === node.run.id && 'run-tree-active')}
      >
        <span className={clsx('status-dot', `status-${node.run.status}`)} />
        <span className="run-tree-title">{node.run.id}</span>
        <span className="run-tree-meta">
          {node.run.agent} · {node.run.status}
        </span>
      </button>
      {node.run.previous_run_id && (
        <div className="run-tree-previous">Restarted from: {node.run.previous_run_id}</div>
      )}
      {node.children.map((child) => renderNode(child, depth + 1))}
    </div>
  )

  if (!runs.length) {
    return <div className="empty-state">No runs yet.</div>
  }

  return (
    <div className="run-tree">
      {restartHint && (
        <div className={clsx('run-tree-restart-hint', restartHint.state)}>
          <span className="run-tree-restart-title">{restartHint.title}</span>
          <span>{restartHint.detail}</span>
        </div>
      )}
      {tree.map((node) => renderNode(node, 0))}
    </div>
  )
}
