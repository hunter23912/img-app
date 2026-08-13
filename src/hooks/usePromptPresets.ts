import { useCallback, useEffect, useState } from 'react'

import {
  createPromptPreset,
  deletePromptPreset,
  fetchPromptPresets,
  importPromptPresets,
  updatePromptPreset,
} from '../api/images'
import type { PromptPreset, PromptPresetDraft } from '../types/image'
import {
  loadLegacyCustomPresets,
  normalizePromptPresetDraft,
  promptPresetMigrationKey,
  validatePromptPresetDraft,
} from '../utils/promptPresets'

export function usePromptPresets() {
  const [presets, setPresets] = useState<PromptPreset[]>([])
  const [storageWarning, setStorageWarning] = useState('')
  const [isLoading, setIsLoading] = useState(true)

  const reload = useCallback(async () => {
    const next = await fetchPromptPresets()
    setPresets(next)
    return next
  }, [])

  useEffect(() => {
    let active = true

    async function load() {
      try {
        const next = await fetchPromptPresets()
        if (!active) return
        setPresets(next)

        let alreadyMigrated = false
        try { alreadyMigrated = window.localStorage.getItem(promptPresetMigrationKey) === '1' } catch { /* storage is optional */ }
        if (!alreadyMigrated) {
          const legacy = loadLegacyCustomPresets()
          if (legacy.warning) setStorageWarning(legacy.warning)
          const existingNames = new Set(next.map((preset) => preset.name.trim().toLocaleLowerCase()))
          const customPresets = legacy.presets.filter((preset) => !existingNames.has(preset.name.trim().toLocaleLowerCase()))
          if (customPresets.length > 0) {
            await importPromptPresets(customPresets)
            if (!active) return
            await reload()
          }
          try { window.localStorage.setItem(promptPresetMigrationKey, '1') } catch { /* migration stays idempotent */ }
        }
      } catch {
        if (active) setStorageWarning('预设同步失败，当前页面暂时无法使用预设。')
      } finally {
        if (active) setIsLoading(false)
      }
    }
    void load()
    return () => { active = false }
  }, [reload])

  const createPreset = useCallback(async (draft: PromptPresetDraft) => {
    const validationError = validatePromptPresetDraft(draft, presets)
    if (validationError) return { ok: false, error: validationError }
    try {
      const created = await createPromptPreset(normalizePromptPresetDraft(draft))
      setPresets((current) => [created, ...current])
      setStorageWarning('')
      return { ok: true }
    } catch (error) {
      return { ok: false, error: error instanceof Error ? error.message : '预设保存失败。' }
    }
  }, [presets])

  const updatePreset = useCallback(async (id: string, draft: PromptPresetDraft) => {
    if (!presets.some((preset) => preset.id === id)) return { ok: false, error: '要编辑的预设已不存在。' }
    const validationError = validatePromptPresetDraft(draft, presets, id)
    if (validationError) return { ok: false, error: validationError }
    try {
      const updated = await updatePromptPreset(id, normalizePromptPresetDraft(draft))
      setPresets((current) => current.map((preset) => preset.id === id ? updated : preset))
      setStorageWarning('')
      return { ok: true }
    } catch (error) {
      return { ok: false, error: error instanceof Error ? error.message : '预设保存失败。' }
    }
  }, [presets])

  const deletePreset = useCallback(async (id: string) => {
    if (!presets.some((preset) => preset.id === id)) return { ok: false, error: '要删除的预设已不存在。' }
    try {
      await deletePromptPreset(id)
      setPresets((current) => current.filter((preset) => preset.id !== id))
      setStorageWarning('')
      return { ok: true }
    } catch (error) {
      return { ok: false, error: error instanceof Error ? error.message : '预设删除失败。' }
    }
  }, [presets])

  return { presets, storageWarning, isLoading, createPreset, updatePreset, deletePreset }
}
