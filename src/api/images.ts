import { defaultModel } from '../constants/image'
import type {
  DownloadFormat,
  HealthResponse,
  HistoryPage,
  ImageResponse,
  PromptPreset,
  PromptPresetDraft,
} from '../types/image'

const healthTimeoutMs = 5_000
const historyTimeoutMs = 10_000
const imageRequestTimeoutMs = 345_000
const downloadTimeoutMs = 45_000

export async function fetchHealth() {
  const response = await fetchWithTimeout('/api/health', undefined, healthTimeoutMs, '后端连接超时。')
  if (!response.ok) throw new Error('后端状态异常。')

  try {
    return (await response.json()) as HealthResponse
  } catch {
    throw new Error('后端状态响应无效。')
  }
}

export async function generateImage(prompt: string, size: string, model = defaultModel) {
  const response = await fetchWithTimeout(
    '/api/generate',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        model,
        prompt,
        size,
        quality: 'auto',
      }),
    },
    imageRequestTimeoutMs,
    '图片生成超时，请稍后重试。',
  )

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

  const response = await fetchWithTimeout(
    '/api/edit',
    {
      method: 'POST',
      body: formData,
    },
    imageRequestTimeoutMs,
    '图片编辑超时，请稍后重试。',
  )

  return parseImageResponse(response)
}

export async function fetchImageHistory(cursor = ''): Promise<HistoryPage> {
  const query = new URLSearchParams({ limit: '5' })
  if (cursor) query.set('cursor', cursor)
  const response = await fetchWithTimeout(
    `/api/history?${query.toString()}`,
    undefined,
    historyTimeoutMs,
    '历史记录同步超时。',
  )

  if (!response.ok) {
    throw new Error(await parseErrorResponse(response, '历史记录同步失败'))
  }

  let data: unknown
  try {
    data = await response.json()
  } catch {
    throw new Error('历史记录响应无效。')
  }

  if (!isHistoryPage(data)) {
    throw new Error('历史记录响应无效。')
  }

  return data
}

export async function deleteImageHistory(id: string) {
  const response = await fetchWithTimeout(
    `/api/history/${encodeURIComponent(id)}`,
    {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
      },
    },
    historyTimeoutMs,
    '历史记录删除超时。',
  )

  if (!response.ok) {
    throw new Error(await parseErrorResponse(response, '历史记录删除失败'))
  }
}

export async function fetchPromptPresets() {
  const response = await fetchWithTimeout('/api/presets', undefined, historyTimeoutMs, '预设同步超时。')
  if (!response.ok) throw new Error(await parseErrorResponse(response, '预设同步失败'))
  const data = (await response.json()) as { presets?: PromptPreset[] }
  if (!data || !Array.isArray(data.presets)) throw new Error('预设响应无效。')
  return data.presets
}

export async function createPromptPreset(draft: PromptPresetDraft) {
  return requestPreset('/api/presets', 'POST', draft)
}

export async function updatePromptPreset(id: string, draft: PromptPresetDraft) {
  return requestPreset(`/api/presets/${encodeURIComponent(id)}`, 'PUT', draft)
}

export async function deletePromptPreset(id: string) {
  const response = await fetchWithTimeout(`/api/presets/${encodeURIComponent(id)}`, { method: 'DELETE' }, historyTimeoutMs, '预设删除超时。')
  if (!response.ok) throw new Error(await parseErrorResponse(response, '预设删除失败'))
}

export async function importPromptPresets(presets: PromptPreset[]) {
  const response = await fetchWithTimeout('/api/presets/import', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ presets }),
  }, historyTimeoutMs, '预设迁移超时。')
  if (!response.ok) throw new Error(await parseErrorResponse(response, '预设迁移失败'))
}

export async function downloadImage(input: {
  source: string
  format: DownloadFormat
  quality: number
}) {
  const response = await fetchWithTimeout(
    '/api/download/image',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(input),
    },
    downloadTimeoutMs,
    '图片处理超时，请稍后重试。',
  )

  if (!response.ok) {
    throw new Error(await parseErrorResponse(response, '图片下载失败'))
  }

  return {
    response,
    filename: getDownloadFilename(response.headers.get('Content-Disposition'), input.format),
  }
}

