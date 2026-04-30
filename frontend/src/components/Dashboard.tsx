import { useEffect, useState } from 'react'
import { useWebSocket } from '@/hooks/useWebSocket'
import { Activity, Wifi, WifiOff, Cpu, HardDrive, Monitor, MapPin, Clock, Trash2, GripVertical, Bell, CloudSun, Send } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { NotificationSettings } from '@/components/NotificationSettings'

interface ClientOrder {
  client_id: string
  sort_order: number
  weather_enabled: boolean
  channel_id: number
}

interface Disk {
  name: string
  label?: string
  total_gb: number
  used_gb: number
  available_gb: number
  load_percent: number
}

interface ClientData {
  client_id: string
  last_seen: string
  sort_order?: number
  data: {
    timestamp?: string
    location?: {
      city?: string
      region?: string
      district?: string
      country?: string
      ip?: string
    }
    memory?: {
      used_gb?: number
      available_gb?: number
      load_percent?: number
      page_used_gb?: number
      page_available_gb?: number
    }
    cpu?: {
      name?: string
      load_percent?: number
      temperature_c?: number
      power_w?: number
    }
    gpu?: {
      name?: string
      load_percent?: number
      temperature_c?: number
      power_w?: number
      memory_used_mb?: number
      memory_total_mb?: number
    }
    disks?: Disk[]
  }
}

interface WSEvent {
  id: number
  client_id: string
  event_type: string
  data: ClientData['data']
  status: string
  created_at: string
}

