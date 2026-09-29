import type { Book, BookSearch, BookSearchResults, Device, RemarkableDocument } from './types'

async function send(path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(`/api${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    throw new Error(body?.message ?? `request to ${path} failed with status ${res.status}`)
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

  searchBooks: (search: BookSearch, signal?: AbortSignal) => {
    const params = new URLSearchParams()
    for (const [key, value] of Object.entries(search)) {
      if (value !== undefined && value !== '') params.set(key, String(value))
    }
    return request<BookSearchResults>(`/search/books?${params}`, { signal })
  },
  // downloadBook fetches a public domain work's ebook, with the file name the
  // server gives it.
  downloadBook: async (openLibraryId: string) => {
    const res = await send(`/search/books/${openLibraryId}/download`)
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
  unlinkDocument: (deviceId: string, uuid: string) =>
    request<void>(`/devices/${deviceId}/documents/${uuid}/link`, { method: 'DELETE' }),
  linkDocument: (deviceId: string, uuid: string, bookId: string) =>
    request<void>(`/devices/${deviceId}/documents/${uuid}/link`, {
      method: 'POST',
      body: JSON.stringify({ bookId }),
    }),
}
