import type { ModelOption, SizeOption } from '../types/image'

export const keepOriginalSize = 'original'

export const defaultModel = 'gpt-image-2'
export const defaultSize = '720x1280'
export const defaultAspectRatio = '9:16'
export const defaultResolution = '1k'

export const builtInModelOptions: ModelOption[] = [
  { id: 'builtin:gpt-image-2', value: 'gpt-image-2', label: 'gpt-image-2', built_in: true },
  { id: 'builtin:grok-imagine-image-2.0', value: 'grok-imagine-image-2.0', label: 'grok-imagine-image-2.0', built_in: true },
  { id: 'builtin:gemini3.1-flash-image', value: 'gemini3.1-flash-image', label: 'gemini3.1-flash-image', built_in: true },
]

export const aspectRatioOptions: SizeOption[] = [
  { value: '1:1', label: '1:1' },
  { value: '9:16', label: '9:16' },
]

export const resolutionOptions: SizeOption[] = [
  { value: '1k', label: '1k' },
  { value: '2k', label: '2k' },
]

const standardSizes: Record<string, Record<string, string>> = {
  '1:1': {
    '1k': '1024x1024',
    '2k': '2048x2048',
  },
  '9:16': {
    '1k': '720x1280',
    '2k': '1440x2560',
  },
}

export function getStandardImageSize(aspectRatio: string, resolution: string) {
  return standardSizes[aspectRatio]?.[resolution] ?? defaultSize
}

export function getResolutionOptions(aspectRatio: string) {
  const sizes = standardSizes[aspectRatio] ?? standardSizes[defaultAspectRatio]
  return resolutionOptions.map((option) => ({
    ...option,
    label: sizes[option.value]?.replace('x', '*') ?? option.label,
  }))
}
