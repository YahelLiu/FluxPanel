// 消息类型定义

export interface Message {
  role: 'user' | 'assistant'
  content: string
  time: Date
  streaming?: boolean
}

export interface ReminderPayload {
  type: 'reminder'
  content: string
  time: string
}

export interface SSEEvent {
  event: string
  data: {
    content?: string
    error?: string
  }
}
