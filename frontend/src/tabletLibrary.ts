import type { RemarkableDocument, RemarkableFolder } from './api/types'

// The tablet's own names for the top of its library and its trash.
export const rootFolder = ''
export const trashFolder = 'trash'

export interface TabletLibrary {
  // foldersIn and documentsIn list what a folder holds directly.
  foldersIn: (folderId: string) => RemarkableFolder[]
  documentsIn: (folderId: string) => RemarkableDocument[]
  // path lists the folders from the top of the library down to folderId.
  path: (folderId: string) => RemarkableFolder[]
  exists: (folderId: string) => boolean
}

// tabletLibrary arranges a tablet's folders and documents as its library
// shows them. An item whose folder wasn't synced sits at the top, so
// nothing on the tablet goes missing from the page.
export function tabletLibrary(folders: RemarkableFolder[], documents: RemarkableDocument[]): TabletLibrary {
  const byId = new Map(folders.map((folder) => [folder.uuid, folder]))
  const parentOf = (parentUuid?: string) =>
    parentUuid === trashFolder || (parentUuid && byId.has(parentUuid)) ? parentUuid : rootFolder
  const folderChildren = new Map<string, RemarkableFolder[]>()
  for (const folder of folders) {
    const parent = parentOf(folder.parentUuid)
    folderChildren.set(parent, [...(folderChildren.get(parent) ?? []), folder])
  }
  const documentChildren = new Map<string, RemarkableDocument[]>()
  for (const document of documents) {
    const parent = parentOf(document.parentUuid)
    documentChildren.set(parent, [...(documentChildren.get(parent) ?? []), document])
  }
  return {
    foldersIn: (folderId) => folderChildren.get(folderId) ?? [],
    documentsIn: (folderId) => documentChildren.get(folderId) ?? [],
    path: (folderId) => {
      const path: RemarkableFolder[] = []
      // Guards against a cycle in the synced folders.
      const seen = new Set<string>()
      for (let folder = byId.get(folderId); folder && !seen.has(folder.uuid); folder = byId.get(folder.parentUuid ?? '')) {
        seen.add(folder.uuid)
        path.unshift(folder)
      }
      return path
    },
    exists: (folderId) => folderId === rootFolder || folderId === trashFolder || byId.has(folderId),
  }
}
