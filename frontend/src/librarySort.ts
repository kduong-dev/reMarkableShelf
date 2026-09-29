import type { Book } from './api/types'
import { progressPercent } from './progress'

export type LibrarySort = 'updated' | 'added' | 'progress' | 'read' | 'title' | 'author'

export const librarySorts: Array<{ label: string; value: LibrarySort }> = [
  { label: 'Recently updated', value: 'updated' },
  { label: 'Recently added', value: 'added' },
  { label: 'Most read', value: 'progress' },
  { label: 'Recently read', value: 'read' },
  { label: 'Title', value: 'title' },
  { label: 'Author', value: 'author' },
]

export type TabletFilter = 'all' | 'on' | 'off'

export const tabletFilters: Array<{ label: string; value: TabletFilter }> = [
  { label: 'Anywhere', value: 'all' },
  { label: 'On reMarkable', value: 'on' },
  { label: 'Not on reMarkable', value: 'off' },
]

export function onTablet(book: Book): boolean {
  return (book.tabletDevices?.length ?? 0) > 0
}

const byText = (first: string, second: string) =>
  first.localeCompare(second, undefined, { numeric: true, sensitivity: 'base' })

// newestFirst orders timestamps latest first, with missing ones last.
const newestFirst = (first?: string, second?: string) => {
  if (!first || !second) return first ? -1 : second ? 1 : 0
  return second.localeCompare(first)
}

// compareBooks orders books by sort, falling back to title so ties keep a
// steady order. Books whose progress or reading time is unknown go last.
function compareBooks(sort: LibrarySort): (first: Book, second: Book) => number {
  const primary = {
    updated: (first: Book, second: Book) => newestFirst(first.updatedAt, second.updatedAt),
    added: (first: Book, second: Book) => newestFirst(first.createdAt, second.createdAt),
    progress: (first: Book, second: Book) => (progressPercent(second) ?? -1) - (progressPercent(first) ?? -1),
    read: (first: Book, second: Book) => newestFirst(first.progressUpdatedAt, second.progressUpdatedAt),
    title: () => 0,
    author: (first: Book, second: Book) => {
      if (!first.author || !second.author) return first.author ? -1 : second.author ? 1 : 0
      return byText(first.author, second.author)
    },
  }[sort]
  return (first, second) => primary(first, second) || byText(first.title, second.title)
}

export function sortBooks(books: Book[], sort: LibrarySort): Book[] {
  return [...books].sort(compareBooks(sort))
}
