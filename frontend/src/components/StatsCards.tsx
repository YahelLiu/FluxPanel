import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Activity, Users, AlertTriangle, CheckCircle } from 'lucide-react'

interface Summary {
  online_clients: number
  today_events: number
  today_errors: number
  event_type_counts: Record<string, number>
  status_counts: Record<string, number>
}

interface StatsCardsProps {
  summary: Summary | null
  loading: boolean
}

export function StatsCards({ summary, loading }: StatsCardsProps) {
  const cards = [
    {
      title: '在线客户端',
      value: summary?.online_clients ?? 0,
      icon: Users,
      color: 'text-blue-600',
      bgColor: 'bg-blue-100',
    },
    {
      title: '今日事件数',
      value: summary?.today_events ?? 0,
      icon: Activity,
      color: 'text-green-600',
      bgColor: 'bg-green-100',
    },
    {
      title: '今日错误数',
      value: summary?.today_errors ?? 0,
      icon: AlertTriangle,
      color: 'text-red-600',
      bgColor: 'bg-red-100',
    },
    {
      title: '成功率',
      value: summary?.today_events
        ? ((1 - (summary.today_errors / summary.today_events)) * 100).toFixed(1) + '%'
        : '100%',
      icon: CheckCircle,
      color: 'text-emerald-600',
      bgColor: 'bg-emerald-100',
    },
  ]

  if (loading) {
    return (
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <Card key={i} className="animate-pulse">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <div className="h-4 w-24 bg-muted rounded" />
            </CardHeader>
            <CardContent>
              <div className="h-8 w-16 bg-muted rounded" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {cards.map((card) => (
        <Card key={card.title}>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {card.title}
            </CardTitle>
            <div className={`p-2 rounded-full ${card.bgColor}`}>
              <card.icon className={`h-4 w-4 ${card.color}`} />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{card.value}</div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
