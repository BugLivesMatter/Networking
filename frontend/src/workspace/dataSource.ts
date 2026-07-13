import type {
  Attachment,
  CreateIncidentInput,
  Incident,
  IncidentFilters,
  IncidentPage,
  IncidentStatus,
  PatchIncidentInput,
  Role,
  Severity,
  TimelineEvent,
  WorkspaceDataSource,
  WorkspaceUser,
} from './types'
import { initialEvents } from '../dashboard/demoCluster'

const mode = ((import.meta.env.VITE_NEUROOPS_MODE as string | undefined) || 'demo') === 'api' ? 'api' : 'demo'
const apiRoot = ((import.meta.env.VITE_CLUSTER_API_URL as string | undefined) || '').replace(/\/$/, '')
const storeKey = 'neuroops.workspace.v1'
const sessionKey = 'neuroops.session.v1'
const changeEvent = 'neuroops:workspace-change'

interface DemoStore {
  schemaVersion: 1
  users: WorkspaceUser[]
  incidents: Incident[]
  timeline: TimelineEvent[]
  attachments: Attachment[]
  sequence: number
}

const now = (offsetMinutes = 0) => new Date(Date.UTC(2026, 6, 14, 9, offsetMinutes)).toISOString()
const demoUsers: WorkspaceUser[] = [
  { id: '00000000-0000-4000-8000-000000000001', email: 'admin@neuroops.demo', displayName: 'Alex Morgan', role: 'admin', createdAt: now(), updatedAt: now() },
  { id: '00000000-0000-4000-8000-000000000002', email: 'responder@neuroops.demo', displayName: 'Sam Rivera', role: 'responder', createdAt: now(), updatedAt: now() },
  { id: '00000000-0000-4000-8000-000000000003', email: 'viewer@neuroops.demo', displayName: 'Taylor Chen', role: 'viewer', createdAt: now(), updatedAt: now() },
]

function seedStore(): DemoStore {
  const incidents: Incident[] = [
    {
      id: '10000000-0000-4000-8000-000000000001', title: 'Redis latency above SLO',
      description: 'Cache latency is affecting the Core API request path.', service: 'redis', severity: 'high', status: 'investigating',
      assigneeId: demoUsers[1].id, creatorId: demoUsers[0].id, version: 2, createdAt: now(1), updatedAt: now(12),
    },
    {
      id: '10000000-0000-4000-8000-000000000002', title: 'API readiness probe failures',
      description: 'One API replica repeatedly failed readiness checks.', service: 'api', severity: 'critical', status: 'mitigated',
      assigneeId: demoUsers[0].id, creatorId: demoUsers[0].id, version: 3, createdAt: now(-70), updatedAt: now(-15),
    },
  ]
  return {
    schemaVersion: 1,
    users: demoUsers.map(user => ({ ...user })),
    incidents,
    timeline: [
      { id: '20000000-0000-4000-8000-000000000001', incidentId: incidents[0].id, type: 'created', actorId: demoUsers[0].id, message: 'Incident created', createdAt: now(1) },
      { id: '20000000-0000-4000-8000-000000000002', incidentId: incidents[0].id, type: 'status.changed', actorId: demoUsers[1].id, message: 'Investigation started', createdAt: now(12) },
      { id: '20000000-0000-4000-8000-000000000003', incidentId: incidents[1].id, type: 'created', actorId: demoUsers[0].id, message: 'Incident created', createdAt: now(-70) },
      { id: '20000000-0000-4000-8000-000000000004', incidentId: incidents[1].id, type: 'commented', actorId: demoUsers[0].id, message: 'Replica recycled and probes are stable.', createdAt: now(-15) },
    ],
    attachments: [],
    sequence: 10,
  }
}

function readStore(): DemoStore {
  try {
    const parsed = JSON.parse(localStorage.getItem(storeKey) || '') as DemoStore
    if (parsed.schemaVersion === 1 && Array.isArray(parsed.incidents)) return parsed
  } catch { /* seed invalid or missing data */ }
  const seeded = seedStore()
  localStorage.setItem(storeKey, JSON.stringify(seeded))
  return seeded
}

function writeStore(store: DemoStore) {
  localStorage.setItem(storeKey, JSON.stringify(store))
  window.dispatchEvent(new Event(changeEvent))
}

function nextID(store: DemoStore, prefix = '30000000') {
  store.sequence += 1
  return `${prefix}-0000-4000-8000-${store.sequence.toString().padStart(12, '0')}`
}

const currentDemoUser = (store = readStore()) => store.users.find(user => user.id === localStorage.getItem(sessionKey)) || null

