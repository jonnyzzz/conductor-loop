import { useEffect, useRef } from 'react'

type EventHandler = (event: MessageEvent) => void

interface UseSSEOptions {
  onOpen?: (event: Event) => void
  onError?: (event: Event) => void
  events?: Record<string, EventHandler>
  withCredentials?: boolean
  reconnectBaseDelayMs?: number
  reconnectMaxDelayMs?: number
}

export function useSSE(url?: string, options: UseSSEOptions = {}) {
  const reconnectAttempt = useRef(0)
  const reconnectTimer = useRef<number | null>(null)
  const { onOpen, onError, events, withCredentials, reconnectBaseDelayMs, reconnectMaxDelayMs } = options

  useEffect(() => {
    if (!url) {
      return undefined
    }

    let closed = false
    let source: EventSource | null = null

    const connect = () => {
      if (closed) {
        return
      }
      source = new EventSource(url, { withCredentials })
      if (onOpen) {
        source.addEventListener('open', onOpen)
      }
      if (events) {
        Object.entries(events).forEach(([event, handler]) => {
          source?.addEventListener(event, handler)
        })
      }
      source.onerror = (event) => {
        onError?.(event)
        if (closed) {
          return
        }
        source?.close()
        reconnectAttempt.current += 1
        const base = reconnectBaseDelayMs ?? 500
        const max = reconnectMaxDelayMs ?? 8000
        const delay = Math.min(max, base * 2 ** reconnectAttempt.current)
        reconnectTimer.current = window.setTimeout(connect, delay)
      }
    }

    connect()

    return () => {
      closed = true
      if (reconnectTimer.current) {
        window.clearTimeout(reconnectTimer.current)
      }
      source?.close()
    }
  }, [
    url,
    events,
    onError,
    onOpen,
    reconnectBaseDelayMs,
    reconnectMaxDelayMs,
    withCredentials,
  ])
}
