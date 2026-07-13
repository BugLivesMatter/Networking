export type Role = 'viewer' | 'responder' | 'incident-manager' | 'admin'
export type Severity = 'critical' | 'high' | 'medium' | 'low'
export type IncidentStatus = 'open' | 'investigating' | 'mitigated' | 'resolved'

export interface WorkspaceUser {
  id: string
  email: string
  displayName?: string
  role: Role
  createdAt?: string
  updatedAt?: string
}

export interface SourceEventSnapshot {
  id: string
  serviceId: string
  status: string
  title: string
  detail: string
  timestamp: string
}

export interface Incident {
  id: string
  title: string
  description: string
  service: string
  severity: Severity
  status: IncidentStatus
  assigneeId?: string
  creatorId: string
  sourceEvent?: SourceEventSnapshot
  version: number
  createdAt: string
  updatedAt: string
}

export interface TimelineEvent {
  id: string
  incidentId: string
  type: 'created' | 'commented' | 'status.changed' | 'severity.changed' | 'assignee.changed' | 'incident.edited' | 'attachment.added'
  actorId: string
  message?: string
  changes?: Record<string, unknown>
  fileId?: string
  createdAt: string
}

export interface Attachment {
  id: string
  userId: string
  originalName: string
  size: number
  mimetype: string
  scope?: string
  incidentId?: string
  createdAt: string
  updatedAt: string
}

export interface IncidentFilters {
  status?: IncidentStatus | ''
  severity?: Severity | ''
  service?: string
  assigneeId?: string
}

export interface CreateIncidentInput {
  title?: string
  description?: string
  service?: string
  severity?: Severity
  assigneeId?: string
  sourceEventId?: string
}

export interface PatchIncidentInput {
  version: number
  title?: string
  description?: string
  service?: string
  severity?: Severity
  status?: IncidentStatus
  assigneeId?: string
  unassign?: boolean
}

export interface IncidentPage {
  data: Incident[]
  meta: { page: number; limit: number; total: number; totalPages: number }
}

export interface WorkspaceDataSource {
  mode: 'demo' | 'api'
  whoami(): Promise<WorkspaceUser | null>
  login(email?: string, password?: string): Promise<WorkspaceUser>
  logout(): Promise<void>
  reset(): Promise<void>
  users(): Promise<WorkspaceUser[]>
  incidents(filters?: IncidentFilters): Promise<IncidentPage>
  incident(id: string): Promise<Incident>
  timeline(id: string): Promise<TimelineEvent[]>
  create(input: CreateIncidentInput): Promise<Incident>
  patch(id: string, input: PatchIncidentInput): Promise<Incident>
  comment(id: string, message: string): Promise<TimelineEvent>
  attach(id: string, file: File): Promise<Attachment>
  downloadAttachment(incidentId: string, fileId: string, filename: string): Promise<void>
  updateRole(userId: string, role: Role): Promise<WorkspaceUser>
  subscribe(onEvent: () => void, onSessionExpired: () => void): () => void
}

export const canRespond = (role?: Role) => role === 'responder' || role === 'incident-manager' || role === 'admin'
export const canManage = (role?: Role) => role === 'incident-manager' || role === 'admin'

