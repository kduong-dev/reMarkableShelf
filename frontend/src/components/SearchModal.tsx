import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { Book, BookSearchResult, SearchSort } from '../api/types'

// Open Library's search refuses anything shorter.
const minimumQueryLength = 3
// How long typing has to pause before searching, so each keystroke doesn't
// send a request.
const searchDelayMs = 350

const sorts: Array<{ label: string; value: SearchSort }> = [
  { label: 'Relevance', value: '' },
  { label: 'Top rated', value: 'rating' },
  { label: 'Newest', value: 'new' },
  { label: 'Oldest', value: 'old' },
]

// Open Library filters on MARC language codes.
const languages: Array<{ label: string; value: string }> = [
  { label: 'Any language', value: '' },
  { label: 'English', value: 'eng' },
  { label: 'Spanish', value: 'spa' },
  { label: 'French', value: 'fre' },
  { label: 'German', value: 'ger' },
  { label: 'Italian', value: 'ita' },
  { label: 'Portuguese', value: 'por' },
  { label: 'Japanese', value: 'jpn' },
  { label: 'Chinese', value: 'chi' },
]

const eras: Array<{ label: string; from?: number; to?: number }> = [
  { label: 'Any time' },
  { label: 'Before 1900', to: 1899 },
  { label: '1900–1949', from: 1900, to: 1949 },
  { label: '1950s–70s', from: 1950, to: 1979 },
  { label: '1980s–90s', from: 1980, to: 1999 },
  { label: '2000s', from: 2000, to: 2009 },
  { label: '2010 onwards', from: 2010 },
]

function resultDetails(result: BookSearchResult): string {
  return [
    result.author,
    result.firstPublishYear,
    result.averageRating && `★ ${result.averageRating.toFixed(1)}`,
    result.pageCount && `${result.pageCount} pages`,
  ]
    .filter(Boolean)
    .join(' · ')
}

export function SearchModal({
  shelf,
  onClose,
  onAdded,
}: {
  shelf: Book[]
  onClose: () => void
  onAdded: (book: Book) => void
}) {
  const [query, setQuery] = useState('')
  const [sort, setSort] = useState<SearchSort>('')
  const [language, setLanguage] = useState('')
  const [eraIndex, setEraIndex] = useState(0)
  const [results, setResults] = useState<BookSearchResult[]>([])
  const [searchedFor, setSearchedFor] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [addingId, setAddingId] = useState<string | null>(null)

  const trimmed = query.trim()
  const era = eras[eraIndex]

  // Maps a result to the book already on the shelf, by Open Library work or
  // by ISBN, so it can't be added twice.
  const shelved = useMemo(() => {
    const byKey = new Map<string, Book>()
    for (const book of shelf) {
      if (book.openLibraryId) byKey.set(`work:${book.openLibraryId}`, book)
      if (book.isbn) byKey.set(`isbn:${book.isbn}`, book)
    }
    return (result: BookSearchResult) =>
      byKey.get(`work:${result.openLibraryId}`) ?? (result.isbn ? byKey.get(`isbn:${result.isbn}`) : undefined)
  }, [shelf])

  useEffect(() => {
    if (trimmed.length < minimumQueryLength) return
    const controller = new AbortController()
    const timer = setTimeout(async () => {
      setLoading(true)
      setError(null)
      try {
        const found = await api.searchBooks(
          { q: trimmed, sort, language, publishedFrom: era.from, publishedTo: era.to },
          controller.signal,
        )
        setResults(found)
        setSearchedFor(trimmed)
      } catch (err) {
        if (controller.signal.aborted) return
        setError(err instanceof Error ? err.message : 'search failed')
      } finally {
        if (!controller.signal.aborted) setLoading(false)
      }
    }, searchDelayMs)
    return () => {
      clearTimeout(timer)
      controller.abort()
    }
  }, [trimmed, sort, language, era])

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

  const tooShort = trimmed.length < minimumQueryLength
  const visibleResults = tooShort ? [] : results

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal search-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Add a book</h2>
          <button className="icon-button" onClick={onClose} aria-label="Close">
            &times;
          </button>
        </div>
        <input
          autoFocus
          type="search"
          className="search-input"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search by title, author, or ISBN"
          aria-label="Search Open Library"
        />
        <div className="search-filters">
          <div className="filter-tabs" role="group" aria-label="Sort">
            {sorts.map((option) => (
              <button
                key={option.value}
                className={sort === option.value ? 'tab active' : 'tab'}
                onClick={() => setSort(option.value)}
              >
                {option.label}
              </button>
            ))}
          </div>
          <div className="search-selects">
            <select value={language} onChange={(e) => setLanguage(e.target.value)} aria-label="Language">
              {languages.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            <select value={eraIndex} onChange={(e) => setEraIndex(Number(e.target.value))} aria-label="First published">
              {eras.map((option, index) => (
                <option key={option.label} value={index}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
        </div>
        <p className="search-status" aria-live="polite">
          {tooShort
            ? `Type at least ${minimumQueryLength} characters to search Open Library.`
            : loading
              ? 'Searching…'
              : !error && searchedFor === trimmed && results.length === 0
                ? 'No books match. Try a different spelling or fewer filters.'
                : ' '}
        </p>
        {error && <p className="error">{error}</p>}
        <ul className={loading ? 'search-results stale' : 'search-results'}>
          {visibleResults.map((result) => {
            const onShelf = shelved(result)
            return (
              <li key={result.openLibraryId} className="search-result">
                <div className="search-result-cover">
                  {result.coverUrl && <img src={result.coverUrl} alt="" loading="lazy" />}
                </div>
                <div className="search-result-info">
                  <div className="search-result-title">{result.title}</div>
                  <div className="search-result-author">{resultDetails(result)}</div>
                </div>
                {onShelf ? (
                  <Link to={`/books/${onShelf.id}`} className="on-shelf">
                    On your shelf
                  </Link>
                ) : (
                  <button onClick={() => add(result)} disabled={addingId === result.openLibraryId}>
                    {addingId === result.openLibraryId ? 'Adding…' : 'Add'}
                  </button>
                )}
              </li>
            )
          })}
        </ul>
      </div>
    </div>
  )
}
