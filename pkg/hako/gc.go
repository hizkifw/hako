package hako

import (
	"context"
	"log"
	"time"
)

type GC struct {
	db   *DB
	fs   FS
	done chan struct{}
}

func NewGC(db *DB, fs FS) *GC {
	return &GC{db, fs, make(chan struct{})}
}

// LoopForever runs the garbage collection loop.
func (g *GC) LoopForever(ctx context.Context) {
	log.Printf("[GC] Start")
	for {
		// Check if the context is cancelled
		select {
		case <-ctx.Done():
			log.Printf("[GC] Stop")
			g.done <- struct{}{}
			return
		default:
		}

		removed, err := g.RunGC(ctx)
		if err != nil {
			log.Printf("[GC] Failed to run garbage collection: %v", err)
		} else {
			if removed > 0 {
				log.Printf("[GC] Removed %d files", removed)
			}
		}

		// Sleep for a while
		SleepWithContext(ctx, 1*time.Minute)
	}
}

// RunGC runs the garbage collection process.
func (g *GC) RunGC(ctx context.Context) (int, error) {
	// Get a list of expired files
	files, err := g.db.ListExpiredFiles()
	if err != nil {
		return 0, err
	}

	// Delete expired files
	removed := 0
	for _, expired := range files {
		// Check if the context is cancelled
		select {
		case <-ctx.Done():
			return removed, nil
		default:
		}

		// Delete the file
		if err := g.fs.DeleteFile(expired.FilePath); err != nil {
			log.Printf("[GC] Failed to delete file %d (%s): %v", expired.ID, expired.FilePath, err)
		} else {
			g.db.RemoveFile(expired.ID)
			log.Printf("[GC] Deleted file %d (%s)", expired.ID, expired.FilePath)
			removed++
		}
	}

	return removed, nil
}

// Done returns a channel that will be closed when the garbage collection loop
// is done.
func (g *GC) Done() <-chan struct{} {
	return g.done
}
