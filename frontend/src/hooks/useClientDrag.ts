import { useState, useCallback } from 'react'

interface DragOrder {
  client_id: string
  sort_order: number
}

interface UseClientDragOptions {
  onReorder: (orders: DragOrder[]) => Promise<void>
}

export function useClientDrag(options: UseClientDragOptions) {
  const [draggedClient, setDraggedClient] = useState<string | null>(null)
  const { onReorder } = options

  const handleDragStart = useCallback((clientId: string) => {
    setDraggedClient(clientId)
  }, [])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
  }, [])

  const handleDrop = useCallback(async (
    targetClientId: string,
    getSortedClients: () => { client_id: string }[]
  ) => {
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

    // Call the reorder callback
    await onReorder(orders)

    setDraggedClient(null)
  }, [draggedClient, onReorder])

  return {
    draggedClient,
    handleDragStart,
    handleDragOver,
    handleDrop,
  }
}
