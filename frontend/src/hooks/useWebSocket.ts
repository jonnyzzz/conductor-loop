import { useEffect } from 'react'

export function useWebSocket(url?: string) {
  useEffect(() => {
    if (!url) {
      return undefined
    }

    const socket = new WebSocket(url)

    socket.onopen = () => {
      // Placeholder for future use.
    }

    socket.onerror = () => {
      // Placeholder for future use.
    }

    return () => {
      socket.close()
    }
  }, [url])
}
