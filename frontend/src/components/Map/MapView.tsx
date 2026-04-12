import 'leaflet/dist/leaflet.css'
import { MapContainer, TileLayer } from 'react-leaflet'
import { useStore } from '../../store'
import { useTerminals } from '../../hooks/useTerminals'
import { RouteLine } from './RouteLine'
import { TerminalMarker } from './TerminalMarker'

// South Korea center
const KR_CENTER: [number, number] = [36.5, 127.8]
const INITIAL_ZOOM = 7

export function MapView() {
  const terminals = useTerminals()
  const selectedResult = useStore((s) => s.selectedResult)

  return (
    <MapContainer
      center={KR_CENTER}
      zoom={INITIAL_ZOOM}
      className="h-full w-full"
      zoomControl={true}
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />

      {/* Terminal / station markers */}
      {terminals.map((t) => (
        <TerminalMarker key={t.id} terminal={t} />
      ))}

      {/* Route polyline for the selected search result */}
      {selectedResult && <RouteLine result={selectedResult} />}
    </MapContainer>
  )
}
