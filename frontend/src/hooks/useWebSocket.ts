import { useEffect, useRef, useState } from 'react'

interface Event {
  id: number
  client_id: string
  event_type: string
  data: Record<string, unknown>
  status: string
  created_at: string
}

export function useWebSocket(url: string) {
  const [isConnected, setIsConnected] = useState(false)
  const [lastEvent, setLastEvent] = useState<Event | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => {
      setIsConnected(true)
      console.log('WebSocket connected')
    }

    ws.onclose = () => {
      setIsConnected(false)
      console.log('WebSocket disconnected')
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'event') {
          setLastEvent(data.event)
        }
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e)
      }
    }

    return () => {
      ws.close()
    }
  }, [url])

  return { isConnected, lastEvent }
}
