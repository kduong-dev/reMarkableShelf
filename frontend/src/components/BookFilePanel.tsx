import { useEffect, useRef, useState } from 'react'
import { api } from '../api/client'
import type { Book, BookFile, Device, SourceEbook, SourceEbooks } from '../api/types'
import { timeAgo } from '../timeAgo'

function fileSize(bytes: number): string {
  if (bytes >= 1_000_000) return `${(bytes / 1_000_000).toFixed(1)} MB`
  return `${Math.max(1, Math.round(bytes / 1_000))} KB`
}

// fileOrigin says where a saved file came from.
function fileOrigin(file: BookFile): string {
  if (file.source === 'upload') return 'uploaded'
  if (file.source === 'open_library') return 'from Open Library'
  return `from ${file.sourceName ?? 'a source'}`
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
  // choices is what each source found of the book, once asked for; null
  // while they're being asked.
  const [choices, setChoices] = useState<SourceEbooks[] | null | undefined>(undefined)

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

  async function chooseSource() {
    setChoices(null)
    setError(null)
    try {
      setChoices(await api.listBookEbooks(book.id))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed to ask the sources')
      setChoices(undefined)
    }
  }

  function fetchChoice(sourceId: string, ebook: SourceEbook) {
    act('Fetching…', async () => {
      const fetched = await api.fetchBookFile(book.id, { sourceId, ebookId: ebook.id })
      setChoices(undefined)
      return fetched
    })
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
            {file.format.toUpperCase()} · {fileSize(file.size)} · {fileOrigin(file)} {timeAgo(file.savedAt)}
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
        {!file && (
          <button onClick={() => act('Fetching…', () => api.fetchBookFile(book.id))} disabled={busy !== null}>
            {busy === 'Fetching…' && !choices ? busy : 'Fetch ebook'}
          </button>
        )}
        <button className="text-button" onClick={chooseSource} disabled={busy !== null || choices === null}>
          {choices === null ? 'Asking sources…' : file ? 'Fetch from a source…' : 'Choose source…'}
        </button>
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
      {!file && !choices && (
        <p className="book-file-hint">
          Fetch ebook asks your sources in order, set on the Sources page; the Internet Archive has public domain
          books matched on Open Library.
        </p>
      )}
      {choices && (
        <ul className="source-choices">
          {choices.length === 0 && <li className="book-file-hint">All your sources are disabled.</li>}
          {choices.map(({ source, ebooks, error: sourceError }) => (
            <li key={source.id}>
              <strong>{source.name}</strong>
              {sourceError ? (
                <div className="source-choice-detail">{sourceError}</div>
              ) : ebooks.length === 0 ? (
                <div className="source-choice-detail">Doesn't have this book</div>
              ) : (
                ebooks.map((ebook) => (
                  <div key={ebook.id} className="source-choice">
                    <span className="source-choice-detail">
                      {[ebook.format.toUpperCase(), ebook.size ? fileSize(ebook.size) : null, ebook.description || ebook.title]
                        .filter(Boolean)
                        .join(' · ')}
                    </span>
                    <button onClick={() => fetchChoice(source.id, ebook)} disabled={busy !== null}>
                      {file ? 'Replace with this' : 'Fetch this'}
                    </button>
                  </div>
                ))
              )}
            </li>
          ))}
        </ul>
      )}
      {error && <p className="error">{error}</p>}
    </div>
  )
}
