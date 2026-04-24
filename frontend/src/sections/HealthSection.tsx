import { useState } from 'react'
import { api } from '../api'
import { Card, Btn, JsonView, SectionHeader } from './shared'

export default function HealthSection({ showToast }: { showToast: (t: string, type?: 'success' | 'error') => void }) {
  const [redisData, setRedisData] = useState<unknown>(null)
  const [diagData, setDiagData] = useState<unknown>(null)
  const [loading, setLoading] = useState('')
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [timer, setTimer] = useState<ReturnType<typeof setInterval> | null>(null)

  async function fetchRedis() {
    setLoading('redis')
    try {
      const d = await api.health.redis(); setRedisData(d); showToast('Redis OK')
    } catch (e: unknown) {
      const d = { error: e instanceof Error ? e.message : String(e) }; setRedisData(d); showToast(d.error, 'error')
    } finally { setLoading('') }
  }

  async function fetchDiag() {
    setLoading('diag')
    try {
      const d = await api.health.diagnosis(); setDiagData(d); showToast('Диагностика готова')
    } catch (e: unknown) {
      const d = { error: e instanceof Error ? e.message : String(e) }; setDiagData(d); showToast(d.error, 'error')
    } finally { setLoading('') }
  }

  function toggleAutoRefresh() {
    if (autoRefresh) {
      if (timer) clearInterval(timer)
      setTimer(null); setAutoRefresh(false)
    } else {
      fetchRedis()
      const t = setInterval(fetchRedis, 5000)
      setTimer(t); setAutoRefresh(true)
    }
  }

  const redis = redisData as Record<string, unknown> | null

  return (
    <div className="p-6 space-y-6">
      <SectionHeader icon="💚" title="Health" sub="Мониторинг Redis и диагностика производительности" />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Redis */}
        <Card title="Redis Status" badge="GET /health/redis">
          <div className="flex gap-2">
            <Btn small loading={loading === 'redis'} onClick={fetchRedis}>Проверить</Btn>
            <Btn small variant={autoRefresh ? 'danger' : 'secondary'} onClick={toggleAutoRefresh}>
              {autoRefresh ? '⏹ Стоп (5s)' : '▶ Авто (5s)'}
            </Btn>
          </div>

          {redis && (
            <div className="space-y-2.5">
              <div className="flex items-center gap-2">
                <span className={`w-2.5 h-2.5 rounded-full ${redis.connected ? 'bg-emerald-400 shadow-lg shadow-emerald-500/30' : 'bg-rose-400'}`} />
                <span className={`text-sm font-semibold ${redis.connected ? 'text-emerald-300' : 'text-rose-300'}`}>
                  {redis.connected ? 'Подключён' : 'Недоступен'}
                </span>
                {redis.pingLatencyMs !== undefined && (
                  <span className="text-xs text-slate-500 font-mono">{String(redis.pingLatencyMs)} ms</span>
                )}
              </div>

              {typeof redis.server === 'object' && redis.server !== null ? (() => {
                const s = redis.server as Record<string, unknown>
                const rows: [string, string][] = [
                  ['Версия', String(s.redisVersion ?? '—')],
                  ['Память', String(s.usedMemoryHuman ?? '—')],
                  ['Клиенты', String(s.connectedClients ?? '—')],
                  ['Uptime', `${Math.floor(((s.uptime_seconds as number) ?? 0) / 60)} мин`],
                ]
                return (
                  <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                    {rows.map(([k, v]) => (
                      <div key={k} className="flex justify-between">
                        <span className="text-slate-500">{k}</span>
                        <span className="text-slate-300 font-mono">{v}</span>
                      </div>
                    ))}
                  </div>
                )
              })() : null}

              {typeof redis.usage === 'object' && redis.usage !== null ? (() => {
                const u = redis.usage as Record<string, unknown>
                const hit = (u.hitRatioEstimate as number) ?? 0
                const ops: [string, string][] = [
                  ['GET', String(u.getRequests ?? 0)],
                  ['SET', String(u.setWrites ?? 0)],
                  ['DEL', String(u.delSingle ?? 0)],
                  ['Всего', String(u.totalCacheOperations ?? 0)],
                ]
                return (
                  <div className="space-y-1.5">
                    <div className="flex justify-between text-xs">
                      <span className="text-slate-500">Hit rate</span>
                      <span className={`font-mono font-semibold ${hit > 0.7 ? 'text-emerald-400' : hit > 0.4 ? 'text-amber-400' : 'text-rose-400'}`}>
                        {(hit * 100).toFixed(1)}%
                      </span>
                    </div>
                    <div className="w-full bg-slate-800 rounded-full h-1.5">
                      <div className={`h-1.5 rounded-full transition-all ${hit > 0.7 ? 'bg-emerald-500' : hit > 0.4 ? 'bg-amber-500' : 'bg-rose-500'}`}
                        style={{ width: `${Math.min(hit * 100, 100)}%` }} />
                    </div>
                    <div className="grid grid-cols-2 gap-x-4 text-xs">
                      {ops.map(([k, v]) => (
                        <div key={k} className="flex justify-between">
                          <span className="text-slate-500">{k}</span>
                          <span className="text-slate-300 font-mono">{v}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )
              })() : null}
            </div>
          )}
          <JsonView data={redisData} />
        </Card>

        {/* Diagnosis */}
        <Card title="Диагностика MongoDB vs Redis" badge="GET /health/diagnosis">
          <p className="text-xs text-slate-500">Сравнивает латентность прямого запроса к MongoDB и запроса через Redis-кеш.</p>
          <Btn loading={loading === 'diag'} onClick={fetchDiag}>Запустить диагностику</Btn>
          <JsonView data={diagData} />
        </Card>
      </div>
    </div>
  )
}
