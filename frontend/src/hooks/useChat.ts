// useChat Hook - 管理聊天状态和逻辑

import { useState, useCallback } from 'react'
import type { Message } from '../types/chat'
import { sendChatMessage } from '../services/chat'

export function useChat(userId: string) {
  const [messages, setMessages] = useState<Message[]>([])
  const [loading, setLoading] = useState(false)

  // 发送消息
  const sendMessage = useCallback(async (content: string) => {
    if (!content.trim() || loading) return

    const userMessage = content.trim()

    // 添加用户消息
    setMessages(prev => [...prev, {
      role: 'user',
      content: userMessage,
      time: new Date()
    }])
    setLoading(true)

    // 添加空的助手消息用于流式输出
    setMessages(prev => [...prev, {
      role: 'assistant',
      content: '',
      time: new Date(),
      streaming: true
    }])

    try {
      await sendChatMessage(
        { user_id: userId, message: userMessage },
        // onChunk - 流式内容
        (chunk) => {
          setMessages(prev => {
            const newMessages = [...prev]
            const lastMsg = newMessages[newMessages.length - 1]
            if (lastMsg.role === 'assistant') {
              lastMsg.content += chunk
            }
            return newMessages
          })
        },
        // onComplete - 完成
        (fullContent) => {
          setMessages(prev => {
            const newMessages = [...prev]
            const lastMsg = newMessages[newMessages.length - 1]
            if (lastMsg.role === 'assistant') {
              if (!lastMsg.content && fullContent) {
                lastMsg.content = fullContent
              }
              lastMsg.streaming = false
            }
            return newMessages
          })
        },
        // onError - 错误
        (error) => {
          setMessages(prev => {
            const newMessages = [...prev]
            const lastMsg = newMessages[newMessages.length - 1]
            if (lastMsg.role === 'assistant') {
              lastMsg.content = `错误: ${error}`
              lastMsg.streaming = false
            }
            return newMessages
          })
        }
      )
    } catch {
      setMessages(prev => {
        const newMessages = [...prev]
        const lastMsg = newMessages[newMessages.length - 1]
        if (lastMsg.role === 'assistant') {
          lastMsg.content = '请求失败，请检查后端服务是否启动'
          lastMsg.streaming = false
        }
        return newMessages
      })
    }

    setLoading(false)
  }, [userId, loading])

  // 清空消息
  const clearMessages = useCallback(() => {
    setMessages([])
  }, [])

  // 添加提醒消息（来自 WebSocket）
  const addReminderMessage = useCallback((content: string) => {
    setMessages(prev => [...prev, {
      role: 'assistant',
      content,
      time: new Date()
    }])
  }, [])

  return {
    messages,
    loading,
    sendMessage,
    clearMessages,
    addReminderMessage
  }
}
