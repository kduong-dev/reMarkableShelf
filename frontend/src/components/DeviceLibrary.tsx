import { useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import type { Book, RemarkableDocument, RemarkableFolder } from '../api/types'
import { rootFolder, tabletLibrary, trashFolder } from '../tabletLibrary'

type Sort = 'name' | 'recent'

const byTitle = (first: { title: string }, second: { title: string }) =>
  first.title.localeCompare(second.title, undefined, { numeric: true, sensitivity: 'base' })

// DeviceLibrary browses a tablet's library as the tablet shows it: folders
// first, then the documents in the open folder, which the URL keeps so the
// back button leaves a folder.
export function DeviceLibrary({
  deviceId,
  folders,
  documents,
  books,
  onUnlink,
  onLink,
  onFind,
  onAddAsIs,
}: {
  deviceId: string
  folders: RemarkableFolder[]
  documents: RemarkableDocument[]
  books: Book[]
  onUnlink: (document: RemarkableDocument) => void
  onLink: (document: RemarkableDocument, bookId: string) => void
  onFind: (document: RemarkableDocument) => void
  onAddAsIs: (document: RemarkableDocument) => void
}) {
  const [searchParams, setSearchParams] = useSearchParams()
  const [sort, setSort] = useState<Sort>('name')
  const library = useMemo(() => tabletLibrary(folders, documents), [folders, documents])
  const requested = searchParams.get('folder') ?? rootFolder
  // A folder deleted on the tablet since the link was made falls back to the top.
  const folderId = library.exists(requested) ? requested : rootFolder

  function open(id: string) {
    setSearchParams((params) => {
      if (id === rootFolder) params.delete('folder')
      else params.set('folder', id)
      return params
    })
  }

  const childFolders = [...library.foldersIn(folderId)].sort(byTitle)
  const childDocuments = [...library.documentsIn(folderId)].sort(
    sort === 'name' ? byTitle : (first, second) => second.lastModified.localeCompare(first.lastModified),
  )
  const itemCount = (id: string) => library.foldersIn(id).length + library.documentsIn(id).length
  const trashCount = itemCount(trashFolder)
  const inTrash = folderId === trashFolder || library.path(folderId)[0]?.parentUuid === trashFolder

  return (
    <div className="device-library">
      <div className="library-toolbar">
        <nav className="breadcrumbs" aria-label="Folder">
          <button className="text-button" onClick={() => open(rootFolder)} aria-current={folderId === rootFolder}>
            My files
          </button>
          {inTrash && (
            <>
              <span aria-hidden="true">/</span>
              <button className="text-button" onClick={() => open(trashFolder)} aria-current={folderId === trashFolder}>
                Trash
              </button>
            </>
          )}
          {library.path(folderId).map((folder) => (
            <span key={folder.uuid} className="breadcrumb">
              <span aria-hidden="true">/</span>
              <button className="text-button" onClick={() => open(folder.uuid)} aria-current={folder.uuid === folderId}>
                {folder.title}
              </button>
            </span>
          ))}
        </nav>
        <select value={sort} onChange={(e) => setSort(e.target.value as Sort)} aria-label="Sort documents">
          <option value="name">Name</option>
          <option value="recent">Last modified</option>
        </select>
      </div>
      <ul className="doc-list">
        {childFolders.map((folder) => (
          <li key={folder.uuid} className="doc-row folder-row">
            <button className="folder-open" onClick={() => open(folder.uuid)}>
              <span className="folder-icon" aria-hidden="true" />
              <span className="doc-title">{folder.title}</span>
              <span className="doc-position">{itemCount(folder.uuid)} items</span>
            </button>
          </li>
        ))}
        {childDocuments.map((document) => (
          <li key={document.uuid} className="doc-row">
            <span className="doc-cover">
              {document.hasCover && (
                <img src={`/api/devices/${deviceId}/documents/${document.uuid}/cover`} alt="" loading="lazy" />
              )}
            </span>
            <span className="doc-title">{document.title}</span>
            {document.currentPage && document.pageCount && (
              <span className="doc-position">
                p. {document.currentPage} / {document.pageCount}
              </span>
            )}
            <span className="badge badge-filetype">{document.fileType}</span>
            {document.fileType === 'notebook' ? null : document.linkedBookId ? (
              <span className="linked">
                Linked to{' '}
                <Link to={`/books/${document.linkedBookId}`}>
                  {books.find((book) => book.id === document.linkedBookId)?.title ?? 'a book'}
                </Link>
                <button className="text-button" onClick={() => onUnlink(document)}>
                  Unlink
                </button>
              </span>
            ) : (
              <div className="doc-actions">
                <select onChange={(e) => onLink(document, e.target.value)} defaultValue="">
                  <option value="" disabled>
                    Link to existing book…
                  </option>
                  {books.map((book) => (
                    <option key={book.id} value={book.id}>
                      {book.title}
                    </option>
                  ))}
                </select>
                <button className="primary" onClick={() => onFind(document)}>
                  Find book
                </button>
                <button className="text-button" onClick={() => onAddAsIs(document)}>
                  Add as is
                </button>
              </div>
            )}
          </li>
        ))}
        {folderId === rootFolder && trashCount > 0 && (
          <li className="doc-row folder-row">
            <button className="folder-open" onClick={() => open(trashFolder)}>
              <span className="folder-icon folder-icon-trash" aria-hidden="true" />
              <span className="doc-title">Trash</span>
              <span className="doc-position">{trashCount} items</span>
            </button>
          </li>
        )}
      </ul>
      {childFolders.length === 0 && childDocuments.length === 0 && (
        <p className="empty-state">{folderId === rootFolder ? 'Nothing synced from this tablet yet.' : 'This folder is empty.'}</p>
      )}
    </div>
  )
}
