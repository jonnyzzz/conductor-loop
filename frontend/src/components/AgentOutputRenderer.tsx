import { useMemo } from 'react'

interface Props {
  content: string
}

export function AgentOutputRenderer({ content }: Props) {
  const lines = useMemo(() => content.split('\n').filter(Boolean), [content])
  return (
    <div className="agent-output">
      {lines.map((line, i) => (
        <AgentOutputLine key={i} raw={line} />
      ))}
    </div>
  )
}

function truncate(s: string, max: number): string {
  if (s.length <= max) return s
  return s.slice(0, max) + '…'
}

function AgentOutputLine({ raw }: { raw: string }) {
  try {
    const obj = JSON.parse(raw) as Record<string, unknown>
    return <>{renderObj(obj)}</>
  } catch {
    return <span className="ao-plain">{raw}</span>
  }
}

function renderObj(obj: Record<string, unknown>): React.ReactNode {
  const type = (obj.type as string) ?? ''
  const subtype = (obj.subtype as string) ?? ''

  if (type === 'system' && subtype === 'init') {
    const tools = Array.isArray(obj.tools) ? obj.tools.length : '?'
    return (
      <span className="ao-init">
        [Init] model={String(obj.model ?? '')}  tools={String(tools)}  cwd={String(obj.cwd ?? '')}
      </span>
    )
  }

  if (type === 'system' && subtype === 'task_started') {
    return (
      <span className="ao-subtask">
        → Subtask: {String(obj.description ?? '')}
      </span>
    )
  }

  if (type === 'system' && subtype === 'task_notification') {
    return <span className="ao-subtask">← Subtask done</span>
  }

  if (type === 'result') {
    const resultText = truncate(String(obj.result ?? ''), 300)
    if (subtype === 'success') {
      const durationS = obj.duration_ms != null ? (Number(obj.duration_ms) / 1000).toFixed(1) : '?'
      return (
        <span className="ao-result ao-result-success">
          ✅ Done in {durationS}s  ({String(obj.num_turns ?? '?')} turns): {resultText}
        </span>
      )
    }
    return (
      <span className="ao-result ao-result-error">
        ❌ Error: {resultText}
      </span>
    )
  }

  if (type === 'assistant') {
    const msg = obj.message as { content?: unknown[] } | undefined
    const blocks = Array.isArray(msg?.content) ? msg.content : []
    if (blocks.length === 0) {
      return <span className="ao-plain">{JSON.stringify(obj)}</span>
    }
    return (
      <>
        {blocks.map((block, i) => {
          const b = block as Record<string, unknown>
          const btype = b.type as string
          if (btype === 'text') {
            return (
              <span key={i} className="ao-text">
                {String(b.text ?? '')}
              </span>
            )
          }
          if (btype === 'thinking') {
            return (
              <details key={i} className="ao-thinking">
                <summary>💭 Thinking…</summary>
                <pre>{String(b.thinking ?? '')}</pre>
              </details>
            )
          }
          if (btype === 'tool_use') {
            const inputStr = truncate(JSON.stringify(b.input), 60)
            return (
              <span key={i} className="ao-tool">
                🔧 {String(b.name ?? '')}({inputStr})
              </span>
            )
          }
          return null
        })}
      </>
    )
  }

  if (type === 'user') {
    const msg = obj.message as { content?: unknown[] } | undefined
    const blocks = Array.isArray(msg?.content) ? msg.content : []
    return (
      <>
        {blocks.map((block, i) => {
          const b = block as Record<string, unknown>
          if (b.type === 'tool_result') {
            const content = b.content
            const text = typeof content === 'string'
              ? content
              : Array.isArray(content)
              ? (content as Array<{ text?: string }>).map((c) => c.text ?? '').join('')
              : JSON.stringify(content)
            return (
              <span key={i} className="ao-tool-result">
                ↳ {truncate(text, 120)}
              </span>
            )
          }
          return null
        })}
      </>
    )
  }

  return (
    <details className="ao-unknown">
      <summary className="ao-unknown-summary">[{type || 'unknown'}]</summary>
      <pre>{JSON.stringify(obj, null, 2)}</pre>
    </details>
  )
}
