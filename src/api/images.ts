import { imageModel } from '../constants/image'
import type { HealthResponse, ImageResponse } from '../types/image'
import type { CompressionOutputType } from '../utils/compressImage'

export async function fetchHealth() {
  const response = await fetch('/api/health')
  return (await response.json()) as HealthResponse
}

export async function generateImage(prompt: string, size: string) {
  const response = await fetch('/api/generate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      model: imageModel,
      prompt,
      size,
      quality: 'auto',
    }),
  })

  return parseImageResponse(response)
}

export async function editImage(input: {
  prompt: string
  size: string
  image: File
}) {
  const formData = new FormData()
  formData.append('model', imageModel)
  formData.append('prompt', input.prompt)
  formData.append('quality', 'auto')
  formData.append('image', input.image)

  if (input.size) {
    formData.append('size', input.size)
  }

  const response = await fetch('/api/edit', {
    method: 'POST',
    body: formData,
  })

  return parseImageResponse(response)
}

export async function compressImageOnServer(input: {
  image: File
  outputType: CompressionOutputType
  quality: number
}) {
  const formData = new FormData()
  formData.append('image', input.image)
  formData.append(
    'output',
    input.outputType === 'auto' ? 'auto' : input.outputType === 'image/jpeg' ? 'jpg' : 'png'
  )
  formData.append('quality', String(Math.round(input.quality * 100)))

  const response = await fetch('/api/compress', {
    method: 'POST',
    body: formData,
  })

  if (!response.ok) {
    const data = (await response.json().catch(() => null)) as { error?: string } | null
    throw new Error(data?.error || `压缩失败：${response.status}`)
  }

  const blob = await response.blob()
  return {
    blob,
    url: URL.createObjectURL(blob),
    width: Number(response.headers.get('X-Image-Width')) || 0,
    height: Number(response.headers.get('X-Image-Height')) || 0,
    sourceFormat: response.headers.get('X-Image-Source-Format') || '',
    output: response.headers.get('X-Image-Output') || '',
    originalBytes: Number(response.headers.get('X-Original-Bytes')) || input.image.size,
    compressedBytes: Number(response.headers.get('X-Compressed-Bytes')) || blob.size,
    savedBytes: Number(response.headers.get('X-Saved-Bytes')) || input.image.size - blob.size,
  }
}

export async function removeWatermark(input: {
  image: File
  mask: Blob
}) {
  const formData = new FormData()
  formData.append('image', input.image)
  formData.append('mask', input.mask, 'mask.png')

  const response = await fetch('/api/watermark/remove', {
    method: 'POST',
    body: formData,
  })

  if (!response.ok) {
    const data = (await response.json().catch(() => null)) as { error?: string } | null
    throw new Error(data?.error || `去水印失败：${response.status}`)
  }

  const blob = await response.blob()
  return {
    blob,
    url: URL.createObjectURL(blob),
    mode: response.headers.get('X-Watermark-Mode') || '',
  }
}

async function parseImageResponse(response: Response) {
  const data = (await response.json()) as ImageResponse

  if (!response.ok) {
    throw new Error(data.error || `请求失败：${response.status}`)
  }

  if (!data.image) {
    throw new Error('后端没有返回图片数据。')
  }

  return data.image
}
