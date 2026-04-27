// useChatWebSocket Hook - 专门用于聊天场景的 WebSocket 连接

import { useEffect, useState, useRef, useCallback } from 'react'
import { createWebSocketService, WebSocketService } from '../services/websocket'

export function useChatWebSocket(userId: string, onReminder: (content: string) => void) {
  const [connected, setConnected] = useState(false)
  const wsServiceRef = useRef<WebSocketService | null>(null)

  // 用 useCallback 包装，避免 useEffect 依赖变化
  const handleReminder = useCallback(onReminder, [onReminder])

  useEffect(() => {
    const wsService = createWebSocketService(userId)
    wsServiceRef.current = wsService

    const unsubConnection = wsService.onConnection(setConnected)
    const unsubMessage = wsService.onMessage((data) => {
      if (data.type === 'reminder' && data.content) {
        handleReminder(data.content)
      }
    })

    wsService.connect()

    return () => {
      unsubConnection()
      unsubMessage()
      wsService.disconnect()
    }
  }, [userId, handleReminder])

  return { connected }
}