export function Dashboard() {
  const [clients, setClients] = useState<Map<string, ClientData>>(new Map())
  const [clientOrders, setClientOrders] = useState<Map<string, ClientOrder>>(new Map())
  const [loading, setLoading] = useState(true)
  const [draggedClient, setDraggedClient] = useState<string | null>(null)
  const [showNotificationSettings, setShowNotificationSettings] = useState(false)
  const [now, setNow] = useState(() => Date.now())

  // WebSocket connection
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${wsProtocol}//${window.location.host}/ws`
  const { isConnected, lastEvent, connectionStatus, reconnectAttempts } = useWebSocket(wsUrl)
  const websocketFailed = connectionStatus === 'failed'

  // Fetch client orders
  const fetchClientOrders = async () => {
    try {
      const res = await fetch('/api/clients/orders')
      const orders = await res.json()
      const orderMap = new Map<string, ClientOrder>()
      for (const order of orders) {
        orderMap.set(order.client_id, order)
      }
      setClientOrders(orderMap)
    } catch (error) {
      console.error('Failed to fetch client orders:', error)
    }
  }

  // Fetch initial client data
  useEffect(() => {
    const fetchClients = async () => {
      try {
        const [clientsRes] = await Promise.all([
          fetch('/api/clients/latest'),
          fetchClientOrders()
        ])
        const json = await clientsRes.json()
        const clientMap = new Map<string, ClientData>()

        for (const client of json.clients || []) {
          clientMap.set(client.client_id, {
            client_id: client.client_id,
            last_seen: client.last_seen,
            data: client.data || {}
          })
        }

        setClients(clientMap)
        setLoading(false)
      } catch (error) {
        console.error('Failed to fetch clients:', error)
        setLoading(false)
      }
    }

    fetchClients()
  }, [])

  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 15000)
    return () => window.clearInterval(interval)
  }, [])

  // Update when new event arrives via WebSocket
  useEffect(() => {
    if (lastEvent) {
      const event = lastEvent as WSEvent
      setClients(prev => {
        const newMap = new Map(prev)
        newMap.set(event.client_id, {
          client_id: event.client_id,
          last_seen: event.created_at,
          data: event.data || {}
        })
        return newMap
      })
    }
  }, [lastEvent])

  // Delete client
  const deleteClient = async (clientId: string) => {
    if (!confirm(`确定要删除客户端 "${clientId}" 吗？此操作将删除该客户端的所有数据。`)) {
      return
    }

    try {
      const res = await fetch(`/api/clients/${encodeURIComponent(clientId)}`, {
        method: 'DELETE',
      })
      const data = await res.json()
      if (data.success) {
        setClients(prev => {
          const newMap = new Map(prev)
          newMap.delete(clientId)
          return newMap
        })
        setClientOrders(prev => {
          const newMap = new Map(prev)
          newMap.delete(clientId)
          return newMap
        })
      }
    } catch (error) {
      console.error('Failed to delete client:', error)
    }
  }

  // Drag and drop handlers
  const handleDragStart = (clientId: string) => {
    setDraggedClient(clientId)
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
  }

  const handleDrop = async (targetClientId: string) => {
    if (!draggedClient || draggedClient === targetClientId) {
      setDraggedClient(null)
      return
    }

    // Get sorted client list
    const sortedClients = getSortedClients()
    const draggedIndex = sortedClients.findIndex(c => c.client_id === draggedClient)
    const targetIndex = sortedClients.findIndex(c => c.client_id === targetClientId)

    // Reorder
    const newOrder = [...sortedClients]
    const [removed] = newOrder.splice(draggedIndex, 1)
    newOrder.splice(targetIndex, 0, removed)

    // Update orders
    const orders = newOrder.map((c, index) => ({
      client_id: c.client_id,
      sort_order: index
    }))

    // Save to backend
    try {
      await fetch('/api/clients/orders', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ orders })
      })

      // Update local state
      setClientOrders(prev => {
        const newMap = new Map(prev)
        orders.forEach(o => {
          const existing = newMap.get(o.client_id) || { client_id: o.client_id, sort_order: 0, weather_enabled: false, channel_id: 0 }
          newMap.set(o.client_id, { ...existing, sort_order: o.sort_order })
        })
        return newMap
      })
    } catch (error) {
      console.error('Failed to update order:', error)
    }

    setDraggedClient(null)
  }

  // Get sorted clients
  const getSortedClients = () => {
    const clientList = Array.from(clients.values())
    return clientList.sort((a, b) => {
      const orderA = clientOrders.get(a.client_id)?.sort_order ?? 999999
      const orderB = clientOrders.get(b.client_id)?.sort_order ?? 999999
      return orderA - orderB
    })
  }

  // Update client weather settings
  const updateClientWeather = async (clientId: string, weatherEnabled: boolean) => {
    try {
      await fetch(`/api/clients/${encodeURIComponent(clientId)}/weather`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          weather_enabled: weatherEnabled,
        })
      })
      // Update local state
      setClientOrders(prev => {
        const newMap = new Map(prev)
        const existing = newMap.get(clientId) || { client_id: clientId, sort_order: 999999, weather_enabled: false, channel_id: 0 }
        newMap.set(clientId, { ...existing, weather_enabled: weatherEnabled })
        return newMap
      })
    } catch (error) {
      console.error('Failed to update weather settings:', error)
    }
  }

  // Send weather notification to a specific client
  const sendWeatherToClient = async (clientId: string) => {
    try {
      const res = await fetch('/api/weather/send', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ client_id: clientId })
      })
      const data = await res.json()
      alert(data.message || '发送成功')
    } catch (error) {
      alert('发送失败')
    }
  }

  // Format time ago
  const formatTimeAgo = (dateStr: string) => {
    const date = new Date(dateStr)
    const seconds = Math.floor((now - date.getTime()) / 1000)

    if (seconds < 5) return '刚刚'
    if (seconds < 60) return `${seconds}秒前`
    if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟前`
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}小时前`
    return `${Math.floor(seconds / 86400)}天前`
  }

  // Get status color based on load
  const getLoadColor = (load?: number) => {
    if (load === undefined) return 'text-gray-400'
    if (load < 50) return 'text-green-500'
    if (load < 80) return 'text-yellow-500'
    return 'text-red-500'
  }

  // Get temperature color
  const getTempColor = (temp?: number) => {
    if (temp === undefined) return 'text-gray-400'
    if (temp < 60) return 'text-green-500'
    if (temp < 80) return 'text-yellow-500'
    return 'text-red-500'
  }

  // Check if client is online (active in last 1 minute)
  const isOnline = (lastSeen: string) => {
    const date = new Date(lastSeen)
    const minutes = (now - date.getTime()) / 1000 / 60
    return minutes < 1
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-lg">加载中...</div>
      </div>
    )
  }

  return (
    <div className={`min-h-screen bg-background transition duration-300 ${websocketFailed ? 'grayscale opacity-60' : ''}`}>
      {/* Header */}
      <header className="border-b">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Activity className="h-8 w-8 text-primary" />
              <h1 className="text-2xl font-bold">客户端监控面板</h1>
            </div>
            <div className="flex items-center gap-4">
              <button
                onClick={() => setShowNotificationSettings(true)}
                className="flex items-center gap-1 px-3 py-1.5 border rounded-md hover:bg-gray-50 dark:hover:bg-gray-800 text-sm"
              >
                <Bell className="h-4 w-4" />
                通知设置
              </button>
              <div className="text-sm text-muted-foreground">
                在线客户端: {Array.from(clients.values()).filter(c => isOnline(c.last_seen)).length} / {clients.size}
              </div>
              {isConnected ? (
                <div className="flex items-center gap-1 text-green-600">
                  <Wifi className="h-4 w-4" />
                  <span className="text-sm">实时连接</span>
                </div>
              ) : websocketFailed ? (
                <div className="flex items-center gap-1 text-red-600">
                  <WifiOff className="h-4 w-4" />
                  <span className="text-sm">连接失败</span>
                </div>
              ) : (
                <div className="flex items-center gap-1 text-yellow-600">
                  <WifiOff className="h-4 w-4" />
                  <span className="text-sm">重连中 {reconnectAttempts > 0 && `(${reconnectAttempts}/5)`}</span>
                </div>
              )}
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-6">
        {clients.size === 0 ? (
          <div className="text-center text-muted-foreground py-20">
            暂无客户端数据，等待客户端上报...
          </div>
        ) : (
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {getSortedClients().map((client) => (
              <Card
                key={client.client_id}
                className={`${isOnline(client.last_seen) ? '' : 'opacity-60'} ${draggedClient === client.client_id ? 'ring-2 ring-primary' : ''}`}
                draggable
                onDragStart={() => handleDragStart(client.client_id)}
                onDragOver={handleDragOver}
                onDrop={() => handleDrop(client.client_id)}
              >
                <CardHeader className="pb-2">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg flex items-center gap-2">
                      <GripVertical className="h-4 w-4 text-gray-400 cursor-grab" />
                      <Monitor className="h-5 w-5" />
                      {client.client_id}
                    </CardTitle>
                    <div className="flex items-center gap-2">
                      {isOnline(client.last_seen) ? (
                        <span className="text-xs px-2 py-1 bg-green-100 text-green-700 rounded-full">在线</span>
                      ) : (
                        <span className="text-xs px-2 py-1 bg-gray-100 text-gray-500 rounded-full">离线</span>
                      )}
                      <button
                        onClick={() => deleteClient(client.client_id)}
                        className="p-1 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded transition-colors"
                        title="删除客户端"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                  <div className="flex items-center gap-1 text-xs text-muted-foreground">
                    <Clock className="h-3 w-3" />
                    {formatTimeAgo(client.last_seen)}
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

                  {/* Weather Settings */}
                  <div className="flex items-center gap-2 p-2 bg-gray-50 dark:bg-gray-800/50 rounded-lg text-sm">
                    <CloudSun className="h-4 w-4 text-blue-500" />
                    <label className="flex items-center gap-1 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={clientOrders.get(client.client_id)?.weather_enabled || false}
                        onChange={(e) => {
                          updateClientWeather(client.client_id, e.target.checked)
                        }}
                        className="rounded"
                      />
                      <span className="text-xs">天气推送</span>
                    </label>
                    {clientOrders.get(client.client_id)?.weather_enabled && client.data.location && (
                      <button
                        onClick={() => sendWeatherToClient(client.client_id)}
                        className="p-1 text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded"
                        title="立即发送天气通知"
                      >
                        <Send className="h-4 w-4" />
                      </button>
                    )}
                  </div>

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
                      {/* CPU Load Bar */}
                      {client.data.cpu.load_percent !== undefined && (
                        <div className="w-full h-3 bg-gray-300 dark:bg-gray-700 rounded-full overflow-hidden border border-gray-400 dark:border-gray-600">
                          <div
                            className={`h-full rounded-full transition-all ${
                              client.data.cpu.load_percent < 50 ? 'bg-green-500' :
                              client.data.cpu.load_percent < 80 ? 'bg-yellow-500' : 'bg-red-500'
                            }`}
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
                      {/* GPU Load Bar */}
                      {client.data.gpu.load_percent !== undefined && (
                        <div>
                          <div className="flex justify-between text-xs text-muted-foreground mb-1">
                            <span>负载</span>
                            <span>{client.data.gpu.load_percent?.toFixed(1)}%</span>
                          </div>
                          <div className="w-full h-3 bg-gray-300 dark:bg-gray-700 rounded-full overflow-hidden border border-gray-400 dark:border-gray-600">
                            <div
                              className={`h-full rounded-full transition-all ${
                                client.data.gpu.load_percent < 50 ? 'bg-green-500' :
                                client.data.gpu.load_percent < 80 ? 'bg-yellow-500' : 'bg-red-500'
                              }`}
                              style={{ width: `${Math.min(client.data.gpu.load_percent, 100)}%` }}
                            />
                          </div>
                        </div>
                      )}
                      {/* GPU Memory Bar */}
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
                      {/* Memory Bar */}
                      {client.data.memory.load_percent !== undefined && (
                        <div className="w-full h-3 bg-gray-300 dark:bg-gray-700 rounded-full overflow-hidden border border-gray-400 dark:border-gray-600">
                          <div
                            className={`h-full rounded-full transition-all ${
                              client.data.memory.load_percent < 50 ? 'bg-green-500' :
                              client.data.memory.load_percent < 80 ? 'bg-yellow-500' : 'bg-red-500'
                            }`}
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
                              className={`h-full rounded-full transition-all ${
                                disk.load_percent < 50 ? 'bg-green-500' :
                                disk.load_percent < 80 ? 'bg-yellow-500' : 'bg-red-500'
                              }`}
                              style={{ width: `${Math.min(disk.load_percent, 100)}%` }}
                            />
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </main>

      {/* Notification Settings Modal */}
      {showNotificationSettings && (
        <NotificationSettings onClose={() => setShowNotificationSettings(false)} />
      )}
    </div>
  )
}
