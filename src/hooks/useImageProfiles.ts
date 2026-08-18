import { useCallback, useEffect, useState } from 'react'

import {
  activateImageProfile,
  createImageProfile,
  deleteImageProfile,
  fetchImageProfiles,
  updateImageProfile,
} from '../api/images'
import type { ImageProfile, ImageProfileDraft } from '../types/image'

export function useImageProfiles() {
  const [profiles, setProfiles] = useState<ImageProfile[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState('')

  const refresh = useCallback(async () => {
    const loaded = orderProfiles(await fetchImageProfiles())
    setProfiles(loaded)
    setError('')
    return loaded
  }, [])

  useEffect(() => {
    let ignore = false
    async function load() {
      try {
        const loaded = orderProfiles(await fetchImageProfiles())
        if (!ignore) {
          setProfiles(loaded)
          setError('')
        }
      } catch (loadError) {
        if (!ignore) setError(loadError instanceof Error ? loadError.message : '中转站配置读取失败。')
      } finally {
        if (!ignore) setIsLoading(false)
      }
    }
    void load()
    return () => { ignore = true }
  }, [])

  const run = useCallback(async <T,>(operation: () => Promise<T>) => {
    setIsSaving(true)
    setError('')
    try {
      return await operation()
    } catch (saveError) {
      const message = saveError instanceof Error ? saveError.message : '中转站配置操作失败。'
      setError(message)
      throw saveError
    } finally {
      setIsSaving(false)
    }
  }, [])

  const create = useCallback((draft: ImageProfileDraft) => run(async () => {
    await createImageProfile(draft)
    return refresh()
  }), [refresh, run])

  const update = useCallback((id: string, draft: ImageProfileDraft) => run(async () => {
    await updateImageProfile(id, draft)
    return refresh()
  }), [refresh, run])

  const activate = useCallback((id: string) => run(async () => {
    await activateImageProfile(id)
    return refresh()
  }), [refresh, run])

  const remove = useCallback((id: string) => run(async () => {
    await deleteImageProfile(id)
    return refresh()
  }), [refresh, run])

  return { profiles, isLoading, isSaving, error, create, update, activate, remove }
}

function orderProfiles(profiles: ImageProfile[]) {
  return [...profiles].sort((left, right) => {
    const createdAtOrder = (left.created_at ?? '').localeCompare(right.created_at ?? '')
    return createdAtOrder || left.id.localeCompare(right.id)
  })
}
