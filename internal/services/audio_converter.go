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

// AudioConverter handles audio conversion with anti-fingerprinting
type AudioConverter struct {
	workerPool *pool.WorkerPool
	bufferPool *pool.BufferPool
	mu         sync.RWMutex
	stats      AudioStats
}

// AudioStats tracks conversion metrics
type AudioStats struct {
	TotalConversions  int64
	FailedConversions int64
	AvgConversionTime time.Duration
}

// NewAudioConverter creates a new audio converter
func NewAudioConverter(workerPool *pool.WorkerPool, bufferPool *pool.BufferPool) *AudioConverter {
	return &AudioConverter{
		workerPool: workerPool,
		bufferPool: bufferPool,
	}
}

// Convert processes audio with anti-fingerprinting
func (ac *AudioConverter) Convert(ctx context.Context, inputData []byte, level string, outputPath string) error {
	start := time.Now()

	// Validate input
	if len(inputData) == 0 {
		return fmt.Errorf("empty input data")
	}

	// Get randomized parameters based on level
	params := ac.getRandomizedParams(level)

	// Build FFmpeg command with anti-fingerprinting
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", "pipe:0", // Input from stdin
		"-vn",           // No video
		"-map", "0:a:0", // First audio stream
		"-c:a", "libopus",
		"-b:a", params.bitrate,
		"-vbr", "on",
		"-compression_level", strconv.Itoa(params.compression),
		"-application", "voip",
		"-ar", "48000",
		"-ac", "1", // Mono
	)

	// Add anti-fingerprint filters
	filters := []string{}

	// Add silence padding (basic, moderate, paranoid)
	if params.silencePadding > 0 {
		filters = append(filters, fmt.Sprintf("adelay=%d:all=1", params.silencePadding))
	}

	// Add pitch shift (moderate, paranoid)
	if params.pitchShift != 0 {
		filters = append(filters, fmt.Sprintf("asetrate=48000*%.6f,aresample=48000", params.pitchShift))
	}

	// Add subtle noise (paranoid only)
	if params.addNoise {
		filters = append(filters, fmt.Sprintf("anoisesrc=d=%d:c=pink:r=48000:a=0.001,amix=inputs=2:weights=1 %.6f",
			len(inputData)/1000, params.noiseLevel))
	}

	if len(filters) > 0 {
		cmd.Args = append(cmd.Args, "-af", strings.Join(filters, ","))
	}

	// Output settings
	cmd.Args = append(cmd.Args,
		"-f", "opus",
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
		ac.recordFailure()
		return fmt.Errorf("ffmpeg error: %v, stderr: %s", err, errorBuffer.String())
	}

	output := outputBuffer.Bytes()
	if len(output) == 0 {
		ac.recordFailure()
		return fmt.Errorf("ffmpeg produced no output")
	}

	// Write to file
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		ac.recordFailure()
		return fmt.Errorf("failed to write output file: %w", err)
	}

	ac.recordSuccess(time.Since(start))
	return nil
}

