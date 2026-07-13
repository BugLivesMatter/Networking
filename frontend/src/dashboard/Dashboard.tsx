import { lazy, Suspense, useMemo, useState } from 'react'
import {
  Activity,
  Box,
  ChevronRight,
  CircleDot,
  DatabaseZap,
  GitFork,
  LogIn,
  Maximize2,
  Network,
  Play,
  RotateCcw,
  ServerCrash,
  Sparkles,
  Waves,
  Zap,
} from 'lucide-react'
import { statusLabel } from './demoCluster'
import type { DemoScenario, HealthStatus } from './types'
import { useClusterSource } from './useClusterSource'

const statusClass: Record<HealthStatus, string> = {
  healthy: 'status-healthy',
  degraded: 'status-degraded',
  unhealthy: 'status-unhealthy',
  starting: 'status-starting',
  unknown: 'status-unknown',
}

const scenarioActions: Array<{
  id: DemoScenario
  label: string
  hint: string
  icon: typeof Zap
}> = [
  { id: 'latency', label: 'Redis latency', hint: 'Inject 286 ms', icon: Waves },
  { id: 'crash', label: 'Crash API pod', hint: 'Fail readiness', icon: ServerCrash },
  { id: 'scale', label: 'Scale API', hint: 'Add replica', icon: Maximize2 },
  { id: 'recover', label: 'Recover', hint: 'Restore cluster', icon: RotateCcw },
]

const compactNumber = new Intl.NumberFormat('en', { notation: 'compact', maximumFractionDigits: 1 })
const ClusterScene = lazy(() => import('./ClusterScene'))

