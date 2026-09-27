import { useEffect, useMemo, useState } from 'react'
import { api } from '../api/client'
import { BookCard } from '../components/BookCard'
import { SearchModal } from '../components/SearchModal'
import { useRefreshOnFocus } from '../useRefreshOnFocus'
import type { Book, BookStatus } from '../api/types'

const filters: Array<{ label: string; value: BookStatus | 'all' }> = [
  { label: 'All', value: 'all' },
  { label: 'Want to Read', value: 'want_to_read' },
  { label: 'Reading', value: 'reading' },
  { label: 'Finished', value: 'finished' },
]

export function Library() {
  const [books, setBooks] = useState<Book[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState<BookStatus | 'all'>('all')
  const [query, setQuery] = useState('')
  const [showAdd, setShowAdd] = useState(false)

  function load() {
    api
      .listBooks()
      .then(setBooks)
      .catch((err) => setError(err instanceof Error ? err.message : 'failed to load books'))
      .finally(() => setLoading(false))
  }
  useEffect(load, [])
  useRefreshOnFocus(load)

  const visible = useMemo(() => {
    return books
      .filter((b) => filter === 'all' || b.status === filter)
      .filter((b) => {
        const q = query.trim().toLowerCase()
        if (!q) return true
        return b.title.toLowerCase().includes(q) || b.author.toLowerCase().includes(q)
      })
  }, [books, filter, query])

  return (
    <section>
      <div className="page-header">
        <h1>Library</h1>
        <button className="primary" onClick={() => setShowAdd(true)}>
          Add book
        </button>
      </div>

      <div className="toolbar">
        <input
          className="library-search"
          placeholder="Filter by title or author…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <div className="filter-tabs">
          {filters.map((f) => (
            <button
              key={f.value}
              className={filter === f.value ? 'tab active' : 'tab'}
              onClick={() => setFilter(f.value)}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {loading && <p>Loading…</p>}
      {error && <p className="error">{error}</p>}
      {!loading && !error && visible.length === 0 && (
        <p className="empty-state">
          No books yet. Add one manually, or sync a tablet from the Sync tab.
        </p>
      )}

      <div className="book-grid">
        {visible.map((book) => (
          <BookCard key={book.id} book={book} />
        ))}
      </div>

      {showAdd && (
        <SearchModal
          shelf={books}
          onClose={() => setShowAdd(false)}
          onAdded={(book) => {
            setBooks((prev) => [book, ...prev])
            setShowAdd(false)
          }}
        />
      )}
    </section>
  )
}
