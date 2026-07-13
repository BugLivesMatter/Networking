import type { ClusterEvent, ClusterService, ClusterSnapshot, HealthStatus } from './types'

const startedAt = new Date(Date.now() - 1000 * 60 * 60 * 24 * 18).toISOString()

const instance = (name: string, latency: number) => ({
  id: name,
  name,
  status: 'healthy' as HealthStatus,
  latency,
  restarts: 0,
  startedAt,
})

export const initialServices: ClusterService[] = [
  {
    id: 'gateway',
    name: 'Edge Gateway',
    description: 'Public ingress and request routing',
    kind: 'edge',
    status: 'healthy',
    position: [-2.45, 0.65, 1.05],
    latency: 18,
    uptime: 99.99,
    requestsPerMinute: 1842,
    errorRate: 0.03,
    version: 'nginx/1.27',
    instances: [instance('gateway-7db9f-2rk8m', 17), instance('gateway-7db9f-x9tq4', 19)],
    dependencies: ['api'],
  },
  {
    id: 'api',
    name: 'Core API',
    description: 'Authentication, incidents and cluster topology',
    kind: 'compute',
    status: 'healthy',
    position: [-0.85, 0.55, 0.2],
    latency: 32,
    uptime: 99.97,
    requestsPerMinute: 1396,
    errorRate: 0.08,
    version: 'neuro-api/0.1.0',
    instances: [instance('api-68f7d-k5jrm', 30), instance('api-68f7d-p82nc', 34)],
    dependencies: ['mongodb', 'redis', 'rabbitmq', 'minio'],
  },
  {
    id: 'mongodb',
    name: 'MongoDB',
    description: 'Persistent cluster state and incident history',
    kind: 'database',
    status: 'healthy',
    position: [1.35, 1.35, -0.85],
    latency: 24,
    uptime: 99.98,
    requestsPerMinute: 884,
    errorRate: 0.01,
    version: 'mongo/7.0',
    instances: [instance('mongodb-0', 24)],
    dependencies: [],
  },
  {
    id: 'redis',
    name: 'Redis',
    description: 'Snapshots, cache and distributed locks',
    kind: 'cache',
    status: 'healthy',
    position: [1.65, 0.05, 0.65],
    latency: 7,
    uptime: 99.99,
    requestsPerMinute: 2234,
    errorRate: 0,
    version: 'redis/7.2',
    instances: [instance('redis-6dc8b-m2v7s', 7)],
    dependencies: [],
  },
  {
    id: 'rabbitmq',
    name: 'RabbitMQ',
    description: 'Cluster events and notification delivery',
    kind: 'queue',
    status: 'healthy',
    position: [-0.35, -1.35, -0.9],
    latency: 12,
    uptime: 99.96,
    requestsPerMinute: 428,
    errorRate: 0.02,
    version: 'rabbitmq/3.12',
    instances: [instance('rabbitmq-0', 12)],
    dependencies: [],
  },
  {
    id: 'minio',
    name: 'MinIO',
    description: 'Logs, screenshots and incident attachments',
    kind: 'storage',
    status: 'healthy',
    position: [1.55, -1.25, 0.9],
    latency: 41,
    uptime: 99.95,
    requestsPerMinute: 96,
    errorRate: 0.04,
    version: 'minio/2025.01',
    instances: [instance('minio-0', 41)],
    dependencies: [],
  },
]

export const initialEvents: ClusterEvent[] = [
  {
    id: 'boot-1',
    serviceId: 'api',
    status: 'healthy',
    title: 'Topology synchronized',
    detail: '6 services · 8 instances discovered',
    timestamp: new Date(Date.now() - 1000 * 18),
  },
  {
    id: 'boot-2',
    serviceId: 'gateway',
    status: 'healthy',
    title: 'Health probes stable',
    detail: 'All readiness checks are passing',
    timestamp: new Date(Date.now() - 1000 * 52),
  },
]

export const createInitialSnapshot = (): ClusterSnapshot => ({
  services: structuredClone(initialServices),
  events: initialEvents.map(event => ({ ...event })),
  generatedAt: new Date(),
})

export const statusLabel: Record<HealthStatus, string> = {
  healthy: 'Operational',
  degraded: 'Degraded',
  unhealthy: 'Unhealthy',
  starting: 'Starting',
  unknown: 'Unknown',
}

export const statusColor: Record<HealthStatus, string> = {
  healthy: '#f4f4f5',
  degraded: '#f5c451',
  unhealthy: '#ff5c5c',
  starting: '#4fa7ff',
  unknown: '#71717a',
}
