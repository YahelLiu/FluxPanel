import { useEffect, useState } from 'react'
import { AlertTriangle, X } from 'lucide-react'
import { AlertTab, LogsTab, ThresholdForm } from './settings'
import type { NotificationChannel } from './settings/ChannelsTab'
import type { AlertThreshold } from './settings/ThresholdForm'

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

interface AlertSettingsProps {
  onClose: () => void
}

export function AlertSettings({ onClose }: AlertSettingsProps) {
  const [thresholds, setThresholds] = useState<AlertThreshold[]>([])
  const [alertRecords, setAlertRecords] = useState<AlertRecord[]>([])
  const [channels, setChannels] = useState<NotificationChannel[]>([])
  const [activeTab, setActiveTab] = useState<'rules' | 'logs'>('rules')
  const [editingThreshold, setEditingThreshold] = useState<AlertThreshold | null>(null)
  const [showThresholdForm, setShowThresholdForm] = useState(false)

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

  const fetchChannels = async () => {
    try {
      const res = await fetch('/api/notifications/channels')
      setChannels(await res.json() || [])
    } catch (error) {
      console.error('Failed to fetch channels:', error)
    }
  }

  useEffect(() => {
    fetchThresholds()
    fetchAlertRecords()
    fetchChannels()
  }, [])

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

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-4xl border dark:border-gray-700 flex flex-col" style={{ height: '80vh' }}>
        {/* Header - 固定高度 */}
        <div className="flex items-center justify-between p-4 border-b dark:border-gray-700 flex-shrink-0">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5" />
            <h2 className="text-lg font-semibold">告警设置</h2>
          </div>
          <button onClick={onClose} className="p-1 hover:bg-gray-100 dark:hover:bg-gray-800 rounded">
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Tabs - 固定高度 */}
        <div className="flex border-b dark:border-gray-700 flex-shrink-0">
          {[
            { key: 'rules', label: '告警规则', count: thresholds.length },
            { key: 'logs', label: '告警记录', count: alertRecords.length },
          ].map(tab => (
            <button
              key={tab.key}
              className={`px-4 py-2 text-sm font-medium ${
                activeTab === tab.key ? 'border-b-2 border-primary text-primary' : 'text-gray-600 dark:text-gray-400'
              }`}
              onClick={() => setActiveTab(tab.key as any)}
            >
              {tab.label}
              {tab.count > 0 && ` (${tab.count})`}
            </button>
          ))}
        </div>

        {/* Content - 固定高度，超出滚动 */}
        <div className="flex-1 overflow-y-auto p-4">
          {activeTab === 'rules' && (
            <AlertTab
              thresholds={thresholds}
              alertRecords={alertRecords}
              channels={channels}
              onAdd={() => { setEditingThreshold(null); setShowThresholdForm(true) }}
              onEdit={(threshold) => { setEditingThreshold(threshold); setShowThresholdForm(true) }}
              onToggle={toggleThreshold}
              onDelete={deleteThreshold}
              onTest={testThreshold}
              onResolveAlert={resolveAlert}
              onDeleteAlert={deleteAlertRecord}
            />
          )}

          {activeTab === 'logs' && (
            <LogsTab
              alertRecords={alertRecords}
              onResolve={resolveAlert}
              onDelete={deleteAlertRecord}
            />
          )}
        </div>

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
