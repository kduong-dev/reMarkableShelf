import { useState } from 'react'
import { api } from '../api/client'
import type { Book, BookStatus } from '../api/types'
import { progressPercent } from '../progress'

// statusForPage moves a book along the shelf as its bookmark moves: starting
// it marks it Reading, reaching the last page marks it Finished, and moving
// back from the end reopens it.
function statusForPage(book: Book, currentPage: number, pageCount?: number): BookStatus {
  if (pageCount && currentPage >= pageCount) return 'finished'
  if (currentPage > 0 && book.status !== 'reading') return 'reading'
  return book.status
}

export function Bookmark({ book, onSaved }: { book: Book; onSaved: () => void }) {
  const [page, setPage] = useState(String(book.currentPage ?? ''))
  const [total, setTotal] = useState(String(book.pageCount ?? ''))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const percent = progressPercent(book)
  const pageNumber = Number(page)
  const totalNumber = total ? Number(total) : undefined
  const unchanged = page === String(book.currentPage ?? '') && total === String(book.pageCount ?? '')
  const invalid =
    page === '' ||
    !Number.isInteger(pageNumber) ||
    pageNumber < 0 ||
    (totalNumber !== undefined && (!Number.isInteger(totalNumber) || totalNumber < 1 || pageNumber > totalNumber))

  async function save(e: React.FormEvent) {
    e.preventDefault()
    if (invalid) return
    setSaving(true)
    setError(null)
    try {
      await api.updateBook(book.id, {
        currentPage: pageNumber,
        pageCount: totalNumber,
        status: statusForPage(book, pageNumber, totalNumber),
      })
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed to save bookmark')
      setSaving(false)
    }
  }

  return (
    <div className="bookmark">
      <div className="bookmark-summary">
        <span className="bookmark-label">Bookmark</span>
        <span className="bookmark-status">
          {book.currentPage
            ? `Page ${book.currentPage}${book.pageCount ? ` of ${book.pageCount}` : ''}`
            : 'Not started'}
          {percent !== null && ` · ${percent}%`}
        </span>
      </div>
      {book.pageCount ? (
        <div className="progress" role="progressbar" aria-valuenow={percent ?? 0} aria-valuemin={0} aria-valuemax={100}>
          <div className="progress-fill" style={{ width: `${percent ?? 0}%` }} />
        </div>
      ) : null}
      <form className="bookmark-form" onSubmit={save}>
        <label>
          On page
          <input
            type="number"
            inputMode="numeric"
            min={0}
            max={totalNumber}
            value={page}
            onChange={(e) => setPage(e.target.value)}
          />
        </label>
        <label>
          of
          <input
            type="number"
            inputMode="numeric"
            min={1}
            placeholder="?"
            value={total}
            onChange={(e) => setTotal(e.target.value)}
          />
        </label>
        <button type="submit" className="primary" disabled={saving || invalid || unchanged}>
          {saving ? 'Saving…' : 'Save'}
        </button>
      </form>
      {!book.pageCount && (
        <p className="bookmark-hint">Add the page count to track your progress as a percentage.</p>
      )}
      {error && <p className="error">{error}</p>}
    </div>
  )
}
