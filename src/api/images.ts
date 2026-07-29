import { defaultModel } from '../constants/image'
import type { HealthResponse, ImageResponse } from '../types/image'

export async function fetchHealth() {
  const response = await fetch('/api/health')
  return (await response.json()) as HealthResponse
}

export async function generateImage(prompt: string, size: string, model = defaultModel, n = 1) {
  const response = await fetch('/api/generate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      model,
      prompt,
      size,
      quality: 'auto',
      n,
    }),
  })

  return parseImageResponse(response)
}

export async function editImage(input: {
  prompt: string
  size: string
  image: File
  model?: string
}) {
  const formData = new FormData()
  formData.append('model', input.model ?? defaultModel)
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
