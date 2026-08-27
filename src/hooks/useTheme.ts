import { useCallback, useEffect, useState } from 'react'

import type { Theme } from '../types/image'

export const themeStorageKey = 'img-app.theme.v1'
const themeCookieKey = 'img-app.theme.v1'

function isTheme(value: string | null): value is Theme {
  return value === 'light' || value === 'dark'
}

function readCookieTheme(): Theme | null {
  try {
    const value = document.cookie
      .split('; ')
      .find((item) => item.startsWith(`${themeCookieKey}=`))
      ?.slice(themeCookieKey.length + 1)
    const candidate = value ?? null
    return isTheme(candidate) ? candidate : null
  } catch {
    return null
  }
}

function readStoredTheme(): Theme | null {
  try {
    const saved = window.localStorage.getItem(themeStorageKey)
    if (isTheme(saved)) return saved
  } catch { /* cookie remains as a fallback */ }

  return readCookieTheme()
}

function persistTheme(theme: Theme) {
  try {
    window.localStorage.setItem(themeStorageKey, theme)
  } catch { /* cookie remains as a fallback */ }

  try {
    document.cookie = `${themeCookieKey}=${theme}; Max-Age=31536000; Path=/; SameSite=Lax`
  } catch { /* theme still changes for this session */ }
}

function systemTheme(): Theme {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function readTheme(): Theme {
  return readStoredTheme() ?? systemTheme()
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(readTheme)

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
    document.documentElement.dataset.theme = theme
  }, [theme])

  useEffect(() => {
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const handleChange = (event: MediaQueryListEvent) => {
      // Once the user has chosen a theme, a system change must not override it.
      if (readStoredTheme()) return
      setTheme(event.matches ? 'dark' : 'light')
    }
    media.addEventListener('change', handleChange)
    return () => media.removeEventListener('change', handleChange)
  }, [])

  const toggleTheme = useCallback(() => {
    const next = theme === 'dark' ? 'light' : 'dark'
    persistTheme(next)
    setTheme(next)
  }, [theme])

  return { theme, toggleTheme }
}
