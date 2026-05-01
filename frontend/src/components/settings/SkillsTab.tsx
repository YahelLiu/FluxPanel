import { useEffect, useState } from 'react'
import { Upload, Link, Trash2, Settings, Check, RefreshCw, X, Plus } from 'lucide-react'

interface Skill {
  id: string
  name: string
  description: string
  type: string
  source: string
  enabled: boolean
  trusted: boolean
  allowed_tools: string[]
  triggers: string[]
  created_at: string
  updated_at: string
}

interface Tool {
  name: string
  description: string
  parameters: Record<string, { type: string; description: string; required: boolean }>
}

export function SkillsTab() {
  const [skills, setSkills] = useState<Skill[]>([])
  const [tools, setTools] = useState<Tool[]>([])
  const [loading, setLoading] = useState(true)
  const [uploading, setUploading] = useState(false)
  const [installUrl, setInstallUrl] = useState('')
  const [installing, setInstalling] = useState(false)
  const [configSkill, setConfigSkill] = useState<Skill | null>(null)
  const [selectedTools, setSelectedTools] = useState<string[]>([])

  const fetchSkills = async () => {
    try {
      const res = await fetch('/api/skills')
      const data = await res.json()
      setSkills(data.skills || [])
    } catch (error) {
      console.error('Failed to fetch skills:', error)
    }
  }

  const fetchTools = async () => {
    try {
      const res = await fetch('/api/skills/tools')
      const data = await res.json()
      setTools(data.tools || [])
    } catch (error) {
      console.error('Failed to fetch tools:', error)
    }
  }

  useEffect(() => {
    const init = async () => {
      await Promise.all([fetchSkills(), fetchTools()])
      setLoading(false)
    }
    init()
  }, [])

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setUploading(true)
    try {
      const formData = new FormData()
      formData.append('file', file)

      const res = await fetch('/api/skills/upload', {
        method: 'POST',
        body: formData,
      })
      const data = await res.json()

      if (data.success) {
        alert(`技能 "${data.skill.name}" 安装成功！`)
        await fetchSkills()
      } else {
        alert(data.error || '安装失败')
      }
    } catch (error) {
      alert('上传失败')
    }
    setUploading(false)
    e.target.value = ''
  }

  const handleInstallFromUrl = async () => {
    if (!installUrl.trim()) return

    setInstalling(true)
    try {
      const res = await fetch('/api/skills/install', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url: installUrl }),
      })
      const data = await res.json()

      if (data.success) {
        alert(`技能 "${data.skill.name}" 安装成功！`)
        setInstallUrl('')
        await fetchSkills()
      } else {
        alert(data.error || '安装失败')
      }
    } catch (error) {
      alert('安装失败')
    }
    setInstalling(false)
  }

  const toggleSkill = async (skillId: string, enabled: boolean) => {
    try {
      const res = await fetch(`/api/skills/${skillId}/enable`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled }),
      })
      const data = await res.json()

      if (data.success) {
        setSkills(prev =>
          prev.map(s => (s.id === skillId ? { ...s, enabled } : s))
        )
      }
    } catch (error) {
      console.error('Failed to toggle skill:', error)
    }
  }

  const deleteSkill = async (skillId: string) => {
    if (!confirm('确定要删除这个技能吗？')) return

    try {
      const res = await fetch(`/api/skills/${skillId}`, {
        method: 'DELETE',
      })
      const data = await res.json()

      if (data.success) {
        setSkills(prev => prev.filter(s => s.id !== skillId))
      }
    } catch (error) {
      console.error('Failed to delete skill:', error)
    }
  }

  const openConfig = (skill: Skill) => {
    setConfigSkill(skill)
    setSelectedTools(skill.allowed_tools || [])
  }

  const saveToolConfig = async () => {
    if (!configSkill) return

    try {
      const res = await fetch(`/api/skills/${configSkill.id}/tools`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tools: selectedTools }),
      })
      const data = await res.json()

      if (data.success) {
        setSkills(prev =>
          prev.map(s =>
            s.id === configSkill.id ? { ...s, allowed_tools: selectedTools } : s
          )
        )
        setConfigSkill(null)
      }
    } catch (error) {
      console.error('Failed to save tool config:', error)
    }
  }

  const toggleTool = (toolName: string) => {
    setSelectedTools(prev =>
      prev.includes(toolName)
        ? prev.filter(t => t !== toolName)
        : [...prev, toolName]
    )
  }

  if (loading) {
    return <div className="text-center py-10 text-gray-500">加载中...</div>
  }

  return (
    <div className="space-y-4">
      {/* 安装区域 */}
      <div className="flex justify-between items-center">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          安装和管理 AI 技能，扩展助手能力
        </p>
        <div className="flex gap-2">
          <label className="flex items-center gap-1 px-3 py-1.5 bg-blue-500 text-white rounded-md cursor-pointer hover:bg-blue-600 text-sm">
            <Upload className="h-4 w-4" />
            {uploading ? '上传中...' : '上传 ZIP'}
            <input
              type="file"
              accept=".zip"
              onChange={handleUpload}
              disabled={uploading}
              className="hidden"
            />
          </label>
        </div>
      </div>

      {/* URL 安装 */}
      <div className="flex gap-2">
        <div className="flex-1 relative">
          <Link className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <input
            type="url"
            placeholder="输入技能包 URL 安装"
            value={installUrl}
            onChange={e => setInstallUrl(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border rounded-md dark:bg-gray-800 dark:border-gray-600"
          />
        </div>
        <button
          onClick={handleInstallFromUrl}
          disabled={installing || !installUrl.trim()}
          className="px-4 py-2 bg-green-500 text-white rounded-md hover:bg-green-600 disabled:opacity-50 flex items-center gap-2"
        >
          {installing ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
          {installing ? '安装中' : '安装'}
        </button>
      </div>

      {/* 已安装技能列表 */}
      {skills.length === 0 ? (
        <div className="text-center text-gray-500 dark:text-gray-400 py-10 border-2 border-dashed rounded-lg border-gray-300 dark:border-gray-600">
          暂无已安装的技能
        </div>
      ) : (
        <div className="space-y-2">
          {skills.map(skill => (
            <div
              key={skill.id}
              className={`flex items-center justify-between p-3 border rounded-lg ${
                skill.enabled
                  ? 'bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700'
                  : 'bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700 opacity-60'
              }`}
            >
              <div className="flex items-center gap-3">
                <div className={`w-2.5 h-2.5 rounded-full ${skill.enabled ? 'bg-green-500' : 'bg-gray-300'}`} />
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-900 dark:text-gray-100">{skill.name}</span>
                    <span className="text-xs px-2 py-0.5 bg-gray-100 dark:bg-gray-700 rounded">{skill.type}</span>
                    {skill.trusted && (
                      <span className="text-xs px-2 py-0.5 bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300 rounded">可信</span>
                    )}
                  </div>
                  <p className="text-sm text-gray-500 mt-0.5">{skill.description}</p>
                </div>
              </div>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => toggleSkill(skill.id, !skill.enabled)}
                  className={`px-2 py-1 text-xs rounded ${
                    skill.enabled
                      ? 'bg-yellow-100 text-yellow-700 hover:bg-yellow-200'
                      : 'bg-green-100 text-green-700 hover:bg-green-200'
                  }`}
                >
                  {skill.enabled ? '禁用' : '启用'}
                </button>
                <button onClick={() => openConfig(skill)} className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-gray-500" title="配置">
                  <Settings className="h-4 w-4" />
                </button>
                <button onClick={() => deleteSkill(skill.id)} className="p-1.5 hover:bg-red-50 dark:hover:bg-red-900/20 text-red-500 rounded" title="删除">
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 工具配置模态框 */}
      {configSkill && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60] p-4">
          <div className="bg-white dark:bg-gray-800 rounded-lg w-full max-w-md">
            <div className="flex items-center justify-between p-4 border-b dark:border-gray-700">
              <h3 className="font-medium">配置工具权限 - {configSkill.name}</h3>
              <button onClick={() => setConfigSkill(null)} className="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded">
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="p-4 max-h-[400px] overflow-auto">
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">选择此技能允许使用的工具：</p>
              <div className="space-y-2">
                {tools.map(tool => (
                  <label
                    key={tool.name}
                    className="flex items-start gap-3 p-3 border rounded-lg cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                  >
                    <input
                      type="checkbox"
                      checked={selectedTools.includes(tool.name)}
                      onChange={() => toggleTool(tool.name)}
                      className="mt-1"
                    />
                    <div className="flex-1">
                      <div className="font-medium text-sm">{tool.name}</div>
                      <div className="text-xs text-gray-500">{tool.description}</div>
                    </div>
                    {selectedTools.includes(tool.name) && <Check className="h-4 w-4 text-green-500" />}
                  </label>
                ))}
              </div>
            </div>
            <div className="flex justify-end gap-2 p-4 border-t dark:border-gray-700">
              <button onClick={() => setConfigSkill(null)} className="px-4 py-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md">
                取消
              </button>
              <button onClick={saveToolConfig} className="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600">
                保存
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
