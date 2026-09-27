import { useState } from 'react'
import { api } from '../api/client'
import type { Book, BookSearchResult } from '../api/types'

// Open Library's search refuses anything shorter.
const minimumQueryLength = 3

export function SearchModal({
  onClose,
  onAdded,
}: {
  onClose: () => void
  onAdded: (book: Book) => void
}) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<BookSearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [addingId, setAddingId] = useState<string | null>(null)

  async function search(e: React.FormEvent) {
    e.preventDefault()
    if (query.trim().length < minimumQueryLength) {
      setError(`Search for at least ${minimumQueryLength} characters`)
      return
    }
    setLoading(true)
    setError(null)
    try {
      setResults(await api.searchBooks(query.trim()))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'search failed')
    } finally {
      setLoading(false)
    }
  }

  async function add(result: BookSearchResult) {
    setAddingId(result.openLibraryId)
    try {
      const book = await api.createBook({
        title: result.title,
        author: result.author,
        isbn: result.isbn,
        coverUrl: result.coverUrl,
        pageCount: result.pageCount,
        openLibraryId: result.openLibraryId,
        source: 'manual',
      })
      onAdded(book)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed to add book')
    } finally {
      setAddingId(null)
    }
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Add a book</h2>
          <button className="icon-button" onClick={onClose} aria-label="Close">
            &times;
          </button>
        </div>
        <form onSubmit={search} className="search-form">
          <input
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search by title, author, or ISBN"
          />
          <button type="submit" className="primary" disabled={loading}>
            {loading ? 'Searching…' : 'Search'}
          </button>
        </form>
        {error && <p className="error">{error}</p>}
        <ul className="search-results">
          {results.map((result) => (
            <li key={result.openLibraryId} className="search-result">
              <div className="search-result-cover">
                {result.coverUrl && <img src={result.coverUrl} alt="" />}
              </div>
              <div className="search-result-info">
                <div className="search-result-title">{result.title}</div>
                <div className="search-result-author">
                  {[result.author, result.pageCount && `${result.pageCount} pages`]
                    .filter(Boolean)
                    .join(' · ')}
                </div>
              </div>
              <button onClick={() => add(result)} disabled={addingId === result.openLibraryId}>
                {addingId === result.openLibraryId ? 'Adding…' : 'Add'}
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}
