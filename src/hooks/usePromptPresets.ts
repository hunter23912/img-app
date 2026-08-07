import { useCallback, useEffect, useState } from 'react'

import type { PromptPreset, PromptPresetDraft } from '../types/image'
import {
  createPromptPresetID,
  loadPromptPresets,
  normalizePromptPresetDraft,
  parsePromptPresetStorageValue,
  persistPromptPresets,
  promptPresetStorageKey,
  validatePromptPresetDraft,
} from '../utils/promptPresets'

export function usePromptPresets() {
  const [initial] = useState(loadPromptPresets)
  const [presets, setPresets] = useState(initial.presets)
  const [storageWarning, setStorageWarning] = useState(initial.warning)

  useEffect(() => {
    function handleStorage(event: StorageEvent) {
      if (event.key !== promptPresetStorageKey && event.key !== null) return

      const next = parsePromptPresetStorageValue(event.newValue)
      setPresets(next.presets)
      setStorageWarning(next.warning)
    }

    window.addEventListener('storage', handleStorage)
    return () => window.removeEventListener('storage', handleStorage)
  }, [])

  const createPreset = useCallback(
    (draft: PromptPresetDraft) => {
      const validationError = validatePromptPresetDraft(draft, presets)
      if (validationError) return { ok: false, error: validationError }

      const preset: PromptPreset = {
        id: createPromptPresetID(),
        ...normalizePromptPresetDraft(draft),
      }
      const next = [preset, ...presets]
      const storageError = persistPromptPresets(next)
      if (storageError) return { ok: false, error: storageError }

      setPresets(next)
      setStorageWarning('')
      return { ok: true }
    },
    [presets],
  )

  const updatePreset = useCallback(
    (id: string, draft: PromptPresetDraft) => {
      if (!presets.some(preset => preset.id === id)) {
        return { ok: false, error: '要编辑的预设已不存在。' }
      }

      const validationError = validatePromptPresetDraft(draft, presets, id)
      if (validationError) return { ok: false, error: validationError }

      const normalized = normalizePromptPresetDraft(draft)
      const next = presets.map(preset => (preset.id === id ? { id, ...normalized } : preset))
      const storageError = persistPromptPresets(next)
      if (storageError) return { ok: false, error: storageError }

      setPresets(next)
      setStorageWarning('')
      return { ok: true }
    },
    [presets],
  )

  const deletePreset = useCallback(
    (id: string) => {
      const next = presets.filter(preset => preset.id !== id)
      if (next.length === presets.length) {
        return { ok: false, error: '要删除的预设已不存在。' }
      }

      const storageError = persistPromptPresets(next)
      if (storageError) return { ok: false, error: storageError }

      setPresets(next)
      setStorageWarning('')
      return { ok: true }
    },
    [presets],
  )

  return {
    presets,
    storageWarning,
    createPreset,
    updatePreset,
    deletePreset,
  }
}
