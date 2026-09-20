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
  googleBooksId?: string
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
}

export interface BookSearchResult {
  googleBooksId: string
  title: string
  author: string
  isbn?: string
  coverUrl?: string
}