function demoSource(): WorkspaceDataSource {
  const requireUser = () => {
    const user = currentDemoUser()
    if (!user) throw new Error('Sign in required')
    return user
  }
  const requireResponder = () => {
    const user = requireUser()
    if (user.role === 'viewer') throw new Error('Responder role required')
    return user
  }
  return {
    mode: 'demo',
    async whoami() { return currentDemoUser() },
    async login() {
      const store = readStore()
      localStorage.setItem(sessionKey, store.users[0].id)
      return store.users[0]
    },
    async logout() { localStorage.removeItem(sessionKey) },
    async reset() {
      localStorage.removeItem(storeKey)
      localStorage.removeItem(sessionKey)
      readStore()
      window.dispatchEvent(new Event(changeEvent))
    },
    async users() { requireUser(); return readStore().users },
    async incidents(filters = {}) {
      requireUser()
      const store = readStore()
      const data = store.incidents
        .filter(item => !filters.status || item.status === filters.status)
        .filter(item => !filters.severity || item.severity === filters.severity)
        .filter(item => !filters.service || item.service.toLowerCase().includes(filters.service.toLowerCase()))
        .filter(item => !filters.assigneeId || item.assigneeId === filters.assigneeId)
        .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
      return { data, meta: { page: 1, limit: 100, total: data.length, totalPages: data.length ? 1 : 0 } }
    },
    async incident(id) {
      requireUser()
      const incident = readStore().incidents.find(item => item.id === id)
      if (!incident) throw new Error('Incident not found')
      return incident
    },
    async timeline(id) { requireUser(); return readStore().timeline.filter(event => event.incidentId === id).sort((a, b) => a.createdAt.localeCompare(b.createdAt)) },
    async create(input) {
      const actor = requireResponder()
      const store = readStore()
      const clusterEvent = input.sourceEventId ? initialEvents.find(event => event.id === input.sourceEventId) : undefined
      const severityByHealth: Record<string, Severity> = { unhealthy: 'critical', degraded: 'high', starting: 'medium', healthy: 'low', unknown: 'low' }
      const timestamp = new Date().toISOString()
      const incident: Incident = {
        id: nextID(store), title: input.title?.trim() || clusterEvent?.title || '',
        description: input.description?.trim() || clusterEvent?.detail || '', service: input.service?.trim() || clusterEvent?.serviceId || '',
        severity: input.severity || (clusterEvent ? severityByHealth[clusterEvent.status] : 'medium'), status: 'open',
        assigneeId: input.assigneeId, creatorId: actor.id, version: 1, createdAt: timestamp, updatedAt: timestamp,
        sourceEvent: clusterEvent ? { ...clusterEvent, timestamp: clusterEvent.timestamp.toISOString() } : undefined,
      }
      if (!incident.title || !incident.service) throw new Error('Title and service are required')
      store.incidents.push(incident)
      store.timeline.push({ id: nextID(store, '40000000'), incidentId: incident.id, type: 'created', actorId: actor.id, message: 'Incident created', createdAt: timestamp })
      writeStore(store)
      return incident
    },
    async patch(id, input) {
      const actor = requireResponder()
      const store = readStore()
      const incident = store.incidents.find(item => item.id === id)
      if (!incident) throw new Error('Incident not found')
      if (incident.version !== input.version) throw new Error('Incident was updated. Refresh and retry.')
      const manager = actor.role === 'admin' || actor.role === 'incident-manager'
      if ((input.title !== undefined || input.description !== undefined || input.service !== undefined || input.severity !== undefined) && !manager) throw new Error('Incident manager role required')
      if (input.status === 'resolved' && !manager) throw new Error('Incident manager role required')
      if (input.assigneeId !== undefined && !manager && input.assigneeId !== actor.id) throw new Error('Responders can only self-assign')
      const timestamp = new Date().toISOString()
      const addChange = (type: TimelineEvent['type'], message: string) => store.timeline.push({ id: nextID(store, '40000000'), incidentId: id, type, actorId: actor.id, message, createdAt: timestamp })
      if (input.status && input.status !== incident.status) { addChange('status.changed', `${incident.status} → ${input.status}`); incident.status = input.status }
      if (input.severity && input.severity !== incident.severity) { addChange('severity.changed', `${incident.severity} → ${input.severity}`); incident.severity = input.severity }
      if (input.assigneeId !== undefined || input.unassign) { incident.assigneeId = input.unassign ? undefined : input.assigneeId; addChange('assignee.changed', incident.assigneeId ? 'Assignee updated' : 'Incident unassigned') }
      if (input.title !== undefined) incident.title = input.title
      if (input.description !== undefined) incident.description = input.description
      if (input.service !== undefined) incident.service = input.service
      incident.version += 1; incident.updatedAt = timestamp
      writeStore(store)
      return incident
    },
    async comment(id, message) {
      const actor = requireResponder()
      if (!message.trim()) throw new Error('Comment cannot be empty')
      const store = readStore()
      if (!store.incidents.some(item => item.id === id)) throw new Error('Incident not found')
      const event: TimelineEvent = { id: nextID(store, '40000000'), incidentId: id, type: 'commented', actorId: actor.id, message: message.trim(), createdAt: new Date().toISOString() }
      store.timeline.push(event); writeStore(store); return event
    },
    async attach(id, file) {
      const actor = requireResponder()
      const store = readStore()
      const timestamp = new Date().toISOString()
      const attachment: Attachment = { id: nextID(store, '50000000'), userId: actor.id, originalName: file.name, size: file.size, mimetype: file.type || 'application/octet-stream', scope: 'incident', incidentId: id, createdAt: timestamp, updatedAt: timestamp }
      store.attachments.push(attachment)
      store.timeline.push({ id: nextID(store, '40000000'), incidentId: id, type: 'attachment.added', actorId: actor.id, message: file.name, fileId: attachment.id, createdAt: timestamp })
      writeStore(store); return attachment
    },
    async downloadAttachment() { throw new Error('Demo mode stores attachment metadata only') },
    async updateRole(userId, role) {
      const actor = requireUser()
      if (actor.role !== 'admin') throw new Error('Admin role required')
      const store = readStore(); const user = store.users.find(item => item.id === userId)
      if (!user) throw new Error('User not found')
      user.role = role; user.updatedAt = new Date().toISOString(); writeStore(store); return user
    },
    subscribe(onEvent) {
      const listener = () => onEvent()
      window.addEventListener(changeEvent, listener)
      window.addEventListener('storage', listener)
      return () => { window.removeEventListener(changeEvent, listener); window.removeEventListener('storage', listener) }
    },
  }
}

