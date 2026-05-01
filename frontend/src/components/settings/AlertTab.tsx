import { Plus, Trash2, Edit2, Check, X, Send, AlertCircle, CheckCircle, Gauge } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

// 从 ChannelsTab 导入类型
import { NotificationChannel } from './ChannelsTab'

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

interface AlertTabProps {
  thresholds: AlertThreshold[]
  alertRecords: AlertRecord[]
  channels: NotificationChannel[]
  onAdd: () => void
  onEdit: (threshold: AlertThreshold) => void
  onToggle: (threshold: AlertThreshold) => void
  onDelete: (id: number) => void
  onTest: (id: number) => void
  onResolveAlert: (id: number) => void
  onDeleteAlert: (id: number) => void
}

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

export function AlertTab({
  thresholds,
  alertRecords,
  channels,
  onAdd,
  onEdit,
  onToggle,
  onDelete,
  onTest,
  onResolveAlert,
  onDeleteAlert,
}: AlertTabProps) {
  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          配置告警阈值，当 CPU/内存/硬盘使用率超过阈值时自动发送通知
        </p>
        <button
          onClick={onAdd}
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
                      onClick={() => onTest(threshold.id)}
                      className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-gray-700 dark:text-gray-300"
                      title="发送测试"
                    >
                      <Send className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => onToggle(threshold)}
                      className={`p-1.5 rounded ${threshold.enabled ? 'text-green-600 hover:bg-green-50 dark:hover:bg-green-900/20' : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'}`}
                      title={threshold.enabled ? '禁用' : '启用'}
                    >
                      {threshold.enabled ? <Check className="h-4 w-4" /> : <X className="h-4 w-4" />}
                    </button>
                    <button
                      onClick={() => onEdit(threshold)}
                      className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-gray-700 dark:text-gray-300"
                      title="编辑"
                    >
                      <Edit2 className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => onDelete(threshold.id)}
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
                    onClick={() => onResolveAlert(record.id)}
                    className="text-xs px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400"
                  >
                    解决
                  </button>
                )}
                <button
                  onClick={() => onDeleteAlert(record.id)}
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
  )
}
