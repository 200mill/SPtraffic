import { useEffect, useState } from 'react'
import { api } from '../../api/client'
import type { RealtimeArrival } from '../../types'

interface Props {
  code: string
}

const REFRESH_MS = 30_000

/** Shows live next-train arrivals for a metro station, refreshed every 30 s. */
export function MetroRealtimePanel({ code }: Props) {
  const [arrivals, setArrivals] = useState<RealtimeArrival[]>([])
  const [loading, setLoading] = useState(true)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)

  useEffect(() => {
    let cancelled = false

    const load = () => {
      api.terminals.realtime(code).then((data) => {
        if (!cancelled) {
          setArrivals(data ?? [])
          setLastUpdated(new Date())
          setLoading(false)
        }
      }).catch(() => {
        if (!cancelled) {
          setArrivals([])
          setLoading(false)
        }
      })
    }

    setLoading(true)
    load()
    const timer = setInterval(load, REFRESH_MS)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [code])

  return (
    <div>
      {/* Header */}
      <div className="flex items-center gap-1.5 mb-1.5">
        <span className="text-xs font-semibold text-purple-700">실시간 도착</span>
        {!loading && arrivals.length > 0 && (
          <span className="flex items-center gap-0.5">
            <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" />
            <span className="text-[10px] text-green-600">LIVE</span>
          </span>
        )}
        {lastUpdated && (
          <span className="ml-auto text-[10px] text-gray-400">
            {lastUpdated.toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
          </span>
        )}
      </div>

      {loading && (
        <p className="text-xs text-gray-400 animate-pulse">불러오는 중...</p>
      )}

      {!loading && arrivals.length === 0 && (
        <p className="text-xs text-gray-400">실시간 정보 없음</p>
      )}

      {!loading && arrivals.length > 0 && (
        <table className="w-full text-xs border-collapse">
          <thead>
            <tr className="text-gray-400 border-b">
              <th className="text-left pb-1 pr-2">노선</th>
              <th className="text-left pb-1 pr-2">방향</th>
              <th className="text-left pb-1">도착</th>
            </tr>
          </thead>
          <tbody>
            {arrivals.map((a, i) => (
              <tr key={i} className="border-b last:border-0">
                <td className="py-1 pr-2 text-purple-700 font-medium">{a.line}</td>
                <td className="py-1 pr-2 text-gray-600 truncate max-w-[80px]">{a.direction}</td>
                <td className="py-1 text-gray-800 whitespace-nowrap">
                  {a.message || formatSecs(a.arriveSecs)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

function formatSecs(secs: number): string {
  if (secs <= 0) return '곧 도착'
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return m > 0 ? `${m}분 ${s}초` : `${s}초`
}
