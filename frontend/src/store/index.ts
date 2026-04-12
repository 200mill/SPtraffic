import { create } from 'zustand'
import type { SearchResult, Terminal } from '../types'

interface AppState {
  // All terminals loaded from the API (used for map markers)
  terminals: Terminal[]
  setTerminals: (t: Terminal[]) => void

  // Current search results
  searchResults: SearchResult[]
  setSearchResults: (r: SearchResult[]) => void

  // The result the user has selected from the list (highlighted on map)
  selectedResult: SearchResult | null
  selectResult: (r: SearchResult | null) => void

  // Terminal whose schedule popup is open
  selectedTerminal: Terminal | null
  selectTerminal: (t: Terminal | null) => void

  // Whether a search is in progress
  searching: boolean
  setSearching: (v: boolean) => void

  // Last search error
  searchError: string | null
  setSearchError: (e: string | null) => void
}

export const useStore = create<AppState>((set) => ({
  terminals: [],
  setTerminals: (terminals) => set({ terminals }),

  searchResults: [],
  setSearchResults: (searchResults) => set({ searchResults }),

  selectedResult: null,
  selectResult: (selectedResult) => set({ selectedResult }),

  selectedTerminal: null,
  selectTerminal: (selectedTerminal) => set({ selectedTerminal }),

  searching: false,
  setSearching: (searching) => set({ searching }),

  searchError: null,
  setSearchError: (searchError) => set({ searchError }),
}))
