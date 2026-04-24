import { useState } from 'react'
import { api } from '../api'
import type { UserResponse } from '../types'
import { Card, Field, Btn, JsonView, SectionHeader } from './shared'

interface Props {
  setUser: (u: UserResponse | null) => void
  showToast: (t: string, type?: 'success' | 'error') => void
}

export default function AuthSection({ setUser, showToast }: Props) {
  const [loginEmail, setLoginEmail] = useState('')
  const [loginPass, setLoginPass] = useState('')
  const [regEmail, setRegEmail] = useState('')
  const [regPass, setRegPass] = useState('')
  const [regPhone, setRegPhone] = useState('')
  const [whoamiData, setWhoamiData] = useState<unknown>(null)
  const [res, setRes] = useState<Record<string, unknown>>({})
  const [loading, setLoading] = useState<string>('')

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
    } finally {
      setLoading('')
    }
  }

  return (
    <div className="p-6 space-y-6">
      <SectionHeader icon="🔐" title="Авторизация" sub="Вход, регистрация, управление сессией" />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Login */}
        <Card title="Вход" badge="POST /auth/login">
          <Field label="Email" type="email" value={loginEmail} onChange={setLoginEmail} placeholder="user@example.com" />
          <Field label="Пароль" type="password" value={loginPass} onChange={setLoginPass} placeholder="••••••••" />
          <Btn loading={loading === 'login'} onClick={async () => {
            const data = await run('login', () => api.auth.login(loginEmail, loginPass))
            if (data) api.auth.whoami().then(setUser).catch(() => {})
          }}>Войти</Btn>
          <JsonView data={res['login']} />
        </Card>

        {/* Register */}
        <Card title="Регистрация" badge="POST /auth/register">
          <Field label="Email" type="email" value={regEmail} onChange={setRegEmail} placeholder="new@example.com" />
          <Field label="Пароль (мин. 8 символов)" type="password" value={regPass} onChange={setRegPass} placeholder="••••••••" />
          <Field label="Телефон (опционально)" value={regPhone} onChange={setRegPhone} placeholder="+79991234567" />
          <Btn loading={loading === 'register'} onClick={() => run('register', () => api.auth.register(regEmail, regPass, regPhone || undefined))}>
            Зарегистрироваться
          </Btn>
          <JsonView data={res['register']} />
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Whoami */}
        <Card title="Текущий пользователь" badge="GET /auth/whoami">
          <Btn variant="secondary" loading={loading === 'whoami'} onClick={async () => {
            const data = await run('whoami', () => api.auth.whoami())
            if (data) { setUser(data as UserResponse); setWhoamiData(data) }
          }}>Whoami</Btn>
          <JsonView data={whoamiData || res['whoami']} />
        </Card>

        {/* Logout */}
        <Card title="Выход" badge="POST /auth/logout">
          <Btn variant="danger" loading={loading === 'logout'} onClick={async () => {
            await run('logout', () => api.auth.logout())
            setUser(null)
          }}>Выйти из сессии</Btn>
          <Btn variant="danger" loading={loading === 'logoutAll'} onClick={async () => {
            await run('logoutAll', () => api.auth.logoutAll())
            setUser(null)
          }}>Выйти из всех сессий</Btn>
          <JsonView data={res['logout'] || res['logoutAll']} />
        </Card>

        {/* Refresh */}
        <Card title="Обновить токен" badge="POST /auth/refresh">
          <p className="text-xs text-slate-500 mb-3">Обновляет access-токен через refresh cookie.</p>
          <Btn variant="secondary" loading={loading === 'refresh'} onClick={() => run('refresh', () => api.auth.refresh())}>
            Refresh Token
          </Btn>
          <JsonView data={res['refresh']} />
        </Card>
      </div>
    </div>
  )
}
