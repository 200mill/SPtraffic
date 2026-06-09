import L from 'leaflet'
import { Marker, Popup } from 'react-leaflet'
import { useStore } from '../../store'
import type { Terminal } from '../../types'
import { SchedulePopup } from '../Schedule/SchedulePopup'

interface Props {
  terminal: Terminal
}

/** Creates a small colored marker — blue circle (bus), red square (rail), purple diamond (metro). */
function makeIcon(type: Terminal['type']) {
  const color = type === 'bus' ? '#3b82f6' : type === 'metro' ? '#a855f7' : '#ef4444'
  const shape =
    type === 'bus'
      ? 'border-radius:50%'
      : type === 'metro'
        ? 'border-radius:2px;transform:rotate(45deg)'
        : 'border-radius:2px'
  return L.divIcon({
    className: '',
    html: `<div style="
      width:10px;height:10px;${shape};
      background:${color};border:2px solid white;
      box-shadow:0 1px 4px rgba(0,0,0,0.35);
    "></div>`,
    iconSize: [10, 10],
    iconAnchor: [5, 5],
  })
}

export function TerminalMarker({ terminal }: Props) {
  const selectTerminal = useStore((s) => s.selectTerminal)
  const icon = makeIcon(terminal.type)

  return (
    <Marker
      position={[terminal.lat, terminal.lon]}
      icon={icon}
      eventHandlers={{ click: () => selectTerminal(terminal) }}
    >
      <Popup eventHandlers={{ remove: () => selectTerminal(null) }} minWidth={240}>
        <SchedulePopup terminal={terminal} />
      </Popup>
    </Marker>
  )
}
