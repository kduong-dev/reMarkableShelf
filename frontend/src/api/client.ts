import type { Book, BookSearchResult, Device, RemarkableDocument } from './types'

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

export const api = {
  listBooks: () => request<Book[]>('/books'),
  getBook: (id: string) => request<Book>(`/books/${id}`),
  createBook: (book: Partial<Book>) =>
    request<Book>('/books', { method: 'POST', body: JSON.stringify(book) }),
  updateBook: (id: string, patch: Partial<Book>) =>
    request<Book>(`/books/${id}`, { method: 'PUT', body: JSON.stringify(patch) }),
  deleteBook: (id: string) => request<void>(`/books/${id}`, { method: 'DELETE' }),

  searchBooks: (q: string) =>
    request<BookSearchResult[]>(`/search/books?q=${encodeURIComponent(q)}`),

  listDevices: () => request<Device[]>('/devices'),
  createDevice: (device: Partial<Device>) =>
    request<Device>('/devices', { method: 'POST', body: JSON.stringify(device) }),
  syncDevice: (id: string) =>
    request<RemarkableDocument[]>(`/devices/${id}/sync`, { method: 'POST' }),
  listDocuments: (deviceId: string) =>
    request<RemarkableDocument[]>(`/devices/${deviceId}/documents`),
  linkDocument: (deviceId: string, uuid: string, bookId: string) =>
    request<void>(`/devices/${deviceId}/documents/${uuid}/link`, {
      method: 'POST',
      body: JSON.stringify({ bookId }),
    }),
}
