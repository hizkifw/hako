package hako

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalFS struct {
	Root string
}

func NewLocalFS(root string) (FS, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}

	return &LocalFS{Root: root}, nil
}

func FxNewLocalFS(config *Config) (FS, error) {
	return NewLocalFS(config.FsRoot)
}

// ReadFile implements FS.
func (l *LocalFS) ReadFile(filename string) (io.ReadSeeker, error) {
	filePath := filepath.Join(l.Root, filename)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// WriteFile implements FS.
func (l *LocalFS) WriteFile(data io.Reader) (string, error) {
	// Create a temporary file
	file, err := os.CreateTemp(l.Root, "")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Hash and write the file concurrently
	hash := sha256.New()
	multiWriter := io.MultiWriter(file, hash)
	_, err = io.Copy(multiWriter, data)
	file.Close()
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}

	// Get the hash value
	hashValue := hex.EncodeToString(hash.Sum(nil))
	dirPath := filepath.Join(l.Root, hashValue[:2])
	relPath := filepath.Join(hashValue[:2], hashValue)

	// Create subdirectories
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		return "", err
	}

	// Rename the temporary file
	err = os.Rename(file.Name(), filepath.Join(l.Root, relPath))
	if err != nil {
		return "", err
	}

	// Return the hash value
	return relPath, nil
}

// DeleteFile implements FS.
func (l *LocalFS) DeleteFile(filename string) error {
	filePath := filepath.Join(l.Root, filename)
	return os.Remove(filePath)
}
