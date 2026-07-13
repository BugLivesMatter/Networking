import { useMemo, useState } from 'react'
import { Activity, ArrowRight, Box, GitBranch, Plus, Server } from 'lucide-react'
import { statusLabel } from '../dashboard/demoCluster'
import type { ClusterSnapshot, HealthStatus } from '../dashboard/types'

const statusClass = (status: HealthStatus) => `status-${status}`

export function WorkloadsView({ snapshot }: { snapshot: ClusterSnapshot }) {
  const [filter, setFilter] = useState<HealthStatus | 'all'>('all')
  const [selected, setSelected] = useState(snapshot.services[0]?.id)
  const services = snapshot.services.filter(service => filter === 'all' || service.status === filter)
  const detail = snapshot.services.find(service => service.id === selected) || services[0]
  return <WorkspacePage eyebrow="Cluster inventory" title="Workloads" description="Services, runtime health and replica-level details.">
    <div className="workspace-toolbar" aria-label="Workload filters">
      {(['all', 'healthy', 'degraded', 'unhealthy', 'starting'] as const).map(status => <button key={status} className={filter === status ? 'is-active' : ''} onClick={() => setFilter(status)}>{status}</button>)}
    </div>
    <div className="workspace-split">
      <section className="workspace-card table-card">
        <table><thead><tr><th>Service</th><th>Status</th><th>Replicas</th><th>p95</th><th>Version</th></tr></thead>
          <tbody>{services.map(service => <tr key={service.id} className={detail?.id === service.id ? 'selected-row' : ''} onClick={() => setSelected(service.id)}><td><Box size={14} /> {service.name}<small>{service.description}</small></td><td><span className={`workspace-status ${statusClass(service.status)}`}><i />{statusLabel[service.status]}</span></td><td>{service.instances.length}</td><td>{service.latency} ms</td><td>{service.version}</td></tr>)}</tbody>
        </table>
      </section>
      {detail && <aside className="workspace-card detail-card"><div className="card-kicker"><Server size={14} /> Replica details</div><h2>{detail.name}</h2><p>{detail.description}</p>{detail.instances.map(instance => <div className="replica-detail" key={instance.id}><i className={statusClass(instance.status)} /><span>{instance.name}<small>{instance.restarts} restarts</small></span><strong>{instance.latency} ms</strong></div>)}</aside>}
    </div>
  </WorkspacePage>
}

export function EventsView({ snapshot, canCreate, onCreate }: { snapshot: ClusterSnapshot; canCreate: boolean; onCreate: (eventID: string) => void }) {
  const [status, setStatus] = useState<HealthStatus | 'all'>('all')
  const [service, setService] = useState('all')
  const events = snapshot.events.filter(event => (status === 'all' || event.status === status) && (service === 'all' || event.serviceId === service))
  return <WorkspacePage eyebrow="Cluster activity" title="Events" description="The complete local event stream, ready to promote into incidents.">
    <div className="workspace-toolbar"><select aria-label="Event status" value={status} onChange={event => setStatus(event.target.value as HealthStatus | 'all')}><option value="all">All statuses</option><option value="healthy">Healthy</option><option value="degraded">Degraded</option><option value="unhealthy">Unhealthy</option><option value="starting">Starting</option></select><select aria-label="Event service" value={service} onChange={event => setService(event.target.value)}><option value="all">All services</option>{snapshot.services.map(item => <option value={item.id} key={item.id}>{item.name}</option>)}</select></div>
    <section className="workspace-card event-feed">{events.map(event => <article key={event.id}><span className={`event-indicator ${statusClass(event.status)}`}><Activity size={15} /></span><div><strong>{event.title}</strong><p>{event.detail}</p><small>{event.serviceId} · {event.timestamp.toLocaleString()}</small></div>{canCreate && <button className="secondary-action" onClick={() => onCreate(event.id)}><Plus size={13} /> Create incident</button>}</article>)}</section>
  </WorkspacePage>
}

export function ArchitectureView({ snapshot }: { snapshot: ClusterSnapshot }) {
  const links = useMemo(() => snapshot.services.flatMap(service => service.dependencies.map(target => ({ source: service, target: snapshot.services.find(item => item.id === target) }))).filter(link => link.target), [snapshot.services])
  return <WorkspacePage eyebrow="Runtime topology" title="Architecture" description="A readable dependency map derived from the live service model.">
    <section className="workspace-card architecture-map"><div className="architecture-root"><GitBranch size={18} /><strong>Request path</strong></div>{links.map(link => <div className="dependency-link" key={`${link.source.id}-${link.target?.id}`}><div><span className={`node-dot ${statusClass(link.source.status)}`} /><strong>{link.source.name}</strong><small>{link.source.kind}</small></div><ArrowRight size={18} /><div><span className={`node-dot ${statusClass(link.target!.status)}`} /><strong>{link.target!.name}</strong><small>{link.target!.kind}</small></div></div>)}</section>
  </WorkspacePage>
}

export function WorkspacePage({ eyebrow, title, description, children }: { eyebrow: string; title: string; description: string; children: React.ReactNode }) {
  return <div className="workspace-page"><header className="workspace-heading"><span>{eyebrow}</span><h1>{title}</h1><p>{description}</p></header>{children}</div>
}

