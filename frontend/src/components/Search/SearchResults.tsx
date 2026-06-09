import { useStore } from '../../store'
import { minsToLabel, minsToTime } from '../../types'

interface Props {
  onSelect?: () => void
}

export function SearchResults({ onSelect }: Props) {
  const results = useStore((s) => s.searchResults)
  const selected = useStore((s) => s.selectedResult)
  const selectResult = useStore((s) => s.selectResult)
  const searching = useStore((s) => s.searching)

  if (searching) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">
        <span className="animate-pulse">검색 중…</span>
      </div>
    )
  }

  if (results.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-400 text-sm px-4 text-center">
        출발지와 도착지를 입력하고 검색하세요
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto px-4 pb-4 space-y-2">
      <p className="text-xs text-gray-400 pt-2">{results.length}개 결과</p>
      {results.map((r, i) => {
        const isSelected = selected === r
        const depTime = minsToTime(r.legs[0].departureMins)
        const arrTime = minsToTime(r.legs[r.legs.length - 1].arrivalMins)
        const from = r.legs[0].from.name
        const to = r.legs[r.legs.length - 1].to.name

        return (
          <div
            key={i}
            onClick={() => {
              selectResult(isSelected ? null : r)
              if (!isSelected) onSelect?.()
            }}
            className={`rounded-xl border p-3 cursor-pointer transition-all
              ${isSelected
                ? 'border-blue-500 bg-blue-50 shadow-sm'
                : 'border-gray-200 bg-white hover:border-blue-300 hover:shadow-sm'
              }`}
          >
            {/* Time row */}
            <div className="flex items-baseline justify-between mb-1">
              <span className="text-lg font-bold text-gray-900">{depTime}</span>
              <span className="text-xs text-gray-400 mx-2">→</span>
              <span className="text-lg font-bold text-gray-900">{arrTime}</span>
              <span className="ml-auto text-sm font-semibold text-blue-600">
                {minsToLabel(r.totalMinutes)}
              </span>
            </div>

            {/* Route summary */}
            <div className="text-xs text-gray-500 mb-2">
              {from} → {to}
            </div>

            {/* Legs */}
            <div className="space-y-1">
              {r.legs.map((leg, j) => (
                <div key={j} className="flex items-center gap-1 text-xs">
                  <span className={`px-1.5 py-0.5 rounded text-white font-medium
                    ${leg.routeType === 'rail' ? 'bg-red-500' : leg.routeType === 'metro' ? 'bg-purple-500' : 'bg-blue-500'}`}>
                    {leg.routeType === 'rail' ? '철도' : leg.routeType === 'metro' ? '지하철' : leg.routeType === 'express' ? '고속' : '시외'}
                  </span>
                  <span className="text-gray-600">{leg.from.name}</span>
                  <span className="text-gray-400">→</span>
                  <span className="text-gray-600">{leg.to.name}</span>
                  {leg.operator && (
                    <span className="text-gray-400 ml-auto">{leg.operator}</span>
                  )}
                </div>
              ))}
            </div>

            {/* Transfers badge */}
            {r.transfers > 0 && (
              <div className="mt-2">
                <span className="text-xs bg-amber-100 text-amber-700 px-2 py-0.5 rounded-full">
                  {r.transfers}회 환승
                </span>
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
