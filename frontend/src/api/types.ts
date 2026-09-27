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
  createdAt: string
  updatedAt: string
}

export interface Device {
  id: string
  name: string
  host: string
  lastSyncedAt?: string
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
