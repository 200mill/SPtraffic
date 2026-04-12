import { useEffect } from 'react'
import { Polyline, useMap } from 'react-leaflet'
import type { SearchResult } from '../../types'

interface Props {
  result: SearchResult
}

/** Draws a polyline on the map for each leg of a search result,
 *  and pans/zooms the map to fit the full route. */
export function RouteLine({ result }: Props) {
  const map = useMap()

  // Build positions: [from, to] for each leg (deduplicated at transfer points)
  const positions: [number, number][] = [
    [result.legs[0].from.lat, result.legs[0].from.lon],
    ...result.legs.map((leg) => [leg.to.lat, leg.to.lon] as [number, number]),
  ]

  useEffect(() => {
    if (positions.length < 2) return
    map.fitBounds(positions as [number, number][], { padding: [60, 60] })
  }, [result])  // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <>
      {/* Main route line */}
      <Polyline
        positions={positions}
        pathOptions={{ color: '#3b82f6', weight: 4, opacity: 0.85 }}
      />

      {/* Transfer points — amber dots */}
      {result.transfers > 0 &&
        result.legs.slice(0, -1).map((leg, i) => (
          <Polyline
            key={i}
            positions={[[leg.to.lat, leg.to.lon]]}
            pathOptions={{ color: '#f59e0b', weight: 8, opacity: 1 }}
          />
        ))}

      {/* Departure / arrival labels shown as tooltips on first and last marker */}
      <Polyline
        positions={[positions[0]]}
        pathOptions={{ color: '#16a34a', weight: 10, opacity: 1 }}
      />
      <Polyline
        positions={[positions[positions.length - 1]]}
        pathOptions={{ color: '#dc2626', weight: 10, opacity: 1 }}
      />
    </>
  )
}
