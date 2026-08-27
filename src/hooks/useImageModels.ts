import { useCallback, useEffect, useState } from 'react'

import { createImageModel, deleteImageModel, fetchImageModels } from '../api/images'
import { builtInModelOptions } from '../constants/image'
import type { ModelOption } from '../types/image'

export function useImageModels(scopeKey: string) {
  const [models, setModels] = useState<ModelOption[]>(builtInModelOptions)
  const [loadedScopeKey, setLoadedScopeKey] = useState<string | null>(null)
  const [isSaving, setIsSaving] = useState(false)
  const [errorState, setErrorState] = useState<{ scopeKey: string; message: string } | null>(null)
  const isLoading = loadedScopeKey !== scopeKey
  const error = errorState?.scopeKey === scopeKey ? errorState.message : ''

  const refresh = useCallback(async () => {
    const loaded = await fetchImageModels()
    setModels(mergeModels(loaded))
    setErrorState(null)
    setLoadedScopeKey(scopeKey)
    return loaded
  }, [scopeKey])

  useEffect(() => {
    let active = true
    async function load() {
      try {
        const loaded = await fetchImageModels()
        if (active) {
          setModels(mergeModels(loaded))
          setErrorState(null)
        }
      } catch (loadError) {
        if (active) {
          setModels(builtInModelOptions)
          setErrorState({
            scopeKey,
            message: loadError instanceof Error ? loadError.message : '模型列表读取失败。',
          })
        }
      } finally {
        if (active) setLoadedScopeKey(scopeKey)
      }
    }
    void load()
    return () => { active = false }
  }, [scopeKey])

  const add = useCallback(async (modelName: string) => {
    setIsSaving(true)
    setErrorState(null)
    try {
      const created = await createImageModel(modelName)
      setModels((current) => mergeModels([...current, created]))
      return created
    } catch (saveError) {
      const message = saveError instanceof Error ? saveError.message : '自定义模型添加失败。'
      setErrorState({ scopeKey, message })
      throw saveError
    } finally {
      setIsSaving(false)
    }
  }, [scopeKey])

  const remove = useCallback(async (id: string) => {
    setIsSaving(true)
    setErrorState(null)
    try {
      await deleteImageModel(id)
      setModels((current) => current.filter((model) => model.id !== id || model.built_in))
    } catch (deleteError) {
      const message = deleteError instanceof Error ? deleteError.message : '自定义模型删除失败。'
      setErrorState({ scopeKey, message })
      throw deleteError
    } finally {
      setIsSaving(false)
    }
  }, [scopeKey])

  return { models: isLoading ? builtInModelOptions : models, isLoading, isSaving, error, add, remove, refresh }
}

function mergeModels(models: ModelOption[]) {
  const seen = new Set<string>()
  const result: ModelOption[] = []
  for (const model of [...builtInModelOptions, ...models]) {
    const key = model.value.trim().toLocaleLowerCase()
    if (!key || seen.has(key)) continue
    seen.add(key)
    result.push(model)
  }
  return result
}
