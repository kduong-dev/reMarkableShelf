import type { RemarkableDocument } from './api/types'

// cleanFileName turns a file name into search words by dropping the
// extension, bracketed notes such as publishers or "(z-lib.org)", and
// separators, e.g. "Dale Carnegie - How to Win Friends (Veridian Digital
// Press).pdf" becomes "Dale Carnegie How to Win Friends".
export function cleanFileName(name: string): string {
  return name
    .replace(/\.(pdf|epub)$/i, '')
    .replace(/[([{][^)\]}]*[)\]}]/g, ' ')
    .replace(/[_–—]+|\s-\s|\s-$|^-\s/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

// documentSearchQuery is what to search Open Library for to find a tablet
// document's book: the title and author the tablet read from the file when
// it could, otherwise its cleaned-up file name.
export function documentSearchQuery(document: RemarkableDocument): string {
  if (document.bookTitle) return [document.bookTitle, document.bookAuthor].filter(Boolean).join(' ')
  return cleanFileName(document.title)
}
