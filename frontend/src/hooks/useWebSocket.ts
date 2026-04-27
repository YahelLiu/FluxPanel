import { useEffect, useRef, useState } from 'react'

interface Event {
  id: number
  client_id: string
  event_type: string
  data: Record<string, unknown>
  status: string
  created_at: string
}

type ConnectionStatus = 'connecting' | 'connected' | 'reconnecting' | 'failed'

export function useWebSocket(url: string) {
  const [isConnected, setIsConnected] = useState(false)
  const [lastEvent, setLastEvent] = useState<Event | null>(null)
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('connecting')
  const [reconnectAttempts, setReconnectAttempts] = useState(0)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<number | null>(null)

  useEffect(() => {
    let closedByCleanup = false
    let attempts = 0
    const maxReconnectAttempts = 5

    const connect = () => {
      if (closedByCleanup) return

      setConnectionStatus(attempts === 0 ? 'connecting' : 'reconnecting')
      setReconnectAttempts(attempts)

      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        attempts = 0
        setIsConnected(true)
        setConnectionStatus('connected')
        setReconnectAttempts(0)
      }

      ws.onclose = () => {
        setIsConnected(false)
        if (closedByCleanup) return

        if (attempts >= maxReconnectAttempts) {
          setConnectionStatus('failed')
          setReconnectAttempts(attempts)
          return
        }

        attempts += 1
        setConnectionStatus('reconnecting')
        setReconnectAttempts(attempts)
        const delay = Math.min(1000 * 2 ** (attempts - 1), 10000)
        reconnectTimerRef.current = window.setTimeout(connect, delay)
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        ws.close()
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
    }

    connect()

    return () => {
      closedByCleanup = true
      if (reconnectTimerRef.current !== null) {
        window.clearTimeout(reconnectTimerRef.current)
      }
      wsRef.current?.close()
    }
  }, [url])

  return { isConnected, lastEvent, connectionStatus, reconnectAttempts }
}
