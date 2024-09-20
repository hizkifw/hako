package hako

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	db        *sql.DB
	snowflake *snowflake.Node
}

func NewDB(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	node, err := snowflake.NewNode(1)
	if err != nil {
		return nil, fmt.Errorf("failed to create snowflake node: %v", err)
	}

	return &DB{
		db:        db,
		snowflake: node,
	}, nil
}

func FxNewDB(cfg *Config) (*DB, error) {
	return NewDB(cfg.DbLocation)
}

func (d *DB) Migrate() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY,
			file_path TEXT,
			original_filename TEXT,
			mime_type TEXT,
			expires_at INTEGER,
			removed BOOLEAN DEFAULT FALSE,
			ip_address TEXT,
			user_agent TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	return nil
}

// CreateFile creates a new file record in the database.
func (d *DB) CreateFile(filePath, originalFilename, mimeType string, expiresAt time.Time, ipAddress, userAgent string) (int64, error) {
	id := d.snowflake.Generate().Int64()
	_, err := d.db.Exec(`
		INSERT INTO files (id, file_path, original_filename, mime_type, expires_at, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, filePath, originalFilename, mimeType, expiresAt.UnixMilli(), ipAddress, userAgent)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %v", err)
	}

	return id, nil
}

// DbFile represents a file record in the database.
type DbFile struct {
	ID               int64
	FilePath         string
	OriginalFilename string
	MimeType         string
	ExpiresAt        time.Time
	Removed          bool
	IPAddress        string
	UserAgent        string
}

// GetFile returns a file record from the database based on the given file ID.
func (d *DB) GetFile(id int64) (*DbFile, error) {
	var file DbFile
	var expiresAt int64

	row := d.db.QueryRow(`SELECT id, file_path, original_filename, mime_type, expires_at, removed, ip_address, user_agent FROM files WHERE id = ?`, id)
	err := row.Scan(&file.ID, &file.FilePath, &file.OriginalFilename, &file.MimeType, &expiresAt, &file.Removed, &file.IPAddress, &file.UserAgent)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("file not found")
		}
		return nil, fmt.Errorf("failed to get file: %v", err)
	}

	file.ExpiresAt = time.Unix(0, expiresAt*int64(time.Millisecond))

	return &file, nil
}

type ExpiredFile struct {
	ID       int64
	FilePath string
}

// ListExpiredFiles returns a list of files that have expired.
func (d *DB) ListExpiredFiles() ([]ExpiredFile, error) {
	var expiredFiles []ExpiredFile

	rows, err := d.db.Query(`SELECT id, file_path FROM files WHERE expires_at < ? AND removed = FALSE`,
		time.Now().UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("failed to list expired files: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var expiredFile ExpiredFile
		err := rows.Scan(&expiredFile.ID, &expiredFile.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}
		expiredFiles = append(expiredFiles, expiredFile)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over rows: %v", err)
	}

	return expiredFiles, nil
}

// RemoveFile marks a file as removed in the database.
func (d *DB) RemoveFile(id int64) error {
	_, err := d.db.Exec(`UPDATE files SET removed = TRUE WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to remove file: %v", err)
	}

	return nil
}
