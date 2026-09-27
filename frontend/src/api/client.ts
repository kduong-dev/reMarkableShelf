import type { Book, BookSearch, BookSearchResult, Device, RemarkableDocument } from './types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    throw new Error(body?.message ?? `request to ${path} failed with status ${res.status}`)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

// requestList wraps request for endpoints that return a JSON array, and
// normalizes a `null` body (Go's encoding/json for a nil/empty slice) to
// `[]` so callers can always safely .filter/.map the result.
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
    return requestList<BookSearchResult>(`/search/books?${params}`, { signal })
  },

  listDevices: () => requestList<Device>('/devices'),
  createDevice: (device: Partial<Device>) =>
    request<Device>('/devices', { method: 'POST', body: JSON.stringify(device) }),
  syncDevice: (id: string) => requestList<RemarkableDocument>(`/devices/${id}/sync`, { method: 'POST' }),
  listDocuments: (deviceId: string) => requestList<RemarkableDocument>(`/devices/${deviceId}/documents`),
  linkDocument: (deviceId: string, uuid: string, bookId: string) =>
    request<void>(`/devices/${deviceId}/documents/${uuid}/link`, {
      method: 'POST',
      body: JSON.stringify({ bookId }),
    }),
}
