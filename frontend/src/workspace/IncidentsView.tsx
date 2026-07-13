import { useCallback, useEffect, useMemo, useState } from 'react'
import { AlertTriangle, MessageSquare, Paperclip, Plus, RefreshCw, ShieldCheck, UserRound } from 'lucide-react'
import type { ClusterSnapshot } from '../dashboard/types'
import type { CreateIncidentInput, Incident, IncidentFilters, IncidentStatus, Role, Severity, TimelineEvent, WorkspaceDataSource, WorkspaceUser } from './types'
import { canManage, canRespond } from './types'
import { WorkspacePage } from './ClusterViews'

const statuses: IncidentStatus[] = ['open', 'investigating', 'mitigated', 'resolved']
const severities: Severity[] = ['critical', 'high', 'medium', 'low']

interface Props {
  source: WorkspaceDataSource
  user: WorkspaceUser
  snapshot: ClusterSnapshot
  initialEventID?: string
  clearInitialEvent(): void
  onSessionExpired(): void
}

export default function IncidentsView({ source, user, snapshot, initialEventID, clearInitialEvent, onSessionExpired }: Props) {
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [users, setUsers] = useState<WorkspaceUser[]>([])
  const [selectedID, setSelectedID] = useState<string>()
  const [selected, setSelected] = useState<Incident>()
  const [timeline, setTimeline] = useState<TimelineEvent[]>([])
  const [filters, setFilters] = useState<IncidentFilters>({})
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const refreshList = useCallback(async () => {
    try {
      const [page, workspaceUsers] = await Promise.all([source.incidents(filters), source.users()])
      setIncidents(page.data); setUsers(workspaceUsers)
      setSelectedID(current => current || page.data[0]?.id)
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'Unable to load incidents') }
  }, [filters, source])

  const refreshSelected = useCallback(async () => {
    if (!selectedID) { setSelected(undefined); setTimeline([]); return }
    try { const [incident, events] = await Promise.all([source.incident(selectedID), source.timeline(selectedID)]); setSelected(incident); setTimeline(events) }
    catch (reason) { setError(reason instanceof Error ? reason.message : 'Unable to load incident') }
  }, [selectedID, source])

  useEffect(() => { void refreshList() }, [refreshList])
  useEffect(() => { void refreshSelected() }, [refreshSelected])
  useEffect(() => source.subscribe(() => { void refreshList(); void refreshSelected() }, onSessionExpired), [onSessionExpired, refreshList, refreshSelected, source])
  useEffect(() => { if (initialEventID) setCreating(true) }, [initialEventID])

  const mutate = async (action: () => Promise<unknown>) => {
    setBusy(true); setError('')
    try { await action(); await refreshList(); await refreshSelected() } catch (reason) { setError(reason instanceof Error ? reason.message : 'Operation failed') } finally { setBusy(false) }
  }

  return <WorkspacePage eyebrow="Response operations" title="Incidents" description="Own the incident lifecycle, evidence and immutable response history.">
    <div className="incident-actions"><div className="workspace-toolbar"><select aria-label="Incident status filter" value={filters.status || ''} onChange={event => setFilters(current => ({ ...current, status: event.target.value as IncidentStatus | '' }))}><option value="">All statuses</option>{statuses.map(status => <option key={status}>{status}</option>)}</select><select aria-label="Incident severity filter" value={filters.severity || ''} onChange={event => setFilters(current => ({ ...current, severity: event.target.value as Severity | '' }))}><option value="">All severities</option>{severities.map(severity => <option key={severity}>{severity}</option>)}</select><input aria-label="Service filter" placeholder="Filter service" value={filters.service || ''} onChange={event => setFilters(current => ({ ...current, service: event.target.value }))} /></div>{canRespond(user.role) && <button className="primary-action" onClick={() => { clearInitialEvent(); setCreating(true) }}><Plus size={14} /> New incident</button>}</div>
    {error && <div className="workspace-error"><AlertTriangle size={14} />{error}<button onClick={() => setError('')}>Dismiss</button></div>}
    <div className="incident-layout">
      <section className="workspace-card incident-list"><div className="card-kicker"><RefreshCw size={13} /> {incidents.length} incidents</div>{incidents.map(incident => <button key={incident.id} className={selectedID === incident.id ? 'incident-row is-selected' : 'incident-row'} onClick={() => setSelectedID(incident.id)}><span className={`severity-mark severity-${incident.severity}`} /><span><strong>{incident.title}</strong><small>{incident.service} · v{incident.version}</small></span><span className={`incident-status status-${incident.status}`}>{incident.status}</span></button>)}</section>
      <section className="workspace-card incident-detail">{selected ? <IncidentDetail incident={selected} timeline={timeline} users={users} user={user} busy={busy} source={source} mutate={mutate} snapshot={snapshot} /> : <div className="empty-state">Select an incident to inspect its response timeline.</div>}</section>
    </div>
    {creating && <CreateIncidentModal source={source} users={users} snapshot={snapshot} sourceEventID={initialEventID} onClose={() => { setCreating(false); clearInitialEvent() }} onCreated={incident => { setCreating(false); clearInitialEvent(); setSelectedID(incident.id); void refreshList() }} />}
  </WorkspacePage>
}

