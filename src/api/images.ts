import { defaultModel } from '../constants/image'
import type {
  DownloadFormat,
  ModelOption,
  HealthResponse,
  HistoryPage,
  ImageResponse,
  ImageProfile,
  ImageProfileDraft,
  ImageSettings,
  PromptPreset,
  PromptPresetDraft,
} from '../types/image'

const healthTimeoutMs = 5_000
const historyTimeoutMs = 10_000
const imageRequestTimeoutMs = 600_000
const downloadTimeoutMs = 6 * 60_000

type ImagePartialCallback = (image: string) => void

export async function fetchImageModels(): Promise<ModelOption[]> {
  const response = await fetchWithTimeout('/api/models', undefined, healthTimeoutMs, '模型列表读取超时。')
  if (!response.ok) throw new Error(await parseErrorResponse(response, '模型列表读取失败'))

  let data: { models?: ModelOption[] }
  try {
    data = (await response.json()) as { models?: ModelOption[] }
  } catch {
    throw new Error('模型列表响应无效。')
  }
  if (!data || !Array.isArray(data.models) || data.models.some((model) => !isModelOption(model))) {
    throw new Error('模型列表响应无效。')
  }
  return data.models
}

export async function createImageModel(model: string): Promise<ModelOption> {
  const response = await fetchWithTimeout(
    '/api/models',
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ model }),
    },
    historyTimeoutMs,
    '自定义模型添加超时。',
  )
  if (!response.ok) throw new Error(await parseErrorResponse(response, '自定义模型添加失败'))
  let data: unknown
  try {
    data = await response.json()
  } catch {
    throw new Error('模型响应无效。')
  }
  if (!isModelOption(data)) throw new Error('模型响应无效。')
  return data
}

export async function deleteImageModel(id: string): Promise<void> {
  const response = await fetchWithTimeout(
    `/api/models/${encodeURIComponent(id)}`,
    { method: 'DELETE' },
    historyTimeoutMs,
    '自定义模型删除超时。',
  )
  if (!response.ok) throw new Error(await parseErrorResponse(response, '自定义模型删除失败'))
}

export async function fetchHealth() {
  const response = await fetchWithTimeout('/api/health', undefined, healthTimeoutMs, '后端连接超时。')
  if (!response.ok) throw new Error('后端状态异常。')

  try {
    return (await response.json()) as HealthResponse
  } catch {
    throw new Error('后端状态响应无效。')
  }
}

export async function fetchImageSettings(): Promise<ImageSettings> {
  const response = await fetchWithTimeout(
    '/api/settings/image',
    undefined,
    healthTimeoutMs,
    '图片服务配置读取超时。',
  )
  if (!response.ok) throw new Error(await parseErrorResponse(response, '图片服务配置读取失败'))

  let data: ImageSettings
  try {
    data = (await response.json()) as ImageSettings
  } catch {
    throw new Error('图片服务配置响应无效。')
  }
  if (typeof data.endpoint !== 'string' || typeof data.api_key !== 'string') {
    throw new Error('图片服务配置响应无效。')
  }
  return data
}

export async function saveImageSettings(settings: ImageSettings): Promise<ImageSettings> {
  const response = await fetchWithTimeout(
    '/api/settings/image',
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(settings),
    },
    historyTimeoutMs,
    '图片服务配置保存超时。',
  )
  if (!response.ok) throw new Error(await parseErrorResponse(response, '图片服务配置保存失败'))

  let data: ImageSettings
  try {
    data = (await response.json()) as ImageSettings
  } catch {
    throw new Error('图片服务配置响应无效。')
  }
  if (typeof data.endpoint !== 'string' || typeof data.api_key !== 'string') {
    throw new Error('图片服务配置响应无效。')
  }
  return data
}

export async function fetchImageProfiles(): Promise<ImageProfile[]> {
  const response = await fetchWithTimeout('/api/image-profiles', undefined, healthTimeoutMs, '中转站配置读取超时。')
  if (!response.ok) throw new Error(await parseErrorResponse(response, '中转站配置读取失败'))
  let data: { profiles?: ImageProfile[] }
  try {
    data = (await response.json()) as { profiles?: ImageProfile[] }
  } catch {
    throw new Error('中转站配置响应无效。')
  }
  if (!data || !Array.isArray(data.profiles)) throw new Error('中转站配置响应无效。')
  return data.profiles
}

export async function createImageProfile(draft: ImageProfileDraft): Promise<ImageProfile> {
  return requestImageProfile('/api/image-profiles', 'POST', draft, '中转站配置创建失败')
}

