// WebSocket 服务 - 独立模块，处理连接、心跳、重连

export interface WebSocketMessage {
  type: string
  content?: string
  [key: string]: unknown
}

export type MessageHandler = (data: WebSocketMessage) => void
export type ConnectionHandler = (connected: boolean) => void

export class WebSocketService {
  private ws: WebSocket | null = null
  private url: string
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private shouldReconnect = true
  private reconnectDelay = 3000
  private messageHandlers: Set<MessageHandler> = new Set()
  private connectionHandlers: Set<ConnectionHandler> = new Set()

  constructor(url: string) {
    this.url = url
  }

  // 连接
  connect(): void {
    this.ws = new WebSocket(this.url)

    this.ws.onopen = () => {
      this.notifyConnection(true)
      console.log('WebSocket connected')
    }

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as WebSocketMessage
        this.handleMessage(data)
      } catch (e) {
        console.error('WebSocket message parse error:', e)
      }
    }

    this.ws.onclose = () => {
      this.notifyConnection(false)
      console.log('WebSocket disconnected')
      this.scheduleReconnect()
    }

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }
  }

  // 断开连接
  disconnect(): void {
    this.shouldReconnect = false
    this.clearReconnectTimer()
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  // 发送消息
  send(data: WebSocketMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data))
    }
  }

  // 订阅消息
  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler)
    return () => this.messageHandlers.delete(handler)
  }

  // 订阅连接状态
  onConnection(handler: ConnectionHandler): () => void {
    this.connectionHandlers.add(handler)
    return () => this.connectionHandlers.delete(handler)
  }

  // 获取连接状态
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  // 处理消息
  private handleMessage(data: WebSocketMessage): void {
    // 响应心跳
    if (data.type === 'ping') {
      this.send({ type: 'pong' })
      return
    }

    // 通知所有订阅者
    this.messageHandlers.forEach(handler => handler(data))
  }

  // 通知连接状态变化
  private notifyConnection(connected: boolean): void {
    this.connectionHandlers.forEach(handler => handler(connected))
  }

  // 安排重连
  private scheduleReconnect(): void {
    if (!this.shouldReconnect) return

    this.clearReconnectTimer()
    this.reconnectTimer = setTimeout(() => {
      console.log('WebSocket reconnecting...')
      this.connect()
    }, this.reconnectDelay)
  }

  // 清除重连定时器
  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }
}

// 创建 WebSocket 服务实例
export function createWebSocketService(userId: string): WebSocketService {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${protocol}//${window.location.host}/ws?user_id=${userId}`
  return new WebSocketService(url)
}
