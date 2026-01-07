package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"fingerprint-converter/internal/models"
	"fingerprint-converter/internal/services"
	"fingerprint-converter/internal/storage"
)

// ProcessHandler handles simplified processing requests
type ProcessHandler struct {
	audioConverter *services.AudioConverter
	imageConverter *services.ImageConverter
	videoConverter *services.VideoConverter
	downloader     *services.Downloader
	tempStorage    *storage.TempStorage
	baseURL        string // e.g., "http://localhost:4000"
	requestTimeout time.Duration
}

// NewProcessHandler creates a new process handler
func NewProcessHandler(
	audioConverter *services.AudioConverter,
	imageConverter *services.ImageConverter,
	videoConverter *services.VideoConverter,
	downloader *services.Downloader,
	tempStorage *storage.TempStorage,
	baseURL string,
	requestTimeout time.Duration,
) *ProcessHandler {
	if requestTimeout <= 0 {
		requestTimeout = 5 * time.Minute
	}

	return &ProcessHandler{
		audioConverter: audioConverter,
		imageConverter: imageConverter,
		videoConverter: videoConverter,
		downloader:     downloader,
		tempStorage:    tempStorage,
		baseURL:        baseURL,
		requestTimeout: requestTimeout,
	}
}

// Process handles POST /api/process
func (h *ProcessHandler) Process(c fiber.Ctx) error {
	// Parse request
	var req models.ProcessRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ProcessResponse{
			Success: false,
			Message: "Invalid request body",
		})
	}

	// Validate URL
	if req.Arquivo == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ProcessResponse{
			Success: false,
			Message: "arquivo (URL) is required",
		})
	}

	// Detect media type and format from URL
	mediaType, inputFormat := detectMediaTypeAndFormatFromURL(req.Arquivo)
	if mediaType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ProcessResponse{
			Success: false,
			Message: "Could not detect media type from URL. Supported: .mp3, .opus, .mp4, .jpg, .jpeg, .png",
		})
	}

	log.Printf("🔄 Processing: type=%s, format=%s, url=%s", mediaType, inputFormat, truncateURL(req.Arquivo))

	ctx, cancel := context.WithTimeout(context.Background(), h.requestTimeout)
	defer cancel()

	// Download file
	log.Printf("📥 Downloading file...")
	inputData, err := h.downloader.Download(ctx, req.Arquivo)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ProcessResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to download file: %v", err),
		})
	}

	// Save original file temporarily
	originalPath := h.tempStorage.GenerateTempPath(mediaType) + ".original"
	if err := os.WriteFile(originalPath, inputData, 0644); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ProcessResponse{
			Success: false,
			Message: "Failed to save original file",
		})
	}

	// Generate output path with original format extension
	outputPath := h.tempStorage.GenerateTempPathWithFormat(mediaType, inputFormat)

	// Process file with script techniques (always use "script" level)
	log.Printf("🧬 Applying fingerprint techniques...")
	processingStart := time.Now()

	switch mediaType {
	case "audio":
		err = h.audioConverter.ConvertWithScriptTechniques(ctx, inputData, outputPath, inputFormat)
	case "image":
		err = h.imageConverter.ConvertWithScriptTechniques(ctx, inputData, outputPath)
	case "video":
		err = h.videoConverter.ConvertWithScriptTechniques(ctx, inputData, outputPath)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(models.ProcessResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported media type: %s", mediaType),
		})
	}

	if err != nil {
		// Cleanup original file on error
		os.Remove(originalPath)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ProcessResponse{
			Success: false,
			Message: fmt.Sprintf("Processing failed: %v", err),
		})
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		os.Remove(originalPath)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ProcessResponse{
			Success: false,
			Message: "Output file was not created",
		})
	}

	log.Printf("📁 Output file created: %s", outputPath)

	// Store in temp storage
	fileID, err := h.tempStorage.Store(outputPath, originalPath, mediaType)
	if err != nil {
		os.Remove(outputPath)
		os.Remove(originalPath)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ProcessResponse{
			Success: false,
			Message: "Failed to store processed file",
		})
	}

	// Generate URL with original format extension
	extension := getExtensionForFormat(inputFormat)
	novaURL := fmt.Sprintf("%s/api/files/%s%s", h.baseURL, fileID, extension)

	log.Printf("✅ Processed: type=%s, format=%s, id=%s, path=%s, time=%dms",
		mediaType, inputFormat, fileID, outputPath, time.Since(processingStart).Milliseconds())

	return c.JSON(models.ProcessResponse{
		Success:   true,
		Message:   "arquivo modificado com sucesso!",
		NovaURL:   novaURL,
		MediaType: mediaType,
		FileID:    fileID,
	})
}