export default function Dashboard() {
  const { snapshot, runScenario, sourceMode } = useClusterSource()
  const [selectedId, setSelectedId] = useState('api')
  const [scenario, setScenario] = useState<DemoScenario | null>(null)
  const selected = snapshot.services.find(service => service.id === selectedId) ?? snapshot.services[0]

  const totals = useMemo(() => {
    const instances = snapshot.services.flatMap(service => service.instances)
    return {
      services: snapshot.services.filter(service => service.status !== 'unhealthy').length,
      totalServices: snapshot.services.length,
      pods: instances.length,
      latency: Math.round(snapshot.services.reduce((sum, service) => sum + service.latency, 0) / snapshot.services.length),
      incidents: instances.filter(instance => instance.status === 'unhealthy').length,
    }
  }, [snapshot.services])

  const executeScenario = (next: DemoScenario) => {
    setScenario(next)
    runScenario(next)
    window.setTimeout(() => setScenario(null), 520)
  }

  return (
    <main className="dashboard-shell">
      <div className="ambient ambient-one" />
      <div className="ambient ambient-two" />
      <div className="noise" />

      <header className="topbar glass-panel">
        <a href="#" className="brand" aria-label="NeuroOps home">
          <span className="brand-mark"><Network size={17} /></span>
          <span className="brand-word">NEURO<span>OPS</span></span>
          <span className="preview-pill">PUBLIC PREVIEW</span>
        </a>
        <nav className="topnav" aria-label="Primary navigation">
          <a href="#overview" className="active">Overview</a>
          <a href="#services">Services</a>
          <a href="#events">Events</a>
          <a href="#about">Architecture</a>
        </nav>
        <div className="topbar-actions">
          <a className="icon-button" href="https://github.com/BugLivesMatter/Networking" target="_blank" rel="noreferrer" aria-label="GitHub repository"><GitFork size={17} /></a>
          <button className="login-button"><LogIn size={15} /> Sign in</button>
        </div>
      </header>

      <section className="hero-copy" id="overview">
        <div className="eyebrow"><span className="live-dot" /> LIVE DEMO CLUSTER <span className="eyebrow-divider" /> {sourceMode}</div>
        <h1>Your infrastructure,<br /><em>alive.</em></h1>
        <p>A living topology of services, pods and dependencies.<br />Rotate the cluster. Break it. Watch it recover.</p>
      </section>

      <section className="scene-wrap" aria-label="Interactive cluster topology">
        <Suspense fallback={<div className="scene-loader"><Network size={20} /><span>Initializing topology</span></div>}>
          <ClusterScene services={snapshot.services} selectedId={selected.id} onSelect={setSelectedId} />
        </Suspense>
        <div className="scene-hint"><CircleDot size={13} /> Drag to orbit · Scroll to zoom · Select a node</div>
      </section>

      <aside className="summary-panel glass-panel">
        <div className="panel-kicker"><Activity size={14} /> Cluster pulse</div>
        <div className="pulse-state"><span className={totals.incidents ? 'pulse-orb pulse-orb--alert' : 'pulse-orb'} />{totals.incidents ? 'Action required' : 'All systems nominal'}</div>
        <div className="metric-grid">
          <div><strong>{totals.services}<small>/{totals.totalServices}</small></strong><span>Services</span></div>
          <div><strong>{totals.pods}</strong><span>Instances</span></div>
          <div><strong>{totals.latency}<small>ms</small></strong><span>Avg latency</span></div>
          <div><strong>{totals.incidents}</strong><span>Failures</span></div>
        </div>
        <div className="last-sync"><span>Last topology sync</span><time>{snapshot.generatedAt.toLocaleTimeString('en-GB')}</time></div>
      </aside>

      <aside className="details-panel glass-panel">
        <div className="details-title-row">
          <div className={`service-glyph ${statusClass[selected.status]}`}><Box size={17} /></div>
          <div><span>Selected service</span><h2>{selected.name}</h2></div>
          <span className={`status-badge ${statusClass[selected.status]}`}><i />{statusLabel[selected.status]}</span>
        </div>
        <p className="service-description">{selected.description}</p>
        <div className="details-metrics">
          <div><span>Latency p95</span><strong>{selected.latency} ms</strong></div>
          <div><span>Uptime 30d</span><strong>{selected.uptime}%</strong></div>
          <div><span>Requests/min</span><strong>{compactNumber.format(selected.requestsPerMinute)}</strong></div>
          <div><span>Error rate</span><strong>{selected.errorRate}%</strong></div>
        </div>
        <div className="instances-head"><span>Instances</span><small>{selected.version}</small></div>
        <div className="instance-list">
          {selected.instances.map(instance => (
            <button key={instance.id} className="instance-row">
              <i className={statusClass[instance.status]} />
              <span>{instance.name}</span>
              <small>{instance.status === 'starting' ? 'pending' : instance.status === 'unhealthy' ? 'timeout' : `${instance.latency} ms`}</small>
              <ChevronRight size={13} />
            </button>
          ))}
        </div>
      </aside>

      <section className="demo-controls glass-panel">
        <div className="controls-heading">
          <span><Play size={13} /> CHAOS LAB</span>
          <small>Safe simulated actions</small>
        </div>
        <div className="scenario-grid">
          {scenarioActions.map(action => {
            const Icon = action.icon
            return (
              <button
                key={action.id}
                className={scenario === action.id ? 'scenario-button is-running' : 'scenario-button'}
                onClick={() => executeScenario(action.id)}
              >
                <Icon size={15} />
                <span>{action.label}<small>{action.hint}</small></span>
              </button>
            )
          })}
        </div>
      </section>

      <section className="services-strip glass-panel" id="services">
        <div className="strip-header">
          <span><DatabaseZap size={14} /> SERVICE HEALTH</span>
          <small>Click any row to focus the node</small>
        </div>
        <div className="services-row">
          {snapshot.services.map(service => (
            <button
              key={service.id}
              className={service.id === selected.id ? 'service-chip is-selected' : 'service-chip'}
              onClick={() => setSelectedId(service.id)}
            >
              <i className={statusClass[service.status]} />
              <span>{service.name}<small>{service.instances.length} pods · {service.latency} ms</small></span>
              <Activity size={14} />
            </button>
          ))}
        </div>
      </section>

      <section className="event-toast glass-panel" id="events">
        <Sparkles size={14} />
        <div><strong>{snapshot.events[0].title}</strong><span>{snapshot.events[0].detail}</span></div>
        <time>{snapshot.events[0].timestamp.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</time>
      </section>
    </main>
  )
}
