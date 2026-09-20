package googlebooks

// searchResponse mirrors the subset of the Google Books volumes list
// response (https://www.googleapis.com/books/v1/volumes) needed to build a
// Result.
type searchResponse struct {
	Items []volumeItem `json:"items"`
}

type volumeItem struct {
	ID         string     `json:"id"`
	VolumeInfo volumeInfo `json:"volumeInfo"`
}

type volumeInfo struct {
	Title               string               `json:"title"`
	Authors             []string             `json:"authors"`
	IndustryIdentifiers []industryIdentifier `json:"industryIdentifiers"`
	ImageLinks          imageLinks           `json:"imageLinks"`
}

type industryIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

type imageLinks struct {
	Thumbnail string `json:"thumbnail"`
}

// toResult picks the fields the frontend needs out of a volume, preferring
// ISBN_13 when a volume lists more than one industry identifier.
func (item volumeItem) toResult() Result {
	author := ""
	if len(item.VolumeInfo.Authors) > 0 {
		author = item.VolumeInfo.Authors[0]
	}

	isbn := ""
	for _, identifier := range item.VolumeInfo.IndustryIdentifiers {
		if identifier.Type == "ISBN_13" {
			isbn = identifier.Identifier
			break
		}
		if isbn == "" {
			isbn = identifier.Identifier
		}
	}

	return Result{
		GoogleBooksID: item.ID,
		Title:         item.VolumeInfo.Title,
		Author:        author,
		ISBN:          isbn,
		CoverURL:      item.VolumeInfo.ImageLinks.Thumbnail,
	}
}
