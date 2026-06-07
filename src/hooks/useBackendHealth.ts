import { useEffect, useMemo, useState } from 'react'

import { fetchHealth } from '../api/images'
import type { HealthState } from '../types/image'

export function useBackendHealth() {
  const [health, setHealth] = useState<HealthState>('checking')
  const [isConfigured, setIsConfigured] = useState(false)

  useEffect(() => {
    let ignore = false

    async function checkBackend() {
      try {
        const data = await fetchHealth()

        if (!ignore) {
          setHealth(data.ok ? 'online' : 'offline')
          setIsConfigured(Boolean(data.configured))
        }
      } catch {
        if (!ignore) {
          setHealth('offline')
          setIsConfigured(false)
        }
      }
    }

    checkBackend()

    return () => {
      ignore = true
    }
  }, [])

  const healthLabel = useMemo(() => {
    if (health === 'checking') return '检测中'
    if (health === 'online') return '后端在线'
    return '后端未连接'
  }, [health])

  const healthClass = useMemo(() => {
    if (health === 'checking') return 'badge-warning'
    if (health === 'online') return 'badge-success'
    return 'badge-error'
  }, [health])

  return {
    health,
    healthLabel,
    healthClass,
    isConfigured,
  }
}
