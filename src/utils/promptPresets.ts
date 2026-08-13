import type {
  PromptApplyMode,
  PromptPreset,
  PromptPresetDraft,
  PromptPresetScope,
} from '../types/image'

export const promptPresetStorageKey = 'img-app.prompt-presets.v1'
export const promptApplyModeStorageKey = 'img-app.prompt-apply-mode.v1'
export const maxPromptPresets = 50
export const maxPresetNameLength = 40
export const maxPresetPromptLength = 5_000

const storageVersion = 2
const legacyStorageVersion = 1

const starterPresets: PromptPreset[] = [
  {
    id: 'starter-remove-text-watermark',
    name: '去除水印',
    scope: 'edit',
    prompt:
      '移除图片中的所有文字、Logo 和水印，并根据周围纹理、光影和透视自然补全缺失区域。',
  },
  {
    id: 'starter-remove-face-sticker-mosaic',
    name: '去脸部遮挡',
    scope: 'edit',
    prompt:
      '去除人物脸部的贴图、表情贴纸、马赛克和遮挡元素，根据脸部轮廓、五官、皮肤纹理、光影和透视自然还原。',
  },
  {
    id: 'starter-enhance-quality',
    name: '提升画质',
    scope: 'edit',
    prompt:
      '超清画质，提升图片清晰度、细节和质感，修复模糊、噪点、压缩痕迹和锯齿。',
  },
  {
    id: 'starter-reduce-gpt-image-texture',
    name: '减少鱼鳞纹',
    scope: 'edit',
    prompt:
      '减少鱼鳞状纹理、碎片状褶皱、重复片状细节和不自然的表面噪声。',
  },
  {
    id: 'starter-natural-body-details',
    name: '人物细节协调',
    scope: 'edit',
    prompt:
      '优化人物的手部、手指、脚部、脚趾、四肢比例、头发和发丝结构，使其自然、合理、协调并符合人体逻辑。保持人物姿态、表情、服装、构图和原有画风不变，避免新增肢体、手指粘连或发丝杂乱。',
  },
  {
    id: 'starter-anime-clean-lines',
    name: '漫画清线',
    scope: 'edit',
    prompt:
      '保持或转换为干净的二次元漫画画风，统一线稿粗细和结构，减少画面中杂乱、断裂、重复和无意义的线条，整理背景与服装细节，强化主体轮廓，保持构图、人物特征、色彩和主要内容不变。',
  },
  {
    id: 'female-generate',
    name: '角色生成',
    scope: 'generate',
    prompt:
      '生成一张高冷御姐图，长发，穿着网红网纱披肩式防晒衣，衣服具有层次感，垂坠感，人物手脚合理协调流畅，符合逻辑，减少画面褶皱噪点，保持干净。',
  },
]

const starterPresetNameMigrations: Record<
  string,
  { from: string[]; to: string }
> = {
  'starter-remove-text-watermark': {
    from: ['去除文字水印', '去除图片文字水印'],
    to: '去除水印',
  },
  'starter-remove-face-sticker-mosaic': {
    from: ['去除脸部贴图/马赛克'],
    to: '去脸部遮挡',
  },
  'starter-reduce-gpt-image-texture': {
    from: ['减少鱼鳞纹理/碎片褶皱'],
    to: '减少鱼鳞纹',
  },
  'starter-natural-body-details': {
    from: ['人物手脚头发协调'],
    to: '人物细节协调',
  },
  'starter-anime-clean-lines': {
    from: ['二次元漫画清理线条'],
    to: '漫画清线',
  },
}

interface PromptPresetStore {
  version: number
  presets: PromptPreset[]
}

interface ParsedPromptPresetStore {
  version: number
  presets: PromptPreset[]
}

export interface PromptPresetLoadResult {
  presets: PromptPreset[]
  warning: string
}

export interface PromptPresetOperationResult {
  ok: boolean
  error?: string
}

export function loadPromptPresets(): PromptPresetLoadResult {
  try {
    const raw = window.localStorage.getItem(promptPresetStorageKey)
    if (raw === null) {
      const presets = cloneStarterPresets()
      const error = persistPromptPresets(presets)
      return {
        presets,
        warning: error ?? '',
      }
    }

    const parsed = parsePromptPresetStore(raw)
    const mergedPresets =
      parsed.version === legacyStorageVersion
        ? mergeStarterPresets(parsed.presets)
        : parsed.presets
    const normalized = normalizeStarterPresetNames(mergedPresets)
    const warning =
      parsed.version === legacyStorageVersion || normalized.changed
        ? persistPromptPresets(normalized.presets) ?? ''
        : ''

    return { presets: normalized.presets, warning }
  } catch {
    return {
      presets: cloneStarterPresets(),
      warning: '本地预设数据异常，已加载默认预设。保存后将覆盖异常数据。',
    }
  }
}

export function parsePromptPresetStorageValue(raw: string | null): PromptPresetLoadResult {
  if (raw === null) {
    return { presets: cloneStarterPresets(), warning: '' }
  }

  try {
    const parsed = parsePromptPresetStore(raw)
    const mergedPresets =
      parsed.version === legacyStorageVersion
        ? mergeStarterPresets(parsed.presets)
        : parsed.presets
    return {
      presets: normalizeStarterPresetNames(mergedPresets).presets,
      warning: '',
    }
  } catch {
    return {
      presets: cloneStarterPresets(),
      warning: '其他页面写入了无效预设数据，已加载默认预设。',
    }
  }
}

export function persistPromptPresets(presets: PromptPreset[]): string | null {
  try {
    const value: PromptPresetStore = { version: storageVersion, presets }
    window.localStorage.setItem(promptPresetStorageKey, JSON.stringify(value))
    return null
  } catch {
    return '无法保存到本机，请检查浏览器存储权限或剩余空间。'
  }
}

