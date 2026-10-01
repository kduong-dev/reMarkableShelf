import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { EbookSource, EbookSourceKind } from '../api/types'

const kindLabel: Record<EbookSourceKind, string> = {
  internet_archive: 'Built in',
  plugin: 'Plugin',
  opds: 'OPDS catalog',
}

// Sources lists where ebooks are fetched from, in the order they're tried,
// and installs plugins and OPDS catalogs.
export function Sources() {
  const [sources, setSources] = useState<EbookSource[]>([])
  const [error, setError] = useState<string | null>(null)
  const [kind, setKind] = useState<EbookSourceKind>('opds')
  const [url, setUrl] = useState('')
  const [adding, setAdding] = useState(false)

  useEffect(() => {
    api
      .listSources()
      .then(setSources)
      .catch((err) => setError(err instanceof Error ? err.message : 'failed to load sources'))
  }, [])

  async function attempt(action: () => Promise<void>) {
    setError(null)
    try {
      await action()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'something went wrong')
    }
  }

  async function add(e: React.FormEvent) {
    e.preventDefault()
    if (!url.trim()) return
    setAdding(true)
    await attempt(async () => {
      const added = await api.addSource(kind, url.trim())
      setSources((prev) => [...prev, added])
      setUrl('')
    })
    setAdding(false)
  }

  function move(index: number, by: number) {
    const reordered = [...sources]
    const [moved] = reordered.splice(index, 1)
    reordered.splice(index + by, 0, moved)
    setSources(reordered)
    attempt(async () => setSources(await api.reorderSources(reordered.map((source) => source.id))))
  }

  function toggle(source: EbookSource) {
    attempt(async () => {
      const updated = await api.setSourceEnabled(source.id, !source.enabled)
      setSources((prev) => prev.map((each) => (each.id === updated.id ? updated : each)))
    })
  }

  function remove(source: EbookSource) {
    if (!confirm(`Remove ${source.name}? Ebooks already fetched from it stay on the server.`)) return
    attempt(async () => {
      await api.removeSource(source.id)
      setSources((prev) => prev.filter((each) => each.id !== source.id))
    })
  }

  return (
    <section>
      <h1>Sources</h1>
      <p className="page-intro">
        Where "Fetch ebook" looks for a book's file, top first. Each book's page also lets you pick a file from any
        source.
      </p>

      <ol className="source-list">
        {sources.map((source, index) => (
          <li key={source.id} className={source.enabled ? 'source-row' : 'source-row source-row-disabled'}>
            <span className="source-position">{index + 1}</span>
            <div className="source-info">
              <div className="source-name">{source.name}</div>
              <div className="source-detail">
                {kindLabel[source.kind]}
                {source.url && ` · ${source.url}`}
                {!source.enabled && ' · Disabled'}
              </div>
            </div>
            <div className="source-actions">
              <button
                className="icon-button"
                onClick={() => move(index, -1)}
                disabled={index === 0}
                aria-label={`Move ${source.name} up`}
              >
                &uarr;
              </button>
              <button
                className="icon-button"
                onClick={() => move(index, 1)}
                disabled={index === sources.length - 1}
                aria-label={`Move ${source.name} down`}
              >
                &darr;
              </button>
              <button className="text-button" onClick={() => toggle(source)}>
                {source.enabled ? 'Disable' : 'Enable'}
              </button>
              {source.kind !== 'internet_archive' && (
                <button className="text-button" onClick={() => remove(source)}>
                  Remove
                </button>
              )}
            </div>
          </li>
        ))}
      </ol>

      <h2>Add a source</h2>
      <form className="source-form" onSubmit={add}>
        <select value={kind} onChange={(e) => setKind(e.target.value as EbookSourceKind)} aria-label="Kind of source">
          <option value="opds">OPDS catalog</option>
          <option value="plugin">Plugin</option>
        </select>
        <input
          type="url"
          placeholder={kind === 'opds' ? 'Catalog URL, e.g. https://m.gutenberg.org/ebooks.opds/' : 'Plugin URL, e.g. http://folder-source:8090'}
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          aria-label="Source URL"
        />
        <button type="submit" className="primary" disabled={adding || !url.trim()}>
          {adding ? 'Checking…' : 'Add'}
        </button>
      </form>
      <p className="device-form-hint">
        {kind === 'opds'
          ? 'Any searchable OPDS catalog, such as Project Gutenberg or your own Calibre-Web or Kavita library.'
          : 'A plugin is a small web service answering the contract in the backend’s pkg/sourceplugin package; the server checks it answers before adding it.'}
      </p>
      {error && <p className="error">{error}</p>}
    </section>
  )
}
