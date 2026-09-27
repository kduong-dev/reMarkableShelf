import type { Book } from './api/types'

// bookCover is the cover to show for a book: its own, else the one synced
// from a linked tablet document.
export function bookCover(book: Book): string | undefined {
  return book.coverUrl || book.tabletCoverUrl
}
