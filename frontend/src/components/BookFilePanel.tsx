import { useEffect, useRef, useState } from 'react'
import { api } from '../api/client'
import type { Book, BookFile, Device } from '../api/types'
import { timeAgo } from '../timeAgo'

function fileSize(bytes: number): string {
  if (bytes >= 1_000_000) return `${(bytes / 1_000_000).toFixed(1)} MB`
  return `${Math.max(1, Math.round(bytes / 1_000))} KB`
}

// deliveryStatus says where the book's copy stands on a paired tablet.
function deliveryStatus(file: BookFile, device: Device): string {
  const delivery = file.deliveries.find((each) => each.deviceId === device.id)
  if (!delivery) return 'Copies on the next sync'
  if (!delivery.loadedAt) return 'Copied — press Sync now to show it'
  return 'On the tablet'
}

// BookFilePanel keeps an ebook of the book on the server, which sync copies
// to every paired tablet.
export function BookFilePanel({ book }: { book: Book }) {
  const [file, setFile] = useState<BookFile | null | undefined>(undefined)
  const [devices, setDevices] = useState<Device[]>([])
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const uploadInput = useRef<HTMLInputElement>(null)

  useEffect(() => {
    api
      .getBookFile(book.id)
      .then(setFile)
      .catch((err) => setError(err instanceof Error ? err.message : 'failed to load the book file'))
    api
      .listDevices()
      .then((listed) => setDevices(listed.filter((device) => device.pairedAt)))
      .catch(() => setDevices([]))
  }, [book.id])

  async function act(label: string, action: () => Promise<BookFile | null>) {
    setBusy(label)
    setError(null)
    try {
      setFile(await action())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'something went wrong')
    } finally {
      setBusy(null)
    }
  }

  function upload(e: React.ChangeEvent<HTMLInputElement>) {
    const chosen = e.target.files?.[0]
    e.target.value = ''
    if (chosen) act('Uploading…', () => api.uploadBookFile(book.id, chosen))
  }

  function remove() {
    if (!confirm('Remove this file from the server? Copies already on your tablets stay there.')) return
    act('Removing…', async () => {
      await api.deleteBookFile(book.id)
      return null
    })
  }

  if (file === undefined && !error) return null

  return (
    <div className="book-file">
      <div className="book-file-summary">
        <span className="book-file-label">Ebook</span>
        {file && (
          <span className="book-file-status">
            {file.format.toUpperCase()} · {fileSize(file.size)} ·{' '}
            {file.source === 'open_library' ? 'from Open Library' : 'uploaded'} {timeAgo(file.savedAt)}
          </span>
        )}
      </div>
      {file ? (
        devices.length > 0 ? (
          <ul className="book-file-devices">
            {devices.map((device) => (
              <li key={device.id}>
                <span>{device.name}</span>
                <span className="book-file-delivery">{deliveryStatus(file, device)}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="book-file-hint">Pair a tablet on the Sync page and sync will copy this book to it.</p>
        )
      ) : (
        <p className="book-file-hint">
          Keep an EPUB or PDF of this book on the server, and sync copies it to your tablets.
        </p>
      )}
      <div className="book-file-actions">
        {!file && book.openLibraryId && (
          <button onClick={() => act('Fetching…', () => api.fetchBookFile(book.id))} disabled={busy !== null}>
            {busy === 'Fetching…' ? busy : 'Fetch free ebook'}
          </button>
        )}
        <button onClick={() => uploadInput.current?.click()} disabled={busy !== null}>
          {busy === 'Uploading…' ? busy : file ? 'Replace file' : 'Upload EPUB or PDF'}
        </button>
        {file && (
          <>
            <a className="button" href={api.bookFileContentUrl(book.id)} download>
              Download
            </a>
            <button className="text-button" onClick={remove} disabled={busy !== null}>
              {busy === 'Removing…' ? busy : 'Remove from server'}
            </button>
          </>
        )}
        <input
          ref={uploadInput}
          type="file"
          accept=".epub,.pdf,application/epub+zip,application/pdf"
          hidden
          onChange={upload}
        />
      </div>
      {!file && book.openLibraryId && (
        <p className="book-file-hint">Free ebooks are only available for public domain books.</p>
      )}
      {error && <p className="error">{error}</p>}
    </div>
  )
}