function IncidentDetail({ incident, timeline, users, user, busy, source, mutate }: { incident: Incident; timeline: TimelineEvent[]; users: WorkspaceUser[]; user: WorkspaceUser; busy: boolean; source: WorkspaceDataSource; mutate(action: () => Promise<unknown>): Promise<void>; snapshot: ClusterSnapshot }) {
  const [comment, setComment] = useState('')
  const assignee = users.find(item => item.id === incident.assigneeId)
  const responder = canRespond(user.role); const manager = canManage(user.role)
  const nextStatuses = statuses.filter(status => statuses.indexOf(status) > statuses.indexOf(incident.status) && (manager || status !== 'resolved'))
  const update = (input: Partial<{ status: IncidentStatus; severity: Severity; assigneeId: string; unassign: boolean }>) => mutate(() => source.patch(incident.id, { version: incident.version, ...input }))
  return <><header className="incident-title"><div><span className={`severity-pill severity-${incident.severity}`}>{incident.severity}</span><h2>{incident.title}</h2><p>{incident.description || 'No description provided.'}</p></div><span className={`incident-status status-${incident.status}`}>{incident.status}</span></header>
    <div className="incident-meta"><div><span>Service</span><strong>{incident.service}</strong></div><div><span>Assignee</span><strong>{assignee?.displayName || assignee?.email || 'Unassigned'}</strong></div><div><span>Version</span><strong>{incident.version}</strong></div></div>
    {responder && <div className="incident-controls">
      {nextStatuses.length > 0 && <label>Status<select disabled={busy} value="" onChange={event => event.target.value && void update({ status: event.target.value as IncidentStatus })}><option value="">Move to…</option>{nextStatuses.map(status => <option key={status}>{status}</option>)}</select></label>}
      {manager ? <><label>Severity<select disabled={busy} value={incident.severity} onChange={event => void update({ severity: event.target.value as Severity })}>{severities.map(severity => <option key={severity}>{severity}</option>)}</select></label><label>Assignee<select disabled={busy} value={incident.assigneeId || ''} onChange={event => void update(event.target.value ? { assigneeId: event.target.value } : { unassign: true })}><option value="">Unassigned</option>{users.map(item => <option value={item.id} key={item.id}>{item.displayName || item.email}</option>)}</select></label></> : !incident.assigneeId && <button className="secondary-action" onClick={() => void update({ assigneeId: user.id })}><UserRound size={13} /> Self-assign</button>}
      <label className="attachment-button"><Paperclip size={13} /> Add attachment<input type="file" disabled={busy} onChange={event => { const file = event.target.files?.[0]; if (file) void mutate(() => source.attach(incident.id, file)) }} /></label>
    </div>}
    <div className="timeline"><div className="card-kicker"><ShieldCheck size={13} /> Immutable timeline</div>{timeline.map(event => <article key={event.id}><i /><div><strong>{event.type.replace('.', ' ')}</strong><p>{event.message || summarizeChanges(event.changes)}</p><small>{users.find(item => item.id === event.actorId)?.displayName || 'Workspace user'} · {new Date(event.createdAt).toLocaleString()}</small>{event.type === 'attachment.added' && event.fileId && <button className="text-action" onClick={() => void source.downloadAttachment(incident.id, event.fileId!, event.message || 'attachment')}>Download</button>}</div></article>)}</div>
    {responder && <form className="comment-form" onSubmit={event => { event.preventDefault(); const message = comment; if (!message.trim()) return; setComment(''); void mutate(() => source.comment(incident.id, message)) }}><MessageSquare size={14} /><input aria-label="Incident comment" value={comment} onChange={event => setComment(event.target.value)} placeholder="Add a response note…" /><button disabled={busy || !comment.trim()}>Comment</button></form>}
  </>
}

