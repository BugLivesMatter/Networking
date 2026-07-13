import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from './App'

vi.mock('./dashboard/ClusterScene', () => ({ default: () => <div data-testid="cluster-scene" /> }))

describe('NeuroOps workspace', () => {
  beforeEach(() => localStorage.clear())

  it('switches tabs without changing the URL', async () => {
    const original = window.location.href
    const view = render(<App />)
    await userEvent.click(screen.getByRole('button', { name: 'Workloads' }))
    expect(await screen.findByRole('heading', { name: 'Workloads' })).toBeInTheDocument()
    expect(window.location.href).toBe(original)
  })

  it('opens Incidents after one-click demo login and restores that session', async () => {
    const view = render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: /Sign in/i }))
    expect(screen.getByRole('heading', { name: 'Enter demo workspace' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Enter workspace' }))
    expect(await screen.findByRole('heading', { name: 'Incidents' })).toBeInTheDocument()
    view.unmount()
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Incidents' })).toBeInTheDocument()
  })

  it('creates and persists a manual incident', async () => {
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: /Sign in/i }))
    await userEvent.click(screen.getByRole('button', { name: 'Enter workspace' }))
    await userEvent.click(await screen.findByRole('button', { name: /New incident/i }))
    await userEvent.type(screen.getByLabelText('Title'), 'Synthetic checkout outage')
    await userEvent.type(screen.getByLabelText('Service'), 'gateway')
    await userEvent.click(screen.getByRole('button', { name: 'Create incident' }))
    expect((await screen.findAllByText('Synthetic checkout outage')).length).toBeGreaterThan(0)
    expect(localStorage.getItem('neuroops.workspace.v1')).toContain('Synthetic checkout outage')
  })

  it('reset removes the demo session and recreates deterministic seeds', async () => {
    localStorage.setItem('neuroops.session.v1', 'stale')
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: /Sign in/i }))
    await userEvent.click(screen.getByRole('button', { name: /Reset demo data/i }))
    await waitFor(() => expect(localStorage.getItem('neuroops.session.v1')).toBeNull())
    expect(localStorage.getItem('neuroops.workspace.v1')).toContain('Redis latency above SLO')
  })

  it('hides response actions from a viewer', async () => {
    const view = render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: /Sign in/i }))
    await userEvent.click(screen.getByRole('button', { name: 'Enter workspace' }))
    const store = JSON.parse(localStorage.getItem('neuroops.workspace.v1')!)
    store.users[0].role = 'viewer'
    localStorage.setItem('neuroops.workspace.v1', JSON.stringify(store))
    view.unmount()
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Incidents' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /New incident/i })).not.toBeInTheDocument()
  })
})
