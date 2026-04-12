import { useEffect, useRef, useState } from 'react'
import { api } from '../../api/client'
import type { Terminal } from '../../types'

interface Props {
  label: string
  placeholder?: string
  value: Terminal | null
  onChange: (t: Terminal | null) => void
}

/** Autocomplete input that searches terminals by name via the API. */
export function TerminalInput({ label, placeholder, value, onChange }: Props) {
  const [query, setQuery] = useState(value?.name ?? '')
  const [results, setResults] = useState<Terminal[]>([])
  const [open, setOpen] = useState(false)
  const debounce = useRef<ReturnType<typeof setTimeout> | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // Keep input text in sync when value is cleared externally
  useEffect(() => {
    setQuery(value?.name ?? '')
  }, [value])

  // Close dropdown when clicking outside
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const handleInput = (text: string) => {
    setQuery(text)
    onChange(null)
    if (debounce.current) clearTimeout(debounce.current)
    if (text.trim().length < 1) {
      setResults([])
      setOpen(false)
      return
    }
    debounce.current = setTimeout(async () => {
      try {
        const list = await api.terminals.search(text.trim())
        setResults(list ?? [])
        setOpen(true)
      } catch {
        setResults([])
      }
    }, 250)
  }

  const select = (t: Terminal) => {
    onChange(t)
    setQuery(t.name)
    setOpen(false)
    setResults([])
  }

  const typeLabel = (type: Terminal['type']) => (type === 'bus' ? '🚌' : '🚆')

  return (
    <div ref={containerRef} className="relative">
      <label className="block text-xs font-medium text-gray-500 mb-1">{label}</label>
      <input
        type="text"
        value={query}
        onChange={(e) => handleInput(e.target.value)}
        onFocus={() => results.length > 0 && setOpen(true)}
        placeholder={placeholder ?? '도시 또는 터미널 이름'}
        className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
      />
      {open && results.length > 0 && (
        <ul className="absolute z-50 w-full mt-1 bg-white border border-gray-200
                        rounded-lg shadow-lg max-h-48 overflow-y-auto">
          {results.map((t) => (
            <li
              key={t.id}
              onMouseDown={() => select(t)}
              className="px-3 py-2 text-sm cursor-pointer hover:bg-blue-50 flex items-center gap-2"
            >
              <span>{typeLabel(t.type)}</span>
              <span className="font-medium">{t.name}</span>
              <span className="text-gray-400 text-xs ml-auto">{t.regionCode}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
