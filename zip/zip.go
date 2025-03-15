package zip

import (
	"archive/zip"
	"io"
)

// Writer wraps the standard zip.Writer
type Writer struct {
	*zip.Writer
}

// NewWriter creates a new Writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{zip.NewWriter(w)}
}

// OpenReader opens a zip archive for reading
func OpenReader(filename string) (*zip.ReadCloser, error) {
	return zip.OpenReader(filename)
} 