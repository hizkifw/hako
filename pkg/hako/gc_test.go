package hako_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/hizkifw/hako/pkg/hako"
	"github.com/stretchr/testify/assert"
)

func TestGC(t *testing.T) {
	assert := assert.New(t)

	db, err := hako.NewDB(":memory:")
	assert.Nil(err, "Failed to create database")

	err = db.Migrate()
	assert.Nil(err, "Failed to migrate database")

	tempDir := t.TempDir()
	fs, err := hako.NewLocalFS(tempDir)
	assert.Nil(err, "Failed to create LocalFS")

	gc := hako.NewGC(db, fs)
	ctx := context.Background()

	// Test running GC with no expired files
	removed, err := gc.RunGC(ctx)
	assert.Nil(err, "Failed to run GC")
	assert.Zero(removed, "No files should be removed")

	// Create an expired file
	filePath, err := fs.WriteFile(bytes.NewReader([]byte("Hello, World!")))
	assert.Nil(err, "Failed to write file")
	assert.NotEmpty(filePath, "File path should not be empty")
	fileId, err := db.CreateFile(filePath, "file.txt", "text/plain", time.Now().Add(-1*time.Hour), "127.0.0.1", "TestAgent")
	assert.Nil(err, "Failed to create expired file")
	assert.NotZero(fileId, "File ID should not be zero")

	// Test running GC with an expired file
	removed, err = gc.RunGC(ctx)
	assert.Nil(err, "Failed to run GC")
	assert.Equal(1, removed, "One file should be removed")

	// File should be deleted from the filesystem
	_, err = fs.ReadFile(filePath)
	assert.Error(err, "File should not exist")

	// Test running GC with no expired files
	removed, err = gc.RunGC(ctx)
	assert.Nil(err, "Failed to run GC")
	assert.Zero(removed, "No files should be removed")
}
