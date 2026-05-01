// 共享类型定义

export interface Channel {
  id: number
  name: string
  type: string
}

export interface ClientOrder {
  client_id: string
  sort_order: number
  weather_enabled: boolean
  channel_ids: number[]
  is_primary: boolean
  hidden: boolean
}

export interface Disk {
  name: string
  label?: string
  total_gb: number
  used_gb: number
  available_gb: number
  load_percent: number
}

export interface ClientData {
  client_id: string
  last_seen: string
  data: {
    location?: {
      city?: string
      district?: string
      ip?: string
    }
    memory?: {
      used_gb?: number
      available_gb?: number
      load_percent?: number
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
