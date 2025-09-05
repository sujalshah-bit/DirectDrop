package config

const (
	SERVER_ADDRESS = 0
	CODE           = 1
	ACTION         = 2
	PATH           = 3
	TIMEOUT        = 5
	DIR            = "dir"
	FILE           = "file"
)

// Meta represents both file and directory metadata
type Meta struct {
	Type     string `json:"type"`               // "file" or "dir"
	Path     string `json:"path,omitempty"`     // relative path for directories or files
	Filename string `json:"filename,omitempty"` // optional filename for single file transfers
	Size     int    `json:"size,omitempty"`     // size of compressed content (only for files)
	Checksum string `json:"checksum,omitempty"` // checksum of compressed content (only for files)
}
