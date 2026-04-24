import { useState, useEffect, useCallback } from 'react'
import { api } from '../api'
import type { ProductResponse, CategoryResponse } from '../types'
import { Card, Field, Select, Btn, JsonView, SectionHeader, Pagination } from './shared'

const STATUS_OPTS = [
  { value: 'available', label: 'Available' },
  { value: 'out_of_stock', label: 'Out of stock' },
  { value: 'discontinued', label: 'Discontinued' },
]

const STATUS_COLORS: Record<string, string> = {
  available: 'bg-emerald-500/10 text-emerald-400',
  out_of_stock: 'bg-amber-500/10 text-amber-400',
  discontinued: 'bg-rose-500/10 text-rose-400',
}

export default function ProductsSection({ showToast }: { showToast: (t: string, type?: 'success' | 'error') => void }) {
  const [list, setList] = useState<ProductResponse[]>([])
  const [categories, setCategories] = useState<CategoryResponse[]>([])
  const [meta, setMeta] = useState({ page: 1, totalPages: 1, total: 0 })
  const [page, setPage] = useState(1)
  const [filterCat, setFilterCat] = useState('')

  const [name, setName] = useState(''); const [desc, setDesc] = useState('')
  const [price, setPrice] = useState('0'); const [catId, setCatId] = useState(''); const [status, setStatus] = useState('available')
  const [editId, setEditId] = useState(''); const [editName, setEditName] = useState('')
  const [editDesc, setEditDesc] = useState(''); const [editPrice, setEditPrice] = useState('0')
  const [editCatId, setEditCatId] = useState(''); const [editStatus, setEditStatus] = useState('available')
  const [getId, setGetId] = useState('')
  const [res, setRes] = useState<Record<string, unknown>>({})
  const [loading, setLoading] = useState('')

  useEffect(() => {
    api.categories.list(1, 100).then(d => setCategories(d.data)).catch(() => {})
  }, [])

  const catOptions = [{ value: '', label: '— Выбрать категорию —' }, ...categories.map(c => ({ value: c.id, label: c.name }))]

  const load = useCallback(async (p = page, cat = filterCat) => {
    try {
      const data = await api.products.list(p, 10, cat || undefined)
      setList(data.data); setMeta({ page: data.meta.page, totalPages: data.meta.totalPages, total: data.meta.total })
    } catch (e: unknown) { showToast(e instanceof Error ? e.message : 'Ошибка', 'error') }
  }, [page, filterCat, showToast])

  useEffect(() => { load() }, [load])

  async function run(key: string, fn: () => Promise<unknown>, reload = false) {
    setLoading(key)
    try {
      const data = await fn()
      setRes(r => ({ ...r, [key]: data })); showToast('Успешно')
      if (reload) await load()
      return data
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setRes(r => ({ ...r, [key]: { error: msg } })); showToast(msg, 'error')
    } finally { setLoading('') }
  }

  function startEdit(p: ProductResponse) {
    setEditId(p.id); setEditName(p.name); setEditDesc(p.description ?? '')
    setEditPrice(String(p.price)); setEditCatId(p.categoryId); setEditStatus(p.status)
  }

  return (
    <div className="p-6 space-y-6">
      <SectionHeader icon="📦" title="Продукты" sub="CRUD-операции над продуктами с категориями" />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Create */}
        <Card title="Создать продукт" badge="POST /products">
          <Field label="Название *" value={name} onChange={setName} placeholder="iPhone 15" />
          <Field label="Описание" value={desc} onChange={setDesc} placeholder="Описание" />
          <Field label="Цена *" type="number" value={price} onChange={setPrice} placeholder="0" />
          <Select label="Категория *" value={catId} onChange={setCatId} options={catOptions} />
          <Select label="Статус" value={status} onChange={setStatus} options={STATUS_OPTS} />
          <Btn loading={loading === 'create'} onClick={() => run('create', () =>
            api.products.create({ name, description: desc || undefined, price: parseFloat(price), categoryId: catId, status }), true)}>
            Создать
          </Btn>
          <JsonView data={res['create']} />
        </Card>

        {/* Get / Delete */}
        <Card title="Получить / Удалить по ID" badge="GET · DELETE /products/:id">
          <Field label="ID продукта" value={getId} onChange={setGetId} placeholder="uuid" />
          <div className="flex gap-2">
            <Btn small variant="secondary" loading={loading === 'get'} onClick={() => run('get', () => api.products.get(getId))}>Получить</Btn>
            <Btn small variant="danger" loading={loading === 'del'} onClick={() => run('del', () => api.products.delete(getId), true)}>Удалить</Btn>
          </div>
          <JsonView data={res['get'] || res['del']} />
        </Card>
      </div>

      {/* Edit */}
      <Card title="Редактировать продукт" badge="PUT / PATCH /products/:id">
        <div className="grid grid-cols-2 gap-3">
          <Field label="ID *" value={editId} onChange={setEditId} placeholder="uuid" />
          <Field label="Название" value={editName} onChange={setEditName} placeholder="Название" />
          <Field label="Цена" type="number" value={editPrice} onChange={setEditPrice} placeholder="0" />
          <Select label="Категория" value={editCatId} onChange={setEditCatId} options={catOptions} />
          <Field label="Описание" value={editDesc} onChange={setEditDesc} placeholder="Описание" />
          <Select label="Статус" value={editStatus} onChange={setEditStatus} options={STATUS_OPTS} />
        </div>
        <div className="flex gap-2">
          <Btn small loading={loading === 'put'} onClick={() => run('put', () =>
            api.products.update(editId, { name: editName, description: editDesc, price: parseFloat(editPrice), categoryId: editCatId, status: editStatus }), true)}>
            PUT
          </Btn>
          <Btn small variant="secondary" loading={loading === 'patch'} onClick={() => run('patch', () =>
            api.products.patch(editId, { ...(editName ? { name: editName } : {}), ...(editDesc ? { description: editDesc } : {}), ...(editPrice ? { price: parseFloat(editPrice) } : {}), ...(editCatId ? { categoryId: editCatId } : {}), ...(editStatus ? { status: editStatus } : {}) }), true)}>
            PATCH
          </Btn>
        </div>
        <JsonView data={res['put'] || res['patch']} />
      </Card>

      {/* List */}
      <Card title={`Список продуктов (всего: ${meta.total})`} badge="GET /products">
        <div className="flex gap-2 items-end">
          <div className="flex-1">
            <Select label="Фильтр по категории" value={filterCat} onChange={v => { setFilterCat(v); setPage(1); load(1, v) }} options={catOptions} />
          </div>
          <Btn small variant="ghost" onClick={() => load()}>↻</Btn>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-slate-800 text-slate-500">
                <th className="text-left py-2 pr-3 font-medium">Название</th>
                <th className="text-left py-2 pr-3 font-medium">Цена</th>
                <th className="text-left py-2 pr-3 font-medium">Категория</th>
                <th className="text-left py-2 pr-3 font-medium">Статус</th>
                <th className="text-left py-2 font-medium">Действия</th>
              </tr>
            </thead>
            <tbody>
              {list.map(p => (
                <tr key={p.id} className="border-b border-slate-800/50 hover:bg-slate-800/30 transition-colors">
                  <td className="py-2 pr-3 text-slate-200 font-medium">{p.name}</td>
                  <td className="py-2 pr-3 text-violet-300 font-mono">{p.price.toFixed(2)} ₽</td>
                  <td className="py-2 pr-3 text-slate-400">{p.categoryName ?? '—'}</td>
                  <td className="py-2 pr-3">
                    <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${STATUS_COLORS[p.status] ?? ''}`}>{p.status}</span>
                  </td>
                  <td className="py-2">
                    <div className="flex gap-1">
                      <button onClick={() => startEdit(p)} className="text-violet-400 hover:text-violet-300 text-[10px] px-1.5 py-0.5 rounded hover:bg-violet-500/10 transition-colors">Изменить</button>
                      <button onClick={() => run('del', () => api.products.delete(p.id), true)} className="text-rose-400 hover:text-rose-300 text-[10px] px-1.5 py-0.5 rounded hover:bg-rose-500/10 transition-colors">Удалить</button>
                    </div>
                  </td>
                </tr>
              ))}
              {list.length === 0 && <tr><td colSpan={5} className="py-6 text-center text-slate-600">Пусто</td></tr>}
            </tbody>
          </table>
        </div>
        <Pagination page={meta.page} totalPages={meta.totalPages} onPage={p => { setPage(p); load(p) }} />
      </Card>
    </div>
  )
}
