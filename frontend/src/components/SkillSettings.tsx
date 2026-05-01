import { useEffect, useState } from 'react'
import { X, Upload, Link, Trash2, Settings, Check, Package, RefreshCw } from 'lucide-react'

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

interface SkillSettingsProps {
  onClose: () => void
}

export function SkillSettings({ onClose }: SkillSettingsProps) {
  const [skills, setSkills] = useState<Skill[]>([])
  const [tools, setTools] = useState<Tool[]>([])
  const [loading, setLoading] = useState(true)
  const [uploading, setUploading] = useState(false)
  const [installUrl, setInstallUrl] = useState('')
  const [installing, setInstalling] = useState(false)
  const [configSkill, setConfigSkill] = useState<Skill | null>(null)
  const [selectedTools, setSelectedTools] = useState<string[]>([])

  // 获取 skills 列表
  const fetchSkills = async () => {
    try {
      const res = await fetch('/api/skills')
      const data = await res.json()
      setSkills(data.skills || [])
    } catch (error) {
      console.error('Failed to fetch skills:', error)
    }
  }

  // 获取可用工具列表
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

  // 上传 zip 文件
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

  // 从 URL 安装
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

  // 启用/禁用 skill
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

  // 删除 skill
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

  // 打开配置
  const openConfig = (skill: Skill) => {
    setConfigSkill(skill)
    setSelectedTools(skill.allowed_tools || [])
  }

  // 保存工具配置
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

  // 切换工具选择
  const toggleTool = (toolName: string) => {
    setSelectedTools(prev =>
      prev.includes(toolName)
        ? prev.filter(t => t !== toolName)
        : [...prev, toolName]
    )
  }

  if (loading) {
    return (
      <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
        <div className="bg-white dark:bg-gray-800 rounded-lg p-6">
          加载中...
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-gray-800 rounded-lg w-full max-w-3xl max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b dark:border-gray-700">
          <h2 className="text-xl font-bold flex items-center gap-2">
            <Package className="h-5 w-5" />
            技能管理
          </h2>
          <button
            onClick={onClose}
            className="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-4 space-y-4">
          {/* 安装区域 */}
          <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4 space-y-3">
            <h3 className="font-medium text-sm text-gray-600 dark:text-gray-400">安装新技能</h3>
            <div className="flex flex-wrap gap-3">
              {/* 上传 ZIP */}
              <label className="flex items-center gap-2 px-4 py-2 bg-blue-500 text-white rounded-md cursor-pointer hover:bg-blue-600 transition-colors">
                <Upload className="h-4 w-4" />
                <span>{uploading ? '上传中...' : '上传 ZIP'}</span>
                <input
                  type="file"
                  accept=".zip"
                  onChange={handleUpload}
                  disabled={uploading}
                  className="hidden"
                />
              </label>

              {/* URL 安装 */}
              <div className="flex-1 flex gap-2 min-w-[300px]">
                <div className="flex-1 relative">
                  <Link className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input
                    type="url"
                    placeholder="输入技能包 URL"
                    value={installUrl}
                    onChange={e => setInstallUrl(e.target.value)}
                    className="w-full pl-10 pr-4 py-2 border rounded-md dark:bg-gray-800 dark:border-gray-600"
                  />
                </div>
                <button
                  onClick={handleInstallFromUrl}
                  disabled={installing || !installUrl.trim()}
                  className="px-4 py-2 bg-green-500 text-white rounded-md hover:bg-green-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                >
                  {installing ? (
                    <>
                      <RefreshCw className="h-4 w-4 animate-spin" />
                      安装中
                    </>
                  ) : (
                    '安装'
                  )}
                </button>
              </div>
            </div>
          </div>

          {/* 已安装技能列表 */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="font-medium">已安装技能 ({skills.length})</h3>
              <button
                onClick={fetchSkills}
                className="text-sm text-blue-500 hover:text-blue-600 flex items-center gap-1"
              >
                <RefreshCw className="h-3 w-3" />
                刷新
              </button>
            </div>

            {skills.length === 0 ? (
              <div className="text-center py-10 text-gray-500">
                暂无已安装的技能
              </div>
            ) : (
              <div className="space-y-3">
                {skills.map(skill => (
                  <div
                    key={skill.id}
                    className={`border rounded-lg p-4 ${
                      skill.enabled
                        ? 'bg-white dark:bg-gray-800'
                        : 'bg-gray-50 dark:bg-gray-900 opacity-60'
                    }`}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <h4 className="font-medium">{skill.name}</h4>
                          <span className="text-xs px-2 py-0.5 bg-gray-100 dark:bg-gray-700 rounded">
                            {skill.type}
                          </span>
                          {skill.trusted && (
                            <span className="text-xs px-2 py-0.5 bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300 rounded">
                              可信
                            </span>
                          )}
                        </div>
                        <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                          {skill.description}
                        </p>

                        {/* 触发词 */}
                        {skill.triggers && skill.triggers.length > 0 && (
                          <div className="flex flex-wrap gap-1 mt-2">
                            {skill.triggers.slice(0, 5).map(trigger => (
                              <span
                                key={trigger}
                                className="text-xs px-2 py-0.5 bg-blue-50 text-blue-600 dark:bg-blue-900 dark:text-blue-300 rounded"
                              >
                                {trigger}
                              </span>
                            ))}
                            {skill.triggers.length > 5 && (
                              <span className="text-xs text-gray-500">
                                +{skill.triggers.length - 5} 更多
                              </span>
                            )}
                          </div>
                        )}

                        {/* 允许的工具 */}
                        {skill.allowed_tools && skill.allowed_tools.length > 0 && (
                          <div className="text-xs text-gray-500 mt-2">
                            工具: {skill.allowed_tools.join(', ')}
                          </div>
                        )}
                      </div>

                      {/* 操作按钮 */}
                      <div className="flex items-center gap-2 ml-4">
                        <button
                          onClick={() => toggleSkill(skill.id, !skill.enabled)}
                          className={`px-3 py-1 text-sm rounded transition-colors ${
                            skill.enabled
                              ? 'bg-yellow-100 text-yellow-700 hover:bg-yellow-200'
                              : 'bg-green-100 text-green-700 hover:bg-green-200'
                          }`}
                        >
                          {skill.enabled ? '禁用' : '启用'}
                        </button>
                        <button
                          onClick={() => openConfig(skill)}
                          className="p-1 text-gray-500 hover:text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded"
                          title="配置工具"
                        >
                          <Settings className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => deleteSkill(skill.id)}
                          className="p-1 text-gray-500 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/30 rounded"
                          title="删除"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="p-4 border-t dark:border-gray-700 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-gray-100 dark:bg-gray-700 rounded-md hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
          >
            关闭
          </button>
        </div>
      </div>

      {/* 工具配置模态框 */}
      {configSkill && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60] p-4">
          <div className="bg-white dark:bg-gray-800 rounded-lg w-full max-w-md">
            <div className="flex items-center justify-between p-4 border-b dark:border-gray-700">
              <h3 className="font-medium">配置工具权限 - {configSkill.name}</h3>
              <button
                onClick={() => setConfigSkill(null)}
                className="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="p-4 max-h-[400px] overflow-auto">
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                选择此技能允许使用的工具：
              </p>
              <div className="space-y-2">
                {tools.map(tool => (
                  <label
                    key={tool.name}
                    className="flex items-start gap-3 p-3 border rounded-lg cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
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
                    {selectedTools.includes(tool.name) && (
                      <Check className="h-4 w-4 text-green-500" />
                    )}
                  </label>
                ))}
              </div>
            </div>
            <div className="flex justify-end gap-2 p-4 border-t dark:border-gray-700">
              <button
                onClick={() => setConfigSkill(null)}
                className="px-4 py-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md"
              >
                取消
              </button>
              <button
                onClick={saveToolConfig}
                className="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600"
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
