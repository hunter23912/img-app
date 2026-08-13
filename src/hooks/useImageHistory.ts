import { useCallback, useEffect, useState } from 'react'

import { deleteImageHistory, fetchImageHistory } from '../api/images'

const historySyncFailureMessage = '历史记录同步失败，当前图片仍可正常使用。'
const historyDeleteFailureMessage = '历史记录删除失败，当前历史记录未修改。'

export function useImageHistory() {
  const [history, setHistory] = useState<string[]>([])
  const [syncWarning, setSyncWarning] = useState('')

  const refreshHistory = useCallback(async () => {
    try {
      const next = await fetchImageHistory()
      setHistory(next)
      setSyncWarning('')
      return true
    } catch {
      setSyncWarning(historySyncFailureMessage)
      return false
    }
  }, [])

  useEffect(() => {
    let active = true

    async function loadHistory() {
      try {
        const next = await fetchImageHistory()
        if (!active) return

        setHistory(next)
        setSyncWarning('')
      } catch {
        if (active) setSyncWarning(historySyncFailureMessage)
      }
    }

    void loadHistory()
    return () => {
      active = false
    }
  }, [])

  const removeImage = useCallback(async (url: string) => {
    try {
      await deleteImageHistory(url)
      setHistory((current) => current.filter((item) => item !== url))
      setSyncWarning('')
    } catch {
      setSyncWarning(historyDeleteFailureMessage)
    }
  }, [])

  return {
    history,
    syncWarning,
    refreshHistory,
    removeImage,
  }
}
