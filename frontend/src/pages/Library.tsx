import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '../api/client'
import { bookFromResult } from '../bookFromResult'
import { BookCard } from '../components/BookCard'
import { SearchModal } from '../components/SearchModal'
import { type LibrarySort, librarySorts, onTablet, sortBooks, type TabletFilter, tabletFilters } from '../librarySort'
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
  // The status, sort and tablet filter live in the URL, so a reload or the
  // back button keeps them.
  const [searchParams, setSearchParams] = useSearchParams()
  const filter = (filters.find((f) => f.value === searchParams.get('status'))?.value ?? 'all') as BookStatus | 'all'
  const sort = librarySorts.find((option) => option.value === searchParams.get('sort'))?.value ?? 'updated'
  const tablet = tabletFilters.find((option) => option.value === searchParams.get('tablet'))?.value ?? 'all'
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

  // setParam sets a URL parameter, dropping it when it's the default.
  function setParam(key: string, value: string, fallback: string) {
    setSearchParams(
      (params) => {
        if (value === fallback) params.delete(key)
        else params.set(key, value)
        return params
      },
      { replace: true },
    )
  }

  const visible = useMemo(() => {
    const matching = books
      .filter((b) => filter === 'all' || b.status === filter)
      .filter((b) => tablet === 'all' || onTablet(b) === (tablet === 'on'))
      .filter((b) => {
        const q = query.trim().toLowerCase()
        if (!q) return true
        return b.title.toLowerCase().includes(q) || b.author.toLowerCase().includes(q)
      })
    return sortBooks(matching, sort)
  }, [books, filter, tablet, sort, query])
  const filtered = filter !== 'all' || tablet !== 'all' || query.trim() !== ''

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
              onClick={() => setParam('status', f.value, 'all')}
            >
              {f.label}
            </button>
          ))}
        </div>
        <div className="library-selects">
          <select
            value={tablet}
            onChange={(e) => setParam('tablet', e.target.value as TabletFilter, 'all')}
            aria-label="Where the book is"
          >
            {tabletFilters.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          <select
            value={sort}
            onChange={(e) => setParam('sort', e.target.value as LibrarySort, 'updated')}
            aria-label="Sort books"
          >
            {librarySorts.map((option) => (
              <option key={option.value} value={option.value}>
                Sort: {option.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {loading && <p>Loading…</p>}
      {error && <p className="error">{error}</p>}
      {!loading && !error && visible.length === 0 && (
        <p className="empty-state">
          {filtered && books.length > 0
            ? 'No books match these filters.'
            : 'No books yet. Add one manually, or sync a tablet from the Sync tab.'}
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
          onChoose={async (result) => {
            const book = await api.createBook(bookFromResult(result, 'manual'))
            setBooks((prev) => [book, ...prev])
            setShowAdd(false)
          }}
        />
      )}
    </section>
  )
}
