import type { Book, BookSearchResult } from './api/types'

// bookFromResult is the book a search result adds to the library.
export function bookFromResult(result: BookSearchResult, source: Book['source']): Partial<Book> {
  return {
    title: result.title,
    author: result.author,
    isbn: result.isbn,
    coverUrl: result.coverUrl,
    pageCount: result.pageCount,
    openLibraryId: result.openLibraryId,
    source,
  }
}
