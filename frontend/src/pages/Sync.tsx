import { useEffect, useState } from 'react'
import { api } from '../api/client'
import { bookFromResult } from '../bookFromResult'
import type { Book, Device, RemarkableDocument, RemarkableFolder } from '../api/types'
import { documentSearchQuery } from '../bookSearchQuery'
import { DeviceLibrary } from '../components/DeviceLibrary'
import { SearchModal } from '../components/SearchModal'

export function Sync() {
  const [devices, setDevices] = useState<Device[]>([])
  const [selected, setSelected] = useState<string>('')
  const [docs, setDocs] = useState<RemarkableDocument[]>([])
  const [folders, setFolders] = useState<RemarkableFolder[]>([])
  const [books, setBooks] = useState<Book[]>([])
  const [syncing, setSyncing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [newDevice, setNewDevice] = useState({ name: '', host: '', password: '' })
  const [registering, setRegistering] = useState(false)
  const [registerError, setRegisterError] = useState<string | null>(null)
  const [pairPassword, setPairPassword] = useState('')
  const [pairing, setPairing] = useState(false)
  const [acceptNewIdentity, setAcceptNewIdentity] = useState(false)
  const [renaming, setRenaming] = useState<string | null>(null)
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
    if (!selected) return
    api.listDocuments(selected).then(setDocs)
    api.listFolders(selected).then(setFolders)
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
      setFolders(await api.listFolders(selected))
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

  async function renameDevice(e: React.FormEvent) {
    e.preventDefault()
    if (!selected || !renaming?.trim()) return
    setError(null)
    try {
      const device = await api.renameDevice(selected, renaming)
      setDevices((prev) => prev.map((d) => (d.id === device.id ? device : d)))
      setRenaming(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed to rename device')
    }
  }

  async function removeDevice() {
    const device = devices.find((d) => d.id === selected)
    if (!device) return
    if (!confirm(`Remove ${device.name} (${device.host})? Its synced documents and their links go with it; your books and bookmarks stay.`)) return
    setError(null)
    try {
      await api.deleteDevice(device.id)
      const remaining = devices.filter((d) => d.id !== device.id)
      setDevices(remaining)
      setDocs([])
      setFolders([])
      setSelected(remaining[0]?.id ?? '')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed to remove device')
    }
  }

  async function unlink(doc: RemarkableDocument) {
    await api.unlinkDocument(selected, doc.uuid)
    setDocs((prev) => prev.map((d) => (d.uuid === doc.uuid ? { ...d, linkedBookId: undefined } : d)))
  }

  async function setNotABook(doc: RemarkableDocument, notABook: boolean) {
    setError(null)
    try {
      await api.setNotABook(selected, doc.uuid, notABook)
      setDocs((prev) =>
        prev.map((d) =>
          d.uuid === doc.uuid ? { ...d, notABook, linkedBookId: notABook ? undefined : d.linkedBookId } : d,
        ),
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed to mark the document')
    }
  }

  async function linkToExisting(doc: RemarkableDocument, bookId: string) {
    if (!bookId) return
    await api.linkDocument(selected, doc.uuid, bookId)
    // Linking clears a not-a-book mark on the server too.
    setDocs((prev) => prev.map((d) => (d.uuid === doc.uuid ? { ...d, linkedBookId: bookId, notABook: false } : d)))
  }

  const selectedDevice = devices.find((d) => d.id === selected)

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
          {renaming !== null ? (
            <form className="rename-form" onSubmit={renameDevice}>
              <input
                autoFocus
                aria-label="Device name"
                value={renaming}
                onChange={(e) => setRenaming(e.target.value)}
                onKeyDown={(e) => e.key === 'Escape' && setRenaming(null)}
              />
              <button type="submit" className="primary" disabled={!renaming.trim()}>
                Save
              </button>
              <button type="button" onClick={() => setRenaming(null)}>
                Cancel
              </button>
            </form>
          ) : (
            <select value={selected} onChange={(e) => setSelected(e.target.value)}>
              {devices.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name} ({d.host})
                </option>
              ))}
            </select>
          )}
          <button className="primary" onClick={sync} disabled={syncing || !selectedDevice?.pairedAt}>
            {syncing ? 'Syncing…' : 'Sync now'}
          </button>
          {devices.find((d) => d.id === selected)?.lastSyncedAt && (
            <span className="last-synced">
              Last synced {new Date(devices.find((d) => d.id === selected)!.lastSyncedAt!).toLocaleString()}
            </span>
          )}
          <div className="device-actions">
            <button className="text-button" onClick={() => setRenaming(selectedDevice?.name ?? '')}>
              Rename
            </button>
            <button className="text-button" onClick={removeDevice}>
              Remove device
            </button>
          </div>
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

      {selectedDevice && (
        <DeviceLibrary
          deviceId={selected}
          folders={folders}
          documents={docs}
          books={books}
          onUnlink={unlink}
          onLink={linkToExisting}
          onFind={setFinding}
          onAddAsIs={addAsNewBook}
          onSetNotABook={setNotABook}
        />
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
