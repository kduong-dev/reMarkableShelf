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
