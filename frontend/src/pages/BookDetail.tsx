import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import { Bookmark } from '../components/Bookmark'
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

  useEffect(() => {
    if (!id) return
    api
      .getBook(id)
      .then(setBook)
      .catch((err) => setError(err instanceof Error ? err.message : 'failed to load book'))
  }, [id])

  async function updateStatus(status: BookStatus) {
    if (!book) return
    // Finishing a book of known length moves the bookmark to its last page.
    const currentPage = status === 'finished' && book.pageCount ? book.pageCount : undefined
    const updated = await api.updateBook(book.id, { status, currentPage })
    setBook(updated)
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
        <div className="book-detail-cover">
          {book.coverUrl ? <img src={book.coverUrl} alt="" /> : book.title.slice(0, 1)}
        </div>
        <div>
          <h1>{book.title}</h1>
          <p className="book-detail-author">{book.author}</p>
          {book.source === 'remarkable' && (
            <span className="badge badge-remarkable">Synced from reMarkable</span>
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
          <Bookmark key={`${book.currentPage}-${book.pageCount}`} book={book} onChange={setBook} />
          <button className="danger" onClick={remove}>
            Remove from collection
          </button>
        </div>
      </div>
    </section>
  )
}
