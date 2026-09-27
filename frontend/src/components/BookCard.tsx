import { Link } from 'react-router-dom'
import type { Book } from '../api/types'
import { bookCover } from '../bookCover'
import { progressPercent } from '../progress'

const statusLabel: Record<Book['status'], string> = {
  want_to_read: 'Want to Read',
  reading: 'Reading',
  finished: 'Finished',
}

export function BookCard({ book }: { book: Book }) {
  const percent = progressPercent(book)
  return (
    <Link to={`/books/${book.id}`} className="book-card">
      <div className="book-card-cover">
        {bookCover(book) ? (
          <img src={bookCover(book)} alt="" />
        ) : (
          <span className="book-card-cover-fallback">{book.title.slice(0, 1)}</span>
        )}
      </div>
      {percent !== null && book.status === 'reading' && (
        <div className="progress book-card-progress">
          <div className="progress-fill" style={{ width: `${percent}%` }} />
        </div>
      )}
      <div className="book-card-title">{book.title}</div>
      <div className="book-card-author">{book.author}</div>
      <span className={`badge badge-${book.status}`}>
        {statusLabel[book.status]}
        {percent !== null && book.status === 'reading' && ` · ${percent}%`}
      </span>
    </Link>
  )
}
