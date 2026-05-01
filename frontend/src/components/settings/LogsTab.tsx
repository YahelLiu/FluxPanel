import { AlertCircle, CheckCircle, Trash2 } from 'lucide-react'

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

interface LogsTabProps {
  alertRecords: AlertRecord[]
  onResolve: (id: number) => void
  onDelete: (id: number) => void
}

const getMetricLabel = (metric: string) => {
  switch (metric) {
    case 'cpu': return 'CPU 使用率'
    case 'memory': return '内存使用率'
    case 'disk': return '硬盘使用率'
    default: return metric
  }
}

export function LogsTab({ alertRecords, onResolve, onDelete }: LogsTabProps) {
  return (
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
            {record.status === 'triggered' && (
              <button
                onClick={() => onResolve(record.id)}
                className="text-xs px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400"
              >
                解决
              </button>
            )}
            <button
              onClick={() => onDelete(record.id)}
              className="p-1 hover:bg-red-50 dark:hover:bg-red-900/20 text-red-500 rounded"
              title="删除"
            >
              <Trash2 className="h-3 w-3" />
            </button>
          </div>
        ))
      )}
    </div>
  )
}
