import { imageModel } from '../constants/image'
import type { HealthResponse, ImageResponse } from '../types/image'

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