function CreateIncidentModal({ source, users, snapshot, sourceEventID, onClose, onCreated }: { source: WorkspaceDataSource; users: WorkspaceUser[]; snapshot: ClusterSnapshot; sourceEventID?: string; onClose(): void; onCreated(incident: Incident): void }) {
  const clusterEvent = snapshot.events.find(event => event.id === sourceEventID)
  const [input, setInput] = useState<CreateIncidentInput>({ sourceEventId: sourceEventID, title: clusterEvent?.title || '', description: clusterEvent?.detail || '', service: clusterEvent?.serviceId || '', severity: clusterEvent ? undefined : 'medium' })
  const [error, setError] = useState(''); const [busy, setBusy] = useState(false)
  const submit = async (event: React.FormEvent) => { event.preventDefault(); setBusy(true); setError(''); try { onCreated(await source.create(input)) } catch (reason) { setError(reason instanceof Error ? reason.message : 'Unable to create incident'); setBusy(false) } }
  return <div className="modal-backdrop" role="presentation"><form className="workspace-modal glass-panel" onSubmit={submit}><div className="card-kicker"><Plus size={13} /> {clusterEvent ? 'Create from cluster event' : 'Manual incident'}</div><h2>New incident</h2>{error && <div className="workspace-error">{error}</div>}<label>Title<input autoFocus required value={input.title || ''} onChange={event => setInput(current => ({ ...current, title: event.target.value }))} /></label><label>Description<textarea rows={3} value={input.description || ''} onChange={event => setInput(current => ({ ...current, description: event.target.value }))} /></label><div className="form-grid"><label>Service<input required value={input.service || ''} onChange={event => setInput(current => ({ ...current, service: event.target.value }))} /></label><label>Severity<select value={input.severity || ''} onChange={event => setInput(current => ({ ...current, severity: event.target.value as Severity }))}><option value="">Derived from event</option>{severities.map(item => <option key={item}>{item}</option>)}</select></label></div><label>Assignee<select value={input.assigneeId || ''} onChange={event => setInput(current => ({ ...current, assigneeId: event.target.value || undefined }))}><option value="">Unassigned</option>{users.map(item => <option value={item.id} key={item.id}>{item.displayName || item.email}</option>)}</select></label><div className="modal-actions"><button type="button" className="secondary-action" onClick={onClose}>Cancel</button><button className="primary-action" disabled={busy}>{busy ? 'Creating…' : 'Create incident'}</button></div></form></div>
}

function summarizeChanges(changes?: Record<string, unknown>) { return changes ? Object.keys(changes).join(', ') + ' updated' : 'Workspace state updated' }

export function RoleAdminPanel({ source, users, onChange }: { source: WorkspaceDataSource; users: WorkspaceUser[]; onChange(): void }) {
  return <section className="workspace-card role-panel"><div className="card-kicker"><ShieldCheck size={13} /> Role administration</div>{users.map(user => <label key={user.id}><span>{user.displayName || user.email}<small>{user.email}</small></span><select value={user.role} onChange={event => void source.updateRole(user.id, event.target.value as Role).then(onChange)}><option value="viewer">viewer</option><option value="responder">responder</option><option value="incident-manager">incident-manager</option><option value="admin">admin</option></select></label>)}</section>
}
