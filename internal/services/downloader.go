package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"fingerprint-converter/internal/pool"
)

// Downloader handles file downloads from URLs (S3, HTTP, HTTPS)
type Downloader struct {
	client     *http.Client
	bufferPool *pool.BufferPool
	maxSize    int64
}

// NewDownloader creates a new downloader with optimized HTTP client
func NewDownloader(bufferPool *pool.BufferPool, maxSize int64, timeout time.Duration) *Downloader {
	if timeout <= 0 {
		timeout = 2 * time.Minute // Aumentado de 30s para 2min (vídeos grandes)
	}

	if maxSize <= 0 {
		maxSize = 500 * 1024 * 1024 // 500MB default
	}

	// Optimized HTTP client for high throughput
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			ForceAttemptHTTP2:   true,
		},
	}

	return &Downloader{
		client:     client,
		bufferPool: bufferPool,
		maxSize:    maxSize,
	}
}

// Download fetches a file from URL (S3, HTTP, HTTPS) with retry logic
func (d *Downloader) Download(ctx context.Context, url string) ([]byte, error) {
	// Validate URL
	if url == "" {
		return nil, fmt.Errorf("empty URL")
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("invalid URL scheme: must be http:// or https://")
	}

	// Retry logic: até 3 tentativas
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		data, err := d.downloadWithValidation(ctx, url, attempt)
		if err == nil {
			return data, nil
		}
		lastErr = err

		// Não retenta em erros que não são de rede/timeout
		if !isRetryableError(err) {
			return nil, err
		}

		if attempt < 3 {
			log.Printf("⚠️  Download attempt %d failed: %v, retrying...", attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second) // Backoff: 1s, 2s
		}
	}

	return nil, fmt.Errorf("download failed after 3 attempts: %w", lastErr)
}

// downloadWithValidation performs the actual download with validation
func (d *Downloader) downloadWithValidation(ctx context.Context, url string, attempt int) ([]byte, error) {

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Check content length
	contentLength := resp.ContentLength
	if contentLength > d.maxSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", contentLength, d.maxSize)
	}

	log.Printf("📥 Downloading: size=%d bytes, attempt=%d, url=%s", contentLength, attempt, truncateURL(url))

	// Use buffer pool for efficient memory management
	var data []byte

	if contentLength > 0 {
		// Known size - allocate exact buffer
		expectedSize := int(contentLength)

		if contentLength <= int64(d.bufferPool.GetStats().Allocated) {
			buf := d.bufferPool.GetSized(expectedSize)
			defer d.bufferPool.PutSized(buf)

			// ReadFull garante que leia exatamente o tamanho esperado
			n, err := io.ReadFull(resp.Body, buf[:expectedSize])
			if err != nil {
				if err == io.ErrUnexpectedEOF || err == io.EOF {
					return nil, fmt.Errorf("incomplete download: expected %d bytes, got %d bytes (connection interrupted)", expectedSize, n)
				}
				return nil, fmt.Errorf("read failed: %w", err)
			}

			// Valida que leu o tamanho completo
			if n != expectedSize {
				return nil, fmt.Errorf("incomplete download: expected %d bytes, got %d bytes", expectedSize, n)
			}

			data = make([]byte, n)
			copy(data, buf[:n])
		} else {
			// Too large for pool, read directly with validation
			var readErr error
			data, readErr = io.ReadAll(io.LimitReader(resp.Body, d.maxSize+1))
			if readErr != nil {
				return nil, fmt.Errorf("read failed: %w", readErr)
			}

			// Valida que leu o tamanho completo esperado
			if int64(len(data)) != contentLength {
				return nil, fmt.Errorf("incomplete download: expected %d bytes, got %d bytes (partial file)", contentLength, len(data))
			}

			// Verifica se excedeu o limite
			if int64(len(data)) > d.maxSize {
				return nil, fmt.Errorf("file too large: %d bytes (max: %d)", len(data), d.maxSize)
			}
		}
	} else {
		// Unknown size - use limited reader
		log.Printf("⚠️  Content-Length not provided, reading until EOF (url=%s)", truncateURL(url))
		var readErr error
		data, readErr = io.ReadAll(io.LimitReader(resp.Body, d.maxSize+1))
		if readErr != nil {
			return nil, fmt.Errorf("read failed: %w", readErr)
		}

		// Para tamanho desconhecido, verifica se chegou ao limite (possível truncamento)
		if int64(len(data)) > d.maxSize {
			return nil, fmt.Errorf("file too large: %d bytes (max: %d)", len(data), d.maxSize)
		}
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("downloaded file is empty")
	}

	// Validação adicional: tamanho mínimo esperado para arquivos de mídia
	// Vídeos/áudios muito pequenos provavelmente estão corrompidos
	if len(data) < 100 {
		return nil, fmt.Errorf("file too small: %d bytes (likely corrupted or empty)", len(data))
	}

	// Validação de integridade básica para vídeos MP4
	if contentLength > 0 && isVideoURL(url) {
		if err := validateVideoData(data); err != nil {
			return nil, fmt.Errorf("video validation failed: %w (file may be corrupted or truncated)", err)
		}
	}

	log.Printf("✅ Download complete: size=%d bytes, url=%s", len(data), truncateURL(url))
	return data, nil
}

// isRetryableError checks if error is retryable (network/timeout errors)
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Erros de rede que devem ser retentados
	retryableErrors := []string{
		"connection reset",
		"connection refused",
		"timeout",
		"deadline exceeded",
		"temporary failure",
		"EOF",
		"broken pipe",
		"incomplete download",
		"connection interrupted",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(strings.ToLower(errStr), retryable) {
			return true
		}
	}

	return false
}

// isVideoURL checks if URL is a video based on extension
func isVideoURL(url string) bool {
	urlLower := strings.ToLower(url)
	return strings.HasSuffix(urlLower, ".mp4") ||
		strings.HasSuffix(urlLower, ".avi") ||
		strings.HasSuffix(urlLower, ".mov") ||
		strings.HasSuffix(urlLower, ".mkv") ||
		strings.HasSuffix(urlLower, ".webm")
}

// validateVideoData performs basic validation on video data
func validateVideoData(data []byte) error {
	if len(data) < 32 {
		return fmt.Errorf("file too small to be valid video: %d bytes", len(data))
	}

	// Verifica magic bytes de MP4 (ftyp box)
	// MP4 files start with ftyp box at offset 4-7
	if len(data) >= 12 {
		// Common MP4 signatures
		ftypSignatures := [][]byte{
			{0x66, 0x74, 0x79, 0x70}, // "ftyp"
		}

		hasFtyp := false
		// Check first 32 bytes for ftyp signature
		for i := 0; i < 32 && i+4 <= len(data); i++ {
			for _, sig := range ftypSignatures {
				if bytes.Equal(data[i:i+4], sig) {
					hasFtyp = true
					break
				}
			}
			if hasFtyp {
				break
			}
		}

		if !hasFtyp {
			return fmt.Errorf("invalid MP4 header: missing ftyp box (file may be corrupted)")
		}
	}

	return nil
}

// truncateURL truncates URL for logging
func truncateURL(url string) string {
	if len(url) > 60 {
		return url[:57] + "..."
	}
	return url
}

// DownloadToFile downloads directly to a file (for large files)
func (d *Downloader) DownloadToFile(ctx context.Context, url, destPath string) error {
	// TODO: Implement streaming download to file for very large files
	// This can be used when file size exceeds memory constraints
	return fmt.Errorf("not implemented yet")
}
