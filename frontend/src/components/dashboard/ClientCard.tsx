import { Monitor, MapPin, Clock, Trash2, GripVertical, Cpu, HardDrive } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { ClientData } from './types'

interface ClientCardProps {
  client: ClientData
  isOnline: boolean
  isDragging: boolean
  now: number
  onDragStart: (clientId: string) => void
  onDragOver: (e: React.DragEvent) => void
  onDrop: (clientId: string) => void
  onDelete: (clientId: string) => void
}

const formatTimeAgo = (dateStr: string, now: number) => {
  const seconds = Math.floor((now - new Date(dateStr).getTime()) / 1000)
  if (seconds < 5) return '刚刚'
  if (seconds < 60) return `${seconds}秒前`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟前`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}小时前`
  return `${Math.floor(seconds / 86400)}天前`
}

const getLoadColor = (load?: number) => {
  if (load === undefined) return 'text-gray-400'
  if (load < 50) return 'text-green-500'
  if (load < 80) return 'text-yellow-500'
  return 'text-red-500'
}

const getTempColor = (temp?: number) => {
  if (temp === undefined) return 'text-gray-400'
  if (temp < 60) return 'text-green-500'
  if (temp < 80) return 'text-yellow-500'
  return 'text-red-500'
}

const getBarColor = (value: number) => {
  if (value < 50) return 'bg-green-500'
  if (value < 80) return 'bg-yellow-500'
  return 'bg-red-500'
}

