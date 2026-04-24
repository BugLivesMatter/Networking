import { useState, useCallback, useEffect } from 'react'
import type { UserResponse, ToastMsg, Section } from './types'
import AuthSection from './sections/AuthSection'
import ProfileSection from './sections/ProfileSection'
import CategoriesSection from './sections/CategoriesSection'
import ProductsSection from './sections/ProductsSection'
import FilesSection from './sections/FilesSection'
import HealthSection from './sections/HealthSection'
import { api } from './api'

const NAV: { id: Section; label: string; icon: string }[] = [
  { id: 'auth', label: 'Авторизация', icon: '🔐' },
  { id: 'profile', label: 'Профиль', icon: '👤' },
  { id: 'categories', label: 'Категории', icon: '🗂️' },
  { id: 'products', label: 'Продукты', icon: '📦' },
  { id: 'files', label: 'Файлы', icon: '📁' },
  { id: 'health', label: 'Health', icon: '💚' },
]

export default function App() {
  const [section, setSection] = useState<Section>('auth')
  const [user, setUser] = useState<UserResponse | null>(null)
  const [toast, setToast] = useState<ToastMsg | null>(null)

  const showToast = useCallback((text: string, type: 'success' | 'error' = 'success') => {
    setToast({ text, type })
    setTimeout(() => setToast(null), 3500)
  }, [])

  useEffect(() => {
    api.auth.whoami().then(setUser).catch(() => {})
  }, [])

  return (
    <div className="flex h-screen bg-slate-950 text-slate-100 overflow-hidden">
      {/* Sidebar */}
      <aside className="w-60 flex-shrink-0 flex flex-col bg-slate-900 border-r border-slate-800">
        {/* Logo */}
        <div className="px-5 py-5 border-b border-slate-800">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-violet-500 to-indigo-600 flex items-center justify-center text-sm font-bold">L</div>
            <div>
              <p className="text-sm font-semibold text-slate-100">Lab API Tester</p>
              <p className="text-xs text-slate-500">ЛР7 · MinIO</p>
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav className="flex-1 px-3 py-4 space-y-1">
          {NAV.map(item => (
            <button
              key={item.id}
              onClick={() => setSection(item.id)}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150 ${
                section === item.id
                  ? 'bg-violet-600/20 text-violet-300 border border-violet-500/30'
                  : 'text-slate-400 hover:bg-slate-800 hover:text-slate-200'
              }`}
            >
              <span className="text-base">{item.icon}</span>
              {item.label}
            </button>
          ))}
        </nav>

        {/* User pill */}
        <div className="px-3 py-4 border-t border-slate-800">
          {user ? (
            <div className="px-3 py-2.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
              <p className="text-xs text-emerald-400 font-medium truncate">✓ {user.email}</p>
              {user.displayName && <p className="text-xs text-slate-500 truncate mt-0.5">{user.displayName}</p>}
            </div>
          ) : (
            <div className="px-3 py-2.5 rounded-lg bg-slate-800/50">
              <p className="text-xs text-slate-500">Не авторизован</p>
            </div>
          )}
        </div>
      </aside>

      {/* Main */}
      <main className="flex-1 overflow-y-auto">
        {section === 'auth' && <AuthSection setUser={setUser} showToast={showToast} />}
        {section === 'profile' && <ProfileSection user={user} setUser={setUser} showToast={showToast} />}
        {section === 'categories' && <CategoriesSection showToast={showToast} />}
        {section === 'products' && <ProductsSection showToast={showToast} />}
        {section === 'files' && <FilesSection showToast={showToast} />}
        {section === 'health' && <HealthSection showToast={showToast} />}
      </main>

      {/* Toast */}
      {toast && (
        <div className={`fixed bottom-5 right-5 z-50 px-4 py-3 rounded-xl shadow-2xl border text-sm font-medium flex items-center gap-2 transition-all animate-in ${
          toast.type === 'success'
            ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-300'
            : 'bg-rose-500/10 border-rose-500/30 text-rose-300'
        }`}>
          <span>{toast.type === 'success' ? '✓' : '✕'}</span>
          {toast.text}
        </div>
      )}
    </div>
  )
}
