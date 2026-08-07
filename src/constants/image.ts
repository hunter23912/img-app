import type { ModelOption, SizeOption } from '../types/image'

export const keepOriginalSize = 'original'

export const defaultModel = 'gpt-image-2-lite'
export const defaultSize = '1152x2048'

export const modelOptions: ModelOption[] = [
  { value: 'gpt-image-2',        label: 'gpt-image-2' },
  { value: 'gpt-image-2-eco',    label: 'gpt-image-2-eco' },
  { value: 'gpt-image-2-auto',   label: 'gpt-image-2-auto' },
  { value: 'gpt-image-2-n',      label: 'gpt-image-2-n' },
  { value: 'gpt-image-2-lite',   label: 'gpt-image-2-lite' },
  { value: 'gemini-3-pro-image', label: 'gemini-3-pro-image' },
  { value: 'gemini-3.1-flash-image', label: 'gemini-3.1-flash-image' },
]

export const sizeOptions: SizeOption[] = [
  { value: '2048x2048', label: '方图' },
  { value: '1152x2048', label: '9:16' },
]
