import { useEffect } from 'react'
import { api } from '../api/client'
import { useStore } from '../store'

/** Fetches all terminals once on mount and stores them in global state. */
export function useTerminals() {
  const setTerminals = useStore((s) => s.setTerminals)

  useEffect(() => {
    api.terminals
      .listAll({ limit: 500 })
      .then((paged) => setTerminals(paged.items))
      .catch((err) => console.warn('Failed to load terminals:', err))
  }, [setTerminals])

  return useStore((s) => s.terminals)
}
