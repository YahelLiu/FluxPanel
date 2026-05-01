import { useState } from 'react'

export interface AlertThreshold {
  id: number
  name: string
  metric_type: string
  operator: string
  threshold: number
  duration: number
  channel_ids: number[]
  enabled: boolean
  description?: string
  created_at: string
}

import type { NotificationChannel } from './ChannelsTab'

interface ThresholdFormProps {
  threshold: AlertThreshold | null
  channels: NotificationChannel[]
  onClose: () => void
  onSave: () => void
}

export function ThresholdForm({ threshold, channels, onClose, onSave }: ThresholdFormProps) {
  const [formData, setFormData] = useState({
    name: threshold?.name || '',
    metric_type: threshold?.metric_type || 'disk',
    operator: threshold?.operator || '>=',
    threshold: threshold?.threshold || 80,
    channel_ids: threshold?.channel_ids || [],
    enabled: threshold?.enabled ?? true,
    description: threshold?.description || ''
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const isEditing = threshold && threshold.id
    const url = isEditing ? `/api/alerts/thresholds/${threshold.id}` : '/api/alerts/thresholds'
    const res = await fetch(url, {
      method: isEditing ? 'PUT' : 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        ...formData,
        channel_ids: formData.channel_ids || []
      })
    })
    if (res.ok) onSave()
    else alert(`保存失败: ${(await res.json()).error}`)
  }

  const toggleChannel = (channelId: number) => {
    setFormData({
      ...formData,
      channel_ids: formData.channel_ids.includes(channelId)
        ? formData.channel_ids.filter(id => id !== channelId)
        : [...formData.channel_ids, channelId]
    })
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-lg border dark:border-gray-700">
        <form onSubmit={handleSubmit}>
          <div className="p-4 border-b dark:border-gray-700">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {threshold ? '编辑告警规则' : '添加告警规则'}
            </h3>
          </div>
          <div className="p-4 space-y-4 bg-white dark:bg-gray-900">
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">
                名称 *
              </label>
              <input
                type="text"
                value={formData.name}
                onChange={e => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder="如: 硬盘空间告警"
                required
              />
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">
                  监控指标
                </label>
                <select
                  value={formData.metric_type}
                  onChange={e => setFormData({ ...formData, metric_type: e.target.value })}
                  className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                >
                  <option value="disk">硬盘使用率</option>
                  <option value="memory">内存使用率</option>
                  <option value="cpu">CPU 使用率</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">
                  条件
                </label>
                <select
                  value={formData.operator}
                  onChange={e => setFormData({ ...formData, operator: e.target.value })}
                  className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                >
                  <option value=">=">达到或超过</option>
                  <option value=">">超过</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">
                  阈值 (%)
                </label>
                <input
                  type="number"
                  value={formData.threshold}
                  onChange={e => setFormData({ ...formData, threshold: Number(e.target.value) })}
                  className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                  min="0"
                  max="100"
                />
              </div>
            </div>

            {channels.length > 0 && (
              <div>
                <label className="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-300">
                  通知渠道
                </label>
                <div className="space-y-2">
                  {channels.filter(c => c.enabled).map(channel => (
                    <label key={channel.id} className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={formData.channel_ids.includes(channel.id)}
                        onChange={() => toggleChannel(channel.id)}
                        className="rounded"
                      />
                      <span className="text-sm text-gray-900 dark:text-gray-100">{channel.name}</span>
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        ({channel.type === 'feishu' ? '飞书' : '企业微信'})
                      </span>
                    </label>
                  ))}
                </div>
              </div>
            )}

            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="threshold-enabled"
                checked={formData.enabled}
                onChange={e => setFormData({ ...formData, enabled: e.target.checked })}
                className="rounded"
              />
              <label htmlFor="threshold-enabled" className="text-sm text-gray-700 dark:text-gray-300">
                启用
              </label>
            </div>
          </div>
          <div className="p-4 border-t dark:border-gray-700 flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border rounded-md hover:bg-gray-50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-gray-600"
            >
              取消
            </button>
            <button
              type="submit"
              className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
            >
              保存
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
