package devicesync

import (
	"github.com/kduong-dev/reMarkableShelf/backend/internal/bookstore"
	"github.com/kduong-dev/reMarkableShelf/backend/internal/models"
)

// tabletProgress decides whether a document's position on the tablet should
// move its linked book's bookmark, and to where. It reports false when:
//   - the document has no position (never opened on the tablet),
//   - the book's bookmark changed more recently than the tablet's, or
//   - the book is Finished, since opening a finished book to look something
//     up shouldn't reopen it.
//
// A linked book is read on the tablet, so it takes the tablet's pages as its
// own and the position carries over page for page.
func tabletProgress(book models.Book, document models.RemarkableDocument) (bookstore.SetTabletProgressInput, bool) {
	if document.CurrentPage == nil || document.PageCount == nil || document.PositionUpdatedAt == nil {
		return bookstore.SetTabletProgressInput{}, false
	}
	if book.ProgressUpdatedAt != nil && !document.PositionUpdatedAt.After(*book.ProgressUpdatedAt) {
		return bookstore.SetTabletProgressInput{}, false
	}
	if book.Status == models.StatusFinished {
		return bookstore.SetTabletProgressInput{}, false
	}
	pageCount, currentPage := *document.PageCount, *document.CurrentPage
	status := models.StatusReading
	if currentPage >= pageCount {
		status = models.StatusFinished
	}
	return bookstore.SetTabletProgressInput{
		BookID:      book.ID,
		CurrentPage: currentPage,
		PageCount:   pageCount,
		Status:      status,
		UpdatedAt:   *document.PositionUpdatedAt,
	}, true
}
