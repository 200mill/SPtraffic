import { useEffect, useState } from 'react'
import { api } from '../../api/client'
import type { DepartureInfo, Terminal } from '../../types'
import { minsToTime } from '../../types'

interface Props {
  terminal: Terminal
}

const ROUTE_TYPE_LABEL: Record<string, string> = {
  express: '고속',
  intercity: '시외',
  rail: '철도',
}

/** Shows upcoming departures from a terminal. */
export function SchedulePopup({ terminal }: Props) {
  const [loading, setLoading] = useState(true)
  const [departures, setDepartures] = useState<DepartureInfo[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    setError(null)
    api.terminals
      .departures(terminal.code, { limit: 10 })
      .then(setDepartures)
      .catch(() => setError('시간표를 불러오지 못했습니다.'))
      .finally(() => setLoading(false))
  }, [terminal.code])

  const typeLabel = terminal.type === 'bus' ? '버스터미널' : '철도역'

  return (
    <div className="min-w-[220px] max-w-xs">
      <div className="font-bold text-sm text-gray-900">{terminal.name}</div>
      <div className="text-xs text-gray-500 mb-2">{typeLabel} · {terminal.regionCode}</div>

      {loading && (
        <p className="text-xs text-gray-400 py-2">불러오는 중...</p>
      )}

      {error && (
        <p className="text-xs text-red-500 py-2">{error}</p>
      )}

      {!loading && !error && departures.length === 0 && (
        <p className="text-xs text-gray-400 py-2">오늘 출발 편이 없습니다.</p>
      )}

      {!loading && !error && departures.length > 0 && (
        <table className="w-full text-xs border-collapse">
          <thead>
            <tr className="text-gray-400 border-b">
              <th className="text-left pb-1 pr-2">출발</th>
              <th className="text-left pb-1 pr-2">도착지</th>
              <th className="text-left pb-1">구분</th>
            </tr>
          </thead>
          <tbody>
            {departures.map((d, i) => (
              <tr key={`${d.routeId}-${d.departureMins}-${i}`} className="border-b last:border-0">
                <td className="py-1 pr-2 font-mono font-medium text-gray-900">
                  {minsToTime(d.departureMins)}
                </td>
                <td className="py-1 pr-2 text-gray-700 truncate max-w-[100px]">
                  {d.destination}
                </td>
                <td className="py-1 text-gray-400">
                  {ROUTE_TYPE_LABEL[d.routeType] ?? d.routeType}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
