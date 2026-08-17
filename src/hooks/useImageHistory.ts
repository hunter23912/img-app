import { useCallback, useEffect, useRef, useState } from 'react'

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
  const requestVersion = useRef(0)

  const refreshHistory = useCallback(async () => {
    const version = ++requestVersion.current
    try {
      const page = await fetchImageHistory()
      if (version !== requestVersion.current) return false
      setHistory(page.tasks)
      setNextCursor(page.next_cursor)
      setHasMore(page.has_more)
      setSyncWarning('')
      return true
    } catch {
      if (version !== requestVersion.current) return false
      setSyncWarning(historySyncFailureMessage)
      return false
    }
  }, [])

  useEffect(() => {
    const version = ++requestVersion.current
    let active = true
    async function loadHistory() {
      try {
        const page = await fetchImageHistory()
        if (!active || version !== requestVersion.current) return
        setHistory(page.tasks)
        setNextCursor(page.next_cursor)
        setHasMore(page.has_more)
        setSyncWarning('')
      } catch {
        if (active && version === requestVersion.current) setSyncWarning(historySyncFailureMessage)
      }
    }
    void loadHistory()
    return () => { active = false }
  }, [])

  useEffect(() => {
    function refreshWhenVisible() {
      if (document.visibilityState === 'visible') {
        void refreshHistory()
      }
    }

    document.addEventListener('visibilitychange', refreshWhenVisible)
    window.addEventListener('pageshow', refreshWhenVisible)
    return () => {
      document.removeEventListener('visibilitychange', refreshWhenVisible)
      window.removeEventListener('pageshow', refreshWhenVisible)
    }
  }, [refreshHistory])

  const loadMore = useCallback(async () => {
    if (!hasMore || !nextCursor || isLoadingMore) return false
    const version = ++requestVersion.current
    setIsLoadingMore(true)
    try {
      const page = await fetchImageHistory(nextCursor)
      if (version !== requestVersion.current) return false
      setHistory((current) => [...current, ...page.tasks])
      setNextCursor(page.next_cursor)
      setHasMore(page.has_more)
      setSyncWarning('')
      return true
    } catch {
      if (version !== requestVersion.current) return false
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
