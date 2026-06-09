import type { DepartureInfo, RealtimeArrival, RouteSchedule, SearchQuery, SearchResult, Terminal } from '../types'

const BASE = import.meta.env.VITE_API_BASE ?? ''

async function get<T>(path: string, params?: Record<string, string>): Promise<T> {
  const url = new URL(BASE + path, window.location.origin)
  if (params) {
    Object.entries(params).forEach(([k, v]) => v !== undefined && url.searchParams.set(k, v))
  }
  const res = await fetch(url.toString())
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(`HTTP ${res.status} ${path}: ${text}`)
  }
  return res.json() as Promise<T>
}

export interface PagedTerminals {
  items: Terminal[]
  total: number
  limit: number
  offset: number
}

export const api = {
  terminals: {
    /** Autocomplete / filtered search — always returns a plain Terminal[]. */
    search: (q: string) =>
      get<Terminal[]>('/api/terminals', { q }),
    /** Full paginated list (no filter). */
    listAll: (params?: { limit?: number; offset?: number }) =>
      get<PagedTerminals>('/api/terminals', {
        ...(params?.limit !== undefined ? { limit: String(params.limit) } : {}),
        ...(params?.offset !== undefined ? { offset: String(params.offset) } : {}),
      }),
    get: (code: string) =>
      get<Terminal>(`/api/terminals/${encodeURIComponent(code)}`),
    departures: (code: string, params?: { after?: string; limit?: number }) =>
      get<DepartureInfo[]>(`/api/terminals/${encodeURIComponent(code)}/departures`, {
        ...(params?.after ? { after: params.after } : {}),
        ...(params?.limit !== undefined ? { limit: String(params.limit) } : {}),
      }),
    realtime: (code: string) =>
      get<RealtimeArrival[]>(`/api/terminals/${encodeURIComponent(code)}/realtime`),
  },

  search: (q: SearchQuery) =>
    get<SearchResult[]>('/api/search', {
      from: q.from,
      to: q.to,
      date: q.date,
      mode: q.mode,
      max_legs: String(q.maxLegs),
    }),

  schedule: (routeId: number) =>
    get<RouteSchedule>(`/api/schedule/${routeId}`),

  regions: () => get<string[]>('/api/regions'),
}
