import type { ModelOption, SizeOption } from '../types/image'

export const keepOriginalSize = 'original'

export const defaultModel = 'gpt-image-2-lite'
export const defaultSize = '720x1280'
export const defaultAspectRatio = '9:16'
export const defaultResolution = '1k'
export const defaultSeedVRSize = 'seedvr-1k'
export const seedVRModel = 'seedvr2-7b'

export const modelOptions: ModelOption[] = [
  { value: 'gpt-image-2',        label: 'gpt-image-2' },
  { value: 'gpt-image-2-eco',    label: 'gpt-image-2-eco' },
  { value: 'gpt-image-2-auto',   label: 'gpt-image-2-auto' },
  { value: 'gpt-image-2-n',      label: 'gpt-image-2-n' },
  { value: 'gpt-image-2-lite',   label: 'gpt-image-2-lite' },
  { value: 'gemini-3-pro-image', label: 'gemini-3-pro-image' },
  { value: 'gemini-3.1-flash-image', label: 'gemini-3.1-flash-image' },
  { value: seedVRModel,           label: 'seedvr2-7b（图像超分）', editOnly: true },
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

export const seedVRSizeOptions: SizeOption[] = [
  { value: 'seedvr-1k', label: '1K（保持原图比例）' },
  { value: 'seedvr-2k', label: '2K（保持原图比例）' },
  { value: 'seedvr-4k', label: '4K（保持原图比例）' },
]

const seedVRTargetLongEdges: Record<string, number> = {
  'seedvr-1k': 1024,
  'seedvr-2k': 2048,
  'seedvr-4k': 4096,
}

export function getSeedVRTargetSize(sourceSize: string, target: string) {
  const sourceDimensions = sourceSize.match(/^(\d+)x(\d+)$/)
  const targetLongEdge = seedVRTargetLongEdges[target]
  if (!sourceDimensions || !targetLongEdge) {
    throw new Error('SeedVR 输出尺寸无效。')
  }

  const sourceWidth = Number(sourceDimensions[1])
  const sourceHeight = Number(sourceDimensions[2])
  if (!sourceWidth || !sourceHeight) {
    throw new Error('无法读取原图尺寸。')
  }

  const sourceLongEdge = Math.max(sourceWidth, sourceHeight)
  const sourceShortEdge = Math.min(sourceWidth, sourceHeight)
  // SeedVR 要求输出宽高比与输入一致；按 8 的倍数取整以兼容图像模型的尺寸要求。
  const targetShortEdge = Math.max(
    8,
    Math.round((targetLongEdge * sourceShortEdge) / sourceLongEdge / 8) * 8,
  )

  return sourceWidth >= sourceHeight
    ? `${targetLongEdge}x${targetShortEdge}`
    : `${targetShortEdge}x${targetLongEdge}`
}
