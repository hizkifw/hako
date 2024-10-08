package hako

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

//go:embed web
var webContent embed.FS

type Server struct {
	router *gin.Engine
	config *Config
	done   chan struct{}
}

func NewServer(db *DB, fs FS, cfg *Config) *Server {
	r := gin.Default()

	// Handle file uploads via PUT
	r.PUT("/:name", func(c *gin.Context) {
		// Get expiry from the query string, if it exists
		expiry := c.Query("expiry")
		if expiry == "" {
			expiry = "24h"
		}

		// Parse the expiry
		ttl, err := ParseExpiry(expiry)
		if err != nil {
			log.Printf("parsing expiry: %s", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("parsing expiry: %s", err)})
			return
		}

		// Check if the expiry is within the allowed range
		if ttl > cfg.FsMaxTTL {
			log.Printf("expiry too long (max %s)", cfg.FsMaxTTL)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("expiry too long (max %s)", cfg.FsMaxTTL)})
			return
		}

		// Check if the file size is within the allowed range
		if c.Request.ContentLength > cfg.FsMaxFileSize {
			log.Printf("file too large (%d > %d)", c.Request.ContentLength, cfg.FsMaxFileSize)
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": fmt.Sprintf("file too large (max %d bytes)", cfg.FsMaxFileSize)})
			return
		}

		// Write the file to the filesystem
		filePath, err := fs.WriteFile(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("writing file: %s", err)})
			return
		}

		fileName := c.Param("name")
		contentType := c.GetHeader("Content-Type")
		expiresAt := time.Now().Add(ttl)
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")

		// If content type is empty, sniff the content type from the file
		if contentType == "" {
			file, err := fs.ReadFile(filePath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("opening file for mime type: %s", err)})
				return
			}

			mime, err := mimetype.DetectReader(file)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("detecting mime type: %s", err)})
				return
			}

			contentType = mime.String()
		}

		// Save the file to the database
		id, err := db.CreateFile(filePath, fileName, contentType, expiresAt, clientIP, userAgent)
		if err != nil {
			// Delete the file from the filesystem if saving to the database fails
			fs.DeleteFile(filePath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("creating file record: %s", err)})
		}

		idStr := strconv.FormatInt(id, 36)
		c.JSON(http.StatusOK, gin.H{"id": idStr, "expires_at": expiresAt})
	})

	// Handle root path
	r.GET("/", func(c *gin.Context) {
		c.FileFromFS("web/", http.FS(webContent))
	})

	// Handle file downloads via GET
	r.GET("/:id", func(c *gin.Context) {
		// Check if we can serve the web contents
		fname := c.Param("id")
		_, err := webContent.Open("web/" + fname)
		if err == nil {
			c.FileFromFS("web/"+fname, http.FS(webContent))
			log.Printf("Serving web content: %s", fname)
			return
		}

		// Strip the file extension
		extIdx := strings.Index(fname, ".")
		if extIdx != -1 {
			fname = fname[:extIdx]
		}

		// Parse file ID
		fileId, err := strconv.ParseInt(fname, 36, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
			return
		}

		// Get the file from the database
		file, err := db.GetFile(fileId)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}

		// Check if the file has expired
		if file.ExpiresAt.Before(time.Now()) || file.Removed {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}

		// Read the file from the filesystem
		readSeeker, err := fs.ReadFile(file.FilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Set the response headers
		c.Header("Content-Type", file.MimeType)
		c.Header("Content-Disposition", "inline; filename=\""+file.OriginalFilename+"\"")
		c.Header("X-Hako-Expires-At", file.ExpiresAt.Format(time.RFC3339))

		// Serve the file
		http.ServeContent(c.Writer, c.Request, file.OriginalFilename, time.Now(), readSeeker)
	})

	return &Server{router: r, config: cfg, done: make(chan struct{})}
}

func (s *Server) Run(ctx context.Context) {
	srv := &http.Server{
		Addr:    s.config.HttpListenAddr,
		Handler: s.router.Handler(),
	}

	go func() {
		log.Printf("[HTTP] Listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
		s.done <- struct{}{}
	}()

	<-ctx.Done()
	log.Println("[HTTP] Shutting down server...")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("[HTTP] Server shutdown failed: %+v", err)
	}
}

// Done returns a channel that will be closed when the server has stopped.
func (s *Server) Done() <-chan struct{} {
	return s.done
}

// FxNewServer is a constructor for the Server type that is compatible with
// the fx framework.
func FxNewServer(db *DB, fs FS, cfg *Config, lc fx.Lifecycle) *Server {
	server := NewServer(db, fs, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go server.Run(ctx)
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			<-server.Done()
			return nil
		},
	})
	return server
}
