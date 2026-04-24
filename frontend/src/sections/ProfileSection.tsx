import { useState } from 'react'
import { api } from '../api'
import type { UserResponse, FileResponse } from '../types'
import { Card, Field, Btn, JsonView, SectionHeader } from './shared'

interface Props {
  user: UserResponse | null
  setUser: (u: UserResponse | null) => void
  showToast: (t: string, type?: 'success' | 'error') => void
}

function InfoRow({ label, value }: { label: string; value?: string }) {
  if (!value) return null
  return (
    <div className="flex gap-2 text-sm">
      <span className="text-slate-500 w-28 flex-shrink-0">{label}</span>
      <span className="text-slate-200 break-all">{value}</span>
    </div>
  )
}

export default function ProfileSection({ user, setUser, showToast }: Props) {
  const [displayName, setDisplayName] = useState(user?.displayName ?? '')
  const [bio, setBio] = useState(user?.bio ?? '')
  const [avatarFile, setAvatarFile] = useState<FileResponse | null>(null)
  const [uploadedFile, setUploadedFile] = useState<File | null>(null)
  const [res, setRes] = useState<Record<string, unknown>>({})
  const [loading, setLoading] = useState('')

  async function run(key: string, fn: () => Promise<unknown>) {
    setLoading(key)
    try {
      const data = await fn()
      setRes(r => ({ ...r, [key]: data }))
      showToast('Успешно')
      return data
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setRes(r => ({ ...r, [key]: { error: msg } }))
      showToast(msg, 'error')
    } finally { setLoading('') }
  }

  async function uploadAvatar() {
    if (!uploadedFile) return
    const data = await run('uploadAvatar', () => api.files.upload(uploadedFile)) as { file: FileResponse } | undefined
    if (data?.file) setAvatarFile(data.file)
  }

  async function updateProfile() {
    const data = await run('profile', () =>
      api.profile.update({
        ...(displayName ? { displayName } : {}),
        ...(bio ? { bio } : {}),
        ...(avatarFile ? { avatarFileId: avatarFile.id } : {}),
      })
    )
    if (data) setUser(data as UserResponse)
  }

  return (
    <div className="p-6 space-y-6">
      <SectionHeader icon="👤" title="Профиль" sub="Просмотр и редактирование профиля пользователя" />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Current profile */}
        <Card title="Текущий профиль" badge="GET /profile">
          {user ? (
            <div className="space-y-2.5">
              <InfoRow label="ID" value={user.id} />
              <InfoRow label="Email" value={user.email} />
              <InfoRow label="Телефон" value={user.phone} />
              <InfoRow label="Имя" value={user.displayName} />
              <InfoRow label="Bio" value={user.bio} />
              <InfoRow label="Avatar ID" value={user.avatarFileId} />
              <InfoRow label="Создан" value={new Date(user.createdAt).toLocaleString('ru')} />
            </div>
          ) : (
            <p className="text-sm text-slate-500">Войдите для просмотра профиля</p>
          )}
          <div className="mt-3">
            <Btn variant="secondary" loading={loading === 'getProfile'} onClick={async () => {
              const d = await run('getProfile', () => api.profile.get())
              if (d) setUser(d as UserResponse)
            }}>Обновить</Btn>
          </div>
          <JsonView data={res['getProfile']} />
        </Card>

        {/* Edit profile */}
        <Card title="Редактировать профиль" badge="POST /profile">
          <Field label="Имя (displayName)" value={displayName} onChange={setDisplayName} placeholder="Иван Иванов" />
          <Field label="Bio" value={bio} onChange={setBio} placeholder="Backend разработчик..." />

          <div className="space-y-2">
            <label className="block text-xs font-medium text-slate-400">Аватар (загрузить изображение)</label>
            <input
              type="file" accept="image/png,image/jpeg"
              onChange={e => setUploadedFile(e.target.files?.[0] ?? null)}
              className="block w-full text-xs text-slate-400 file:mr-3 file:py-1.5 file:px-3 file:rounded-lg file:border-0 file:text-xs file:font-medium file:bg-violet-600/20 file:text-violet-300 hover:file:bg-violet-600/30 cursor-pointer"
            />
            {avatarFile && (
              <p className="text-xs text-emerald-400">✓ Загружен: {avatarFile.originalName} ({avatarFile.id})</p>
            )}
            <Btn variant="secondary" loading={loading === 'uploadAvatar'} onClick={uploadAvatar}>
              Загрузить файл в /files
            </Btn>
          </div>

          <Btn loading={loading === 'profile'} onClick={updateProfile}>Сохранить профиль</Btn>
          <JsonView data={res['profile'] || res['uploadAvatar']} />
        </Card>
      </div>
    </div>
  )
}