export async function updateImageProfile(id: string, draft: ImageProfileDraft): Promise<ImageProfile> {
  return requestImageProfile(`/api/image-profiles/${encodeURIComponent(id)}`, 'PUT', draft, '中转站配置保存失败')
}

export async function activateImageProfile(id: string): Promise<ImageProfile> {
  const response = await fetchWithTimeout(
    `/api/image-profiles/${encodeURIComponent(id)}/activate`,
    { method: 'POST' },
    historyTimeoutMs,
    '中转站切换超时。',
  )
  if (!response.ok) throw new Error(await parseErrorResponse(response, '中转站切换失败'))
  return parseImageProfileResponse(response)
}

export async function deleteImageProfile(id: string): Promise<void> {
  const response = await fetchWithTimeout(
    `/api/image-profiles/${encodeURIComponent(id)}`,
    { method: 'DELETE' },
    historyTimeoutMs,
    '中转站配置删除超时。',
  )
  if (!response.ok) throw new Error(await parseErrorResponse(response, '中转站配置删除失败'))
}

async function requestImageProfile(
  url: string,
  method: 'POST' | 'PUT',
  draft: ImageProfileDraft,
  fallbackMessage: string,
) {
  const response = await fetchWithTimeout(
    url,
    {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(draft),
    },
    historyTimeoutMs,
    fallbackMessage,
  )
  if (!response.ok) throw new Error(await parseErrorResponse(response, fallbackMessage))
  return parseImageProfileResponse(response)
}

async function parseImageProfileResponse(response: Response): Promise<ImageProfile> {
  let data: ImageProfile
  try {
    data = (await response.json()) as ImageProfile
  } catch {
    throw new Error('中转站配置响应无效。')
  }
  if (
    !data ||
    typeof data.id !== 'string' ||
    typeof data.name !== 'string' ||
    typeof data.endpoint !== 'string' ||
    typeof data.api_key !== 'string' ||
    typeof data.is_active !== 'boolean'
  ) {
    throw new Error('中转站配置响应无效。')
  }
  return data
}

export async function generateImage(prompt: string, size: string, model = defaultModel, onPartialImage?: ImagePartialCallback) {
  return requestImage(
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
        background: 'auto',
        moderation: 'auto',
        output_format: 'png',
        n: 1,
      }),
    },
    '图片生成超时，请稍后重试。',
    onPartialImage,
  )
}

