import { useEffect, useState } from 'react'
import { Send, Save, Key } from 'lucide-react'

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
    <div className="space-y-4">
      {/* 说明 */}
      <p className="text-sm text-gray-600 dark:text-gray-400">
        配置 LLM 用于微信 iLink AI 助手，支持 Qwen（阿里云百炼）和 OpenAI API
      </p>

      {/* 配置表单 */}
      <div className="border rounded-lg p-4 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 space-y-4">
        <div className="flex items-center gap-2 mb-2">
          <Key className="h-4 w-4 text-primary" />
          <span className="font-medium">LLM 配置</span>
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

        <div className="flex items-center gap-4 pt-2">
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={llmConfig.enabled}
              onChange={e => setLLMConfig({ ...llmConfig, enabled: e.target.checked })}
              className="rounded"
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
      </div>

      {/* 测试 */}
      <div className="border rounded-lg p-4 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 space-y-3">
        <span className="font-medium">测试 LLM 连接</span>
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
          <div className="p-3 bg-gray-50 dark:bg-gray-700 rounded-md">
            <p className="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{llmTestResponse}</p>
          </div>
        )}
      </div>
    </div>
  )
}
