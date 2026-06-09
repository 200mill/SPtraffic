import { useState } from 'react'
import { useSearch } from '../../hooks/useSearch'
import type { SearchQuery, Terminal } from '../../types'
import { TerminalInput } from './TerminalInput'

export function SearchForm() {
  const [from, setFrom] = useState<Terminal | null>(null)
  const [to, setTo] = useState<Terminal | null>(null)
  const [date, setDate] = useState(() => {
    // Default to now
    const now = new Date()
    now.setSeconds(0, 0)
    return now.toISOString().slice(0, 16)  // "YYYY-MM-DDTHH:MM"
  })
  const [mode, setMode] = useState<SearchQuery['mode']>('all')
  const [maxLegs, setMaxLegs] = useState<1 | 2 | 3>(2)

  const { search, searching, error } = useSearch()

  const swap = () => {
    setFrom(to)
    setTo(from)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!from || !to) return
    search({ from: from.code, to: to.code, date, mode, maxLegs })
  }

  return (
    <form onSubmit={handleSubmit} className="p-4 space-y-3">
      {/* From / To */}
      <div className="space-y-2">
        <TerminalInput label="출발지" value={from} onChange={setFrom} />

        <div className="flex justify-center">
          <button
            type="button"
            onClick={swap}
            className="text-gray-400 hover:text-blue-500 text-lg transition-colors"
            title="출발지/도착지 교환"
          >
            ⇅
          </button>
        </div>

        <TerminalInput label="도착지" value={to} onChange={setTo} />
      </div>

      {/* Date */}
      <div>
        <label className="block text-xs font-medium text-gray-500 mb-1">출발 일시</label>
        <input
          type="datetime-local"
          value={date}
          onChange={(e) => setDate(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      {/* Mode + Max legs */}
      <div className="flex gap-2">
        <div className="flex-1">
          <label className="block text-xs font-medium text-gray-500 mb-1">교통수단</label>
          <select
            value={mode}
            onChange={(e) => setMode(e.target.value as SearchQuery['mode'])}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">전체</option>
            <option value="bus">버스만</option>
            <option value="rail">철도만</option>
            <option value="metro">지하철만</option>
          </select>
        </div>

        <div className="flex-1">
          <label className="block text-xs font-medium text-gray-500 mb-1">환승</label>
          <select
            value={maxLegs}
            onChange={(e) => setMaxLegs(Number(e.target.value) as 1 | 2 | 3)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value={1}>직행만</option>
            <option value={2}>1회 환승</option>
            <option value={3}>2회 환승</option>
          </select>
        </div>
      </div>

      {/* Submit */}
      <button
        type="submit"
        disabled={!from || !to || searching}
        className="w-full py-2.5 bg-blue-600 text-white font-semibold rounded-lg
                   hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed
                   transition-colors text-sm"
      >
        {searching ? '검색 중…' : '시간표 검색'}
      </button>

      {error && (
        <div className="flex items-start gap-2 bg-red-50 border border-red-200
                        text-red-700 text-xs rounded-lg px-3 py-2">
          <span className="mt-0.5">⚠</span>
          <span className="flex-1">{error}</span>
        </div>
      )}
    </form>
  )
}