export async function editImage(input: {
  prompt: string
  size: string
  images: File[]
  model?: string
  onPartialImage?: ImagePartialCallback
}) {
  const formData = new FormData()
  formData.append('model', input.model ?? defaultModel)
  formData.append('prompt', input.prompt)
  formData.append('quality', 'auto')
  formData.append('moderation', 'auto')
  formData.append('output_format', 'png')
  formData.append('n', '1')
  for (const image of input.images) {
    formData.append('image', image)
  }

  if (input.size) {
    formData.append('size', input.size)
  }

  return requestImage(
    '/api/edit',
    {
      method: 'POST',
      body: formData,
    },
    '图片编辑超时，请稍后重试。',
    input.onPartialImage,
  )
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
	if (/^https:\/\//i.test(input.source)) {
		try {
			return await downloadExternalImage(input)
		} catch (error) {
			// 没有 CORS、跨域读取失败或 CDN 不支持浏览器下载时，回退到 Go 后端。
			if (!(error instanceof TypeError)) throw error
		}
	}
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

async function downloadExternalImage(input: { source: string; format: DownloadFormat; quality: number }) {
	const controller = new AbortController()
	const timeout = window.setTimeout(() => controller.abort(), downloadTimeoutMs)
	try {
		const response = await fetch(input.source, { signal: controller.signal, cache: 'force-cache' })
		if (!response.ok) throw new TypeError(`external image returned ${response.status}`)
		const blob = await response.blob()
		if (input.format === 'png') {
			return { response: new Response(blob, { headers: { 'Content-Type': 'image/png', 'Content-Disposition': 'attachment; filename="gpt-image.png"' } }), filename: 'gpt-image.png' }
		}
		const bitmap = await createImageBitmap(blob)
		const canvas = document.createElement('canvas')
		canvas.width = bitmap.width
		canvas.height = bitmap.height
		const ctx = canvas.getContext('2d')
		if (!ctx) throw new TypeError('canvas is unavailable')
		ctx.fillStyle = '#fff'
		ctx.fillRect(0, 0, canvas.width, canvas.height)
		ctx.drawImage(bitmap, 0, 0)
		bitmap.close()
		const jpg = await new Promise<Blob>((resolve, reject) => canvas.toBlob((value) => value ? resolve(value) : reject(new TypeError('JPG conversion failed')), 'image/jpeg', input.quality / 100))
		return { response: new Response(jpg, { headers: { 'Content-Type': 'image/jpeg', 'Content-Disposition': 'attachment; filename="gpt-image.jpg"' } }), filename: 'gpt-image.jpg' }
	} finally {
		window.clearTimeout(timeout)
	}
}

async function requestImage(
  input: RequestInfo | URL,
  init: RequestInit,
  timeoutMessage: string,
  onPartialImage?: ImagePartialCallback,
) {
  const controller = new AbortController()
  const timeout = window.setTimeout(() => controller.abort(), imageRequestTimeoutMs)

  try {
    const response = await fetch(input, { ...init, signal: controller.signal, cache: 'no-store' })
    return await parseImageResponse(response, onPartialImage)
  } catch (error) {
    if (controller.signal.aborted) {
      throw new Error(timeoutMessage, { cause: error })
    }

    if (error instanceof TypeError) {
      throw new Error('无法连接服务器，请检查网络或后端服务。', { cause: error })
    }

    throw error instanceof Error ? error : new Error('请求未能完成，请稍后重试。', { cause: error })
  } finally {
    window.clearTimeout(timeout)
  }
}

async function parseImageResponse(response: Response, onPartialImage?: ImagePartialCallback) {
  if (response.headers.get('Content-Type')?.toLowerCase().includes('text/event-stream')) {
    return readImageStream(response, onPartialImage)
  }

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

async function readImageStream(response: Response, onPartialImage?: ImagePartialCallback) {
  if (!response.body) throw new Error('图片服务未返回可读取的流式响应。')

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let completedImage = ''
  let sawEvent = false

  const processBlock = (block: string) => {
    const data = block
      .split(/\r?\n/)
      .filter((line) => line.startsWith('data:'))
      .map((line) => line.slice(5).replace(/^ /, ''))
      .join('\n')
      .trim()
    if (!data || data === '[DONE]') return false

    sawEvent = true
    let event: { type?: unknown; image?: unknown; b64_json?: unknown; partial_image_b64?: unknown; error?: unknown }
    try {
      event = JSON.parse(data) as typeof event
    } catch {
      throw new Error('图片流响应格式无效。')
    }

    const type = typeof event.type === 'string' ? event.type : ''
    const partial = typeof event.image === 'string'
      ? event.image
      : typeof event.b64_json === 'string'
        ? `data:image/png;base64,${event.b64_json}`
        : typeof event.partial_image_b64 === 'string'
          ? `data:image/png;base64,${event.partial_image_b64}`
          : ''
    if (type.includes('partial_image') && partial) {
      onPartialImage?.(partial)
      return false
    }

    if (type.endsWith('.failed') || type.endsWith('.error')) {
      const error = typeof event.error === 'string'
        ? event.error
        : event.error && typeof event.error === 'object' && 'message' in event.error && typeof event.error.message === 'string'
          ? event.error.message
          : '图片流请求失败。'
      throw new Error(error)
    }

    if (type.endsWith('.completed') && typeof event.image === 'string' && event.image) {
      completedImage = event.image
      return true
    }
    return false
  }

  try {
    while (!completedImage) {
      const { value, done } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })

      let separator = buffer.search(/\r?\n\r?\n/)
      while (separator >= 0) {
        const match = buffer.match(/\r?\n\r?\n/)
        const block = buffer.slice(0, separator)
        buffer = buffer.slice(separator + (match?.[0].length ?? 2))
        if (processBlock(block)) break
        separator = buffer.search(/\r?\n\r?\n/)
      }
    }

    if (!completedImage) {
      buffer += decoder.decode()
      if (buffer.trim()) processBlock(buffer)
    }
  } finally {
    await reader.cancel().catch(() => undefined)
  }

  if (completedImage) return completedImage
  if (!sawEvent) throw new Error('图片流响应为空。')
  throw new Error('图片流未返回最终图片。')
}

function isModelOption(value: unknown): value is ModelOption {
  if (!value || typeof value !== 'object') return false
  const model = value as Partial<ModelOption>
  return (
    typeof model.id === 'string' &&
    typeof model.value === 'string' &&
    typeof model.label === 'string' &&
    typeof model.built_in === 'boolean'
  )
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
  if (message.includes('failed to load image models')) {
    return '模型列表加载失败，请检查后端服务是否已重启。'
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
