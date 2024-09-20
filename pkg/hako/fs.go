package hako

import (
	"io"
)

type FS interface {
	// ReadFile reads the file named by filename and returns the contents.
	ReadFile(filename string) (io.ReadSeeker, error)

	// WriteFile writes data to the file with the given expiry.
	WriteFile(data io.Reader) (string, error)

	// DeleteFile deletes the file with the given filename.
	DeleteFile(filename string) error
}
