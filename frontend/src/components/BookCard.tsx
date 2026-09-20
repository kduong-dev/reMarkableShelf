import { Link } from 'react-router-dom'
import type { Book } from '../api/types'

const statusLabel: Record<Book['status'], string> = {
  want_to_read: 'Want to Read',
  reading: 'Reading',
  finished: 'Finished',
}

export function BookCard({ book }: { book: Book }) {
  return (
    <Link to={`/books/${book.id}`} className="book-card">
      <div className="book-card-cover">
        {book.coverUrl ? (
          <img src={book.coverUrl} alt="" />
        ) : (
          <span className="book-card-cover-fallback">{book.title.slice(0, 1)}</span>
        )}
      </div>
      <div className="book-card-title">{book.title}</div>
      <div className="book-card-author">{book.author}</div>
      <span className={`badge badge-${book.status}`}>{statusLabel[book.status]}</span>
    </Link>
  )
}
