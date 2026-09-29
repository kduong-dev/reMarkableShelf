package internetarchive

import (
	"encoding/json"
	"slices"
)

// searchResponse mirrors the subset of an advanced search response
// (https://archive.org/advancedsearch.php) needed to pick a scan.
type searchResponse struct {
	Error    string `json:"error"`
	Response struct {
		Docs []searchDocument `json:"docs"`
	} `json:"response"`
}

type searchDocument struct {
	Identifier string      `json:"identifier"`
	Collection collections `json:"collection"`
}

// priority ranks Project Gutenberg's transcriptions above scans.
func (document searchDocument) priority() int {
	if slices.Contains(document.Collection, gutenbergCollection) {
		return 1
	}
	return 0
}

// collections is an item's collections, which the archive sends as a lone
// string when there's only one.
type collections []string

func (collection *collections) UnmarshalJSON(data []byte) error {
	var single string
	if json.Unmarshal(data, &single) == nil {
		*collection = collections{single}
		return nil
	}
	return json.Unmarshal(data, (*[]string)(collection))
}

// filesResponse mirrors an item's file listing
// (https://archive.org/developers/md-read.html).
type filesResponse struct {
	Result []struct {
		Name    string `json:"name"`
		Format  string `json:"format"`
		Private string `json:"private"`
	} `json:"result"`
}
