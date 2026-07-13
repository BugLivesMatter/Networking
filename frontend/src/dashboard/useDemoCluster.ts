import { useCallback, useEffect, useRef, useState } from 'react'
import { createInitialSnapshot } from './demoCluster'
import type { ClusterEvent, ClusterSnapshot, DemoScenario, HealthStatus } from './types'

const makeEvent = (
  serviceId: string,
  status: HealthStatus,
  title: string,
  detail: string,
): ClusterEvent => ({
  id: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
  serviceId,
  status,
  title,
  detail,
  timestamp: new Date(),
})

export function useDemoCluster() {
  const [snapshot, setSnapshot] = useState<ClusterSnapshot>(createInitialSnapshot)
  const scaleTimer = useRef<number | null>(null)

  useEffect(() => {
    const interval = window.setInterval(() => {
      setSnapshot(current => ({
        ...current,
        generatedAt: new Date(),
        services: current.services.map(service => {
          if (service.status !== 'healthy') return service
          const jitter = Math.round((Math.random() - 0.5) * Math.max(2, service.latency * 0.12))
          return {
            ...service,
            latency: Math.max(2, service.latency + jitter),
            requestsPerMinute: Math.max(0, service.requestsPerMinute + Math.round((Math.random() - 0.5) * 24)),
          }
        }),
      }))
    }, 1800)

    return () => window.clearInterval(interval)
  }, [])

  useEffect(() => () => {
    if (scaleTimer.current) window.clearTimeout(scaleTimer.current)
  }, [])

  const runScenario = useCallback((scenario: DemoScenario) => {
    if (scaleTimer.current) {
      window.clearTimeout(scaleTimer.current)
      scaleTimer.current = null
    }

    if (scenario === 'recover') {
      const clean = createInitialSnapshot()
      setSnapshot(current => ({
        ...clean,
        events: [
          makeEvent('api', 'healthy', 'Cluster recovered', 'All services returned to nominal state'),
          ...current.events,
        ].slice(0, 12),
      }))
      return
    }

    if (scenario === 'latency') {
      setSnapshot(current => ({
        ...current,
        generatedAt: new Date(),
        services: current.services.map(service => {
          if (service.id === 'redis') {
            return {
              ...service,
              status: 'degraded',
              latency: 286,
              errorRate: 2.84,
              instances: service.instances.map(item => ({ ...item, status: 'degraded', latency: 286 })),
            }
          }
          if (service.id === 'api') return { ...service, status: 'degraded', latency: 148, errorRate: 1.16 }
          return service
        }),
        events: [
          makeEvent('redis', 'degraded', 'Latency threshold exceeded', 'p95 reached 286 ms · Core API affected'),
          ...current.events,
        ].slice(0, 12),
      }))
      return
    }

    if (scenario === 'crash') {
      setSnapshot(current => ({
        ...current,
        generatedAt: new Date(),
        services: current.services.map(service => service.id === 'api' ? {
          ...service,
          status: 'degraded',
          latency: 91,
          errorRate: 4.72,
          instances: service.instances.map((item, index) => index === 0
            ? { ...item, status: 'unhealthy', restarts: item.restarts + 1, latency: 0 }
            : item),
        } : service),
        events: [
          makeEvent('api', 'unhealthy', 'Pod stopped responding', 'api-68f7d-k5jrm failed readiness probe'),
          ...current.events,
        ].slice(0, 12),
      }))
      return
    }

    setSnapshot(current => {
      const api = current.services.find(service => service.id === 'api')
      if (!api || api.instances.some(item => item.id === 'api-68f7d-new01')) return current
      return {
        ...current,
        generatedAt: new Date(),
        services: current.services.map(service => service.id === 'api' ? {
          ...service,
          status: 'starting',
          instances: [...service.instances, {
            id: 'api-68f7d-new01',
            name: 'api-68f7d-new01',
            status: 'starting',
            latency: 0,
            restarts: 0,
            startedAt: new Date().toISOString(),
          }],
        } : service),
        events: [
          makeEvent('api', 'starting', 'Scaling deployment', 'New API replica is waiting for readiness'),
          ...current.events,
        ].slice(0, 12),
      }
    })

    scaleTimer.current = window.setTimeout(() => {
      setSnapshot(current => ({
        ...current,
        generatedAt: new Date(),
        services: current.services.map(service => {
          if (service.id !== 'api') return service
          const instances = service.instances.map(item => item.id === 'api-68f7d-new01'
            ? { ...item, status: 'healthy' as const, latency: 29 }
            : item)
          const remainsDegraded = instances.some(item => item.status === 'unhealthy' || item.status === 'degraded')
          return {
            ...service,
            status: remainsDegraded ? 'degraded' : 'healthy',
            latency: remainsDegraded ? service.latency : 27,
            instances,
          }
        }),
        events: [
          makeEvent('api', 'healthy', 'Replica is ready', 'api-68f7d-new01 joined the service mesh'),
          ...current.events,
        ].slice(0, 12),
      }))
      scaleTimer.current = null
    }, 3800)
  }, [])

  return { snapshot, runScenario }
}
