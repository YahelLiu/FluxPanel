import { Plus, Trash2, Edit2, Check, X, Send } from 'lucide-react'
import { Badge } from '@/components/ui/badge'

export interface NotificationChannel {
  id: number
  name: string
  type: 'feishu' | 'wechat_work' | 'wechat_ilink'
  mode: 'webhook' | 'app'
  enabled: boolean
  trigger: 'error' | 'warning' | 'all' | 'custom'
  feishu?: {
    webhook_url?: string
    app_id?: string
    app_secret?: string
    user_ids?: string[]
  }
  wechat_work?: {
    webhook_url?: string
    corp_id?: string
    agent_id?: number
    secret?: string
    user_ids?: string[]
  }
  wechat_ilink?: {
    bot_token?: string
    ilink_bot_id?: string
    base_url?: string
    ilink_user_id?: string
    user_ids?: string[]
    logged_in?: boolean
  }
  description?: string
  created_at: string
}

interface ChannelsTabProps {
  channels: NotificationChannel[]
  onAdd: () => void
  onEdit: (channel: NotificationChannel) => void
  onToggle: (channel: NotificationChannel) => void
  onDelete: (id: number) => void
  onTest: (id: number) => void
}

const getTypeLabel = (type: string) => {
  switch (type) {
    case 'feishu': return '飞书'
    case 'wechat_ilink': return '微信 iLink'
    default: return type
  }
}

const getModeLabel = (mode: string) => mode === 'webhook' ? '群机器人' : '应用消息'

export function ChannelsTab({
  channels,
  onAdd,
  onEdit,
  onToggle,
  onDelete,
  onTest,
}: ChannelsTabProps) {
  return (
    <div className="space-y-4">
      {/* 顶部操作栏 */}
      <div className="flex justify-between items-center">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          配置飞书或微信机器人，用于发送告警通知
        </p>
        <button
          onClick={onAdd}
          className="flex items-center gap-1 px-3 py-1.5 bg-primary text-primary-foreground rounded-md text-sm hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          添加渠道
        </button>
      </div>

      {/* 列表 */}
      {channels.length === 0 ? (
        <div className="text-center text-gray-500 dark:text-gray-400 py-10 border-2 border-dashed rounded-lg border-gray-300 dark:border-gray-600">
          暂无通知渠道，点击上方按钮添加
        </div>
      ) : (
        <div className="space-y-2">
          {channels.map(channel => (
            <div
              key={channel.id}
              className="flex items-center justify-between p-3 border rounded-lg bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700"
            >
              <div className="flex items-center gap-3">
                <div className={`w-2.5 h-2.5 rounded-full ${channel.enabled ? 'bg-green-500' : 'bg-gray-300'}`} />
                <div>
                  <div className="font-medium text-gray-900 dark:text-gray-100">{channel.name}</div>
                  <div className="text-sm text-gray-500 flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">{getTypeLabel(channel.type)}</Badge>
                    <Badge variant="outline" className="text-xs">{getModeLabel(channel.mode)}</Badge>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-1">
                <button onClick={() => onTest(channel.id)} className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-gray-500" title="测试">
                  <Send className="h-4 w-4" />
                </button>
                <button onClick={() => onEdit(channel)} className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-gray-500" title="编辑">
                  <Edit2 className="h-4 w-4" />
                </button>
                <button onClick={() => onToggle(channel)} className={`p-1.5 rounded ${channel.enabled ? 'text-green-600 hover:bg-green-50 dark:hover:bg-green-900/20' : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'}`}>
                  {channel.enabled ? <Check className="h-4 w-4" /> : <X className="h-4 w-4" />}
                </button>
                <button onClick={() => onDelete(channel.id)} className="p-1.5 hover:bg-red-50 dark:hover:bg-red-900/20 text-red-500 rounded" title="删除">
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