export function ClientCard({
  client, isOnline, isDragging, now,
  onDragStart, onDragOver, onDrop, onDelete,
}: ClientCardProps) {
  return (
    <Card
      className={`${isOnline ? '' : 'opacity-60'} ${isDragging ? 'ring-2 ring-primary' : ''}`}
      draggable
      onDragStart={() => onDragStart(client.client_id)}
      onDragOver={onDragOver}
      onDrop={() => onDrop(client.client_id)}
    >
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <GripVertical className="h-4 w-4 text-gray-400 cursor-grab" />
            <Monitor className="h-5 w-5" />
            {client.client_id}
          </CardTitle>
          <div className="flex items-center gap-2">
            {isOnline ? (
              <span className="text-xs px-2 py-1 bg-green-100 text-green-700 rounded-full">在线</span>
            ) : (
              <span className="text-xs px-2 py-1 bg-gray-100 text-gray-500 rounded-full">离线</span>
            )}
            <button
              onClick={() => onDelete(client.client_id)}
              className="p-1 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded transition-colors"
              title="删除客户端"
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        </div>
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <Clock className="h-3 w-3" />
          {formatTimeAgo(client.last_seen, now)}
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Location */}
        {client.data.location && (
          <div className="flex items-center gap-2 text-sm">
            <MapPin className="h-4 w-4 text-muted-foreground" />
            <span>
              {client.data.location.city}
              {client.data.location.district && `, ${client.data.location.district}`}
            </span>
            <span className="text-xs text-muted-foreground ml-auto">
              {client.data.location.ip}
            </span>
          </div>
        )}

        {/* CPU */}
        {client.data.cpu && (
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-2">
                <Cpu className="h-4 w-4 text-blue-500" />
                <span className="font-medium">CPU</span>
              </div>
              <span className="text-xs text-muted-foreground truncate max-w-[150px]">
                {client.data.cpu.name}
              </span>
            </div>
            <div className="grid grid-cols-3 gap-2 text-xs">
              <div>
                <span className="text-muted-foreground">负载</span>
                <div className={`font-medium ${getLoadColor(client.data.cpu.load_percent)}`}>
                  {client.data.cpu.load_percent?.toFixed(1)}%
                </div>
              </div>
              <div>
                <span className="text-muted-foreground">温度</span>
                <div className={`font-medium ${getTempColor(client.data.cpu.temperature_c)}`}>
                  {client.data.cpu.temperature_c}°C
                </div>
              </div>
              <div>
                <span className="text-muted-foreground">功耗</span>
                <div className="font-medium">
                  {client.data.cpu.power_w?.toFixed(1)}W
                </div>
              </div>
            </div>
            {client.data.cpu.load_percent !== undefined && (
              <div className="w-full h-3 bg-gray-300 dark:bg-gray-700 rounded-full overflow-hidden border border-gray-400 dark:border-gray-600">
                <div
                  className={`h-full rounded-full transition-all ${getBarColor(client.data.cpu.load_percent)}`}
                  style={{ width: `${Math.min(client.data.cpu.load_percent, 100)}%` }}
                />
              </div>
            )}
          </div>
        )}

        {/* GPU */}
        {client.data.gpu && (
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-2">
                <Monitor className="h-4 w-4 text-purple-500" />
                <span className="font-medium">GPU</span>
              </div>
              <span className="text-xs text-muted-foreground truncate max-w-[150px]">
                {client.data.gpu.name}
              </span>
            </div>
            <div className="grid grid-cols-3 gap-2 text-xs">
              <div>
                <span className="text-muted-foreground">负载</span>
                <div className={`font-medium ${getLoadColor(client.data.gpu.load_percent)}`}>
                  {client.data.gpu.load_percent?.toFixed(1)}%
                </div>
              </div>
              <div>
                <span className="text-muted-foreground">温度</span>
                <div className={`font-medium ${getTempColor(client.data.gpu.temperature_c)}`}>
                  {client.data.gpu.temperature_c}°C
                </div>
              </div>
              <div>
                <span className="text-muted-foreground">功耗</span>
                <div className="font-medium">
                  {client.data.gpu.power_w?.toFixed(1)}W
                </div>
              </div>
            </div>
            {client.data.gpu.load_percent !== undefined && (
              <div>
                <div className="flex justify-between text-xs text-muted-foreground mb-1">
                  <span>负载</span>
                  <span>{client.data.gpu.load_percent?.toFixed(1)}%</span>
                </div>
                <div className="w-full h-3 bg-gray-300 dark:bg-gray-700 rounded-full overflow-hidden border border-gray-400 dark:border-gray-600">
                  <div
                    className={`h-full rounded-full transition-all ${getBarColor(client.data.gpu.load_percent)}`}
                    style={{ width: `${Math.min(client.data.gpu.load_percent, 100)}%` }}
                  />
                </div>
              </div>
            )}
            {client.data.gpu.memory_used_mb !== undefined && client.data.gpu.memory_total_mb !== undefined && (
              <div>
                <div className="flex justify-between text-xs text-muted-foreground mb-1">
                  <span>显存</span>
                  <span>{client.data.gpu.memory_used_mb} / {client.data.gpu.memory_total_mb} MB</span>
                </div>
                <div className="w-full h-3 bg-gray-300 dark:bg-gray-700 rounded-full overflow-hidden border border-gray-400 dark:border-gray-600">
                  <div
                    className="h-full bg-purple-500 rounded-full transition-all"
                    style={{ width: `${(client.data.gpu.memory_used_mb / client.data.gpu.memory_total_mb) * 100}%` }}
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {/* Memory */}
        {client.data.memory && (
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <HardDrive className="h-4 w-4 text-orange-500" />
              <span className="font-medium">内存</span>
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div>
                <span className="text-muted-foreground">已用</span>
                <div className="font-medium">{client.data.memory.used_gb?.toFixed(2)} GB</div>
              </div>
              <div>
                <span className="text-muted-foreground">可用</span>
                <div className="font-medium">{client.data.memory.available_gb?.toFixed(2)} GB</div>
              </div>
            </div>
            {client.data.memory.load_percent !== undefined && (
              <div className="w-full h-3 bg-gray-300 dark:bg-gray-700 rounded-full overflow-hidden border border-gray-400 dark:border-gray-600">
                <div
                  className={`h-full rounded-full transition-all ${getBarColor(client.data.memory.load_percent)}`}
                  style={{ width: `${Math.min(client.data.memory.load_percent, 100)}%` }}
                />
              </div>
            )}
            <div className="text-xs text-muted-foreground text-right">
              负载: {client.data.memory.load_percent?.toFixed(1)}%
            </div>
          </div>
        )}

        {/* Disks */}
        {client.data.disks && client.data.disks.length > 0 && (
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <HardDrive className="h-4 w-4 text-cyan-500" />
              <span className="font-medium">硬盘</span>
            </div>
            {client.data.disks.map((disk, index) => (
              <div key={index} className="space-y-1">
                <div className="flex justify-between text-xs">
                  <span className="font-medium">{disk.name} {disk.label && `(${disk.label})`}</span>
                  <span className="text-muted-foreground">
                    {disk.used_gb?.toFixed(1)} / {disk.total_gb?.toFixed(1)} GB
                  </span>
                </div>
                <div className="w-full h-3 bg-gray-300 dark:bg-gray-700 rounded-full overflow-hidden border border-gray-400 dark:border-gray-600">
                  <div
                    className={`h-full rounded-full transition-all ${getBarColor(disk.load_percent)}`}
                    style={{ width: `${Math.min(disk.load_percent, 100)}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
