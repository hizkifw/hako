package hako_test

import (
	"testing"
	"time"

	"github.com/hizkifw/hako/pkg/hako"
	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	assert := assert.New(t)

	db, err := hako.NewDB(":memory:")
	assert.Nil(err, "Failed to create database")

	err = db.Migrate()
	assert.Nil(err, "Failed to migrate database")

	// Test creating a file
	filePath := "/path/to/file"
	originalFilename := "file.txt"
	mimeType := "text/plain"
	expiresAt := time.Now().Add(1 * time.Hour)
	ipAddress := "127.0.0.1"
	userAgent := "TestAgent"
	id, err := db.CreateFile(filePath, originalFilename, mimeType, expiresAt, ipAddress, userAgent)
	assert.Nil(err, "Failed to create file")
	assert.NotZero(id, "File ID should not be zero")

	// Test listing expired files
	paths, err := db.ListExpiredFiles()
	assert.Nil(err, "Failed to list expired files")
	assert.Empty(paths, "Expired files should be empty")

	// Create an expired file
	expiresAt = time.Now().Add(-1 * time.Hour)
	_, err = db.CreateFile(filePath, originalFilename, mimeType, expiresAt, ipAddress, userAgent)
	assert.Nil(err, "Failed to create expired file")
	paths, err = db.ListExpiredFiles()
	assert.Nil(err, "Failed to list expired files")
	assert.NotEmpty(paths, "Expired files should not be empty")

	// Test getting file
	file, err := db.GetFile(id)
	assert.Nil(err, "Failed to get file")
	assert.Equal(filePath, file.FilePath, "File path mismatch")
}
