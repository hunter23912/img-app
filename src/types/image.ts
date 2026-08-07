export type ImageMode = 'generate' | 'edit'

export type DownloadFormat = 'png' | 'jpg'

export type DownloadStage = 'idle' | 'processing' | 'downloading'

export type MessageTone = 'info' | 'success' | 'error'

export type PromptPresetScope = ImageMode | 'all'

export type PromptApplyMode = 'replace' | 'append'

export type AppTab = 'image'

export type HealthState = 'checking' | 'online' | 'offline'

export interface SizeOption {
  value: string
  label: string
}

export interface ModelOption {
  value: string
  label: string
}

export interface PromptPreset {
  id: string
  name: string
  prompt: string
  scope: PromptPresetScope
}

export type PromptPresetDraft = Omit<PromptPreset, 'id'>

export interface HealthResponse {
  ok?: boolean
  configured?: boolean
}

export interface ImageResponse {
  image?: string
  error?: string
}
