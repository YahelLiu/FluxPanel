import { useEffect, useState, useRef } from 'react'
import { Send, Trash2, Bot, User, Loader2 } from 'lucide-react'

interface Message {
  role: 'user' | 'assistant'
  content: string
  time: Date
}

export function ChatTest() {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [userId] = useState('test-user-' + Date.now().toString(36))
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  const sendMessage = async () => {
    if (!input.trim() || loading) return

    const userMessage = input.trim()
    setInput('')

    // 添加用户消息
    setMessages(prev => [...prev, { role: 'user', content: userMessage, time: new Date() }])
    setLoading(true)

    try {
      const res = await fetch('/api/wecom/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          user_id: userId,
          message: userMessage,
          send_to_wecom: false,
        }),
      })
      const data = await res.json()

      // 添加助手回复
      setMessages(prev => [...prev, {
        role: 'assistant',
        content: data.success ? data.response : `错误: ${data.error}`,
        time: new Date()
      }])
    } catch (error) {
      setMessages(prev => [...prev, {
        role: 'assistant',
        content: '请求失败，请检查后端服务是否启动',
        time: new Date()
      }])
    }
    setLoading(false)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  const clearChat = () => {
    setMessages([])
  }

  const quickActions = [
    { label: '你好', text: '你好' },
    { label: '记住我', text: '记住我喜欢简洁的回答' },
    { label: '添加Todo', text: '帮我加个todo，明天整理文档' },
    { label: '查看Todo', text: '我有哪些todo' },
    { label: '设置提醒', text: '1分钟后提醒我测试' },
    { label: '查看提醒', text: '我有哪些提醒' },
  ]

  return (
    <div className="flex flex-col h-[600px] bg-white dark:bg-gray-900 rounded-lg border dark:border-gray-700">
      {/* 头部 */}
      <div className="flex items-center justify-between px-4 py-3 border-b dark:border-gray-700">
        <div className="flex items-center gap-2">
          <Bot className="h-5 w-5 text-primary" />
          <span className="font-medium text-gray-900 dark:text-gray-100">AI 助手测试</span>
          <span className="text-xs text-gray-500 dark:text-gray-400">({userId})</span>
        </div>
        <button
          onClick={clearChat}
          className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-800 rounded text-gray-500"
          title="清空对话"
        >
          <Trash2 className="h-4 w-4" />
        </button>
      </div>

      {/* 快捷操作 */}
      <div className="flex gap-2 px-4 py-2 border-b dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 overflow-x-auto">
        {quickActions.map((action, i) => (
          <button
            key={i}
            onClick={() => setInput(action.text)}
            className="px-3 py-1 text-xs bg-white dark:bg-gray-700 border dark:border-gray-600 rounded-full hover:bg-gray-100 dark:hover:bg-gray-600 whitespace-nowrap text-gray-700 dark:text-gray-300"
          >
            {action.label}
          </button>
        ))}
      </div>

      {/* 消息列表 */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.length === 0 && (
          <div className="text-center text-gray-500 dark:text-gray-400 py-10">
            <Bot className="h-12 w-12 mx-auto mb-2 opacity-50" />
            <p>开始对话测试 AI 助手</p>
            <p className="text-sm mt-1">可以测试：聊天、记忆、Todo、提醒</p>
          </div>
        )}
        {messages.map((msg, i) => (
          <div key={i} className={`flex gap-3 ${msg.role === 'user' ? 'justify-end' : ''}`}>
            {msg.role === 'assistant' && (
              <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                <Bot className="h-4 w-4 text-primary" />
              </div>
            )}
            <div className={`max-w-[70%] ${msg.role === 'user' ? 'order-first' : ''}`}>
              <div className={`px-4 py-2 rounded-2xl ${
                msg.role === 'user'
                  ? 'bg-primary text-white rounded-br-sm'
                  : 'bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-gray-100 rounded-bl-sm'
              }`}>
                <p className="text-sm whitespace-pre-wrap">{msg.content}</p>
              </div>
              <p className="text-xs text-gray-400 mt-1 px-1">
                {msg.time.toLocaleTimeString()}
              </p>
            </div>
            {msg.role === 'user' && (
              <div className="w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center flex-shrink-0">
                <User className="h-4 w-4 text-gray-600 dark:text-gray-300" />
              </div>
            )}
          </div>
        ))}
        {loading && (
          <div className="flex gap-3">
            <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
              <Bot className="h-4 w-4 text-primary" />
            </div>
            <div className="px-4 py-2 bg-gray-100 dark:bg-gray-800 rounded-2xl rounded-bl-sm">
              <Loader2 className="h-4 w-4 animate-spin text-gray-500" />
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* 输入框 */}
      <div className="p-4 border-t dark:border-gray-700">
        <div className="flex gap-2">
          <input
            type="text"
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="输入消息测试..."
            className="flex-1 px-4 py-2 border rounded-full bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600 focus:outline-none focus:ring-2 focus:ring-primary/50"
            disabled={loading}
          />
          <button
            onClick={sendMessage}
            disabled={loading || !input.trim()}
            className="px-4 py-2 bg-primary text-white rounded-full hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Send className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  )
}
