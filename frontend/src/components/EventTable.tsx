import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'

interface Event {
  id: number
  client_id: string
  event_type: string
  data: Record<string, unknown>
  status: string
  created_at: string
}

interface EventsResponse {
  total: number
  events: Event[]
}

interface EventTableProps {
  newEvent: Event | null
}

export function EventTable({ newEvent }: EventTableProps) {
  const [data, setData] = useState<EventsResponse>({ total: 0, events: [] })
  const [loading, setLoading] = useState(true)

  const fetchEvents = async () => {
    try {
      const res = await fetch('/api/events?page_size=20')
      const json = await res.json()
      setData(json)
    } catch (error) {
      console.error('Failed to fetch events:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchEvents()
    const interval = setInterval(fetchEvents, 30000) // Refresh every 30s
    return () => clearInterval(interval)
  }, [])

  // Prepend new event from WebSocket
  useEffect(() => {
    if (newEvent) {
      setData((prev) => ({
        total: prev.total + 1,
        events: [newEvent, ...prev.events.slice(0, 19)],
      }))
    }
  }, [newEvent])

  const getStatusBadge = (status: string) => {
    const variants: Record<string, "success" | "destructive" | "secondary"> = {
      success: 'success',
      error: 'destructive',
      warning: 'secondary',
    }
    return (
      <Badge variant={variants[status] || 'secondary'}>
        {status}
      </Badge>
    )
  }

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">最新事件记录</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center justify-center h-48">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>客户端</TableHead>
                <TableHead>事件类型</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>数据</TableHead>
                <TableHead>时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.events.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground">
                    暂无数据
                  </TableCell>
                </TableRow>
              ) : (
                data.events.map((event) => (
                  <TableRow key={event.id}>
                    <TableCell className="font-mono text-xs">{event.id}</TableCell>
                    <TableCell className="font-mono">{event.client_id}</TableCell>
                    <TableCell>
                      <code className="px-2 py-1 bg-muted rounded text-sm">
                        {event.event_type}
                      </code>
                    </TableCell>
                    <TableCell>{getStatusBadge(event.status)}</TableCell>
                    <TableCell className="max-w-xs truncate">
                      <code className="text-xs">
                        {JSON.stringify(event.data)}
                      </code>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatTime(event.created_at)}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}
