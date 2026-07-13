import { useCallback, useEffect, useState } from 'react'
import { GitFork, LogIn, LogOut, Network, RotateCcw, ShieldCheck } from 'lucide-react'
import Dashboard from './dashboard/Dashboard'
import { useClusterSource } from './dashboard/useClusterSource'
import { ArchitectureView, EventsView, WorkloadsView } from './workspace/ClusterViews'
import IncidentsView, { RoleAdminPanel } from './workspace/IncidentsView'
import { neuroOpsMode, workspaceDataSource } from './workspace/dataSource'
import type { WorkspaceUser } from './workspace/types'
import { canRespond } from './workspace/types'

type Tab = 'Overview' | 'Workloads' | 'Events' | 'Architecture' | 'Incidents'
const publicTabs: Tab[] = ['Overview', 'Workloads', 'Events', 'Architecture']

export default function App() {
  const cluster = useClusterSource()
  const [user, setUser] = useState<WorkspaceUser | null>(null)
  const [loadingSession, setLoadingSession] = useState(true)
  const [tab, setTab] = useState<Tab>('Overview')
  const [loginOpen, setLoginOpen] = useState(false)
  const [rolesOpen, setRolesOpen] = useState(false)
  const [roleUsers, setRoleUsers] = useState<WorkspaceUser[]>([])
  const [eventIncidentID, setEventIncidentID] = useState<string>()

  useEffect(() => {
    void workspaceDataSource.whoami().then(current => {
      setUser(current)
      if (current) setTab('Incidents')
    }).finally(() => setLoadingSession(false))
  }, [])

  const expireSession = useCallback(() => {
    setUser(null); setTab('Overview')
  }, [])

  const logout = async () => {
    try { await workspaceDataSource.logout() } finally { expireSession() }
  }

  const openRoles = async () => {
    setRoleUsers(await workspaceDataSource.users()); setRolesOpen(true)
  }

  const createFromEvent = (eventID: string) => {
    setEventIncidentID(eventID); setTab('Incidents')
  }

  const tabs = user ? [...publicTabs, 'Incidents' as const] : publicTabs
  return <main className={tab === 'Overview' ? 'dashboard-shell' : 'dashboard-shell workspace-shell'}>
    <div className="ambient ambient-one" /><div className="ambient ambient-two" /><div className="noise" />
    <header className="topbar glass-panel">
      <button className="brand brand-button" aria-label="NeuroOps home" onClick={() => setTab('Overview')}><span className="brand-mark"><Network size={17} /></span><span className="brand-word">NEURO<span>OPS</span></span><span className="preview-pill">{neuroOpsMode === 'demo' ? 'DEMO WORKSPACE' : 'LIVE WORKSPACE'}</span></button>
      <nav className="topnav" aria-label="Primary navigation">{tabs.map(item => <button key={item} className={tab === item ? 'active' : ''} onClick={() => setTab(item)}>{item}</button>)}</nav>
      <div className="topbar-actions"><a className="icon-button" href="https://github.com/BugLivesMatter/Networking" target="_blank" rel="noreferrer" aria-label="GitHub repository"><GitFork size={17} /></a>{user?.role === 'admin' && <button className="icon-button" onClick={() => void openRoles()} aria-label="Manage roles"><ShieldCheck size={16} /></button>}{user ? <button className="login-button" onClick={() => void logout()}><LogOut size={15} /><span>{user.displayName || user.email}</span></button> : <button className="login-button" disabled={loadingSession} onClick={() => setLoginOpen(true)}><LogIn size={15} /> Sign in</button>}</div>
    </header>

    {tab === 'Overview' && <Dashboard {...cluster} />}
    {tab === 'Workloads' && <WorkloadsView snapshot={cluster.snapshot} />}
    {tab === 'Events' && <EventsView snapshot={cluster.snapshot} canCreate={!!user && canRespond(user.role)} onCreate={createFromEvent} />}
    {tab === 'Architecture' && <ArchitectureView snapshot={cluster.snapshot} />}
    {tab === 'Incidents' && user && <IncidentsView source={workspaceDataSource} user={user} snapshot={cluster.snapshot} initialEventID={eventIncidentID} clearInitialEvent={() => setEventIncidentID(undefined)} onSessionExpired={expireSession} />}

    {loginOpen && <LoginModal onClose={() => setLoginOpen(false)} onReset={() => { expireSession(); setLoginOpen(false) }} onLogin={current => { setUser(current); setTab('Incidents'); setLoginOpen(false) }} />}
    {rolesOpen && <div className="modal-backdrop" role="presentation"><div className="workspace-modal glass-panel"><h2>Workspace roles</h2><RoleAdminPanel source={workspaceDataSource} users={roleUsers} onChange={() => void workspaceDataSource.users().then(setRoleUsers)} /><div className="modal-actions"><button className="secondary-action" onClick={() => setRolesOpen(false)}>Close</button></div></div></div>}
  </main>
}

function LoginModal({ onClose, onReset, onLogin }: { onClose(): void; onReset(): void; onLogin(user: WorkspaceUser): void }) {
  const [email, setEmail] = useState(''); const [password, setPassword] = useState(''); const [error, setError] = useState(''); const [busy, setBusy] = useState(false)
  const login = async (event: React.FormEvent) => { event.preventDefault(); setBusy(true); setError(''); try { onLogin(await workspaceDataSource.login(email, password)) } catch (reason) { setError(reason instanceof Error ? reason.message : 'Sign in failed'); setBusy(false) } }
  const reset = async () => { await workspaceDataSource.reset(); onReset() }
  return <div className="modal-backdrop" role="presentation"><form className="workspace-modal login-modal glass-panel" onSubmit={login}><div className="brand-mark"><Network size={18} /></div><span className="modal-eyebrow">{neuroOpsMode === 'demo' ? 'LOCAL DEMO SESSION' : 'SECURE API SESSION'}</span><h2>{neuroOpsMode === 'demo' ? 'Enter demo workspace' : 'Sign in to NeuroOps'}</h2><p>{neuroOpsMode === 'demo' ? 'A one-click local admin session. Your incidents persist in this browser.' : 'Use your API account. Authentication remains in secure cookies.'}</p>{error && <div className={error === 'Demo data reset' ? 'workspace-notice' : 'workspace-error'}>{error}</div>}{neuroOpsMode === 'api' && <><label>Email<input type="email" required value={email} onChange={event => setEmail(event.target.value)} /></label><label>Password<input type="password" required value={password} onChange={event => setPassword(event.target.value)} /></label></>}<button className="primary-action login-submit" disabled={busy}>{busy ? 'Entering…' : neuroOpsMode === 'demo' ? 'Enter workspace' : 'Sign in'}</button><div className="modal-actions">{neuroOpsMode === 'demo' && <button type="button" className="text-action" onClick={() => void reset()}><RotateCcw size={13} /> Reset demo data</button>}<button type="button" className="text-action" onClick={onClose}>Cancel</button></div></form></div>
}
