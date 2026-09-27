import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { bookFromResult } from '../bookFromResult'
import type { Book, Device, RemarkableDocument } from '../api/types'
import { documentSearchQuery } from '../bookSearchQuery'
import { SearchModal } from '../components/SearchModal'

export function Sync() {
  const [devices, setDevices] = useState<Device[]>([])
  const [selected, setSelected] = useState<string>('')
  const [docs, setDocs] = useState<RemarkableDocument[]>([])
  const [books, setBooks] = useState<Book[]>([])
  const [syncing, setSyncing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [newDevice, setNewDevice] = useState({ name: '', host: '', password: '' })
  const [registering, setRegistering] = useState(false)
  const [registerError, setRegisterError] = useState<string | null>(null)
  const [pairPassword, setPairPassword] = useState('')
  const [pairing, setPairing] = useState(false)
  const [acceptNewIdentity, setAcceptNewIdentity] = useState(false)
  // The tablet document being matched to a book on Open Library.
  const [finding, setFinding] = useState<RemarkableDocument | null>(null)

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
    if (!newDevice.name.trim() || !newDevice.host.trim() || !newDevice.password) return
    setRegistering(true)
    setRegisterError(null)
    try {
      const device = await api.createDevice(newDevice)
      setDevices((prev) => [...prev, device])
      setSelected(device.id)
      setNewDevice({ name: '', host: '', password: '' })
    } catch (err) {
      setRegisterError(err instanceof Error ? err.message : 'failed to pair the tablet')
    } finally {
      setRegistering(false)
    }
  }

  async function pair(e: React.FormEvent) {
    e.preventDefault()
    if (!selected || !pairPassword) return
    setPairing(true)
    setError(null)
    try {
      const device = await api.pairDevice(selected, pairPassword, acceptNewIdentity)
      setDevices((prev) => prev.map((d) => (d.id === device.id ? device : d)))
      setPairPassword('')
      setAcceptNewIdentity(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed to pair the tablet')
      // A refused identity change flags the device, so pick up its state.
      api.listDevices().then(setDevices)
    } finally {
      setPairing(false)
    }
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
      // A rejected key unpairs the device, so pick up its new state.
      api.listDevices().then(setDevices)
    } finally {
      setSyncing(false)
    }
  }

  // addAsNewBook adds a document Open Library doesn't know, such as a
  // personal PDF, under its file name.
  async function addAsNewBook(doc: RemarkableDocument) {
    const book = await api.createBook({ title: doc.title, author: '', source: 'remarkable' })
    setBooks((prev) => [book, ...prev])
    await linkToExisting(doc, book.id)
  }

  async function unlink(doc: RemarkableDocument) {
    await api.unlinkDocument(selected, doc.uuid)
    setDocs((prev) => prev.map((d) => (d.uuid === doc.uuid ? { ...d, linkedBookId: undefined } : d)))
  }

  async function linkToExisting(doc: RemarkableDocument, bookId: string) {
    if (!bookId) return
    await api.linkDocument(selected, doc.uuid, bookId)
    setDocs((prev) => prev.map((d) => (d.uuid === doc.uuid ? { ...d, linkedBookId: bookId } : d)))
  }

  const selectedDevice = devices.find((d) => d.id === selected)
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
        <input
          type="password"
          autoComplete="off"
          placeholder="Tablet password"
          value={newDevice.password}
          onChange={(e) => setNewDevice((d) => ({ ...d, password: e.target.value }))}
        />
        <button type="submit" className="primary" disabled={registering}>
          {registering ? 'Pairing…' : 'Pair device'}
        </button>
        <p className="device-form-hint">
          The password is under Settings → Help → Copyrights and licenses on the tablet. It's used
          once to install this server's key, then discarded.
        </p>
      </form>
      {registerError && <p className="error">{registerError}</p>}

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
          <button className="primary" onClick={sync} disabled={syncing || !selectedDevice?.pairedAt}>
            {syncing ? 'Syncing…' : 'Sync now'}
          </button>
          {devices.find((d) => d.id === selected)?.lastSyncedAt && (
            <span className="last-synced">
              Last synced {new Date(devices.find((d) => d.id === selected)!.lastSyncedAt!).toLocaleString()}
            </span>
          )}
        </div>
      )}

      {selectedDevice && !selectedDevice.pairedAt && (
        <form className={selectedDevice.identityChanged ? 'pair-form pair-form-warning' : 'pair-form'} onSubmit={pair}>
          {selectedDevice.identityChanged ? (
            <>
              <p>
                <strong>The tablet at {selectedDevice.host} identified itself differently</strong> than
                when {selectedDevice.name} was paired, so syncing has stopped. If you factory-reset or
                replaced the tablet, confirm below and pair it again. If you didn't, don't enter the
                password: something on your network may be impersonating it.
              </p>
              <label className="pair-confirm">
                <input
                  type="checkbox"
                  checked={acceptNewIdentity}
                  onChange={(e) => setAcceptNewIdentity(e.target.checked)}
                />
                I factory-reset or replaced this tablet
              </label>
            </>
          ) : (
            <p>
              <strong>{selectedDevice.name}</strong> isn't paired with this server, so it can't sync.
              Enter the tablet's password to pair it.
            </p>
          )}
          <input
            type="password"
            autoComplete="off"
            placeholder="Tablet password"
            value={pairPassword}
            onChange={(e) => setPairPassword(e.target.value)}
          />
          <button
            type="submit"
            className="primary"
            disabled={pairing || !pairPassword || (selectedDevice.identityChanged && !acceptNewIdentity)}
          >
            {pairing ? 'Pairing…' : 'Pair'}
          </button>
        </form>
      )}

      {error && <p className="error">{error}</p>}

      {bookDocs.length > 0 && (
        <>
          <h2>Books &amp; documents ({bookDocs.length})</h2>
          <ul className="doc-list">
            {bookDocs.map((doc) => (
              <li key={doc.uuid} className="doc-row">
                <span className="doc-cover">
                  {doc.hasCover && (
                    <img src={`/api/devices/${selected}/documents/${doc.uuid}/cover`} alt="" loading="lazy" />
                  )}
                </span>
                <span className="doc-title">{doc.title}</span>
                {doc.currentPage && doc.pageCount && (
                  <span className="doc-position">
                    p. {doc.currentPage} / {doc.pageCount}
                  </span>
                )}
                <span className={`badge badge-filetype`}>{doc.fileType}</span>
                {doc.linkedBookId ? (
                  <span className="linked">
                    Linked to{' '}
                    <Link to={`/books/${doc.linkedBookId}`}>
                      {books.find((b) => b.id === doc.linkedBookId)?.title ?? 'a book'}
                    </Link>
                    <button className="text-button" onClick={() => unlink(doc)}>
                      Unlink
                    </button>
                  </span>
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
                    <button className="primary" onClick={() => setFinding(doc)}>
                      Find book
                    </button>
                    <button className="text-button" onClick={() => addAsNewBook(doc)}>
                      Add as is
                    </button>
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
                <span className="doc-cover">
                  {doc.hasCover && (
                    <img src={`/api/devices/${selected}/documents/${doc.uuid}/cover`} alt="" loading="lazy" />
                  )}
                </span>
                <span className="doc-title">{doc.title}</span>
                <span className="badge badge-filetype">notebook</span>
              </li>
            ))}
          </ul>
        </>
      )}

      {finding && (
        <SearchModal
          shelf={books}
          heading="Find this book"
          initialQuery={documentSearchQuery(finding)}
          chooseLabel="Add & link"
          onClose={() => setFinding(null)}
          onChoose={async (result) => {
            const book = await api.createBook(bookFromResult(result, 'remarkable'))
            setBooks((prev) => [book, ...prev])
            await linkToExisting(finding, book.id)
            setFinding(null)
          }}
          onChooseShelved={async (book) => {
            await linkToExisting(finding, book.id)
            setFinding(null)
          }}
        />
      )}
    </section>
  )
}
