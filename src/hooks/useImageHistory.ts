import { useCallback, useEffect, useState } from 'react'

import { deleteImageHistory, fetchImageHistory } from '../api/images'
import type { ImageTask } from '../types/image'

const historySyncFailureMessage = '历史记录同步失败，当前图片仍可正常使用。'
const historyDeleteFailureMessage = '历史记录删除失败，当前历史记录未修改。'

export function useImageHistory() {
  const [history, setHistory] = useState<ImageTask[]>([])
  const [nextCursor, setNextCursor] = useState('')
  const [hasMore, setHasMore] = useState(false)
  const [syncWarning, setSyncWarning] = useState('')
  const [isLoadingMore, setIsLoadingMore] = useState(false)

  const refreshHistory = useCallback(async () => {
    try {
      const page = await fetchImageHistory()
      setHistory(page.tasks)
      setNextCursor(page.next_cursor)
      setHasMore(page.has_more)
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
        const page = await fetchImageHistory()
        if (!active) return
        setHistory(page.tasks)
        setNextCursor(page.next_cursor)
        setHasMore(page.has_more)
        setSyncWarning('')
      } catch {
        if (active) setSyncWarning(historySyncFailureMessage)
      }
    }
    void loadHistory()
    return () => { active = false }
  }, [])

  const loadMore = useCallback(async () => {
    if (!hasMore || !nextCursor || isLoadingMore) return false
    setIsLoadingMore(true)
    try {
      const page = await fetchImageHistory(nextCursor)
      setHistory((current) => [...current, ...page.tasks])
      setNextCursor(page.next_cursor)
      setHasMore(page.has_more)
      setSyncWarning('')
      return true
    } catch {
      setSyncWarning(historySyncFailureMessage)
      return false
    } finally {
      setIsLoadingMore(false)
    }
  }, [hasMore, isLoadingMore, nextCursor])

  const removeTask = useCallback(async (id: string) => {
    try {
      await deleteImageHistory(id)
      setHistory((current) => current.filter((item) => item.id !== id))
      setSyncWarning('')
    } catch {
      setSyncWarning(historyDeleteFailureMessage)
    }
  }, [])

  return { history, hasMore, isLoadingMore, syncWarning, refreshHistory, loadMore, removeTask }
}