class HTTPError extends Error { constructor(public status: number, message: string) { super(message) } }

async function request<T>(path: string, options: RequestInit = {}, retry = true): Promise<T> {
  const form = options.body instanceof FormData
  const response = await fetch(`${apiRoot}${path}`, { ...options, credentials: 'include', headers: { ...(form ? {} : { 'Content-Type': 'application/json' }), ...options.headers } })
  if (response.status === 401 && retry && path !== '/auth/refresh') {
    const refreshed = await fetch(`${apiRoot}/auth/refresh`, { method: 'POST', credentials: 'include' })
    if (refreshed.ok) return request<T>(path, options, false)
  }
  const data = await response.json().catch(() => null) as { error?: string } | null
  if (!response.ok) throw new HTTPError(response.status, data?.error || response.statusText)
  return data as T
}

function apiSource(): WorkspaceDataSource {
  return {
    mode: 'api',
    async whoami() { try { return await request<WorkspaceUser>('/auth/whoami') } catch (error) { if (error instanceof HTTPError && error.status === 401) return null; throw error } },
    async login(email, password) { await request('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }); return request('/auth/whoami') },
    async logout() { await request('/auth/logout', { method: 'POST', body: '{}' }) },
    async reset() { throw new Error('Reset is available in demo mode only') },
    users: () => request('/api/v1/users'),
    incidents(filters = {}) {
      const params = new URLSearchParams()
      Object.entries(filters).forEach(([key, value]) => { if (value) params.set(key, value) })
      return request(`/api/v1/incidents${params.size ? `?${params}` : ''}`)
    },
    incident: id => request(`/api/v1/incidents/${id}`),
    timeline: id => request(`/api/v1/incidents/${id}/timeline`),
    create: input => request('/api/v1/incidents', { method: 'POST', body: JSON.stringify(input) }),
    patch: (id, input) => request(`/api/v1/incidents/${id}`, { method: 'PATCH', body: JSON.stringify(input) }),
    comment: (id, message) => request(`/api/v1/incidents/${id}/comments`, { method: 'POST', body: JSON.stringify({ message }) }),
    async attach(id, file) { const body = new FormData(); body.append('file', file); const result = await request<{ file: Attachment }>(`/api/v1/incidents/${id}/attachments`, { method: 'POST', body }); return result.file },
    async downloadAttachment(incidentId, fileId, filename) {
      const response = await fetch(`${apiRoot}/api/v1/incidents/${incidentId}/attachments/${fileId}`, { credentials: 'include' })
      if (!response.ok) throw new Error('Download failed')
      const objectURL = URL.createObjectURL(await response.blob()); const link = document.createElement('a'); link.href = objectURL; link.download = filename; link.click(); URL.revokeObjectURL(objectURL)
    },
    updateRole: (userId, role) => request(`/api/v1/users/${userId}/role`, { method: 'PATCH', body: JSON.stringify({ role }) }),
    subscribe(onEvent, onSessionExpired) {
      let closed = false; let stream: EventSource | null = null; let refreshed = false
      const connect = () => {
        if (closed) return
        stream = new EventSource(`${apiRoot}/api/v1/incidents/events`, { withCredentials: true })
        const update = () => { refreshed = false; onEvent() }
        ;['incident.created', 'incident.updated', 'timeline.appended'].forEach(type => stream?.addEventListener(type, update))
        stream.onerror = async () => {
          stream?.close()
          if (closed) return
          if (!refreshed) {
            refreshed = true
            try { await request('/auth/refresh', { method: 'POST', body: '{}' }, false); window.setTimeout(connect, 300); return } catch { /* expire below */ }
          }
          onSessionExpired()
        }
      }
      connect()
      return () => { closed = true; stream?.close() }
    },
  }
}

export const workspaceDataSource = mode === 'api' ? apiSource() : demoSource()
export const neuroOpsMode = mode
export type { IncidentStatus, Role }
