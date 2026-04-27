import { useEffect, useState } from 'react'
import { Save, Send, MessageSquare, Key, Globe } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'

interface LLMConfig {
  id?: number
  provider: string
  api_key: string
  base_url: string
  model: string
  enabled: boolean
}

interface WeComConfig {
  id?: number
  corp_id: string
  agent_id: string
  secret: string
  token: string
  encoding_aes_key: string
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
  const [wecomConfig, setWeComConfig] = useState<WeComConfig>({
    corp_id: '',
    agent_id: '',
    secret: '',
    token: '',
    encoding_aes_key: '',
    enabled: false,
  })
  const [saving, setSaving] = useState(false)
  const [testUserID, setTestUserID] = useState('')
  const [testMessage, setTestMessage] = useState('你好')
  const [testResponse, setTestResponse] = useState('')
  const [llmTestMessage, setLlmTestMessage] = useState('你好，请简单介绍一下你自己')
  const [llmTestResponse, setLlmTestResponse] = useState('')
  const [llmTesting, setLlmTesting] = useState(false)

  useEffect(() => {
    fetchConfigs()
  }, [])

  const fetchConfigs = async () => {
    try {
      const [llmRes, wecomRes] = await Promise.all([
        fetch('/api/assistant/llm'),
        fetch('/api/wecom/config'),
      ])
      const llmData = await llmRes.json()
      const wecomData = await wecomRes.json()
      setLLMConfig(llmData)
      setWeComConfig(wecomData)
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

  const saveWeComConfig = async () => {
    setSaving(true)
    try {
      const res = await fetch('/api/wecom/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(wecomConfig),
      })
      if (res.ok) {
        alert('企业微信配置保存成功')
      } else {
        const data = await res.json()
        alert(`保存失败: ${data.error}`)
      }
    } catch (error) {
      alert('保存失败')
    }
    setSaving(false)
  }

  const testChat = async () => {
    if (!testUserID || !testMessage) {
      alert('请填写用户ID和测试消息')
      return
    }
    try {
      const res = await fetch('/api/wecom/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          user_id: testUserID,
          message: testMessage,
          send_to_wecom: false,
        }),
      })
      const data = await res.json()
      if (data.success) {
        setTestResponse(data.response)
      } else {
        setTestResponse(`错误: ${data.error}`)
      }
    } catch (error) {
      setTestResponse('测试失败')
    }
  }

  const testSendToWeCom = async () => {
    if (!testUserID || !testMessage) {
      alert('请填写用户ID和测试消息')
      return
    }
    try {
      const res = await fetch('/api/wecom/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          user_id: testUserID,
          message: testMessage,
        }),
      })
      const data = await res.json()
      if (data.success) {
        alert('消息已发送到企业微信')
      } else {
        alert(`发送失败: ${data.error}`)
      }
    } catch (error) {
      alert('发送失败')
    }
  }

  return (
    <div className="space-y-6">
      {/* 说明 */}
      <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
        <h4 className="font-medium text-blue-800 dark:text-blue-300 mb-2">🤖 AI 助手说明</h4>
        <ul className="text-sm text-blue-700 dark:text-blue-400 space-y-1">
          <li>• 支持通过企业微信与 AI 助手对话</li>
          <li>• 支持 Qwen（阿里云百炼）和 OpenAI API</li>
          <li>• 可以记忆用户偏好、管理待办事项、设置提醒</li>
          <li>• 需要先配置 LLM 和企业微信才能使用</li>
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

      {/* 企业微信配置 */}
      <Card>
        <CardContent className="p-4 bg-white dark:bg-gray-800 space-y-4">
          <div className="flex items-center gap-2">
            <MessageSquare className="h-5 w-5 text-green-500" />
            <h3 className="font-medium text-gray-900 dark:text-gray-100">企业微信配置</h3>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">企业 ID (CorpID)</label>
              <input
                type="text"
                value={wecomConfig.corp_id}
                onChange={e => setWeComConfig({ ...wecomConfig, corp_id: e.target.value })}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder="在企业微信后台获取"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">应用 AgentID</label>
              <input
                type="text"
                value={wecomConfig.agent_id}
                onChange={e => setWeComConfig({ ...wecomConfig, agent_id: e.target.value })}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder="应用的 AgentID"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">应用 Secret</label>
            <input
              type="password"
              value={wecomConfig.secret}
              onChange={e => setWeComConfig({ ...wecomConfig, secret: e.target.value })}
              className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
              placeholder="应用的 Secret"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">Token (回调验证)</label>
              <input
                type="text"
                value={wecomConfig.token}
                onChange={e => setWeComConfig({ ...wecomConfig, token: e.target.value })}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder="可选"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">EncodingAESKey</label>
              <input
                type="text"
                value={wecomConfig.encoding_aes_key}
                onChange={e => setWeComConfig({ ...wecomConfig, encoding_aes_key: e.target.value })}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder="可选"
              />
            </div>
          </div>

          <div className="flex items-center gap-4">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={wecomConfig.enabled}
                onChange={e => setWeComConfig({ ...wecomConfig, enabled: e.target.checked })}
                className="rounded border-gray-300"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">启用</span>
            </label>
            <button
              onClick={saveWeComConfig}
              disabled={saving}
              className="flex items-center gap-1 px-4 py-2 bg-green-500 text-white rounded-md hover:bg-green-600 disabled:opacity-50"
            >
              <Save className="h-4 w-4" />
              {saving ? '保存中...' : '保存配置'}
            </button>
          </div>
        </CardContent>
      </Card>

      {/* 测试 */}
      <Card>
        <CardContent className="p-4 bg-white dark:bg-gray-800 space-y-4">
          <div className="flex items-center gap-2">
            <Globe className="h-5 w-5 text-purple-500" />
            <h3 className="font-medium text-gray-900 dark:text-gray-100">测试 AI 助手</h3>
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">用户 ID</label>
              <input
                type="text"
                value={testUserID}
                onChange={e => setTestUserID(e.target.value)}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder="企业微信用户ID"
              />
            </div>
            <div className="col-span-2">
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">测试消息</label>
              <input
                type="text"
                value={testMessage}
                onChange={e => setTestMessage(e.target.value)}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder="输入测试消息"
              />
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={testChat}
              className="flex items-center gap-1 px-4 py-2 bg-purple-500 text-white rounded-md hover:bg-purple-600"
            >
              <Send className="h-4 w-4" />
              测试对话
            </button>
            <button
              onClick={testSendToWeCom}
              className="flex items-center gap-1 px-4 py-2 bg-green-500 text-white rounded-md hover:bg-green-600"
            >
              <MessageSquare className="h-4 w-4" />
              发送到企业微信
            </button>
          </div>

          {testResponse && (
            <div className="p-3 bg-gray-100 dark:bg-gray-700 rounded-md">
              <p className="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{testResponse}</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
