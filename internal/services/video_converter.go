package services

import (
	"bytes"
	"context"
	"fmt"
	mathrand "math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fingerprint-converter/internal/pool"
)

// VideoConverter handles video conversion with anti-fingerprinting
type VideoConverter struct {
	workerPool *pool.WorkerPool
	bufferPool *pool.BufferPool
	mu         sync.RWMutex
	stats      VideoStats
}

// VideoStats tracks conversion metrics
type VideoStats struct {
	TotalConversions  int64
	FailedConversions int64
	AvgConversionTime time.Duration
}

// NewVideoConverter creates a new video converter
func NewVideoConverter(workerPool *pool.WorkerPool, bufferPool *pool.BufferPool) *VideoConverter {
	return &VideoConverter{
		workerPool: workerPool,
		bufferPool: bufferPool,
	}
}

// Convert processes video with anti-fingerprinting
func (vc *VideoConverter) Convert(ctx context.Context, inputData []byte, level string, outputPath string) error {
	start := time.Now()

	// Validate input
	if len(inputData) == 0 {
		return fmt.Errorf("empty input data")
	}

	// Get original video bitrate
	originalBitrate, err := vc.getVideoBitrate(ctx, inputData)
	if err != nil {
		// If we can't get bitrate, use a default
		originalBitrate = 2000
	}

	// Get randomized parameters based on level
	params := vc.getRandomizedParams(level, originalBitrate)

	// Build FFmpeg command with anti-fingerprinting
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", "pipe:0", // Input from stdin
	)

	// Video filters for anti-fingerprinting
	videoFilters := []string{}

	// Add subtle noise (basic, moderate, paranoid)
	if params.addNoise {
		videoFilters = append(videoFilters, fmt.Sprintf("noise=alls=%d:allf=t+u", params.noiseStrength))
	}

	// Add color adjustment (moderate, paranoid)
	if params.colorAdjust {
		videoFilters = append(videoFilters, fmt.Sprintf("eq=brightness=%.6f:contrast=%.6f:saturation=%.6f",
			params.brightness, params.contrast, params.saturation))
	}

	// Add timestamp in metadata (paranoid)
	if params.addTimestamp {
		videoFilters = append(videoFilters, "drawtext=text='':x=0:y=0:fontsize=1:fontcolor=black@0.01")
	}

	if len(videoFilters) > 0 {
		cmd.Args = append(cmd.Args, "-vf", strings.Join(videoFilters, ","))
	}

	// Video codec settings
	cmd.Args = append(cmd.Args,
		"-c:v", "libx264",
		"-b:v", fmt.Sprintf("%dk", params.bitrate),
		"-crf", strconv.Itoa(params.crf),
		"-preset", params.preset,
		"-g", strconv.Itoa(params.keyframeInterval),
		"-bf", "2", // B-frames
		"-movflags", "+faststart", // WhatsApp compatibility - moov atom at start
	)

	// Audio settings (copy or re-encode depending on level)
	if level == "none" || level == "basic" {
		cmd.Args = append(cmd.Args, "-c:a", "copy") // Copy audio stream
	} else {
		// Re-encode audio with slight variations
		cmd.Args = append(cmd.Args,
			"-c:a", "aac",
			"-b:a", fmt.Sprintf("%dk", 128+mathrand.Intn(16)), // 128-143k
			"-ar", "48000",
		)
	}

	// Output settings
	cmd.Args = append(cmd.Args,
		"-f", "mp4",
		"-threads", "0",
		"pipe:1", // Output to stdout
	)

	// Set up pipes
	cmd.Stdin = bytes.NewReader(inputData)
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &errorBuffer

	// Execute conversion
	if err := cmd.Run(); err != nil {
		vc.recordFailure()
		return fmt.Errorf("ffmpeg error: %v, stderr: %s", err, errorBuffer.String())
	}

	output := outputBuffer.Bytes()
	if len(output) == 0 {
		vc.recordFailure()
		return fmt.Errorf("ffmpeg produced no output")
	}

	// Write to file
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		vc.recordFailure()
		return fmt.Errorf("failed to write output file: %w", err)
	}

	vc.recordSuccess(time.Since(start))
	return nil
}

