package hako

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
)

type Server struct {
	router *gin.Engine
	done   chan struct{}
}

func NewServer(db *DB, fs FS) *Server {
	r := gin.Default()

	// Handle file uploads via PUT
	r.PUT("/:name", func(c *gin.Context) {
		// Get expiry from the query string, if it exists
		expiry := c.Query("expiry")
		if expiry == "" {
			expiry = "1h"
		}

		// Parse the expiry
		ttl, err := ParseExpiry(expiry)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("parsing expiry: %s", err)})
			return
		}

		// Write the file to the filesystem
		filePath, err := fs.WriteFile(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("writing file: %s", err)})
			return
		}

		// Save the file to the database
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

		id, err := db.CreateFile(filePath, fileName, contentType, expiresAt, clientIP, userAgent)
		if err != nil {
			// Delete the file from the filesystem if saving to the database fails
			fs.DeleteFile(filePath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("creating file record: %s", err)})
		}

		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	// Handle file downloads via GET
	r.GET("/:id", func(c *gin.Context) {
		// Get the file from the database
		fileId, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
			return
		}

		file, err := db.GetFile(fileId)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}

		// Check if the file has expired
		if file.ExpiresAt.Before(time.Now()) {
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

		// Serve the file
		http.ServeContent(c.Writer, c.Request, file.OriginalFilename, time.Now(), readSeeker)
	})

	return &Server{router: r, done: make(chan struct{})}
}

func (s *Server) Run(ctx context.Context) {
	srv := &http.Server{
		Addr:    ":8080",
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
