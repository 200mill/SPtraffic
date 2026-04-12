import { useState } from 'react'
import { SearchForm } from './components/Search/SearchForm'
import { SearchResults } from './components/Search/SearchResults'
import { MapView } from './components/Map/MapView'
import { useStore } from './store'

export default function App() {
  const [panelOpen, setPanelOpen] = useState(true)
  const resultCount = useStore((s) => s.searchResults.length)

  return (
    <div className="flex flex-col h-screen bg-gray-50">
      {/* ── Header ─────────────────────────────────────────── */}
      <header className="flex-none h-12 bg-white border-b border-gray-200
                         flex items-center px-4 gap-3 shadow-sm z-20">
        {/* Mobile toggle */}
        <button
          className="sm:hidden p-1 rounded text-gray-500 hover:text-blue-600 hover:bg-gray-100"
          onClick={() => setPanelOpen((v) => !v)}
          title={panelOpen ? '지도 보기' : '검색 열기'}
        >
          {panelOpen ? '🗺' : '🔍'}
        </button>

        <span className="text-lg hidden sm:block">🚌</span>
        <h1 className="text-base font-bold text-gray-900 tracking-tight">SPtraffic</h1>
        <span className="text-xs text-gray-400 hidden sm:block">
          전국 시외·고속버스 및 간선철도 시간표
        </span>

        {/* Mobile result badge */}
        {resultCount > 0 && !panelOpen && (
          <span className="sm:hidden ml-auto text-xs bg-blue-600 text-white
                           px-2 py-0.5 rounded-full">
            {resultCount}개
          </span>
        )}
      </header>

      {/* ── Main layout ────────────────────────────────────── */}
      <div className="flex flex-1 overflow-hidden relative">

        {/* Left panel — search (full-screen overlay on mobile when open) */}
        <aside className={`
          flex flex-col bg-white border-r border-gray-200 shadow-sm z-10
          transition-all duration-200
          ${panelOpen
            ? 'w-full sm:w-80 flex-none'
            : 'w-0 overflow-hidden border-0'
          }
        `}>
          <SearchForm />
          <div className="flex-1 overflow-hidden flex flex-col border-t border-gray-100">
            <SearchResults onSelect={() => {
              // On mobile, close panel when a result is selected so map is visible
              if (window.innerWidth < 640) setPanelOpen(false)
            }} />
          </div>
        </aside>

        {/* Map — always rendered, just hidden behind panel on mobile */}
        <main className={`flex-1 relative ${panelOpen ? 'hidden sm:block' : 'block'}`}>
          <MapView />
        </main>

      </div>
    </div>
  )
}
