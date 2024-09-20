package hako_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/hizkifw/hako/pkg/hako"
	"github.com/stretchr/testify/assert"
)

func TestLocalFS(t *testing.T) {
	assert := assert.New(t)
	tempDir, err := os.MkdirTemp("", "hako_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fs, err := hako.NewLocalFS(tempDir)
	assert.Nil(err, "Failed to create LocalFS")

	// Test reading a nonexistent file
	nonexistentFile := "nonexistent.txt"
	_, err = fs.ReadFile(nonexistentFile)
	assert.Error(err, "Expected error when reading nonexistent file")

	// Test writing a file
	data := []byte("Hello, World!")
	fileID, err := fs.WriteFile(bytes.NewReader(data))
	assert.NoError(err, "Failed to write file")
	assert.NotEmpty(fileID, "File ID should not be empty")

	// Test reading the written file
	file, err := fs.ReadFile(fileID)
	assert.NoError(err, "Failed to read file")
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, file)
	assert.NoError(err, "Failed to copy file contents")
	assert.Equal(data, buf.Bytes(), "File contents mismatch")
}
