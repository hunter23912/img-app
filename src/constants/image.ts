import type { ModelOption, SizeOption } from '../types/image'

export const keepOriginalSize = 'original'

export const defaultModel = 'gpt-image-2'

export const modelOptions: ModelOption[] = [
  { value: 'gpt-image-2',        label: 'gpt-image-2' },
  { value: 'gpt-image-2-eco',    label: 'gpt-image-2-eco' },
  { value: 'gpt-image-2-auto',   label: 'gpt-image-2-auto' },
  { value: 'gpt-image-2-n',      label: 'gpt-image-2-n',      supportsN: true },
  { value: 'gpt-image-2-lite',   label: 'gpt-image-2-lite',   maxSize: '1K', supportsN: true },
  { value: 'gemini-3-pro-image', label: 'gemini-3-pro-image' },
  { value: 'gemini-3.1-flash-image', label: 'gemini-3.1-flash-image' },
]

export const sizeOptions: SizeOption[] = [
  { value: '1024x1024', label: '1:1 方图（1024x1024）' },
  { value: '768x1024',  label: '竖图 3:4（768x1024）' },
  { value: '576x1024',  label: '9:16 竖图（576x1024）' },
  { value: '1152x2048', label: '9:16 高清（1152x2048）' },
  { value: '1312x2848', label: 'iPhone 壁纸（1312×2848）' },
]

export function getAvailableSizes(model: string): SizeOption[] {
  const opt = modelOptions.find(m => m.value === model)
  if (opt?.maxSize === '1K') {
    return sizeOptions.filter(s => {
      const [w, h] = s.value.split('x').map(Number)
      return Math.max(w, h) <= 1024
    })
  }
  return sizeOptions
}
