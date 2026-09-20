package remarkable

import "encoding/json"

// xochitlDir is where documents live on the tablet's filesystem, per the
// setup guide in the repo README.
const xochitlDir = "/home/root/.local/share/remarkable/xochitl"

// metadata mirrors the fields we need from a <uuid>.metadata file.
type metadata struct {
	VisibleName  string `json:"visibleName"`
	Type         string `json:"type"`         // "DocumentType" or "CollectionType" (folder)
	LastModified string `json:"lastModified"` // epoch millis, as a string
}

// content mirrors the fields we need from a <uuid>.content file. fileType
// is the signal that distinguishes an imported PDF/EPUB from a native
// handwritten notebook — see Classify.go.
type content struct {
	FileType string `json:"fileType"`
}

func parseMetadata(raw []byte) (metadata, error) {
	var m metadata
	err := json.Unmarshal(raw, &m)
	return m, err
}

func parseContent(raw []byte) (content, error) {
	if len(raw) == 0 {
		return content{}, nil
	}
	var c content
	err := json.Unmarshal(raw, &c)
	return c, err
}