// ConvertWithScriptTechniques processes video using micro-variation gamma and a safe crop to guarantee binary uniqueness
func (vc *VideoConverter) ConvertWithScriptTechniques(ctx context.Context, inputData []byte, outputPath string) error {
	start := time.Now()

	if len(inputData) == 0 {
		return fmt.Errorf("empty input data")
	}

	// Validate MP4 integrity before processing
	if err := validateMP4Integrity(inputData); err != nil {
		return fmt.Errorf("invalid MP4 file: %w", err)
	}

	// Save to temporary file first (workaround for pipe issues with some MP4 files)
	tempInput := outputPath + ".input.mp4"
	if err := os.WriteFile(tempInput, inputData, 0644); err != nil {
		return fmt.Errorf("failed to write temp input: %w", err)
	}
	defer os.Remove(tempInput)

	// Generate unique nonce for this processing (guarantees uniqueness)
	nonce := GenerateNonce()

	// Create a local RNG seeded with nonce to ensure uniqueness
	localRand := mathrand.New(mathrand.NewSource(nonce.GetSeedForRand()))

	// 1. Crop Aleatório (1-2 pixels) - influenced by nonce
	cropPixels := 1 + localRand.Intn(2)

	// Add micro-variation from timestamp to ensure uniqueness
	cropVariation := int(nonce.Timestamp % 3) // 0-2
	cropPixels = (cropPixels + cropVariation) % 3
	if cropPixels == 0 {
		cropPixels = 1
	}

	cropExprW := fmt.Sprintf("if(gt(iw\\,32)\\,iw-%d\\,iw)", cropPixels*2)
	cropExprH := fmt.Sprintf("if(gt(ih\\,32)\\,ih-%d\\,ih)", cropPixels*2)
	xExpr := "(iw-ow)/2"
	yExpr := "(ih-oh)/2"

	// 2. MICRO-VARIAÇÃO DE GAMMA (0.998 - 1.002) - influenced by nonce
	gamma := 0.998 + localRand.Float64()*0.004

	// Add micro-variation from timestamp for absolute uniqueness
	gamma += float64(nonce.Timestamp%1000) / 1000000.0 // ±0.000999 additional variation
	if gamma > 1.002 {
		gamma = 1.002
	}

	// Add a 1x1 drawbox with very low alpha to guarantee a byte-level change in keyframes
	// Position influenced by nonce for extra uniqueness
	boxX := int(nonce.Timestamp % 2)        // 0 or 1
	boxY := int((nonce.Timestamp / 10) % 2) // 0 or 1
	drawBox := fmt.Sprintf("drawbox=x=%d:y=%d:w=1:h=1:color=black@0.01:t=fill", boxX, boxY)
	vfilter := fmt.Sprintf("crop=w=%s:h=%s:x=%s:y=%s,eq=gamma=%.6f,%s", cropExprW, cropExprH, xExpr, yExpr, gamma, drawBox)

	// 3. Metadata standard field - includes nonce for guaranteed uniqueness
	uniqueTitle := fmt.Sprintf("uid:%s", nonce.Nonce)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", tempInput, // Use temp file instead of pipe for better compatibility
		"-vf", vfilter,
		"-c:v", "libx264",
		"-crf", "20",
		"-preset", "medium",
		"-c:a", "aac",
		"-b:a", "128k",
		"-ar", "48000",
		// Metadata in title field (more portable)
		"-map_metadata", "-1",
		"-metadata", "title="+uniqueTitle,
		"-movflags", "+faststart", // WhatsApp compatibility - moov atom at start
		"-f", "mp4",
		"-threads", "0",
		"pipe:1",
	)

	// No longer need stdin since we're using temp file
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &errorBuffer

	if err := cmd.Run(); err != nil {
		vc.recordFailure()
		return fmt.Errorf("ffmpeg error: %v, stderr: %s", err, errorBuffer.String())
	}

	output := outputBuffer.Bytes()
	if len(output) == 0 {
		vc.recordFailure()
		return fmt.Errorf("ffmpeg produced no output")
	}

	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		vc.recordFailure()
		return fmt.Errorf("failed to write output file: %w", err)
	}

	vc.recordSuccess(time.Since(start))
	return nil
}

type videoParams struct {
	bitrate          int
	crf              int
	preset           string
	keyframeInterval int
	addNoise         bool
	noiseStrength    int
	colorAdjust      bool
	brightness       float64
	contrast         float64
	saturation       float64
	addTimestamp     bool
}

