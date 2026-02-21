import { useEffect, useState } from 'react'

type EventHandler = (event: MessageEvent) => void

interface UseSSEOptions {
  onOpen?: (event: Event) => void
  onError?: (event: Event) => void
  events?: Record<string, EventHandler>
  withCredentials?: boolean
}

export type SSEConnectionState = 'disabled' | 'connecting' | 'open' | 'reconnecting' | 'error'

interface UseSSEResult {
  state: SSEConnectionState
  errorCount: number
}

export function useSSE(url?: string, options: UseSSEOptions = {}): UseSSEResult {
  const [state, setState] = useState<SSEConnectionState>(() => (url ? 'connecting' : 'disabled'))
  const [errorCount, setErrorCount] = useState(0)
  const { onOpen, onError, events, withCredentials } = options

  useEffect(() => {
    if (!url) {
      setState('disabled')
      setErrorCount(0)
      return undefined
    }

    setState('connecting')
    setErrorCount(0)

    const source = new EventSource(url, { withCredentials })
    let hasOpened = false

    const handleOpen = (event: Event) => {
      hasOpened = true
      setState('open')
      onOpen?.(event)
    }

    const handleError = (event: Event) => {
      onError?.(event)
      setErrorCount((prev) => prev + 1)
      setState(hasOpened ? 'reconnecting' : 'error')
    }

    source.addEventListener('open', handleOpen)
    source.addEventListener('error', handleError)

    if (events) {
      Object.entries(events).forEach(([event, handler]) => {
        source.addEventListener(event, handler as EventListener)
      })
    }

    return () => {
      source.removeEventListener('open', handleOpen)
      source.removeEventListener('error', handleError)
      if (events) {
        Object.entries(events).forEach(([event, handler]) => {
          source.removeEventListener(event, handler as EventListener)
        })
      }
      source.close()
    }
  }, [
    url,
    events,
    onError,
    onOpen,
    withCredentials,
  ])

  return { state, errorCount }
}