async function parseImageResponse(response: Response) {
  let data: ImageResponse
  try {
    data = (await response.json()) as ImageResponse
  } catch {
    throw new Error(messageForStatus(response.status, '', '图片服务返回异常，请稍后重试。'))
  }

  if (!response.ok) {
    throw new Error(messageForStatus(response.status, data.error, '图片生成失败，请稍后重试。'))
  }

  if (!data.image) {
    throw new Error('后端没有返回图片数据。')
  }

  return data.image
}

async function parseErrorResponse(response: Response, fallback: string) {
  const contentType = response.headers.get('Content-Type') ?? ''
  if (contentType.includes('application/json')) {
    try {
      const data = (await response.json()) as { error?: string }
      return messageForStatus(response.status, data.error, fallback)
    } catch {
      // Fall through to the status-based message when the error body is malformed.
    }
  }

  return messageForStatus(response.status, '', fallback)
}

function isHistoryPage(value: unknown): value is HistoryPage {
  if (!value || typeof value !== 'object' || !('tasks' in value)) return false
  const page = value as { tasks?: unknown; next_cursor?: unknown; has_more?: unknown }
  return Array.isArray(page.tasks) && typeof page.next_cursor === 'string' && typeof page.has_more === 'boolean'
}

async function requestPreset(path: string, method: 'POST' | 'PUT', draft: PromptPresetDraft) {
  const response = await fetchWithTimeout(path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(draft),
  }, historyTimeoutMs, '预设保存超时。')
  if (!response.ok) throw new Error(await parseErrorResponse(response, '预设保存失败'))
  return (await response.json()) as PromptPreset
}

async function fetchWithTimeout(
  input: RequestInfo | URL,
  init: RequestInit | undefined,
  timeoutMs: number,
  timeoutMessage: string,
) {
  const controller = new AbortController()
  const timeout = window.setTimeout(() => controller.abort(), timeoutMs)

  try {
    return await fetch(input, { ...init, signal: controller.signal })
  } catch (error) {
    if (controller.signal.aborted) {
      throw new Error(timeoutMessage, { cause: error })
    }

    if (error instanceof TypeError) {
      throw new Error('无法连接服务器，请检查网络或后端服务。', { cause: error })
    }

    throw new Error('请求未能完成，请稍后重试。', { cause: error })
  } finally {
    window.clearTimeout(timeout)
  }
}

function messageForStatus(status: number, rawMessage: string | undefined, fallback: string) {
  const detail = rawMessage?.trim() ?? ''
  const message = detail.toLowerCase()

  if (message.includes('img_api_key') || message.includes('api key')) {
    return '图片服务尚未正确配置。'
  }
  if (message.includes('rate limit') || message.includes('concurrent') || status === 429) {
    return '请求过于频繁，请稍后重试。'
  }
  if (message.includes('timeout') || message.includes('timed out') || message.includes('sync wait') || status === 504) {
    return '图片生成超时，请稍后重试。'
  }
  if (message.includes('permission') || message.includes('forbidden') || status === 401 || status === 403) {
    return '图片服务拒绝了请求，请检查 API 权限。'
  }
  if (message.includes('service unavailable') || message.includes('queue') || status === 503) {
    return '图片服务繁忙，请稍后重试。'
  }
  if (message.includes('call relay') || message.includes('network error')) {
    return '暂时无法连接图片服务，请稍后重试。'
  }
  if (message.includes('no image') || message.includes('image data')) {
    return '图片服务未返回有效图片，请重试。'
  }
  if (message.includes('source url was not generated')) {
    return '图片来源已失效，请重新生成后下载。'
  }
  if (message.includes('too large')) {
    return '图片过大，暂时无法处理。'
  }
  if (message.includes('format') || message.includes('quality')) {
    return '下载参数无效，请调整后重试。'
  }

  // 后端可能返回审核拒绝、内容限制等无法穷举的具体原因。
  // 已知的技术错误在上面统一处理，其余后端消息应保留给用户，避免被状态码覆盖。
  if (detail) return detail

  if (status === 400) return '请求内容有误，请检查后重试。'
  if (status === 404) return '请求的服务不存在。'
  if (status >= 500) return '图片服务暂时不可用，请稍后重试。'

  return fallback
}

function getDownloadFilename(contentDisposition: string | null, format: DownloadFormat) {
  const fallback = `gpt-image.${format}`
  if (!contentDisposition) return fallback

  const match = contentDisposition.match(/filename\*?=(?:UTF-8'')?"?([^";]+)"?/i)
  if (!match?.[1]) return fallback

  try {
    return decodeURIComponent(match[1])
  } catch {
    return match[1]
  }
}
