package devicesync

import (
	"math"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/store"
)

// tabletProgress decides whether a document's position on the tablet should
// move its linked book's bookmark, and to where. It reports false when:
//   - the document has no position (never opened on the tablet),
//   - the book's bookmark changed more recently than the tablet's, or
//   - the book is Finished, since opening a finished book to look something
//     up shouldn't reopen it.
//
// Tablet pages rarely match the book's pages (an EPUB is laid out by the
// tablet), so the position carries over as a fraction of the way through.
// A book of unknown length takes the tablet's pages as its own.
func tabletProgress(book models.Book, document models.RemarkableDocument) (store.SetTabletProgressInput, bool) {
	if document.CurrentPage == nil || document.PageCount == nil || document.PositionUpdatedAt == nil {
		return store.SetTabletProgressInput{}, false
	}
	if book.ProgressUpdatedAt != nil && !document.PositionUpdatedAt.After(*book.ProgressUpdatedAt) {
		return store.SetTabletProgressInput{}, false
	}
	if book.Status == models.StatusFinished {
		return store.SetTabletProgressInput{}, false
	}
	pageCount, currentPage := *document.PageCount, *document.CurrentPage
	if book.PageCount != nil {
		pageCount = *book.PageCount
		currentPage = int(math.Round(float64(*document.CurrentPage) / float64(*document.PageCount) * float64(pageCount)))
		currentPage = min(max(currentPage, 1), pageCount)
	}
	status := models.StatusReading
	if currentPage >= pageCount {
		status = models.StatusFinished
	}
	return store.SetTabletProgressInput{
		BookID:      book.ID,
		CurrentPage: currentPage,
		PageCount:   pageCount,
		Status:      status,
		UpdatedAt:   *document.PositionUpdatedAt,
	}, true
}
