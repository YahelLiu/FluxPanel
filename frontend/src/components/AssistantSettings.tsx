import { useEffect, useState } from 'react'
import { Save, Send, Key } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'

interface LLMConfig {
  id?: number
  provider: string
  api_key: string
  base_url: string
  model: string
  enabled: boolean
}

export function AssistantSettings() {
  const [llmConfig, setLLMConfig] = useState<LLMConfig>({
    provider: 'qwen',
    api_key: '',
    base_url: '',
    model: 'qwen-plus',
    enabled: false,
  })
  const [saving, setSaving] = useState(false)
  const [llmTestMessage, setLlmTestMessage] = useState('你好，请简单介绍一下你自己')
  const [llmTestResponse, setLlmTestResponse] = useState('')
  const [llmTesting, setLlmTesting] = useState(false)

  useEffect(() => {
    fetchConfigs()
  }, [])

  const fetchConfigs = async () => {
    try {
      const llmRes = await fetch('/api/assistant/llm')
      const llmData = await llmRes.json()
      setLLMConfig(llmData)
    } catch (error) {
      console.error('Failed to fetch configs:', error)
    }
  }

  const saveLLMConfig = async () => {
    setSaving(true)
    try {
      const res = await fetch('/api/assistant/llm', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(llmConfig),
      })
      if (res.ok) {
        alert('LLM 配置保存成功')
      } else {
        const data = await res.json()
        alert(`保存失败: ${data.error}`)
      }
    } catch (error) {
      alert('保存失败')
    }
    setSaving(false)
  }

  const testLLM = async () => {
    if (!llmTestMessage) {
      alert('请输入测试消息')
      return
    }
    setLlmTesting(true)
    setLlmTestResponse('')
    try {
      const res = await fetch('/api/assistant/llm/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: llmTestMessage }),
      })
      const data = await res.json()
      if (data.success) {
        setLlmTestResponse(data.response)
      } else {
        setLlmTestResponse(`错误: ${data.error}`)
      }
    } catch (error) {
      setLlmTestResponse('测试失败，请检查配置')
    }
    setLlmTesting(false)
  }

  return (
    <div className="space-y-6">
      {/* 说明 */}
      <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
        <h4 className="font-medium text-blue-800 dark:text-blue-300 mb-2">🤖 AI 助手说明</h4>
        <ul className="text-sm text-blue-700 dark:text-blue-400 space-y-1">
          <li>• 通过微信 iLink 与 AI 助手对话</li>
          <li>• 支持 Qwen（阿里云百炼）和 OpenAI API</li>
          <li>• 可以记忆用户偏好、管理待办事项、设置提醒</li>
          <li>• 需要先配置 LLM 才能使用</li>
        </ul>
      </div>

      {/* LLM 配置 */}
      <Card>
        <CardContent className="p-4 bg-white dark:bg-gray-800 space-y-4">
          <div className="flex items-center gap-2">
            <Key className="h-5 w-5 text-primary" />
            <h3 className="font-medium text-gray-900 dark:text-gray-100">LLM 配置</h3>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">提供商</label>
              <select
                value={llmConfig.provider}
                onChange={e => setLLMConfig({ ...llmConfig, provider: e.target.value })}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
              >
                <option value="qwen">阿里云百炼 (Qwen)</option>
                <option value="openai">OpenAI</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">模型</label>
              <input
                type="text"
                value={llmConfig.model}
                onChange={e => setLLMConfig({ ...llmConfig, model: e.target.value })}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder={llmConfig.provider === 'qwen' ? 'qwen-plus' : 'gpt-4o-mini'}
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">API Key *</label>
            <input
              type="password"
              value={llmConfig.api_key}
              onChange={e => setLLMConfig({ ...llmConfig, api_key: e.target.value })}
              className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
              placeholder="输入 API Key"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">Base URL (可选)</label>
            <input
              type="text"
              value={llmConfig.base_url}
              onChange={e => setLLMConfig({ ...llmConfig, base_url: e.target.value })}
              className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
              placeholder={llmConfig.provider === 'qwen' ? 'https://dashscope.aliyuncs.com/compatible-mode/v1' : 'https://api.openai.com/v1'}
            />
          </div>

          <div className="flex items-center gap-4">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={llmConfig.enabled}
                onChange={e => setLLMConfig({ ...llmConfig, enabled: e.target.checked })}
                className="rounded border-gray-300"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">启用</span>
            </label>
            <button
              onClick={saveLLMConfig}
              disabled={saving}
              className="flex items-center gap-1 px-4 py-2 bg-primary text-white rounded-md hover:bg-primary/90 disabled:opacity-50"
            >
              <Save className="h-4 w-4" />
              {saving ? '保存中...' : '保存配置'}
            </button>
          </div>

          {/* LLM 测试 */}
          <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
            <h4 className="text-sm font-medium mb-2 text-gray-700 dark:text-gray-300">测试 LLM 连接</h4>
            <div className="flex gap-2">
              <input
                type="text"
                value={llmTestMessage}
                onChange={e => setLlmTestMessage(e.target.value)}
                className="flex-1 px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder="输入测试消息"
              />
              <button
                onClick={testLLM}
                disabled={llmTesting}
                className="flex items-center gap-1 px-4 py-2 bg-purple-500 text-white rounded-md hover:bg-purple-600 disabled:opacity-50"
              >
                <Send className="h-4 w-4" />
                {llmTesting ? '测试中...' : '测试'}
              </button>
            </div>
            {llmTestResponse && (
              <div className="mt-2 p-3 bg-gray-100 dark:bg-gray-700 rounded-md">
                <p className="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{llmTestResponse}</p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
