import { type ReactNode } from 'react'

export function SectionHeader({ icon, title, sub }: { icon: string; title: string; sub: string }) {
  return (
    <div className="flex items-center gap-4 pb-2 border-b border-slate-800">
      <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-violet-500/20 to-indigo-500/20 border border-violet-500/20 flex items-center justify-center text-xl">
        {icon}
      </div>
      <div>
        <h1 className="text-lg font-semibold text-slate-100">{title}</h1>
        <p className="text-xs text-slate-500">{sub}</p>
      </div>
    </div>
  )
}

export function Card({ title, badge, children }: { title: string; badge?: string; children: ReactNode }) {
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-200">{title}</h2>
        {badge && <span className="text-[10px] font-mono bg-slate-800 text-slate-400 px-2 py-0.5 rounded-md border border-slate-700">{badge}</span>}
      </div>
      {children}
    </div>
  )
}

export function Field({
  label, value, onChange, type = 'text', placeholder,
}: {
  label: string; value: string; onChange: (v: string) => void
  type?: string; placeholder?: string
}) {
  return (
    <div className="space-y-1.5">
      <label className="block text-xs font-medium text-slate-400">{label}</label>
      <input
        type={type} value={value} onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-violet-500 focus:border-violet-500 transition-all"
      />
    </div>
  )
}

export function Select({
  label, value, onChange, options,
}: {
  label: string; value: string; onChange: (v: string) => void
  options: { value: string; label: string }[]
}) {
  return (
    <div className="space-y-1.5">
      <label className="block text-xs font-medium text-slate-400">{label}</label>
      <select
        value={value} onChange={e => onChange(e.target.value)}
        className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-violet-500 focus:border-violet-500 transition-all"
      >
        {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
      </select>
    </div>
  )
}

export function Btn({
  children, onClick, loading, variant = 'primary', small,
}: {
  children: ReactNode; onClick?: () => void; loading?: boolean
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost'; small?: boolean
}) {
  const base = `inline-flex items-center gap-1.5 font-medium rounded-lg transition-all duration-150 disabled:opacity-50 ${small ? 'text-xs px-2.5 py-1.5' : 'text-sm px-3.5 py-2 w-full justify-center'}`
  const variants = {
    primary: 'bg-violet-600 hover:bg-violet-500 text-white',
    secondary: 'bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700',
    danger: 'bg-rose-600/20 hover:bg-rose-600/30 text-rose-300 border border-rose-500/30',
    ghost: 'hover:bg-slate-800 text-slate-400 hover:text-slate-200',
  }
  return (
    <button className={`${base} ${variants[variant]}`} onClick={onClick} disabled={loading}>
      {loading && <span className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" />}
      {children}
    </button>
  )
}

export function JsonView({ data }: { data: unknown }) {
  if (!data) return null
  const isError = data && typeof data === 'object' && 'error' in (data as object)
  return (
    <div className={`mt-1 rounded-lg border text-xs font-mono overflow-auto max-h-48 ${isError ? 'bg-rose-950/30 border-rose-500/20 text-rose-300' : 'bg-slate-950/80 border-slate-700/50 text-emerald-300'}`}>
      <pre className="p-3 whitespace-pre-wrap break-all">{JSON.stringify(data, null, 2)}</pre>
    </div>
  )
}

export function Pagination({
  page, totalPages, onPage,
}: {
  page: number; totalPages: number; onPage: (p: number) => void
}) {
  if (totalPages <= 1) return null
  return (
    <div className="flex items-center gap-2">
      <Btn small variant="secondary" onClick={() => onPage(page - 1)} loading={page <= 1}>‹ Назад</Btn>
      <span className="text-xs text-slate-500">Стр. {page} / {totalPages}</span>
      <Btn small variant="secondary" onClick={() => onPage(page + 1)} loading={page >= totalPages}>Вперёд ›</Btn>
    </div>
  )
}
