import { api } from '../api/client'
import { useStore } from '../store'
import type { SearchQuery } from '../types'

/** Returns a search function that calls the API and updates global state. */
export function useSearch() {
  const { setSearchResults, setSearching, setSearchError, selectResult } = useStore()

  const search = async (query: SearchQuery) => {
    setSearching(true)
    setSearchError(null)
    selectResult(null)

    try {
      const results = await api.search(query)
      setSearchResults(results ?? [])
    } catch (err) {
      setSearchError(err instanceof Error ? err.message : '검색 중 오류가 발생했습니다')
      setSearchResults([])
    } finally {
      setSearching(false)
    }
  }

  return {
    search,
    results: useStore((s) => s.searchResults),
    searching: useStore((s) => s.searching),
    error: useStore((s) => s.searchError),
  }
}