func (vc *VideoConverter) getRandomizedParams(level string, originalBitrate int) videoParams {
	params := videoParams{
		bitrate:          originalBitrate,
		crf:              23,
		preset:           "medium",
		keyframeInterval: 250,
	}

	switch level {
	case "basic":
		// Minimal randomization (recommended for video)
		bitrateVariation := int(float64(originalBitrate) * (0.05 + float64(mathrand.Intn(6))/100.0)) // 5-10%
		params.bitrate = originalBitrate + bitrateVariation - mathrand.Intn(bitrateVariation*2)
		params.crf = 22 + mathrand.Intn(3)                // 22-24
		params.keyframeInterval = 240 + mathrand.Intn(21) // 240-260

	case "moderate":
		// Moderate randomization
		bitrateVariation := int(float64(originalBitrate) * (0.08 + float64(mathrand.Intn(5))/100.0)) // 8-12%
		params.bitrate = originalBitrate + bitrateVariation - mathrand.Intn(bitrateVariation*2)
		params.crf = 22 + mathrand.Intn(4)                // 22-25
		params.keyframeInterval = 230 + mathrand.Intn(41) // 230-270
		params.addNoise = true
		params.noiseStrength = 1 + mathrand.Intn(2) // 1-2
		params.colorAdjust = true
		params.brightness = float64(mathrand.Intn(3)-1) / 1000.0     // ±0.001
		params.contrast = 1.0 + float64(mathrand.Intn(3)-1)/1000.0   // ±0.001
		params.saturation = 1.0 + float64(mathrand.Intn(3)-1)/1000.0 // ±0.001

	case "paranoid":
		// Maximum randomization
		bitrateVariation := int(float64(originalBitrate) * (0.10 + float64(mathrand.Intn(6))/100.0)) // 10-15%
		params.bitrate = originalBitrate + bitrateVariation - mathrand.Intn(bitrateVariation*2)
		params.crf = 21 + mathrand.Intn(5)                                     // 21-25
		params.keyframeInterval = 220 + mathrand.Intn(61)                      // 220-280
		params.preset = []string{"fast", "medium", "medium"}[mathrand.Intn(3)] // Vary preset
		params.addNoise = true
		params.noiseStrength = 2 + mathrand.Intn(4) // 2-5
		params.colorAdjust = true
		params.brightness = float64(mathrand.Intn(5)-2) / 1000.0     // ±0.002
		params.contrast = 1.0 + float64(mathrand.Intn(5)-2)/1000.0   // ±0.002
		params.saturation = 1.0 + float64(mathrand.Intn(5)-2)/1000.0 // ±0.002
		params.addTimestamp = true

	default: // "none"
		params.bitrate = originalBitrate
		params.crf = 23
		params.keyframeInterval = 250
	}

	return params
}

// getVideoBitrate probes the video to get its bitrate
func (vc *VideoConverter) getVideoBitrate(ctx context.Context, inputData []byte) (int, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=bit_rate",
		"-of", "default=noprint_wrappers=1:nokey=1",
		"-i", "pipe:0",
	)

	cmd.Stdin = bytes.NewReader(inputData)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	bitrateStr := strings.TrimSpace(string(output))
	bitrate, err := strconv.Atoi(bitrateStr)
	if err != nil {
		return 0, err
	}

	// Convert to kbps
	return bitrate / 1000, nil
}

func (vc *VideoConverter) recordSuccess(duration time.Duration) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.stats.TotalConversions++
	vc.stats.AvgConversionTime = (vc.stats.AvgConversionTime*time.Duration(vc.stats.TotalConversions-1) + duration) / time.Duration(vc.stats.TotalConversions)
}

func (vc *VideoConverter) recordFailure() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.stats.FailedConversions++
}

// GetStats returns current statistics
func (vc *VideoConverter) GetStats() VideoStats {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.stats
}

// GetOutputExtension returns the file extension for this converter
func (vc *VideoConverter) GetOutputExtension() string {
	return ".mp4"
}

// GenerateOutputPath creates a unique output path
func (vc *VideoConverter) GenerateOutputPath(cacheDir, deviceID, urlHash string) string {
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("%s_%s_%d%s", deviceID, urlHash[:8], timestamp, vc.GetOutputExtension())
	return filepath.Join(cacheDir, filename)
}

// validateMP4Integrity performs basic integrity checks on MP4 data
func validateMP4Integrity(data []byte) error {
	if len(data) < 32 {
		return fmt.Errorf("file too small: %d bytes", len(data))
	}

	// Check for ftyp box (file type box) - must be present in valid MP4
	// Format: [size:4 bytes][type:4 bytes][major_brand:4 bytes][minor_version:4 bytes]
	// ftyp box should appear early in the file (usually at offset 0 or after a few bytes)
	hasFtyp := false
	hasModat := false

	// Scan first 256 bytes for ftyp
	scanLimit := 256
	if len(data) < scanLimit {
		scanLimit = len(data)
	}

	for i := 0; i < scanLimit-8; i++ {
		// Check for "ftyp" signature (0x66747970)
		if data[i] == 0x66 && data[i+1] == 0x74 && data[i+2] == 0x79 && data[i+3] == 0x70 {
			hasFtyp = true
			break
		}
	}

	if !hasFtyp {
		return fmt.Errorf("missing ftyp box - file may be corrupted or not a valid MP4")
	}

	// Check for mdat (media data) or moov (movie metadata) boxes
	// These are essential boxes in MP4 files
	for i := 0; i < len(data)-8; i++ {
		// Check for "mdat" (0x6D646174) or "moov" (0x6D6F6F76)
		if i+4 < len(data) {
			if (data[i] == 0x6D && data[i+1] == 0x64 && data[i+2] == 0x61 && data[i+3] == 0x74) ||
				(data[i] == 0x6D && data[i+1] == 0x6F && data[i+2] == 0x6F && data[i+3] == 0x76) {
				hasModat = true
				break
			}
		}
	}

	if !hasModat {
		return fmt.Errorf("missing mdat/moov box - file may be truncated or corrupted")
	}

	return nil
}
