import { ClientCard } from './ClientCard'
import type { ClientData, ClientOrder } from './types'

export type { ClientData, ClientOrder } from './types'

interface ClientCardListProps {
  clients: Map<string, ClientData>
  clientOrders: Map<string, ClientOrder>
  draggedClient: string | null
  now: number
  onDragStart: (clientId: string) => void
  onDragOver: (e: React.DragEvent) => void
  onDrop: (clientId: string) => void
  onDelete: (clientId: string) => void
  isOnline: (lastSeen: string) => boolean
}

export function ClientCardList({
  clients, clientOrders, draggedClient, now,
  onDragStart, onDragOver, onDrop, onDelete, isOnline,
}: ClientCardListProps) {
  const sortedClients = Array.from(clients.values())
    .filter(c => !clientOrders.get(c.client_id)?.hidden)
    .sort((a, b) => {
      const orderA = clientOrders.get(a.client_id)?.sort_order ?? 999999
      const orderB = clientOrders.get(b.client_id)?.sort_order ?? 999999
      return orderA - orderB
    })

  if (clients.size === 0) {
    return (
      <div className="text-center text-muted-foreground py-20">
        暂无客户端数据，等待客户端上报...
      </div>
    )
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {sortedClients.map(client => (
        <ClientCard
          key={client.client_id}
          client={client}
          isOnline={isOnline(client.last_seen)}
          isDragging={draggedClient === client.client_id}
          now={now}
          onDragStart={onDragStart}
          onDragOver={onDragOver}
          onDrop={onDrop}
          onDelete={onDelete}
        />
      ))}
    </div>
  )
}
