import { useCallback, useEffect, useState } from 'react'

import { fetchImageSettings, saveImageSettings } from '../api/images'
import type { ImageSettings } from '../types/image'

const emptySettings: ImageSettings = { endpoint: '', api_key: '' }

export function useImageSettings() {
  const [settings, setSettings] = useState<ImageSettings>(emptySettings)
  const [savedSettings, setSavedSettings] = useState<ImageSettings>(emptySettings)
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    let ignore = false

    async function loadSettings() {
      try {
        const loaded = await fetchImageSettings()
        if (!ignore) {
          setSettings(loaded)
          setSavedSettings(loaded)
          setError('')
        }
      } catch (loadError) {
        if (!ignore) {
          setError(loadError instanceof Error ? loadError.message : '图片服务配置读取失败。')
        }
      } finally {
        if (!ignore) setIsLoading(false)
      }
    }

    void loadSettings()
    return () => {
      ignore = true
    }
  }, [])

  const updateSettings = useCallback((next: Partial<ImageSettings>) => {
    setSettings((current) => ({ ...current, ...next }))
    setError('')
  }, [])

  const persistSettings = useCallback(async (nextSettings?: ImageSettings) => {
    const toSave = nextSettings ?? settings
    setIsSaving(true)
    setError('')
    try {
      const saved = await saveImageSettings(toSave)
      setSettings(saved)
      setSavedSettings(saved)
      return saved
    } catch (saveError) {
      const message = saveError instanceof Error ? saveError.message : '图片服务配置保存失败。'
      setError(message)
      throw saveError
    } finally {
      setIsSaving(false)
    }
  }, [settings])

  const resetSettings = useCallback(() => persistSettings(emptySettings), [persistSettings])
  const isDirty = settings.endpoint !== savedSettings.endpoint || settings.api_key !== savedSettings.api_key

  return {
    settings,
    isLoading,
    isSaving,
    isDirty,
    error,
    updateSettings,
    persistSettings,
    resetSettings,
  }
}
