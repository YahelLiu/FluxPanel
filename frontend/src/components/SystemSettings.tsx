import { useEffect, useState } from 'react'
import { Settings, X, Bell, CloudSun, Bot, Package } from 'lucide-react'
import { ChannelsTab, ChannelForm } from './settings'
import type { NotificationChannel } from './settings/ChannelsTab'
import { WeatherSettings } from './WeatherSettings'
import { AssistantSettings } from './AssistantSettings'
import { SkillsTab } from './settings/SkillsTab'

interface SystemSettingsProps {
  onClose: () => void
}

export function SystemSettings({ onClose }: SystemSettingsProps) {
  const [channels, setChannels] = useState<NotificationChannel[]>([])
  const [activeTab, setActiveTab] = useState<'channels' | 'weather' | 'ai' | 'skills'>('channels')
  const [editingChannel, setEditingChannel] = useState<NotificationChannel | null>(null)
  const [showChannelForm, setShowChannelForm] = useState(false)

  const fetchChannels = async () => {
    try {
      const res = await fetch('/api/notifications/channels')
      setChannels(await res.json() || [])
    } catch (error) {
      console.error('Failed to fetch channels:', error)
    }
  }

  useEffect(() => {
    fetchChannels()
  }, [])

  const toggleChannel = async (channel: NotificationChannel) => {
    await fetch(`/api/notifications/channels/${channel.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...channel, enabled: !channel.enabled })
    })
    fetchChannels()
  }

  const deleteChannel = async (id: number) => {
    if (!confirm('确定要删除此通知渠道吗？')) return
    await fetch(`/api/notifications/channels/${id}`, { method: 'DELETE' })
    fetchChannels()
  }

  const testChannel = async (id: number) => {
    const res = await fetch(`/api/notifications/channels/${id}/test`, { method: 'POST' })
    const data = await res.json()
    alert(data.success ? '测试通知发送成功！' : `测试失败: ${data.error}`)
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-4xl border dark:border-gray-700 flex flex-col" style={{ height: '80vh' }}>
        {/* Header - 固定高度 */}
        <div className="flex items-center justify-between p-4 border-b dark:border-gray-700 flex-shrink-0">
          <div className="flex items-center gap-2">
            <Settings className="h-5 w-5" />
            <h2 className="text-lg font-semibold">系统设置</h2>
          </div>
          <button onClick={onClose} className="p-1 hover:bg-gray-100 dark:hover:bg-gray-800 rounded">
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Tabs - 固定高度 */}
        <div className="flex border-b dark:border-gray-700 flex-shrink-0">
          {[
            { key: 'channels', label: '通知渠道', icon: <Bell className="h-4 w-4" /> },
            { key: 'weather', label: '天气 API', icon: <CloudSun className="h-4 w-4" /> },
            { key: 'ai', label: 'AI 设置', icon: <Bot className="h-4 w-4" /> },
            { key: 'skills', label: '技能管理', icon: <Package className="h-4 w-4" /> },
          ].map(tab => (
            <button
              key={tab.key}
              className={`px-4 py-2 text-sm font-medium flex items-center gap-1 ${
                activeTab === tab.key ? 'border-b-2 border-primary text-primary' : 'text-gray-600 dark:text-gray-400'
              }`}
              onClick={() => setActiveTab(tab.key as any)}
            >
              {tab.icon}
              {tab.label}
            </button>
          ))}
        </div>

        {/* Content - 固定高度，超出滚动 */}
        <div className="flex-1 overflow-y-auto p-4">
          {activeTab === 'channels' && (
            <ChannelsTab
              channels={channels}
              onAdd={() => { setEditingChannel(null); setShowChannelForm(true) }}
              onEdit={(channel) => { setEditingChannel(channel); setShowChannelForm(true) }}
              onToggle={toggleChannel}
              onDelete={deleteChannel}
              onTest={testChannel}
            />
          )}

          {activeTab === 'weather' && (
            <WeatherSettings channels={channels} />
          )}

          {activeTab === 'ai' && (
            <AssistantSettings />
          )}

          {activeTab === 'skills' && (
            <SkillsTab />
          )}
        </div>

        {showChannelForm && (
          <ChannelForm
            channel={editingChannel}
            onClose={() => { setShowChannelForm(false); setEditingChannel(null) }}
            onSave={() => { setShowChannelForm(false); setEditingChannel(null); fetchChannels() }}
          />
        )}
      </div>
    </div>
  )
}
