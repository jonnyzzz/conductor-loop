import { AgentOutputRenderer } from './AgentOutputRenderer'

interface Props {
  fileName: string
  content: string
}

export function FileViewer({ fileName, content }: Props) {
  if (fileName === 'agent-stdout.txt') {
    return <AgentOutputRenderer content={content} />
  }
  return <pre className="file-viewer">{content}</pre>
}
