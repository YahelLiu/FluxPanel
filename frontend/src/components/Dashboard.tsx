import { useEffect, useState } from 'react'
import { useWebSocket } from '@/hooks/useWebSocket'
import { useClientDrag } from '@/hooks/useClientDrag'
import { Activity, Wifi, WifiOff, AlertTriangle, Settings } from 'lucide-react'
import { ClientCardList } from '@/components/dashboard/ClientCardList'
import type { ClientOrder, ClientData } from '@/components/dashboard/types'
import { AlertSettings } from '@/components/AlertSettings'
import { SystemSettings } from '@/components/SystemSettings'

interface WSEvent {
  client_id: string
  created_at: string
  data: ClientData['data']
}

export function Dashboard() {
  const [clients, setClients] = useState<Map<string, ClientData>>(new Map())
  const [clientOrders, setClientOrders] = useState<Map<string, ClientOrder>>(new Map())
  const [loading, setLoading] = useState(true)
  const [showAlertSettings, setShowAlertSettings] = useState(false)
  const [showSystemSettings, setShowSystemSettings] = useState(false)
  const [now, setNow] = useState(() => Date.now())

  // WebSocket
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${wsProtocol}//${window.location.host}/ws`
  const { isConnected, lastEvent, connectionStatus, reconnectAttempts } = useWebSocket(wsUrl)
  const websocketFailed = connectionStatus === 'failed'

  // Drag and drop
  const { draggedClient, handleDragStart, handleDragOver, handleDrop } = useClientDrag({
    onReorder: async (orders) => {
      await fetch('/api/clients/orders', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ orders })
      })
      setClientOrders(prev => {
        const newMap = new Map(prev)
        orders.forEach(o => {
          const existing = newMap.get(o.client_id) || {
            client_id: o.client_id, sort_order: 0, weather_enabled: false,
            channel_ids: [], is_primary: false, hidden: false
          }
          newMap.set(o.client_id, { ...existing, sort_order: o.sort_order })
        })
        return newMap
      })
    }
  })

  // Fetch data
  useEffect(() => {
    const fetchAll = async () => {
      try {
        const [clientsRes, ordersRes] = await Promise.all([
          fetch('/api/clients/latest'),
          fetch('/api/clients/orders'),
        ])

        const clientsJson = await clientsRes.json()
        const ordersJson = await ordersRes.json()

        const clientMap = new Map<string, ClientData>()
        for (const c of clientsJson.clients || []) {
          clientMap.set(c.client_id, { client_id: c.client_id, last_seen: c.last_seen, data: c.data || {} })
        }

        const orderMap = new Map<string, ClientOrder>()
        for (const o of ordersJson) {
          orderMap.set(o.client_id, o)
        }

        setClients(clientMap)
        setClientOrders(orderMap)
        setLoading(false)
      } catch (error) {
        console.error('Failed to fetch:', error)
        setLoading(false)
      }
    }
    fetchAll()
  }, [])

  // Update time
  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 15000)
    return () => window.clearInterval(interval)
  }, [])

  // WebSocket updates
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

  // Actions
  const deleteClient = async (clientId: string) => {
    if (!confirm(`确定要删除客户端 "${clientId}" 吗？`)) return
    const res = await fetch(`/api/clients/${encodeURIComponent(clientId)}`, { method: 'DELETE' })
    if ((await res.json()).success) {
      setClients(prev => { const m = new Map(prev); m.delete(clientId); return m })
      setClientOrders(prev => { const m = new Map(prev); m.delete(clientId); return m })
    }
  }

  const isOnline = (lastSeen: string) => {
    return (now - new Date(lastSeen).getTime()) / 60000 < 1
  }

  const getSortedClients = () => {
    return Array.from(clients.values()).sort((a, b) => {
      const orderA = clientOrders.get(a.client_id)?.sort_order ?? 999999
      const orderB = clientOrders.get(b.client_id)?.sort_order ?? 999999
      return orderA - orderB
    })
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-lg">加载中...</div>
      </div>
    )
  }

  return (
    <div className={`min-h-screen bg-background ${websocketFailed ? 'grayscale opacity-60' : ''}`}>
      {/* Header */}
      <header className="border-b">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Activity className="h-8 w-8 text-primary" />
              <h1 className="text-2xl font-bold">客户端监控</h1>
            </div>
            <div className="flex items-center gap-4">
              <button onClick={() => setShowAlertSettings(true)} className="flex items-center gap-1 px-3 py-1.5 border rounded-md hover:bg-gray-50 dark:hover:bg-gray-800 text-sm">
                <AlertTriangle className="h-4 w-4" /> 告警设置
              </button>
              <button onClick={() => setShowSystemSettings(true)} className="flex items-center gap-1 px-3 py-1.5 border rounded-md hover:bg-gray-50 dark:hover:bg-gray-800 text-sm">
                <Settings className="h-4 w-4" /> 系统设置
              </button>
              <div className="text-sm text-muted-foreground">
                在线: {Array.from(clients.values()).filter(c => isOnline(c.last_seen)).length} / {clients.size}
              </div>
              {isConnected ? (
                <div className="flex items-center gap-1 text-green-600">
                  <Wifi className="h-4 w-4" /> <span className="text-sm">已连接</span>
                </div>
              ) : websocketFailed ? (
                <div className="flex items-center gap-1 text-red-600">
                  <WifiOff className="h-4 w-4" /> <span className="text-sm">断开</span>
                </div>
              ) : (
                <div className="flex items-center gap-1 text-yellow-600">
                  <WifiOff className="h-4 w-4" /> <span className="text-sm">重连中({reconnectAttempts}/5)</span>
                </div>
              )}
            </div>
          </div>
        </div>
      </header>

      {/* Main */}
      <main className="container mx-auto px-4 py-6">
        <ClientCardList
          clients={clients}
          clientOrders={clientOrders}
          draggedClient={draggedClient}
          now={now}
          onDragStart={handleDragStart}
          onDragOver={handleDragOver}
          onDrop={(clientId) => handleDrop(clientId, getSortedClients)}
          onDelete={deleteClient}
          isOnline={isOnline}
        />
      </main>

      {/* Modals */}
      {showAlertSettings && <AlertSettings onClose={() => setShowAlertSettings(false)} />}
      {showSystemSettings && <SystemSettings onClose={() => setShowSystemSettings(false)} />}
    </div>
  )
}