// GetFile handles GET /api/files/:id
func (h *ProcessHandler) GetFile(c fiber.Ctx) error {
	fileIDWithExt := c.Params("id")
	if fileIDWithExt == "" {
		return c.Status(fiber.StatusBadRequest).SendString("File ID is required")
	}

	// Remove extension from ID (e.g., "abc123.opus" -> "abc123")
	fileID := fileIDWithExt
	if idx := strings.LastIndex(fileIDWithExt, "."); idx > 0 {
		fileID = fileIDWithExt[:idx]
	}

	log.Printf("🔍 GetFile: id_with_ext=%s, id=%s", fileIDWithExt, fileID)

	// Get file from storage
	tf, err := h.tempStorage.Get(fileID)
	if err != nil {
		log.Printf("❌ GetFile: storage.Get failed: %v", err)
		return c.Status(fiber.StatusNotFound).SendString("File not found or expired")
	}

	log.Printf("📂 GetFile: found file path=%s", tf.Path)

	// Check if file exists
	if _, err := os.Stat(tf.Path); os.IsNotExist(err) {
		log.Printf("❌ GetFile: file not found on disk: %s", tf.Path)
		return c.Status(fiber.StatusNotFound).SendString("File not found on disk")
	}

	// Set appropriate content type based on file extension
	contentType := getContentTypeFromPath(tf.Path)
	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(tf.Path)))

	// Send file
	return c.SendFile(tf.Path)
}

// Helper functions

// detectMediaTypeAndFormatFromURL detects both media type and format from URL
func detectMediaTypeAndFormatFromURL(url string) (mediaType string, format string) {
	urlLower := strings.ToLower(url)

	// Audio formats
	if strings.HasSuffix(urlLower, ".mp3") {
		return "audio", "mp3"
	}
	if strings.HasSuffix(urlLower, ".opus") {
		return "audio", "opus"
	}
	if strings.HasSuffix(urlLower, ".ogg") {
		return "audio", "ogg"
	}
	if strings.HasSuffix(urlLower, ".m4a") {
		return "audio", "m4a"
	}
	if strings.HasSuffix(urlLower, ".wav") {
		return "audio", "wav"
	}
	if strings.HasSuffix(urlLower, ".aac") {
		return "audio", "aac"
	}

	// Image formats
	if strings.HasSuffix(urlLower, ".jpg") || strings.HasSuffix(urlLower, ".jpeg") {
		return "image", "jpg"
	}
	if strings.HasSuffix(urlLower, ".png") {
		return "image", "png"
	}
	if strings.HasSuffix(urlLower, ".webp") {
		return "image", "webp"
	}

	// Video formats
	if strings.HasSuffix(urlLower, ".mp4") {
		return "video", "mp4"
	}
	if strings.HasSuffix(urlLower, ".avi") {
		return "video", "avi"
	}
	if strings.HasSuffix(urlLower, ".mov") {
		return "video", "mov"
	}
	if strings.HasSuffix(urlLower, ".mkv") {
		return "video", "mkv"
	}
	if strings.HasSuffix(urlLower, ".webm") {
		return "video", "webm"
	}

	return "", ""
}

// getExtensionForFormat returns extension for a specific format
func getExtensionForFormat(format string) string {
	format = strings.ToLower(format)
	if !strings.HasPrefix(format, ".") {
		return "." + format
	}
	return format
}

func getContentType(mediaType string) string {
	switch mediaType {
	case "audio":
		return "audio/ogg"
	case "image":
		return "image/jpeg"
	case "video":
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

// getContentTypeFromPath returns content type based on file extension
func getContentTypeFromPath(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".opus":
		return "audio/ogg"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	case ".wav":
		return "audio/wav"
	case ".aac":
		return "audio/aac"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".mkv":
		return "video/x-matroska"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

// Health handles GET /api/health
func (h *ProcessHandler) Health(c fiber.Ctx) error {
	// Check FFmpeg availability
	ffmpegVersion := "unknown"
	if output, err := exec.Command("ffmpeg", "-version").Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			ffmpegVersion = strings.TrimSpace(lines[0])
		}
	}

	// Get temp storage stats
	storageStats := h.tempStorage.GetStats()

	return c.JSON(fiber.Map{
		"status":        "healthy",
		"timestamp":     time.Now().Format(time.RFC3339),
		"ffmpeg_version": ffmpegVersion,
		"temp_storage":  storageStats,
	})
}
