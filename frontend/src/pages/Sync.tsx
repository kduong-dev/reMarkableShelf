import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { Book, Device, RemarkableDocument } from '../api/types'

export function Sync() {
  const [devices, setDevices] = useState<Device[]>([])
  const [selected, setSelected] = useState<string>('')
  const [docs, setDocs] = useState<RemarkableDocument[]>([])
  const [books, setBooks] = useState<Book[]>([])
  const [syncing, setSyncing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [newDevice, setNewDevice] = useState({ name: '', host: '' })

  useEffect(() => {
    api.listDevices().then((list) => {
      setDevices(list)
      if (list.length > 0) setSelected(list[0].id)
    })
    api.listBooks().then(setBooks)
  }, [])

  useEffect(() => {
    if (selected) api.listDocuments(selected).then(setDocs)
  }, [selected])

  async function addDevice(e: React.FormEvent) {
    e.preventDefault()
    if (!newDevice.name.trim() || !newDevice.host.trim()) return
    const device = await api.createDevice(newDevice)
    setDevices((prev) => [...prev, device])
    setSelected(device.id)
    setNewDevice({ name: '', host: '' })
  }

  async function sync() {
    if (!selected) return
    setSyncing(true)
    setError(null)
    try {
      const synced = await api.syncDevice(selected)
      setDocs(synced)
      setDevices((prev) =>
        prev.map((d) => (d.id === selected ? { ...d, lastSyncedAt: new Date().toISOString() } : d)),
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'sync failed')
    } finally {
      setSyncing(false)
    }
  }

  async function addAsNewBook(doc: RemarkableDocument) {
    const book = await api.createBook({ title: doc.title, author: '', source: 'remarkable' })
    setBooks((prev) => [book, ...prev])
    await api.linkDocument(selected, doc.uuid, book.id)
    setDocs((prev) => prev.map((d) => (d.uuid === doc.uuid ? { ...d, linkedBookId: book.id } : d)))
  }

  async function linkToExisting(doc: RemarkableDocument, bookId: string) {
    if (!bookId) return
    await api.linkDocument(selected, doc.uuid, bookId)
    setDocs((prev) => prev.map((d) => (d.uuid === doc.uuid ? { ...d, linkedBookId: bookId } : d)))
  }

  const bookDocs = docs.filter((d) => d.fileType !== 'notebook')
  const noteDocs = docs.filter((d) => d.fileType === 'notebook')

  return (
    <section>
      <h1>Sync</h1>

      <form className="device-form" onSubmit={addDevice}>
        <input
          placeholder="Device name (e.g. My Paper Pro)"
          value={newDevice.name}
          onChange={(e) => setNewDevice((d) => ({ ...d, name: e.target.value }))}
        />
        <input
          placeholder="Host / reserved IP"
          value={newDevice.host}
          onChange={(e) => setNewDevice((d) => ({ ...d, host: e.target.value }))}
        />
        <button type="submit">Register device</button>
      </form>

      {devices.length === 0 && (
        <p className="empty-state">
          No devices registered yet. Register your tablet's host above (see the network setup
          guide in the repo README).
        </p>
      )}

      {devices.length > 0 && (
        <div className="toolbar">
          <select value={selected} onChange={(e) => setSelected(e.target.value)}>
            {devices.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name} ({d.host})
              </option>
            ))}
          </select>
          <button onClick={sync} disabled={syncing}>
            {syncing ? 'Syncing…' : 'Sync now'}
          </button>
          {devices.find((d) => d.id === selected)?.lastSyncedAt && (
            <span className="last-synced">
              Last synced {new Date(devices.find((d) => d.id === selected)!.lastSyncedAt!).toLocaleString()}
            </span>
          )}
        </div>
      )}

      {error && <p className="error">{error}</p>}

      {bookDocs.length > 0 && (
        <>
          <h2>Books &amp; documents ({bookDocs.length})</h2>
          <ul className="doc-list">
            {bookDocs.map((doc) => (
              <li key={doc.uuid} className="doc-row">
                <span className="doc-title">{doc.title}</span>
                <span className={`badge badge-filetype`}>{doc.fileType}</span>
                {doc.linkedBookId ? (
                  <span className="linked">Linked to collection</span>
                ) : (
                  <div className="doc-actions">
                    <select onChange={(e) => linkToExisting(doc, e.target.value)} defaultValue="">
                      <option value="" disabled>
                        Link to existing book…
                      </option>
                      {books.map((b) => (
                        <option key={b.id} value={b.id}>
                          {b.title}
                        </option>
                      ))}
                    </select>
                    <button onClick={() => addAsNewBook(doc)}>Add as new book</button>
                  </div>
                )}
              </li>
            ))}
          </ul>
        </>
      )}

      {noteDocs.length > 0 && (
        <>
          <h2>Notebooks &amp; sketches ({noteDocs.length})</h2>
          <ul className="doc-list">
            {noteDocs.map((doc) => (
              <li key={doc.uuid} className="doc-row">
                <span className="doc-title">{doc.title}</span>
                <span className="badge badge-filetype">notebook</span>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  )
}
