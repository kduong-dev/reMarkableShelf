import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import { bookFromResult } from '../bookFromResult'
import { cleanFileName } from '../bookSearchQuery'
import { Bookmark } from '../components/Bookmark'
import { SearchModal } from '../components/SearchModal'
import { useRefreshOnFocus } from '../useRefreshOnFocus'
import type { Book, BookStatus } from '../api/types'

const statusOptions: Array<{ label: string; value: BookStatus }> = [
  { label: 'Want to Read', value: 'want_to_read' },
  { label: 'Reading', value: 'reading' },
  { label: 'Finished', value: 'finished' },
]

export function BookDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [book, setBook] = useState<Book | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [matching, setMatching] = useState(false)
  const [editing, setEditing] = useState<{ title: string; author: string } | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)

  function load() {
    if (!id) return
    api
      .getBook(id)
      .then(setBook)
      .catch((err) => setError(err instanceof Error ? err.message : 'failed to load book'))
  }
  useEffect(load, [id])
  useRefreshOnFocus(load)

  async function updateStatus(status: BookStatus) {
    if (!book) return
    // Finishing a book of known length moves the bookmark to its last page.
    const currentPage = status === 'finished' && book.pageCount ? book.pageCount : undefined
    const updated = await api.updateBook(book.id, { status, currentPage })
    setBook(updated)
  }

  async function saveDetails(e: React.FormEvent) {
    e.preventDefault()
    if (!book || !editing || !editing.title.trim()) return
    setSaveError(null)
    try {
      setBook(await api.updateBook(book.id, { title: editing.title.trim(), author: editing.author.trim() }))
      setEditing(null)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'failed to save')
    }
  }

  async function remove() {
    if (!book) return
    if (!confirm(`Remove "${book.title}" from your collection?`)) return
    await api.deleteBook(book.id)
    navigate('/')
  }

  if (error) return <p className="error">{error}</p>
  if (!book) return <p>Loading…</p>

  return (
    <section className="book-detail">
      <Link to="/" className="back-link">
        &larr; Library
      </Link>
      <div className="book-detail-header">
        {book.coverUrl ? (
          <img className="book-detail-cover" src={book.coverUrl} alt="" />
        ) : (
          <div className="book-detail-cover book-detail-cover-empty">{book.title.slice(0, 1)}</div>
        )}
        <div>
          {editing ? (
            <form className="details-form" onSubmit={saveDetails}>
              <label>
                Title
                <input
                  autoFocus
                  value={editing.title}
                  onChange={(e) => setEditing({ ...editing, title: e.target.value })}
                />
              </label>
              <label>
                Author
                <input value={editing.author} onChange={(e) => setEditing({ ...editing, author: e.target.value })} />
              </label>
              <div className="details-form-actions">
                <button type="submit" className="primary" disabled={!editing.title.trim()}>
                  Save
                </button>
                <button type="button" onClick={() => setEditing(null)}>
                  Cancel
                </button>
              </div>
              {saveError && <p className="error">{saveError}</p>}
            </form>
          ) : (
            <>
              <h1>{book.title}</h1>
              <p className="book-detail-author">{book.author}</p>
            </>
          )}
          {book.source === 'remarkable' && (
            <span className="badge badge-remarkable">Synced from reMarkable</span>
          )}
          {!editing && (
            <div className="detail-actions">
              <button className="text-button" onClick={() => setEditing({ title: book.title, author: book.author })}>
                Edit details
              </button>
              <button className="text-button" onClick={() => setMatching(true)}>
                {book.openLibraryId ? 'Change match' : 'Find on Open Library'}
              </button>
            </div>
          )}
          <div className="status-picker">
            {statusOptions.map((opt) => (
              <button
                key={opt.value}
                className={book.status === opt.value ? 'tab active' : 'tab'}
                onClick={() => updateStatus(opt.value)}
              >
                {opt.label}
              </button>
            ))}
          </div>
          {/* Keyed on the bookmark so its inputs reset when a status change moves it. */}
          <Bookmark
            key={`${book.currentPage}-${book.pageCount}`}
            book={book}
            onSaved={() => navigate('/')}
          />
          <button className="danger" onClick={remove}>
            Remove from collection
          </button>
        </div>
      </div>
      {matching && (
        <SearchModal
          shelf={[]}
          heading="Find on Open Library"
          initialQuery={[cleanFileName(book.title), book.author].filter(Boolean).join(' ')}
          chooseLabel="Use this"
          onClose={() => setMatching(false)}
          onChoose={async (result) => {
            // Takes the edition's details; the bookmark keeps its place.
            const { source: _source, ...details } = bookFromResult(result, book.source)
            setBook(await api.updateBook(book.id, details))
            setMatching(false)
          }}
        />
      )}
    </section>
  )
}
