export type ImageMode = 'generate' | 'edit'

export type AppTab = 'image' | 'compress' | 'watermark'

export type HealthState = 'checking' | 'online' | 'offline'

export interface SizeOption {
  value: string
  label: string
}

export interface HealthResponse {
  ok?: boolean
  configured?: boolean
}

export interface ImageResponse {
  image?: string
  error?: string
}