export function loadPromptApplyMode(): PromptApplyMode {
  try {
    const value = window.localStorage.getItem(promptApplyModeStorageKey)
    return isPromptApplyMode(value) ? value : 'replace'
  } catch {
    return 'replace'
  }
}

export function persistPromptApplyMode(mode: PromptApplyMode) {
  try {
    window.localStorage.setItem(promptApplyModeStorageKey, mode)
  } catch {
    // Applying a preset should still work when browser storage is unavailable.
  }
}

export function applyPromptPreset(
  currentPrompt: string,
  presetPrompt: string,
  mode: PromptApplyMode,
) {
  const normalizedPreset = presetPrompt.trim()
  if (mode === 'replace') return normalizedPreset

  const normalizedCurrent = currentPrompt.trimEnd()
  if (!normalizedCurrent) return normalizedPreset
  if (!normalizedPreset) return normalizedCurrent

  return `${normalizedCurrent}\n\n${normalizedPreset}`
}

export function validatePromptPresetDraft(
  draft: PromptPresetDraft,
  presets: PromptPreset[],
  editingID?: string,
): string | null {
  const name = draft.name.trim()
  const prompt = draft.prompt.trim()

  if (!name) return '请填写名称。'
  if (name.length > maxPresetNameLength) return `名称不能超过 ${maxPresetNameLength} 个字符。`
  if (!prompt) return '请填写提示词内容。'
  if (prompt.length > maxPresetPromptLength) return `提示词不能超过 ${maxPresetPromptLength} 个字符。`
  if (!isPromptPresetScope(draft.scope)) return '请选择有效的适用模式。'
  if (presets.length >= maxPromptPresets && !editingID) return `最多保存 ${maxPromptPresets} 套预设。`

  const duplicate = presets.some(
    preset => preset.id !== editingID && preset.name.trim().toLocaleLowerCase() === name.toLocaleLowerCase(),
  )
  if (duplicate) return '已有同名预设，请换一个名称。'

  return null
}

export function normalizePromptPresetDraft(draft: PromptPresetDraft): PromptPresetDraft {
  return {
    name: draft.name.trim(),
    prompt: draft.prompt.trim(),
    scope: draft.scope,
  }
}

export function createPromptPresetID() {
  if (typeof window.crypto?.randomUUID === 'function') {
    return window.crypto.randomUUID()
  }
  return `preset-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function parsePromptPresetStore(raw: string): ParsedPromptPresetStore {
  const value: unknown = JSON.parse(raw)
  if (
    !isRecord(value) ||
    typeof value.version !== 'number' ||
    (value.version !== storageVersion && value.version !== legacyStorageVersion) ||
    !Array.isArray(value.presets)
  ) {
    throw new Error('invalid preset store')
  }
  if (value.presets.length > maxPromptPresets || !value.presets.every(isPromptPreset)) {
    throw new Error('invalid preset collection')
  }

  const normalized = value.presets.map(preset => ({
    id: preset.id,
    name: preset.name.trim(),
    prompt: preset.prompt.trim(),
    scope: preset.scope,
  }))
  const uniqueIDs = new Set(normalized.map(preset => preset.id))
  const uniqueNames = new Set(normalized.map(preset => preset.name.toLocaleLowerCase()))
  if (uniqueIDs.size !== normalized.length || uniqueNames.size !== normalized.length) {
    throw new Error('duplicate preset values')
  }

  return {
    version: value.version,
    presets: normalized,
  }
}

function mergeStarterPresets(presets: PromptPreset[]) {
  const next = [...presets]
  const ids = new Set(next.map((preset) => preset.id))
  const names = new Set(next.map((preset) => preset.name.toLocaleLowerCase()))

  for (const starter of starterPresets) {
    if (next.length >= maxPromptPresets) break

    const normalizedName = starter.name.toLocaleLowerCase()
    if (ids.has(starter.id) || names.has(normalizedName)) continue

    next.push({ ...starter })
    ids.add(starter.id)
    names.add(normalizedName)
  }

  return next
}

function normalizeStarterPresetNames(presets: PromptPreset[]) {
  const names = new Set(presets.map((preset) => preset.name.toLocaleLowerCase()))
  let changed = false
  const next = presets.map((preset) => {
    const migration = starterPresetNameMigrations[preset.id]
    if (!migration || !migration.from.includes(preset.name)) return preset

    const oldName = preset.name.toLocaleLowerCase()
    const newName = migration.to.toLocaleLowerCase()
    if (oldName !== newName && names.has(newName)) return preset

    names.delete(oldName)
    names.add(newName)
    changed = true
    return { ...preset, name: migration.to }
  })

  return { presets: next, changed }
}

function isPromptPreset(value: unknown): value is PromptPreset {
  if (!isRecord(value)) return false

  return (
    typeof value.id === 'string' &&
    value.id.length > 0 &&
    typeof value.name === 'string' &&
    value.name.trim().length > 0 &&
    value.name.trim().length <= maxPresetNameLength &&
    typeof value.prompt === 'string' &&
    value.prompt.trim().length > 0 &&
    value.prompt.trim().length <= maxPresetPromptLength &&
    isPromptPresetScope(value.scope)
  )
}

function isPromptPresetScope(value: unknown): value is PromptPresetScope {
  return value === 'generate' || value === 'edit' || value === 'all'
}

function isPromptApplyMode(value: unknown): value is PromptApplyMode {
  return value === 'replace' || value === 'append'
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function cloneStarterPresets() {
  return starterPresets.map(preset => ({ ...preset }))
}