// ConvertWithScriptTechniques processes audio using micro-variation volume + delay while maintaining original format
func (ac *AudioConverter) ConvertWithScriptTechniques(ctx context.Context, inputData []byte, outputPath string, inputFormat string) error {
	start := time.Now()

	if len(inputData) == 0 {
		return fmt.Errorf("empty input data")
	}

	// Generate unique nonce for this processing (guarantees uniqueness)
	nonce := GenerateNonce()
	
	// Create a local RNG seeded with nonce to ensure uniqueness
	localRand := mathrand.New(mathrand.NewSource(nonce.GetSeedForRand()))

	// Use title metadata field (more portable for audio) - includes timestamp for guaranteed uniqueness
	uniqueTitle := fmt.Sprintf("uid:%s", nonce.Nonce)

	// 1. Variação de Delay (1-50ms) - influenced by nonce
	delayMs := 1 + localRand.Intn(50)
	
	// Add micro-variation from timestamp to ensure uniqueness even if random values match
	delayMs += int(nonce.Timestamp % 10) // 0-9ms additional variation
	if delayMs > 50 {
		delayMs = 50
	}

	// 2. MICRO-VARIAÇÃO DE VOLUME (0.99 - 1.01) - influenced by nonce
	volume := 0.99 + localRand.Float64()*0.02 // 0.9900 - 1.0100
	
	// Add micro-variation from timestamp for absolute uniqueness
	volume += float64(nonce.Timestamp%100) / 100000.0 // ±0.00099 additional variation

	// Combined filter: resample + delay + volume
	filter := fmt.Sprintf("aresample=48000,adelay=%d:all=1,volume=%.4f", delayMs, volume)

	var codec string
	var format string
	var extraArgs []string

	switch strings.ToLower(inputFormat) {
	case "mp3":
		codec = "libmp3lame"
		format = "mp3"
		extraArgs = []string{"-q:a", "2"}
	case "opus":
		codec = "libopus"
		format = "opus"
		extraArgs = []string{"-vbr", "on", "-application", "voip"}
	case "m4a", "aac":
		codec = "aac"
		format = "m4a"
		extraArgs = []string{"-b:a", "128k"}
	case "ogg":
		codec = "libvorbis"
		format = "ogg"
		extraArgs = []string{"-q:a", "3"}
	case "wav":
		codec = "pcm_s16le"
		format = "wav"
		extraArgs = []string{}
	default:
		codec = "libopus"
		format = "opus"
		extraArgs = []string{"-vbr", "on"}
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", "pipe:0",
		"-vn",
		"-af", filter,
		"-c:a", codec,
		"-ar", "48000",
	)

	cmd.Args = append(cmd.Args, extraArgs...)

	cmd.Args = append(cmd.Args,
		// Remove original metadata and set title
		"-map_metadata", "-1",
		"-metadata", "title="+uniqueTitle,
		"-f", format,
		"-threads", "0",
		"pipe:1",
	)

	cmd.Stdin = bytes.NewReader(inputData)
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &errorBuffer

	if err := cmd.Run(); err != nil {
		ac.recordFailure()
		return fmt.Errorf("ffmpeg error: %v, stderr: %s", err, errorBuffer.String())
	}

	output := outputBuffer.Bytes()
	if len(output) == 0 {
		ac.recordFailure()
		return fmt.Errorf("ffmpeg produced no output")
	}

	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		ac.recordFailure()
		return fmt.Errorf("failed to write output file: %w", err)
	}

	ac.recordSuccess(time.Since(start))
	return nil
}

type audioParams struct {
	bitrate        string
	compression    int
	silencePadding int // milliseconds
	pitchShift     float64
	addNoise       bool
	noiseLevel     float64
}

func (ac *AudioConverter) getRandomizedParams(level string) audioParams {
	params := audioParams{
		bitrate:     "72k",
		compression: 10,
	}

	switch level {
	case "basic":
		// Minimal randomization
		params.bitrate = fmt.Sprintf("%dk", 70+mathrand.Intn(5)) // 70-74k
		params.compression = 8 + mathrand.Intn(3)                // 8-10
		params.silencePadding = 1 + mathrand.Intn(3)             // 1-3ms

	case "moderate":
		// Moderate randomization (default)
		params.bitrate = fmt.Sprintf("%dk", 70+mathrand.Intn(5))            // 70-74k
		params.compression = 8 + mathrand.Intn(3)                           // 8-10
		params.silencePadding = 1 + mathrand.Intn(3)                        // 1-3ms
		params.pitchShift = 1.0 + (float64(mathrand.Intn(20)-10) / 10000.0) // ±0.001

	case "paranoid":
		// Maximum randomization
		params.bitrate = fmt.Sprintf("%dk", 68+mathrand.Intn(9))            // 68-76k
		params.compression = 7 + mathrand.Intn(4)                           // 7-10
		params.silencePadding = 1 + mathrand.Intn(5)                        // 1-5ms
		params.pitchShift = 1.0 + (float64(mathrand.Intn(40)-20) / 10000.0) // ±0.002
		params.addNoise = true
		params.noiseLevel = 0.0005 + float64(mathrand.Intn(10))/100000.0 // 0.0005-0.0006

	default: // "none"
		params.bitrate = "72k"
		params.compression = 10
	}

	return params
}

func (ac *AudioConverter) recordSuccess(duration time.Duration) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.stats.TotalConversions++
	// Update average (simple moving average)
	ac.stats.AvgConversionTime = (ac.stats.AvgConversionTime*time.Duration(ac.stats.TotalConversions-1) + duration) / time.Duration(ac.stats.TotalConversions)
}

func (ac *AudioConverter) recordFailure() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.stats.FailedConversions++
}

// GetStats returns current statistics
func (ac *AudioConverter) GetStats() AudioStats {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.stats
}

// GetOutputExtension returns the file extension for this converter
func (ac *AudioConverter) GetOutputExtension() string {
	return ".opus"
}

// GenerateOutputPath creates a unique output path
func (ac *AudioConverter) GenerateOutputPath(cacheDir, deviceID, urlHash string) string {
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("%s_%s_%d%s", deviceID, urlHash[:8], timestamp, ac.GetOutputExtension())
	return filepath.Join(cacheDir, filename)
}
