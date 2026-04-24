import { useState, useRef } from 'react'
import { api } from '../api'
import type { FileResponse } from '../types'
import { Card, Btn, JsonView, SectionHeader } from './shared'

function formatBytes(n: number) {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}

export default function FilesSection({ showToast }: { showToast: (t: string, type?: 'success' | 'error') => void }) {
  const [files, setFiles] = useState<FileResponse[]>([])
  const [dragOver, setDragOver] = useState(false)
  const [deleteId, setDeleteId] = useState('')
  const [res, setRes] = useState<Record<string, unknown>>({})
  const [loading, setLoading] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  async function uploadFile(file: File) {
    setLoading('upload')
    try {
      const data = await api.files.upload(file)
      setRes(r => ({ ...r, upload: data }))
      setFiles(f => [data.file, ...f])
      showToast(`Загружен: ${data.file.originalName}`)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setRes(r => ({ ...r, upload: { error: msg } })); showToast(msg, 'error')
    } finally { setLoading('') }
  }

  async function deleteFile(id: string, name: string) {
    setLoading(`del-${id}`)
    try {
      await api.files.delete(id)
      setFiles(f => f.filter(x => x.id !== id))
      showToast(`Удалён: ${name}`)
    } catch (e: unknown) { showToast(e instanceof Error ? e.message : 'Ошибка', 'error') }
    finally { setLoading('') }
  }

  async function downloadFile(id: string, name: string) {
    try {
      await api.files.download(id, name)
      showToast(`Скачан: ${name}`)
    } catch { showToast('Ошибка скачивания', 'error') }
  }

  async function deleteById() {
    if (!deleteId) return
    setLoading('delById')
    try {
      await api.files.delete(deleteId)
      setFiles(f => f.filter(x => x.id !== deleteId))
      setRes(r => ({ ...r, delById: { deleted: deleteId } }))
      showToast('Удалён')
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setRes(r => ({ ...r, delById: { error: msg } })); showToast(msg, 'error')
    } finally { setLoading('') }
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault(); setDragOver(false)
    const file = e.dataTransfer.files[0]
    if (file) uploadFile(file)
  }

  const mimeIcon = (mime: string) => mime.includes('png') ? '🖼️' : mime.includes('jpeg') || mime.includes('jpg') ? '📸' : '📄'

  return (
    <div className="p-6 space-y-6">
      <SectionHeader icon="📁" title="Файлы" sub="Загрузка, скачивание и удаление файлов через MinIO" />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Upload zone */}
        <Card title="Загрузить файл" badge="POST /files">
          <div
            onDragOver={e => { e.preventDefault(); setDragOver(true) }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleDrop}
            onClick={() => inputRef.current?.click()}
            className={`border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-all duration-200 ${
              dragOver ? 'border-violet-500 bg-violet-500/10' : 'border-slate-700 hover:border-slate-600 hover:bg-slate-800/30'
            }`}
          >
            <div className="text-3xl mb-2">📤</div>
            <p className="text-sm text-slate-300 font-medium">Перетащите файл или нажмите</p>
            <p className="text-xs text-slate-500 mt-1">PNG, JPEG · макс. 10 МБ</p>
            {loading === 'upload' && (
              <div className="mt-3 flex items-center justify-center gap-2 text-violet-400 text-xs">
                <span className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" />
                Загрузка...
              </div>
            )}
          </div>
          <input ref={inputRef} type="file" accept="image/png,image/jpeg" className="hidden"
            onChange={e => { const f = e.target.files?.[0]; if (f) uploadFile(f); e.target.value = '' }} />
          <JsonView data={res['upload']} />
        </Card>

        {/* Delete by ID */}
        <Card title="Удалить по ID" badge="DELETE /files/:id">
          <p className="text-xs text-slate-500">Введите ID файла вручную или нажмите «Удалить» в списке ниже.</p>
          <div className="flex gap-2">
            <input value={deleteId} onChange={e => setDeleteId(e.target.value)} placeholder="uuid файла"
              className="flex-1 bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-violet-500" />
            <Btn small variant="danger" loading={loading === 'delById'} onClick={deleteById}>Удалить</Btn>
          </div>
          <JsonView data={res['delById']} />
        </Card>
      </div>

      {/* Uploaded files grid */}
      {files.length > 0 && (
        <Card title={`Загруженные файлы (${files.length})`}>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {files.map(f => (
              <div key={f.id} className="bg-slate-800/50 border border-slate-700/50 rounded-xl p-3 space-y-2">
                <div className="flex items-start gap-2">
                  <span className="text-2xl">{mimeIcon(f.mimetype)}</span>
                  <div className="flex-1 min-w-0">
                    <p className="text-xs font-medium text-slate-200 truncate" title={f.originalName}>{f.originalName}</p>
                    <p className="text-[10px] text-slate-500">{formatBytes(f.size)} · {f.mimetype}</p>
                  </div>
                </div>
                <p className="text-[10px] text-slate-600 font-mono truncate" title={f.id}>{f.id}</p>
                <div className="flex gap-1.5">
                  <button onClick={() => downloadFile(f.id, f.originalName)}
                    className="flex-1 text-[10px] py-1 rounded-lg bg-violet-600/20 text-violet-300 hover:bg-violet-600/30 border border-violet-500/20 transition-colors font-medium">
                    ⬇ Скачать
                  </button>
                  <button onClick={() => deleteFile(f.id, f.originalName)}
                    disabled={loading === `del-${f.id}`}
                    className="flex-1 text-[10px] py-1 rounded-lg bg-rose-600/10 text-rose-400 hover:bg-rose-600/20 border border-rose-500/20 transition-colors font-medium disabled:opacity-50">
                    🗑 Удалить
                  </button>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  )
}
