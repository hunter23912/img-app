import { useCallback, useEffect, useState } from 'react'

import type { Theme } from '../types/image'

export const themeStorageKey = 'img-app.theme.v1'

function systemTheme(): Theme {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function readTheme(): Theme {
  try {
    const saved = window.localStorage.getItem(themeStorageKey)
    if (saved === 'light' || saved === 'dark') return saved
  } catch { /* system preference remains a safe fallback */ }
  return systemTheme()
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(readTheme)

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
    document.documentElement.dataset.theme = theme
  }, [theme])

  useEffect(() => {
    try {
      if (window.localStorage.getItem(themeStorageKey)) return
    } catch { return }
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const handleChange = (event: MediaQueryListEvent) => setTheme(event.matches ? 'dark' : 'light')
    media.addEventListener('change', handleChange)
    return () => media.removeEventListener('change', handleChange)
  }, [])

  const toggleTheme = useCallback(() => {
    setTheme((current) => {
      const next = current === 'dark' ? 'light' : 'dark'
      try { window.localStorage.setItem(themeStorageKey, next) } catch { /* theme still changes for this session */ }
      return next
    })
  }, [])

  return { theme, toggleTheme }
}
