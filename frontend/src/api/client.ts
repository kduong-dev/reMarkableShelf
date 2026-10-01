import type {
  Book,
  BookFile,
  BookSearch,
  BookSearchResults,
  Device,
  EbookSource,
  EbookSourceKind,
  RemarkableDocument,
  RemarkableFolder,
  SourceEbooks,
} from './types'

// ApiError is a failed request, with its status so callers can tell a
// missing resource from a failure.
export class ApiError extends Error {
  readonly status: number

  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

async function send(path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(`/api${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    throw new ApiError(body?.message ?? `request to ${path} failed with status ${res.status}`, res.status)
  }
  return res
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await send(path, init)
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

// requestList wraps request for endpoints that return a JSON array, and
// normalizes a `null` body (Go's encoding/json for a nil/empty slice) to
// `[]` so callers can always safely .filter/.map the result.
// fileNameOf reads the file name from a Content-Disposition header, preferring
// the UTF-8 filename* that non-ASCII names are sent as.
function fileNameOf(contentDisposition: string | null): string {
  const encoded = contentDisposition?.match(/filename\*=utf-8''([^;]+)/i)
  if (encoded) return decodeURIComponent(encoded[1])
  return contentDisposition?.match(/filename="([^"]+)"/)?.[1] ?? 'book'
}

async function requestList<T>(path: string, init?: RequestInit): Promise<T[]> {
  return (await request<T[] | null>(path, init)) ?? []
}

export const api = {
  listBooks: () => requestList<Book>('/books'),
  getBook: (id: string) => request<Book>(`/books/${id}`),
  createBook: (book: Partial<Book>) =>
    request<Book>('/books', { method: 'POST', body: JSON.stringify(book) }),
  updateBook: (id: string, patch: Partial<Book>) =>
    request<Book>(`/books/${id}`, { method: 'PUT', body: JSON.stringify(patch) }),
  deleteBook: (id: string) => request<void>(`/books/${id}`, { method: 'DELETE' }),
  // getBookFile resolves to null for a book with no file saved on the server.
  getBookFile: async (bookId: string) => {
    try {
      return await request<BookFile>(`/books/${bookId}/file`)
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) return null
      throw err
    }
  },
  uploadBookFile: (bookId: string, file: File) =>
    request<BookFile>(`/books/${bookId}/file`, {
      method: 'PUT',
      headers: { 'Content-Type': file.type || 'application/octet-stream' },
      body: file,
    }),
  // fetchBookFile saves an ebook of the book on the server: the chosen one,
  // else the preferred ebook of the first source that has the book.
  fetchBookFile: (bookId: string, choice?: { sourceId: string; ebookId: string }) =>
    request<BookFile>(`/books/${bookId}/file/fetch`, {
      method: 'POST',
      body: choice ? JSON.stringify(choice) : undefined,
    }),
  // listBookEbooks asks every enabled source for ebooks of the book.
  listBookEbooks: (bookId: string) => requestList<SourceEbooks>(`/books/${bookId}/ebooks`),

  listSources: () => requestList<EbookSource>('/sources'),
  // addSource installs a plugin or OPDS catalog, named by itself.
  addSource: (kind: EbookSourceKind, url: string) =>
    request<EbookSource>('/sources', { method: 'POST', body: JSON.stringify({ kind, url }) }),
  setSourceEnabled: (id: string, enabled: boolean) =>
    request<EbookSource>(`/sources/${id}`, { method: 'PUT', body: JSON.stringify({ enabled }) }),
  reorderSources: (ids: string[]) =>
    requestList<EbookSource>('/sources/order', { method: 'PUT', body: JSON.stringify({ ids }) }),
  removeSource: (id: string) => request<void>(`/sources/${id}`, { method: 'DELETE' }),
  deleteBookFile: (bookId: string) => request<void>(`/books/${bookId}/file`, { method: 'DELETE' }),
  bookFileContentUrl: (bookId: string) => `/api/books/${bookId}/file/content`,

  searchBooks: (search: BookSearch, signal?: AbortSignal) => {
    const params = new URLSearchParams()
    for (const [key, value] of Object.entries(search)) {
      if (value !== undefined && value !== '') params.set(key, String(value))
    }
    return request<BookSearchResults>(`/search/books?${params}`, { signal })
  },
  // downloadBook fetches an ebook of a search result from the first source
  // that has one, with the file name the server gives it. The title and
  // author let sources other than the Internet Archive look for it.
  downloadBook: async (result: { openLibraryId: string; title: string; author: string }) => {
    const params = new URLSearchParams({ title: result.title, author: result.author })
    const res = await send(`/search/books/${result.openLibraryId}/download?${params}`)
    return { file: await res.blob(), fileName: fileNameOf(res.headers.get('Content-Disposition')) }
  },

  listDevices: () => requestList<Device>('/devices'),
  // The password pairs the tablet with the server's key; it isn't stored.
  createDevice: (device: { name: string; host: string; password: string }) =>
    request<Device>('/devices', { method: 'POST', body: JSON.stringify(device) }),
  renameDevice: (id: string, name: string) =>
    request<Device>(`/devices/${id}`, { method: 'PUT', body: JSON.stringify({ name }) }),
  deleteDevice: (id: string) => request<void>(`/devices/${id}`, { method: 'DELETE' }),
  // acceptNewIdentity confirms a tablet whose identity changed was reset or
  // replaced; without it, pairing refuses a changed identity.
  pairDevice: (id: string, password: string, acceptNewIdentity = false) =>
    request<Device>(`/devices/${id}/pair`, {
      method: 'POST',
      body: JSON.stringify({ password, acceptNewIdentity }),
    }),
  syncDevice: (id: string) => requestList<RemarkableDocument>(`/devices/${id}/sync`, { method: 'POST' }),
  listDocuments: (deviceId: string) => requestList<RemarkableDocument>(`/devices/${deviceId}/documents`),
  listFolders: (deviceId: string) => requestList<RemarkableFolder>(`/devices/${deviceId}/folders`),
  unlinkDocument: (deviceId: string, uuid: string) =>
    request<void>(`/devices/${deviceId}/documents/${uuid}/link`, { method: 'DELETE' }),
  // setNotABook marks a tablet document as not a book, which unlinks it, or
  // clears the mark.
  setNotABook: (deviceId: string, uuid: string, notABook: boolean) =>
    request<void>(`/devices/${deviceId}/documents/${uuid}/not-a-book`, { method: notABook ? 'POST' : 'DELETE' }),
  linkDocument: (deviceId: string, uuid: string, bookId: string) =>
    request<void>(`/devices/${deviceId}/documents/${uuid}/link`, {
      method: 'POST',
      body: JSON.stringify({ bookId }),
    }),
}
