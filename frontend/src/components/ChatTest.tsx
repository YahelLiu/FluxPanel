// 聊天测试组件 - 只负责 UI 渲染，逻辑由 Hooks 管理

import { useState, useRef, useCallback } from 'react'
import { Send, Trash2, Bot, User, Loader2 } from 'lucide-react'
import { useChat } from '../hooks/useChat'
import { useChatWebSocket } from '../hooks/useChatWebSocket'

export function ChatTest() {
  const [userId] = useState('test-user')
  const [input, setInput] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)

  // 聊天逻辑
  const { messages, loading, sendMessage, clearMessages, addReminderMessage } = useChat(userId)

  // WebSocket 连接（用于接收提醒）
  const { connected } = useChatWebSocket(
    userId,
    useCallback((content: string) => {
      addReminderMessage(content)
    }, [addReminderMessage])
  )

  // 发送消息
  const handleSend = () => {
    if (!input.trim() || loading) return
    sendMessage(input)
    setInput('')
  }

  // 快捷操作
  const quickActions = [
    { label: '你好', text: '你好' },
    { label: '记住我', text: '记住我喜欢简洁的回答' },
    { label: '查看记忆', text: '我有哪些记忆' },
    { label: '设置提醒', text: '1分钟后提醒我测试' },
    { label: '查看提醒', text: '我有哪些提醒' },
  ]

  return (
    <div className="flex flex-col h-[600px] bg-white dark:bg-gray-900 rounded-lg border dark:border-gray-700">
      {/* 头部 */}
      <header className="flex items-center justify-between px-4 py-3 border-b dark:border-gray-700">
        <div className="flex items-center gap-2">
          <Bot className="h-5 w-5 text-primary" />
          <span className="font-medium text-gray-900 dark:text-gray-100">AI 助手测试</span>
          <span className="text-xs text-gray-500 dark:text-gray-400">({userId})</span>
          {connected && (
            <span className="w-2 h-2 bg-green-500 rounded-full" title="WebSocket 已连接" />
          )}
        </div>
        <button
          onClick={clearMessages}
          className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-800 rounded text-gray-500"
          title="清空对话"
        >
          <Trash2 className="h-4 w-4" />
        </button>
      </header>

      {/* 快捷操作 */}
      <nav className="flex gap-2 px-4 py-2 border-b dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 overflow-x-auto">
        {quickActions.map((action, i) => (
          <button
            key={i}
            onClick={() => setInput(action.text)}
            className="px-3 py-1 text-xs bg-white dark:bg-gray-700 border dark:border-gray-600 rounded-full hover:bg-gray-100 dark:hover:bg-gray-600 whitespace-nowrap text-gray-700 dark:text-gray-300"
          >
            {action.label}
          </button>
        ))}
      </nav>

      {/* 消息列表 */}
      <main className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.length === 0 && (
          <div className="text-center text-gray-500 dark:text-gray-400 py-10">
            <Bot className="h-12 w-12 mx-auto mb-2 opacity-50" />
            <p>开始对话测试 AI 助手</p>
            <p className="text-sm mt-1">可以测试：聊天、记忆、提醒</p>
          </div>
        )}
        {messages.map((msg, i) => (
          <MessageItem key={i} message={msg} />
        ))}
        <div ref={messagesEndRef} />
      </main>

      {/* 输入框 */}
      <footer className="p-4 border-t dark:border-gray-700">
        <div className="flex gap-2">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && (e.preventDefault(), handleSend())}
            placeholder="输入消息测试..."
            className="flex-1 px-4 py-2 border rounded-full bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600 focus:outline-none focus:ring-2 focus:ring-primary/50"
            disabled={loading}
          />
          <button
            onClick={handleSend}
            disabled={loading || !input.trim()}
            className="px-4 py-2 bg-primary text-white rounded-full hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
          </button>
        </div>
      </footer>
    </div>
  )
}

// 消息项组件
function MessageItem({ message }: { message: { role: string; content: string; time: Date; streaming?: boolean } }) {
  const isUser = message.role === 'user'

  return (
    <div className={`flex gap-3 ${isUser ? 'justify-end' : ''}`}>
      {!isUser && (
        <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
          <Bot className="h-4 w-4 text-primary" />
        </div>
      )}
      <div className={`max-w-[70%] ${isUser ? 'order-first' : ''}`}>
        <div
          className={`px-4 py-2 rounded-2xl ${
            isUser
              ? 'bg-primary text-white rounded-br-sm'
              : 'bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-gray-100 rounded-bl-sm'
          }`}
        >
          <p className="text-sm whitespace-pre-wrap">
            {message.content}
            {message.streaming && <span className="inline-block w-2 h-4 ml-1 bg-gray-500 animate-pulse" />}
          </p>
        </div>
        <p className="text-xs text-gray-400 mt-1 px-1">{message.time.toLocaleTimeString()}</p>
      </div>
      {isUser && (
        <div className="w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center flex-shrink-0">
          <User className="h-4 w-4 text-gray-600 dark:text-gray-300" />
        </div>
      )}
    </div>
  )
}
