import { useEffect, useState } from 'react'

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

  return {
    health,
    isConfigured,
  }
}
