import type { Book } from './api/types'

// progressPercent is how far through a book the bookmark sits, or null when
// the book's length is unknown or it hasn't been started.
export function progressPercent(book: Book): number | null {
  if (!book.pageCount || !book.currentPage) return null
  return Math.round((book.currentPage / book.pageCount) * 100)
}
