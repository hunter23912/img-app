import type { SizeOption } from '../types/image'

export const imageModel = 'gpt-image-2'

export const keepOriginalSize = 'original'

export const sizeOptions: SizeOption[] = [
  { value: '1024x1024', label: '1:1 方图（1024x1024）' },
  { value: '1024x1536', label: '竖图省钱档（1024x1536）' },
  { value: '1152x2048', label: '9:16 手机壁纸（1152x2048）' },
]
