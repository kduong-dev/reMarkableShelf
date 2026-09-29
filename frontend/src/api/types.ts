export type BookStatus = 'want_to_read' | 'reading' | 'finished'
export type BookSource = 'manual' | 'remarkable'

export interface Book {
  id: string
  title: string
  author: string
  isbn?: string
  coverUrl?: string
  status: BookStatus
  rating?: number
  source: BookSource
  openLibraryId?: string
  pageCount?: number
  currentPage?: number
  progressUpdatedAt?: string
  progressSource?: 'app' | 'remarkable'
  tabletCoverUrl?: string
  tabletPageCount?: number
  // tabletDevices names the tablets holding a copy linked to the book.
  tabletDevices?: string[]
  createdAt: string
  updatedAt: string
}

export interface Device {
  id: string
  name: string
  host: string
  lastSyncedAt?: string
  pairedAt?: string
  identityChanged?: boolean
}

export type RemarkableFileType = 'pdf' | 'epub' | 'notebook'

export interface RemarkableDocument {
  uuid: string
  deviceId: string
  title: string
  fileType: RemarkableFileType
  lastModified: string
  linkedBookId?: string
  currentPage?: number
  pageCount?: number
  positionUpdatedAt?: string
  bookTitle?: string
  bookAuthor?: string
  hasCover?: boolean
  // parentUuid is the folder holding the document on the tablet: unset at
  // the top of its library, "trash" in its trash.
  parentUuid?: string
  // notABook marks a PDF or EPUB the user says isn't a book, such as a
  // planner template, so it's never linked to one.
  notABook?: boolean
}

export interface RemarkableFolder {
  uuid: string
  title: string
  parentUuid?: string
}

export interface BookSearchResult {
  openLibraryId: string
  title: string
  author: string
  isbn?: string
  coverUrl?: string
  pageCount?: number
  firstPublishYear?: number
  averageRating?: number
  // downloadable is true for public domain works with a free ebook scan.
  downloadable?: boolean
}

export interface BookSearchResults {
  results: BookSearchResult[]
  total: number
}

export type SearchSort = '' | 'rating' | 'new' | 'old'

export interface BookSearch {
  q?: string
  subject?: string
  page?: number
  sort?: SearchSort
  language?: string
  publishedFrom?: number
  publishedTo?: number
}

export type BookFileFormat = 'epub' | 'pdf'

// Delivery is a book file copied to a tablet; loadedAt is unset until the
// tablet's reading app restarts and shows it.
export interface Delivery {
  deviceId: string
  documentUuid: string
  copiedAt: string
  loadedAt?: string
}

// BookFile is an ebook kept on the server, which sync copies to every paired
// tablet.
export interface BookFile {
  bookId: string
  format: BookFileFormat
  size: number
  source: 'open_library' | 'upload'
  savedAt: string
  deliveries: Delivery[]
}
