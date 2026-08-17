export type ImageMode = 'generate' | 'edit'

export type DownloadFormat = 'png' | 'jpg'

export type DownloadStage = 'idle' | 'processing' | 'downloading'

export type MessageTone = 'info' | 'success' | 'error'

export type PromptPresetScope = ImageMode | 'all'

export type PromptApplyMode = 'replace' | 'append'

export type AppTab = 'image'

export type HealthState = 'checking' | 'online' | 'offline'

export type Theme = 'light' | 'dark'

export interface SizeOption {
  value: string
  label: string
}

export interface ModelOption {
  value: string
  label: string
  editOnly?: boolean
}

export interface PromptPreset {
	id: string
	name: string
	prompt: string
	scope: PromptPresetScope
	created_at?: string
	updated_at?: string
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

export interface ImageTask {
	id: string
	mode: ImageMode
	prompt: string
	model: string
	size: string
	quality: string
	status: 'pending' | 'succeeded' | 'failed'
	image: string
	error: string
	created_at: string
	completed_at: string | null
}

export interface HistoryPage {
	tasks: ImageTask[]
	next_cursor: string
	has_more: boolean
}
