import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { Book, BookSearch, BookSearchResult, SearchSort } from '../api/types'

// Open Library's search refuses anything shorter.
const minimumQueryLength = 3
// Matches the Open Library client's page size on the server.
const pageSize = 20
// How long typing has to pause before searching, so each keystroke doesn't
// send a request.
const searchDelayMs = 350

// Open Library subjects to browse by; each is searched as subject:"<value>".
const genres: Array<{ label: string; value: string }> = [
  { label: 'Fantasy', value: 'fantasy' },
  { label: 'Science fiction', value: 'science fiction' },
  { label: 'Mystery', value: 'mystery' },
  { label: 'Thriller', value: 'thriller' },
  { label: 'Romance', value: 'romance' },
  { label: 'Horror', value: 'horror' },
  { label: 'Classics', value: 'classics' },
  { label: 'Young adult', value: 'young adult fiction' },
  { label: 'Graphic novels', value: 'graphic novels' },
  { label: 'Poetry', value: 'poetry' },
  { label: 'History', value: 'history' },
  { label: 'Biography', value: 'biography' },
  { label: 'Philosophy', value: 'philosophy' },
  { label: 'Self-help', value: 'self-help' },
]

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
  const [genre, setGenre] = useState('')
  const [sort, setSort] = useState<SearchSort>('')
  const [language, setLanguage] = useState('')
  const [eraIndex, setEraIndex] = useState(0)
  const [results, setResults] = useState<BookSearchResult[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [searched, setSearched] = useState<BookSearch | null>(null)
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [addingId, setAddingId] = useState<string | null>(null)
  // Aborts an in-flight "Load more" when the search it extends changes.
  const loadMoreController = useRef<AbortController | null>(null)

  const trimmed = query.trim()
  const era = eras[eraIndex]
  // A genre can be browsed with no query, but a query has to be long enough
  // for Open Library either way.
  const tooShort = trimmed.length > 0 && trimmed.length < minimumQueryLength
  const canSearch = !tooShort && (trimmed !== '' || genre !== '')
  const search = useMemo<BookSearch>(
    () => ({ q: trimmed, subject: genre, sort, language, publishedFrom: era.from, publishedTo: era.to }),
    [trimmed, genre, sort, language, era],
  )

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
    if (!canSearch) return
    const controller = new AbortController()
    const timer = setTimeout(async () => {
      setLoading(true)
      setError(null)
      try {
        const found = await api.searchBooks(search, controller.signal)
        setResults(found.results)
        setTotal(found.total)
        setPage(1)
        setSearched(search)
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
      loadMoreController.current?.abort()
      setLoadingMore(false)
    }
  }, [canSearch, search])

  async function loadMore() {
    if (!searched) return
    const controller = new AbortController()
    loadMoreController.current = controller
    setLoadingMore(true)
    setError(null)
    try {
      const found = await api.searchBooks({ ...searched, page: page + 1 }, controller.signal)
      // Open Library can repeat a work across pages, so skip ones already shown.
      setResults((shown) => {
        const seen = new Set(shown.map((result) => result.openLibraryId))
        return [...shown, ...found.results.filter((result) => !seen.has(result.openLibraryId))]
      })
      setTotal(found.total)
      setPage(page + 1)
    } catch (err) {
      if (controller.signal.aborted) return
      setError(err instanceof Error ? err.message : 'failed to load more')
    } finally {
      if (!controller.signal.aborted) setLoadingMore(false)
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

  const visibleResults = canSearch ? results : []
  const current = searched === search
  const hasMore = canSearch && current && !loading && page * pageSize < total

  let status = ' '
  if (tooShort) status = `Type at least ${minimumQueryLength} characters to search Open Library.`
  else if (!canSearch) status = 'Type a title, author or ISBN, or pick a genre to browse.'
  else if (loading) status = 'Searching…'
  else if (current && !error && results.length === 0)
    status = 'No books match. Try a different spelling or fewer filters.'
  else if (current && !error) status = `Showing ${results.length} of ${total.toLocaleString()} books`

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
        <div className="genre-chips" role="group" aria-label="Genre">
          {genres.map((option) => (
            <button
              key={option.value}
              className={genre === option.value ? 'chip active' : 'chip'}
              aria-pressed={genre === option.value}
              onClick={() => setGenre(genre === option.value ? '' : option.value)}
            >
              {option.label}
            </button>
          ))}
        </div>
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
          {status}
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
        {hasMore && (
          <button className="load-more" onClick={loadMore} disabled={loadingMore}>
            {loadingMore ? 'Loading…' : 'Load more'}
          </button>
        )}
      </div>
    </div>
  )
}
