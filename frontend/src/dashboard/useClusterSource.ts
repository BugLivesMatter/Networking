import { useCallback, useEffect, useState } from 'react'
import type { ClusterSnapshot, DemoScenario } from './types'
import { useDemoCluster } from './useDemoCluster'

const configuredApiUrl = ((import.meta.env.VITE_CLUSTER_API_URL as string | undefined) || '').replace(/\/$/, '')
const apiMode = ((import.meta.env.VITE_NEUROOPS_MODE as string | undefined) || 'demo') === 'api'

export function useClusterSource() {
  const { snapshot: demoSnapshot, runScenario: runDemoScenario } = useDemoCluster()
  const [remoteSnapshot, setRemoteSnapshot] = useState<ClusterSnapshot | null>(null)
  const [remoteAvailable, setRemoteAvailable] = useState(false)

  const refreshRemote = useCallback(async (signal?: AbortSignal) => {
    if (!apiMode) return
    const response = await fetch(`${configuredApiUrl}/api/v1/cluster/topology`, { signal, credentials: 'include' })
    if (!response.ok) throw new Error(`topology request failed: ${response.status}`)
    const snapshot = await response.json() as ClusterSnapshot
    setRemoteSnapshot({
      ...snapshot,
      generatedAt: new Date(snapshot.generatedAt),
      events: snapshot.events.map(event => ({ ...event, timestamp: new Date(event.timestamp) })),
    })
    setRemoteAvailable(true)
  }, [])

  useEffect(() => {
    if (!apiMode) return

    const controller = new AbortController()
    void refreshRemote(controller.signal).catch(() => setRemoteAvailable(false))
    const events = new EventSource(`${configuredApiUrl}/api/v1/cluster/events`, { withCredentials: true })
    events.addEventListener('cluster-event', () => {
      void refreshRemote().catch(() => setRemoteAvailable(false))
    })
    events.onerror = () => setRemoteAvailable(false)

    return () => {
      controller.abort()
      events.close()
    }
  }, [refreshRemote])

  const runScenario = useCallback((scenario: DemoScenario) => {
    if (!apiMode || !remoteAvailable) {
      runDemoScenario(scenario)
      return
    }

    void fetch(`${configuredApiUrl}/api/v1/demo/scenarios/${scenario}`, { method: 'POST', credentials: 'include' })
      .then(response => {
        if (!response.ok) throw new Error(`scenario request failed: ${response.status}`)
        return response.json() as Promise<ClusterSnapshot>
      })
      .then(snapshot => setRemoteSnapshot({
        ...snapshot,
        generatedAt: new Date(snapshot.generatedAt),
        events: snapshot.events.map(event => ({ ...event, timestamp: new Date(event.timestamp) })),
      }))
      .catch(() => setRemoteAvailable(false))
  }, [remoteAvailable, runDemoScenario])

  return {
    snapshot: remoteAvailable && remoteSnapshot ? remoteSnapshot : demoSnapshot,
    runScenario,
    sourceMode: remoteAvailable ? 'API CONNECTED' : 'LOCAL SIMULATION',
  }
}
