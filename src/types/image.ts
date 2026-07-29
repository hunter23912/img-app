export type ImageMode = 'generate' | 'edit'

export type AppTab = 'image'

export type HealthState = 'checking' | 'online' | 'offline'

export interface SizeOption {
  value: string
  label: string
}

export interface ModelOption {
  value: string
  label: string
  supportsN?: boolean
  maxSize?: '1K' | '2K' | '4K'
}

export interface HealthResponse {
  ok?: boolean
  configured?: boolean
}

export interface ImageResponse {
  image?: string
  error?: string
}
