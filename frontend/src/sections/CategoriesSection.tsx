import { useState, useEffect, useCallback } from 'react'
import { api } from '../api'
import type { CategoryResponse } from '../types'
import { Card, Field, Select, Btn, JsonView, SectionHeader, Pagination } from './shared'

const STATUS_OPTS = [
  { value: 'active', label: 'Active' },
  { value: 'hidden', label: 'Hidden' },
]

export default function CategoriesSection({ showToast }: { showToast: (t: string, type?: 'success' | 'error') => void }) {
  const [list, setList] = useState<CategoryResponse[]>([])
  const [meta, setMeta] = useState({ page: 1, totalPages: 1, total: 0 })
  const [page, setPage] = useState(1)

  const [name, setName] = useState(''); const [desc, setDesc] = useState(''); const [status, setStatus] = useState('active')
  const [editId, setEditId] = useState(''); const [editName, setEditName] = useState(''); const [editDesc, setEditDesc] = useState(''); const [editStatus, setEditStatus] = useState('active')
  const [getId, setGetId] = useState('')
  const [res, setRes] = useState<Record<string, unknown>>({})
  const [loading, setLoading] = useState('')

  const load = useCallback(async (p = page) => {
    try {
      const data = await api.categories.list(p)
      setList(data.data); setMeta({ page: data.meta.page, totalPages: data.meta.totalPages, total: data.meta.total })
    } catch (e: unknown) { showToast(e instanceof Error ? e.message : 'Ошибка', 'error') }
  }, [page, showToast])

  useEffect(() => { load() }, [load])

  async function run(key: string, fn: () => Promise<unknown>, reload = false) {
    setLoading(key)
    try {
      const data = await fn()
      setRes(r => ({ ...r, [key]: data }))
      showToast('Успешно')
      if (reload) await load()
      return data
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setRes(r => ({ ...r, [key]: { error: msg } })); showToast(msg, 'error')
    } finally { setLoading('') }
  }

  function startEdit(c: CategoryResponse) {
    setEditId(c.id); setEditName(c.name); setEditDesc(c.description ?? ''); setEditStatus(c.status)
  }

  return (
    <div className="p-6 space-y-6">
      <SectionHeader icon="🗂️" title="Категории" sub="CRUD-операции над категориями" />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Create */}
        <Card title="Создать категорию" badge="POST /categories">
          <Field label="Название *" value={name} onChange={setName} placeholder="Электроника" />
          <Field label="Описание" value={desc} onChange={setDesc} placeholder="Описание категории" />
          <Select label="Статус" value={status} onChange={setStatus} options={STATUS_OPTS} />
          <Btn loading={loading === 'create'} onClick={() => run('create', () => api.categories.create({ name, description: desc || undefined, status }), true)}>
            Создать
          </Btn>
          <JsonView data={res['create']} />
        </Card>

        {/* Get by ID */}
        <Card title="Получить / Удалить по ID" badge="GET · DELETE /categories/:id">
          <Field label="ID категории" value={getId} onChange={setGetId} placeholder="uuid" />
          <div className="flex gap-2">
            <Btn small variant="secondary" loading={loading === 'get'} onClick={() => run('get', () => api.categories.get(getId))}>Получить</Btn>
            <Btn small variant="danger" loading={loading === 'delete'} onClick={() => run('delete', () => api.categories.delete(getId), true)}>Удалить</Btn>
          </div>
          <JsonView data={res['get'] || res['delete']} />
        </Card>
      </div>

      {/* Edit */}
      <Card title="Редактировать категорию" badge="PUT / PATCH /categories/:id">
        <div className="grid grid-cols-2 gap-3">
          <Field label="ID категории *" value={editId} onChange={setEditId} placeholder="uuid" />
          <Field label="Название" value={editName} onChange={setEditName} placeholder="Новое название" />
          <Field label="Описание" value={editDesc} onChange={setEditDesc} placeholder="Описание" />
          <Select label="Статус" value={editStatus} onChange={setEditStatus} options={STATUS_OPTS} />
        </div>
        <div className="flex gap-2">
          <Btn small loading={loading === 'put'} onClick={() => run('put', () => api.categories.update(editId, { name: editName, description: editDesc, status: editStatus }), true)}>PUT (полная замена)</Btn>
          <Btn small variant="secondary" loading={loading === 'patch'} onClick={() => run('patch', () => api.categories.patch(editId, { ...(editName ? { name: editName } : {}), ...(editDesc ? { description: editDesc } : {}), ...(editStatus ? { status: editStatus } : {}) }), true)}>PATCH (частично)</Btn>
        </div>
        <JsonView data={res['put'] || res['patch']} />
      </Card>

      {/* List */}
      <Card title={`Список категорий (всего: ${meta.total})`} badge="GET /categories">
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-slate-800 text-slate-500">
                <th className="text-left py-2 pr-3 font-medium">Название</th>
                <th className="text-left py-2 pr-3 font-medium">Описание</th>
                <th className="text-left py-2 pr-3 font-medium">Статус</th>
                <th className="text-left py-2 font-medium">Действия</th>
              </tr>
            </thead>
            <tbody>
              {list.map(c => (
                <tr key={c.id} className="border-b border-slate-800/50 hover:bg-slate-800/30 transition-colors">
                  <td className="py-2 pr-3 text-slate-200 font-medium">{c.name}</td>
                  <td className="py-2 pr-3 text-slate-400 max-w-xs truncate">{c.description ?? '—'}</td>
                  <td className="py-2 pr-3">
                    <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${c.status === 'active' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-slate-700 text-slate-400'}`}>{c.status}</span>
                  </td>
                  <td className="py-2">
                    <div className="flex gap-1">
                      <button onClick={() => { setGetId(c.id); startEdit(c) }} className="text-violet-400 hover:text-violet-300 text-[10px] px-1.5 py-0.5 rounded hover:bg-violet-500/10 transition-colors">Изменить</button>
                      <button onClick={() => run('delete', () => api.categories.delete(c.id), true)} className="text-rose-400 hover:text-rose-300 text-[10px] px-1.5 py-0.5 rounded hover:bg-rose-500/10 transition-colors">Удалить</button>
                    </div>
                  </td>
                </tr>
              ))}
              {list.length === 0 && <tr><td colSpan={4} className="py-6 text-center text-slate-600">Пусто</td></tr>}
            </tbody>
          </table>
        </div>
        <div className="flex items-center justify-between pt-1">
          <Pagination page={meta.page} totalPages={meta.totalPages} onPage={p => { setPage(p); load(p) }} />
          <Btn small variant="ghost" onClick={() => load()}>↻ Обновить</Btn>
        </div>
      </Card>
    </div>
  )
}
