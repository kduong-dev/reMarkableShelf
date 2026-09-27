package remarkable

import "github.com/kduong-dev/reMarkableShelf/backend/internal/models"

// Classify maps a document's raw .content "fileType" field to our FileType.
//
// On a reMarkable tablet, "pdf" and "epub" mean the document was imported
// from a source file — an actual ebook/PDF. An empty string (or the literal
// "notebook") means it was created natively on-device with no source file:
// a handwritten sketch/notebook. Annotations drawn on top of an imported
// PDF are stored as separate per-page .rm layers and do NOT change
// fileType, so an annotated PDF is still correctly classified as a book.
func Classify(rawFileType string) models.FileType {
	switch rawFileType {
	case "pdf":
		return models.FileTypePDF
	case "epub":
		return models.FileTypeEPUB
	default:
		return models.FileTypeNotebook
	}
}
