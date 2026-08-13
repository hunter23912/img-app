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
export const promptPresetMigrationKey = 'img-app.prompt-presets.migrated.v1'

const storageVersion = 2
const legacyStorageVersion = 1

const starterPresetIDs = new Set([
  'starter-remove-text-watermark',
  'starter-remove-face-sticker-mosaic',
  'starter-enhance-quality',
  'starter-reduce-gpt-image-texture',
  'starter-natural-body-details',
  'starter-anime-clean-lines',
  'female-generate',
])

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

export function loadLegacyCustomPresets(): PromptPresetLoadResult {
  try {
    const raw = window.localStorage.getItem(promptPresetStorageKey)
    if (raw === null) return { presets: [], warning: '' }
    const parsed = parsePromptPresetStore(raw)
    const merged = parsed.presets
    return {
      presets: merged.filter((preset) => !starterPresetIDs.has(preset.id)),
      warning: '',
    }
  } catch {
    return { presets: [], warning: '旧版本地预设数据异常，未执行迁移。' }
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
