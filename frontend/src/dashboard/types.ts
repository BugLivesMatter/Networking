export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy' | 'starting' | 'unknown'

export interface ClusterInstance {
  id: string
  name: string
  status: HealthStatus
  latency: number
  restarts: number
  startedAt: string
}

export interface ClusterService {
  id: string
  name: string
  description: string
  kind: 'edge' | 'compute' | 'database' | 'cache' | 'queue' | 'storage'
  status: HealthStatus
  position: [number, number, number]
  latency: number
  uptime: number
  requestsPerMinute: number
  errorRate: number
  version: string
  instances: ClusterInstance[]
  dependencies: string[]
}

export interface ClusterEvent {
  id: string
  serviceId: string
  status: HealthStatus
  title: string
  detail: string
  timestamp: Date
}

export interface ClusterSnapshot {
  services: ClusterService[]
  events: ClusterEvent[]
  generatedAt: Date
}

export type DemoScenario = 'latency' | 'crash' | 'scale' | 'recover'
