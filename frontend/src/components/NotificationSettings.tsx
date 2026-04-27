import { useEffect, useState } from 'react'
import { Bell, Plus, Trash2, Edit2, Check, X, Send, AlertTriangle, CheckCircle, AlertCircle, Gauge, CloudSun } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { WeatherSettings } from './WeatherSettings'

interface NotificationChannel {
  id: number
  name: string
  type: 'feishu' | 'wechat_work'
  mode: 'webhook' | 'app'
  enabled: boolean
  trigger: 'error' | 'warning' | 'all' | 'custom'
  feishu: {
    webhook_url?: string
    app_id?: string
    app_secret?: string
    user_ids?: string[]
  }
  wechat_work: {
    webhook_url?: string
    corp_id?: string
    agent_id?: number
    secret?: string
    user_ids?: string[]
  }
  description?: string
  created_at: string
}

interface AlertThreshold {
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

interface AlertRecord {
  id: number
  threshold_id: number
  client_id: string
  metric_type: string
  metric_value: number
  threshold: number
  status: string
  notified: boolean
  resolved_at?: string
  created_at: string
}

interface NotificationSettingsProps {
  onClose?: () => void
}

export function NotificationSettings({ onClose }: NotificationSettingsProps) {
  const [channels, setChannels] = useState<NotificationChannel[]>([])
  const [thresholds, setThresholds] = useState<AlertThreshold[]>([])
  const [alertRecords, setAlertRecords] = useState<AlertRecord[]>([])
  const [activeTab, setActiveTab] = useState<'channels' | 'alerts' | 'logs' | 'weather'>('alerts')
  const [editingChannel, setEditingChannel] = useState<NotificationChannel | null>(null)
  const [editingThreshold, setEditingThreshold] = useState<AlertThreshold | null>(null)
  const [showChannelForm, setShowChannelForm] = useState(false)
  const [showThresholdForm, setShowThresholdForm] = useState(false)

  const fetchChannels = async () => {
    try {
      const res = await fetch('/api/notifications/channels')
      setChannels(await res.json() || [])
    } catch (error) {
      console.error('Failed to fetch channels:', error)
    }
  }

  const fetchThresholds = async () => {
    try {
      const res = await fetch('/api/alerts/thresholds')
      setThresholds(await res.json() || [])
    } catch (error) {
      console.error('Failed to fetch thresholds:', error)
    }
  }

  const fetchAlertRecords = async () => {
    try {
      const res = await fetch('/api/alerts/records?page_size=50')
      const data = await res.json()
      setAlertRecords(data.records || [])
    } catch (error) {
      console.error('Failed to fetch alert records:', error)
    }
  }

  useEffect(() => {
    fetchChannels()
    fetchThresholds()
    fetchAlertRecords()
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

  const toggleThreshold = async (threshold: AlertThreshold) => {
    await fetch(`/api/alerts/thresholds/${threshold.id}/toggle`, { method: 'PUT' })
    fetchThresholds()
  }

  const deleteThreshold = async (id: number) => {
    if (!confirm('确定要删除此告警规则吗？')) return
    await fetch(`/api/alerts/thresholds/${id}`, { method: 'DELETE' })
    fetchThresholds()
  }

  const resolveAlert = async (id: number) => {
    await fetch(`/api/alerts/records/${id}/resolve`, { method: 'PUT' })
    fetchAlertRecords()
  }

  const deleteAlertRecord = async (id: number) => {
    if (!confirm('确定要删除此告警记录吗？')) return
    await fetch(`/api/alerts/records/${id}`, { method: 'DELETE' })
    fetchAlertRecords()
  }

  const testThreshold = async (id: number) => {
    const res = await fetch(`/api/alerts/thresholds/${id}/test`, { method: 'POST' })
    const data = await res.json()
    alert(data.success ? '测试告警发送成功！' : `测试失败: ${data.error}`)
  }

  const getTypeLabel = (type: string) => type === 'feishu' ? '飞书' : '企业微信'
  const getModeLabel = (mode: string) => mode === 'webhook' ? '群机器人' : '应用消息'
  const getMetricLabel = (metric: string) => {
    switch (metric) {
      case 'cpu': return 'CPU 使用率'
      case 'memory': return '内存使用率'
      case 'disk': return '硬盘使用率'
      default: return metric
    }
  }
  const getOperatorLabel = (op: string) => {
    switch (op) {
      case '>': return '超过'
      case '>=': return '达到或超过'
      case '<': return '低于'
      case '<=': return '达到或低于'
      default: return op
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] overflow-hidden border dark:border-gray-700">
        <div className="flex items-center justify-between p-4 border-b dark:border-gray-700">
          <div className="flex items-center gap-2">
            <Bell className="h-5 w-5" />
            <h2 className="text-lg font-semibold">通知与告警设置</h2>
          </div>
          <button onClick={onClose} className="p-1 hover:bg-gray-100 dark:hover:bg-gray-800 rounded">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex border-b dark:border-gray-700">
          {[
            { key: 'alerts', label: '告警规则', icon: <AlertTriangle className="h-4 w-4" />, count: thresholds.length },
            { key: 'channels', label: '通知渠道', icon: <Bell className="h-4 w-4" />, count: channels.length },
            { key: 'weather', label: '天气推送', icon: <CloudSun className="h-4 w-4" /> },
            { key: 'logs', label: '告警记录', icon: <CheckCircle className="h-4 w-4" /> },
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
              {tab.count !== undefined && ` (${tab.count})`}
            </button>
          ))}
        </div>

        <div className="p-4 overflow-y-auto max-h-[calc(90vh-130px)]">
          {/* Alerts Tab */}
          {activeTab === 'alerts' && (
            <div className="space-y-4">
              <div className="flex justify-between items-center">
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  配置告警阈值，当 CPU/内存/硬盘使用率超过阈值时自动发送通知
                </p>
                <button
                  onClick={() => { setEditingThreshold(null); setShowThresholdForm(true) }}
                  className="flex items-center gap-1 px-3 py-1.5 bg-primary text-primary-foreground rounded-md text-sm hover:bg-primary/90"
                >
                  <Plus className="h-4 w-4" />
                  添加告警
                </button>
              </div>

              {thresholds.length === 0 ? (
                <div className="text-center text-gray-500 dark:text-gray-400 py-10 border-2 border-dashed rounded-lg border-gray-300 dark:border-gray-600">
                  <Gauge className="h-12 w-12 mx-auto mb-2 opacity-50" />
                  <p>暂无告警规则</p>
                  <p className="text-sm">点击上方按钮添加，如：硬盘使用率超过 80% 时通知</p>
                </div>
              ) : (
                <div className="space-y-3">
                  {thresholds.map(threshold => (
                    <Card key={threshold.id}>
                      <CardContent className="p-4 bg-white dark:bg-gray-800">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <div className={`w-3 h-3 rounded-full ${threshold.enabled ? 'bg-green-500' : 'bg-gray-300'}`} />
                            <div>
                              <div className="font-medium text-gray-900 dark:text-gray-100">{threshold.name}</div>
                              <div className="text-sm text-gray-600 dark:text-gray-400 flex items-center gap-2">
                                <Badge variant="outline">{getMetricLabel(threshold.metric_type)}</Badge>
                                <span>{getOperatorLabel(threshold.operator)} {threshold.threshold}%</span>
                                {(threshold.channel_ids || []).length > 0 && (
                                  <span className="text-xs">
                                    → {(threshold.channel_ids || []).map(id => channels.find(c => c.id === id)?.name || `#${id}`).join(', ')}
                                  </span>
                                )}
                              </div>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            <button
                              onClick={() => testThreshold(threshold.id)}
                              className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-gray-700 dark:text-gray-300"
                              title="发送测试"
                            >
                              <Send className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => toggleThreshold(threshold)}
                              className={`p-1.5 rounded ${threshold.enabled ? 'text-green-600 hover:bg-green-50 dark:hover:bg-green-900/20' : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'}`}
                              title={threshold.enabled ? '禁用' : '启用'}
                            >
                              {threshold.enabled ? <Check className="h-4 w-4" /> : <X className="h-4 w-4" />}
                            </button>
                            <button
                              onClick={() => { setEditingThreshold(threshold); setShowThresholdForm(true) }}
                              className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-gray-700 dark:text-gray-300"
                              title="编辑"
                            >
                              <Edit2 className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => deleteThreshold(threshold.id)}
                              className="p-1.5 hover:bg-red-50 dark:hover:bg-red-900/20 text-red-500 rounded"
                              title="删除"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}

              {/* Recent Alerts */}
              {alertRecords.length > 0 && (
                <div className="mt-6">
                  <h3 className="text-sm font-medium mb-2 text-gray-700 dark:text-gray-300">最近告警</h3>
                  <div className="space-y-2">
                    {alertRecords.slice(0, 5).map(record => (
                      <div key={record.id} className="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
                        {record.status === 'triggered' ? (
                          <AlertCircle className="h-4 w-4 text-red-500" />
                        ) : (
                          <CheckCircle className="h-4 w-4 text-green-500" />
                        )}
                        <div className="flex-1">
                          <div className="text-sm text-gray-900 dark:text-gray-100">
                            <span className="font-medium">{record.client_id}</span>
                            <span className="text-gray-600 dark:text-gray-400 ml-2">
                              {getMetricLabel(record.metric_type)} {record.metric_value.toFixed(1)}%
                            </span>
                          </div>
                        </div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {new Date(record.created_at).toLocaleString()}
                        </div>
                        {record.status === 'triggered' && (
                          <button
                            onClick={() => resolveAlert(record.id)}
                            className="text-xs px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400"
                          >
                            解决
                          </button>
                        )}
                        <button
                          onClick={() => deleteAlertRecord(record.id)}
                          className="p-1 hover:bg-red-50 dark:hover:bg-red-900/20 text-red-500 rounded"
                          title="删除"
                        >
                          <Trash2 className="h-3 w-3" />
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Channels Tab */}
          {activeTab === 'channels' && (
            <div className="space-y-4">
              <div className="flex justify-end">
                <button
                  onClick={() => { setEditingChannel(null); setShowChannelForm(true) }}
                  className="flex items-center gap-1 px-3 py-1.5 bg-primary text-primary-foreground rounded-md text-sm hover:bg-primary/90"
                >
                  <Plus className="h-4 w-4" />
                  添加渠道
                </button>
              </div>

              {channels.length === 0 ? (
                <div className="text-center text-gray-500 dark:text-gray-400 py-10">
                  暂无通知渠道，点击上方按钮添加飞书或企业微信机器人
                </div>
              ) : (
                <div className="space-y-3">
                  {channels.map(channel => (
                    <Card key={channel.id}>
                      <CardContent className="p-4 bg-white dark:bg-gray-800">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <div className={`w-3 h-3 rounded-full ${channel.enabled ? 'bg-green-500' : 'bg-gray-300'}`} />
                            <div>
                              <div className="font-medium text-gray-900 dark:text-gray-100">{channel.name}</div>
                              <div className="text-sm text-gray-600 dark:text-gray-400 flex items-center gap-2">
                                <Badge variant="outline">{getTypeLabel(channel.type)}</Badge>
                                <Badge variant="outline">{getModeLabel(channel.mode)}</Badge>
                              </div>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            <button onClick={() => testChannel(channel.id)} className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-gray-700 dark:text-gray-300" title="发送测试">
                              <Send className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => { setEditingChannel(channel); setShowChannelForm(true) }}
                              className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-gray-700 dark:text-gray-300"
                            >
                              <Edit2 className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => toggleChannel(channel)}
                              className={`p-1.5 rounded ${channel.enabled ? 'text-green-600 hover:bg-green-50 dark:hover:bg-green-900/20' : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'}`}
                            >
                              {channel.enabled ? <Check className="h-4 w-4" /> : <X className="h-4 w-4" />}
                            </button>
                            <button onClick={() => deleteChannel(channel.id)} className="p-1.5 hover:bg-red-50 dark:hover:bg-red-900/20 text-red-500 rounded">
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Logs Tab */}
          {activeTab === 'logs' && (
            <div className="space-y-2">
              {alertRecords.length === 0 ? (
                <div className="text-center text-gray-500 dark:text-gray-400 py-10">暂无告警记录</div>
              ) : (
                alertRecords.map(record => (
                  <div key={record.id} className="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
                    {record.status === 'triggered' ? (
                      <AlertCircle className="h-4 w-4 text-red-500" />
                    ) : (
                      <CheckCircle className="h-4 w-4 text-green-500" />
                    )}
                    <div className="flex-1">
                      <div className="text-sm text-gray-900 dark:text-gray-100">
                        <span className="font-medium">{record.client_id}</span> -
                        {getMetricLabel(record.metric_type)} {record.metric_value.toFixed(1)}% (阈值: {record.threshold}%)
                      </div>
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">
                      {new Date(record.created_at).toLocaleString()}
                    </div>
                  </div>
                ))
              )}
            </div>
          )}

          {/* Weather Tab */}
          {activeTab === 'weather' && (
            <WeatherSettings channels={channels} />
          )}
        </div>

        {showChannelForm && (
          <ChannelForm
            channel={editingChannel}
            onClose={() => { setShowChannelForm(false); setEditingChannel(null) }}
            onSave={() => { setShowChannelForm(false); setEditingChannel(null); fetchChannels() }}
          />
        )}

        {showThresholdForm && (
          <ThresholdForm
            threshold={editingThreshold}
            channels={channels}
            onClose={() => { setShowThresholdForm(false); setEditingThreshold(null) }}
            onSave={() => { setShowThresholdForm(false); setEditingThreshold(null); fetchThresholds() }}
          />
        )}
      </div>
    </div>
  )
}

// Channel Form
function ChannelForm({ channel, onClose, onSave }: { channel: NotificationChannel | null; onClose: () => void; onSave: () => void }) {
  const [formData, setFormData] = useState({
    name: channel?.name || '',
    type: channel?.type || 'feishu',
    mode: channel?.mode || 'webhook',
    enabled: channel?.enabled ?? true,
    trigger: channel?.trigger || 'error',
    description: channel?.description || '',
    feishu: {
      webhook_url: channel?.feishu?.webhook_url || '',
      app_id: channel?.feishu?.app_id || '',
      app_secret: channel?.feishu?.app_secret || '',
      user_ids: channel?.feishu?.user_ids?.join(', ') || ''
    },
    wechat_work: {
      webhook_url: channel?.wechat_work?.webhook_url || '',
      corp_id: channel?.wechat_work?.corp_id || '',
      agent_id: channel?.wechat_work?.agent_id || '',
      secret: channel?.wechat_work?.secret || '',
      user_ids: channel?.wechat_work?.user_ids?.join(', ') || ''
    }
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const payload: any = {
      name: formData.name,
      type: formData.type,
      mode: formData.mode,
      enabled: formData.enabled,
      trigger: formData.trigger,
      description: formData.description
    }

    if (formData.type === 'feishu') {
      payload.feishu = {
        webhook_url: formData.feishu.webhook_url || undefined,
        app_id: formData.feishu.app_id || undefined,
        app_secret: formData.feishu.app_secret || undefined,
        user_ids: formData.feishu.user_ids ? formData.feishu.user_ids.split(',').map(s => s.trim()).filter(Boolean) : []
      }
    } else {
      payload.wechat_work = {
        webhook_url: formData.wechat_work.webhook_url || undefined,
        corp_id: formData.wechat_work.corp_id || undefined,
        agent_id: formData.wechat_work.agent_id ? Number(formData.wechat_work.agent_id) : undefined,
        secret: formData.wechat_work.secret || undefined,
        user_ids: formData.wechat_work.user_ids ? formData.wechat_work.user_ids.split(',').map(s => s.trim()).filter(Boolean) : []
      }
    }

    const url = channel ? `/api/notifications/channels/${channel.id}` : '/api/notifications/channels'
    const res = await fetch(url, {
      method: channel ? 'PUT' : 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
    if (res.ok) onSave()
    else alert(`保存失败: ${(await res.json()).error}`)
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto border dark:border-gray-700">
        <form onSubmit={handleSubmit}>
          <div className="p-4 border-b dark:border-gray-700">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{channel ? '编辑通知渠道' : '添加通知渠道'}</h3>
          </div>
          <div className="p-4 space-y-4 bg-white dark:bg-gray-900">
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">名称 *</label>
              <input type="text" value={formData.name} onChange={e => setFormData({ ...formData, name: e.target.value })} className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600" placeholder="如: 运维告警群" required />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">渠道类型</label>
              <select value={formData.type} onChange={e => setFormData({ ...formData, type: e.target.value as any })} className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600">
                <option value="feishu">飞书</option>
                <option value="wechat_work">企业微信</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">通知方式</label>
              <select value={formData.mode} onChange={e => setFormData({ ...formData, mode: e.target.value as any })} className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600">
                <option value="webhook">群机器人 Webhook</option>
                <option value="app">应用消息 (单聊)</option>
              </select>
            </div>

            {formData.type === 'feishu' && formData.mode === 'webhook' && (
              <div>
                <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">Webhook 地址</label>
                <input type="url" value={formData.feishu.webhook_url} onChange={e => setFormData({ ...formData, feishu: { ...formData.feishu, webhook_url: e.target.value } })} className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..." />
              </div>
            )}

            {formData.type === 'wechat_work' && formData.mode === 'webhook' && (
              <div>
                <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">Webhook 地址</label>
                <input type="url" value={formData.wechat_work.webhook_url} onChange={e => setFormData({ ...formData, wechat_work: { ...formData.wechat_work, webhook_url: e.target.value } })} className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600" placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..." />
              </div>
            )}

            <div className="flex items-center gap-2">
              <input type="checkbox" id="enabled" checked={formData.enabled} onChange={e => setFormData({ ...formData, enabled: e.target.checked })} className="rounded" />
              <label htmlFor="enabled" className="text-sm text-gray-700 dark:text-gray-300">启用</label>
            </div>
          </div>
          <div className="p-4 border-t dark:border-gray-700 flex justify-end gap-2">
            <button type="button" onClick={onClose} className="px-4 py-2 border rounded-md hover:bg-gray-50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-gray-600">取消</button>
            <button type="submit" className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90">保存</button>
          </div>
        </form>
      </div>
    </div>
  )
}

// Threshold Form
function ThresholdForm({ threshold, channels, onClose, onSave }: { threshold: AlertThreshold | null; channels: NotificationChannel[]; onClose: () => void; onSave: () => void }) {
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
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{threshold ? '编辑告警规则' : '添加告警规则'}</h3>
          </div>
          <div className="p-4 space-y-4 bg-white dark:bg-gray-900">
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">名称 *</label>
              <input type="text" value={formData.name} onChange={e => setFormData({ ...formData, name: e.target.value })} className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600" placeholder="如: 硬盘空间告警" required />
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">监控指标</label>
                <select value={formData.metric_type} onChange={e => setFormData({ ...formData, metric_type: e.target.value })} className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600">
                  <option value="disk">硬盘使用率</option>
                  <option value="memory">内存使用率</option>
                  <option value="cpu">CPU 使用率</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">条件</label>
                <select value={formData.operator} onChange={e => setFormData({ ...formData, operator: e.target.value })} className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600">
                  <option value=">=">达到或超过</option>
                  <option value=">">超过</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">阈值 (%)</label>
                <input type="number" value={formData.threshold} onChange={e => setFormData({ ...formData, threshold: Number(e.target.value) })} className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600" min="0" max="100" />
              </div>
            </div>

            {channels.length > 0 && (
              <div>
                <label className="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-300">通知渠道</label>
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
                      <span className="text-xs text-gray-500 dark:text-gray-400">({channel.type === 'feishu' ? '飞书' : '企业微信'})</span>
                    </label>
                  ))}
                </div>
              </div>
            )}

            <div className="flex items-center gap-2">
              <input type="checkbox" id="threshold-enabled" checked={formData.enabled} onChange={e => setFormData({ ...formData, enabled: e.target.checked })} className="rounded" />
              <label htmlFor="threshold-enabled" className="text-sm text-gray-700 dark:text-gray-300">启用</label>
            </div>
          </div>
          <div className="p-4 border-t dark:border-gray-700 flex justify-end gap-2">
            <button type="button" onClick={onClose} className="px-4 py-2 border rounded-md hover:bg-gray-50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-gray-600">取消</button>
            <button type="submit" className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90">保存</button>
          </div>
        </form>
      </div>
    </div>
  )
}
