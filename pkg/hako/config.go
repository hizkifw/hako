package hako

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HttpListenAddr string
	DbLocation     string
	FsRoot         string
	FsMaxFileSize  int64
	FsMaxTTL       time.Duration
}

func ConfigFromEnv() *Config {
	fileSizeMax, err := strconv.ParseInt(os.Getenv("HAKO_FS_MAX_FILE_SIZE"), 10, 64)
	if err != nil {
		log.Printf("failed to parse HAKO_FS_MAX_FILE_SIZE: %v", err)
		fileSizeMax = 0
	}

	ttlMax, err := ParseExpiry(os.Getenv("HAKO_FS_MAX_TTL"))
	if err != nil {
		log.Printf("failed to parse HAKO_FS_MAX_TTL: %v", err)
		ttlMax = 0
	}

	return &Config{
		HttpListenAddr: os.Getenv("HAKO_HTTP_LISTEN_ADDR"),
		DbLocation:     os.Getenv("HAKO_DB_LOCATION"),
		FsRoot:         os.Getenv("HAKO_FS_ROOT"),
		FsMaxFileSize:  fileSizeMax,
		FsMaxTTL:       ttlMax,
	}
}
