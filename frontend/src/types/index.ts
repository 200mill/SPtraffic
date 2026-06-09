export interface Terminal {
  id: number
  code: string
  name: string
  type: 'bus' | 'rail' | 'metro'
  regionCode: string
  lat: number
  lon: number
}

export interface RealtimeArrival {
  line: string
  direction: string
  destination: string
  arriveSecs: number
  message: string
}

export interface Leg {
  from: Terminal
  to: Terminal
  departureMins: number
  arrivalMins: number
  routeType: string
  operator: string
}

export interface SearchResult {
  legs: Leg[]
  totalMinutes: number
  transfers: number
}

export interface Schedule {
  id: number
  routeId: number
  departureMins: number
  arrivalMins: number
  durationMin: number
  daysOfWeek: number
}

export interface Route {
  id: number
  code: string
  type: string
  originId: number
  destId: number
  operator: string
}

export interface RouteSchedule {
  route: Route
  origin: Terminal
  destination: Terminal
  schedules: Schedule[]
}

export interface DepartureInfo {
  routeId: number
  destination: string
  destCode: string
  routeType: string
  operator: string
  departureMins: number
  arrivalMins: number
  durationMin: number
}

export interface SearchQuery {
  from: string        // terminal code
  to: string
  date: string        // "YYYY-MM-DDTHH:MM"
  mode: 'all' | 'bus' | 'rail' | 'metro'
  maxLegs: 1 | 2 | 3
}

/** minutes since midnight → "HH:MM" */
export function minsToTime(mins: number): string {
  const h = Math.floor(mins / 60).toString().padStart(2, '0')
  const m = (mins % 60).toString().padStart(2, '0')
  return `${h}:${m}`
}

/** total minutes → "Xh Ym" */
export function minsToLabel(mins: number): string {
  const h = Math.floor(mins / 60)
  const m = mins % 60
  return h > 0 ? `${h}시간 ${m}분` : `${m}분`
}
