// 聊天 API 服务 - 处理与后端的通信

export interface ChatRequest {
  user_id: string
  message: string
  send_to_wecom?: boolean
  stream?: boolean
}

// 发送聊天消息（流式响应）
export async function sendChatMessage(
  req: ChatRequest,
  onChunk: (content: string) => void,
  onComplete: (content: string) => void,
  onError: (error: string) => void
): Promise<void> {
  const res = await fetch('/api/wecom/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...req, stream: true, send_to_wecom: false }),
  })

  if (!res.ok) {
    onError('请求失败')
    return
  }

  const reader = res.body?.getReader()
  if (!reader) {
    onError('无法读取响应')
    return
  }

  const decoder = new TextDecoder()
  let buffer = ''
  let currentEvent = ''
  let fullContent = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })

    const lines = buffer.split('\n')
    buffer = lines.pop() || ''

    for (const line of lines) {
      if (line.startsWith('event:')) {
        currentEvent = line.slice(6).trim()
        continue
      }
      if (line.startsWith('data:')) {
        try {
          const data = JSON.parse(line.slice(5))

          if (currentEvent === 'error') {
            onError(data.error || '未知错误')
            return
          } else if (currentEvent === 'message') {
            if (data.content) {
              fullContent += data.content
              onChunk(data.content)
            }
          } else if (currentEvent === 'done') {
            if (data.content && !fullContent) {
              fullContent = data.content
              onComplete(data.content)
            } else {
              onComplete(fullContent)
            }
            return
          }
        } catch {
          // JSON 解析失败，忽略
        }
      }
    }
  }

  onComplete(fullContent)
}
