package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TempFile represents a temporary file with expiration
type TempFile struct {
	ID          string
	Path        string
	OriginalPath string // Path to original downloaded file
	MediaType   string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Size        int64
}

// TempStorage manages temporary files with automatic expiration
type TempStorage struct {
	baseDir    string
	files      map[string]*TempFile
	mu         sync.RWMutex
	ttl        time.Duration // 10 minutes
	cleanupTicker *time.Ticker
	stopCleanup chan struct{}
}

// NewTempStorage creates a new temporary storage manager
func NewTempStorage(baseDir string, ttl time.Duration) *TempStorage {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}

	// Create base directory
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Printf("Warning: Failed to create temp storage directory %s: %v", baseDir, err)
	}

	ts := &TempStorage{
		baseDir:    baseDir,
		files:      make(map[string]*TempFile),
		ttl:        ttl,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine (runs every minute)
	ts.cleanupTicker = time.NewTicker(1 * time.Minute)
	go ts.cleanupLoop()

	log.Printf("✅ Temp storage initialized: TTL=%v, Dir=%s", ttl, baseDir)

	return ts
}

// Store stores a file and returns a unique ID for access
func (ts *TempStorage) Store(filePath, originalPath, mediaType string) (string, error) {
	// Generate unique ID
	id := generateID()

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	now := time.Now()
	tf := &TempFile{
		ID:           id,
		Path:         filePath,
		OriginalPath: originalPath,
		MediaType:    mediaType,
		CreatedAt:    now,
		ExpiresAt:    now.Add(ts.ttl),
		Size:         fileInfo.Size(),
	}

	ts.mu.Lock()
	ts.files[id] = tf
	ts.mu.Unlock()

	// Schedule deletion
	go ts.scheduleDeletion(id, filePath, originalPath, ts.ttl)

	log.Printf("📦 Stored temp file: id=%s, type=%s, expires=%v", id, mediaType, tf.ExpiresAt.Format("15:04:05"))

	return id, nil
}

// Get retrieves a temporary file by ID
func (ts *TempStorage) Get(id string) (*TempFile, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	tf, exists := ts.files[id]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", id)
	}

	// Check if expired
	if time.Now().After(tf.ExpiresAt) {
		return nil, fmt.Errorf("file expired: %s", id)
	}

	return tf, nil
}

// scheduleDeletion deletes files after TTL
func (ts *TempStorage) scheduleDeletion(id, filePath, originalPath string, ttl time.Duration) {
	time.Sleep(ttl)

	// Remove from map
	ts.mu.Lock()
	delete(ts.files, id)
	ts.mu.Unlock()

	// Delete processed file
	if err := os.Remove(filePath); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("⚠️  Failed to delete processed file %s: %v", filePath, err)
		}
	}

	// Delete original file if different
	if originalPath != "" && originalPath != filePath {
		if err := os.Remove(originalPath); err != nil {
			if !os.IsNotExist(err) {
				log.Printf("⚠️  Failed to delete original file %s: %v", originalPath, err)
			}
		}
	}

	log.Printf("🗑️  Deleted expired files: id=%s", id)
}

// cleanupLoop runs periodic cleanup
func (ts *TempStorage) cleanupLoop() {
	for {
		select {
		case <-ts.cleanupTicker.C:
			ts.cleanup()
		case <-ts.stopCleanup:
			ts.cleanupTicker.Stop()
			return
		}
	}
}

// cleanup removes expired entries and files
func (ts *TempStorage) cleanup() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	now := time.Now()
	expiredFiles := []*TempFile{}

	for id, tf := range ts.files {
		if now.After(tf.ExpiresAt) {
			expiredFiles = append(expiredFiles, tf)
			delete(ts.files, id)
		}
	}

	// Delete physical files outside lock
	if len(expiredFiles) > 0 {
		go func() {
			for _, tf := range expiredFiles {
				// Delete processed file
				if err := os.Remove(tf.Path); err != nil && !os.IsNotExist(err) {
					log.Printf("⚠️  Cleanup failed to delete %s: %v", tf.Path, err)
				}
				// Delete original file if different
				if tf.OriginalPath != "" && tf.OriginalPath != tf.Path {
					if err := os.Remove(tf.OriginalPath); err != nil && !os.IsNotExist(err) {
						log.Printf("⚠️  Cleanup failed to delete %s: %v", tf.OriginalPath, err)
					}
				}
			}
			log.Printf("🧹 Cleanup: removed %d expired files", len(expiredFiles))
		}()
	}
}

// Stop gracefully shuts down the storage
func (ts *TempStorage) Stop() {
	close(ts.stopCleanup)
	log.Println("🛑 Temp storage stopped")
}

// GetStats returns storage statistics
func (ts *TempStorage) GetStats() map[string]interface{} {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	totalSize := int64(0)
	for _, tf := range ts.files {
		totalSize += tf.Size
	}

	return map[string]interface{}{
		"total_files": len(ts.files),
		"total_size_mb": float64(totalSize) / (1024 * 1024),
		"ttl_minutes": ts.ttl.Minutes(),
	}
}

// Helper function to generate unique ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetFileExtension returns extension based on media type
func GetFileExtension(mediaType string) string {
	switch mediaType {
	case "audio":
		return ".opus"
	case "image":
		return ".jpg" // Will be adjusted based on input format
	case "video":
		return ".mp4"
	default:
		return ".bin"
	}
}

// GenerateTempPath creates a temporary file path
func (ts *TempStorage) GenerateTempPath(mediaType string) string {
	id := generateID()
	ext := GetFileExtension(mediaType)
	filename := fmt.Sprintf("%s%s", id[:12], ext)
	return filepath.Join(ts.baseDir, filename)
}

// GenerateTempPathWithFormat creates a temporary file path with specific format
func (ts *TempStorage) GenerateTempPathWithFormat(mediaType string, format string) string {
	id := generateID()
	ext := getExtensionForFormat(format)
	filename := fmt.Sprintf("%s%s", id[:12], ext)
	return filepath.Join(ts.baseDir, filename)
}

// getExtensionForFormat returns extension for a specific format
func getExtensionForFormat(format string) string {
	format = strings.ToLower(format)
	if !strings.HasPrefix(format, ".") {
		return "." + format
	}
	return format
}
