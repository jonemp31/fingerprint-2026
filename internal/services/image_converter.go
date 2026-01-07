package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
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

// ImageConverter handles image conversion with anti-fingerprinting
type ImageConverter struct {
	workerPool *pool.WorkerPool
	bufferPool *pool.BufferPool
	mu         sync.RWMutex
	stats      ImageStats
}

// ImageStats tracks conversion metrics
type ImageStats struct {
	TotalConversions  int64
	FailedConversions int64
	AvgConversionTime time.Duration
}

// NewImageConverter creates a new image converter
func NewImageConverter(workerPool *pool.WorkerPool, bufferPool *pool.BufferPool) *ImageConverter {
	return &ImageConverter{
		workerPool: workerPool,
		bufferPool: bufferPool,
	}
}

// Convert processes image with anti-fingerprinting
func (ic *ImageConverter) Convert(ctx context.Context, inputData []byte, level string, outputPath string) error {
	start := time.Now()

	// Validate input
	if len(inputData) == 0 {
		return fmt.Errorf("empty input data")
	}

	// Detect input format
	inputFormat := ic.detectFormat(inputData)

	// Get randomized parameters based on level
	params := ic.getRandomizedParams(level, inputFormat)

	// Build FFmpeg command with anti-fingerprinting
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", "pipe:0", // Input from stdin
	)

	// Add anti-fingerprint filters
	filters := []string{}

	// Add noise based on level and format
	if params.addNoise {
		filters = append(filters, fmt.Sprintf("noise=alls=%d:allf=t", params.noiseStrength))
	}

	// Add subtle color adjustment (moderate, paranoid)
	if params.colorAdjust {
		filters = append(filters, fmt.Sprintf("eq=brightness=%.6f:contrast=%.6f",
			params.brightness, params.contrast))
	}

	// Add slight blur (paranoid only)
	if params.addBlur {
		filters = append(filters, fmt.Sprintf("unsharp=3:3:%.2f", params.blurAmount))
	}

	if len(filters) > 0 {
		cmd.Args = append(cmd.Args, "-vf", strings.Join(filters, ","))
	}

	// Determine output format (always output as input format or fallback to JPEG)
	outputFormat := inputFormat
	if outputFormat != "png" && outputFormat != "jpeg" && outputFormat != "jpg" && outputFormat != "webp" {
		outputFormat = "jpeg" // Fallback to JPEG for unsupported formats
	}

	// Output codec and quality settings
	switch outputFormat {
	case "png":
		cmd.Args = append(cmd.Args,
			"-c:v", "png",
			"-compression_level", strconv.Itoa(params.compressionLevel),
		)
	case "webp":
		cmd.Args = append(cmd.Args,
			"-c:v", "libwebp",
			"-quality", strconv.Itoa(params.quality),
		)
	default: // jpeg/jpg
		cmd.Args = append(cmd.Args,
			"-c:v", "mjpeg",
			"-q:v", strconv.Itoa(params.jpegQScale),
		)
	}

	// Output settings
	cmd.Args = append(cmd.Args,
		"-f", "image2",
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
		ic.recordFailure()
		return fmt.Errorf("ffmpeg error: %v, stderr: %s", err, errorBuffer.String())
	}

	output := outputBuffer.Bytes()
	if len(output) == 0 {
		ic.recordFailure()
		return fmt.Errorf("ffmpeg produced no output")
	}

	// Write to file with correct extension
	finalPath := ic.adjustOutputPath(outputPath, outputFormat)
	if err := os.WriteFile(finalPath, output, 0644); err != nil {
		ic.recordFailure()
		return fmt.Errorf("failed to write output file: %w", err)
	}

	ic.recordSuccess(time.Since(start))
	return nil
}

// modifyImageLSBWithNonce makes very small LSB changes using nonce for guaranteed uniqueness
func modifyImageLSBWithNonce(data []byte, format string, nonce *ProcessingNonce) ([]byte, error) {
	if len(data) == 0 {
		return data, fmt.Errorf("empty data")
	}

	// Only handle JPEG and PNG via stdlib to avoid adding deps
	if format != "jpeg" && format != "png" {
		return data, fmt.Errorf("format not supported for LSB modification: %s", format)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, fmt.Errorf("decode failed: %w", err)
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return data, fmt.Errorf("invalid dimensions")
	}

	// Create local RNG seeded with nonce for unique pixel modifications
	localRand := mathrand.New(mathrand.NewSource(nonce.GetSeedForRand()))

	// Choose up to 3 pixels near the center to avoid being removed by small crops
	cx := w / 2
	cy := h / 2
	coords := [][2]int{{cx, cy}, {cx + 1, cy}, {cx, cy + 1}}
	// Create an NRGBA image to edit pixels safely
	nimg := image.NewNRGBA(bounds)
	draw.Draw(nimg, bounds, img, bounds.Min, draw.Src)

	for _, c := range coords {
		x := c[0]
		y := c[1]
		if x >= w || y >= h {
			continue
		}
		off := nimg.PixOffset(x+bounds.Min.X, y+bounds.Min.Y)
		// Pix is [R,G,B,A]
		// Tweak each channel LSB by +/-1 randomly (using nonce-seeded RNG)
		for i := 0; i < 3; i++ {
			v := nimg.Pix[off+i]
			if localRand.Intn(2) == 0 {
				if v < 255 {
					v = v + 1
				}
			} else {
				if v > 0 {
					v = v - 1
				}
			}
			nimg.Pix[off+i] = v
		}
	}

	// Re-encode
	var buf bytes.Buffer
	if format == "jpeg" {
		if err := jpeg.Encode(&buf, nimg, &jpeg.Options{Quality: 95}); err != nil {
			return data, fmt.Errorf("jpeg encode failed: %w", err)
		}
	} else {
		if err := png.Encode(&buf, nimg); err != nil {
			return data, fmt.Errorf("png encode failed: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// modifyImageLSB makes very small LSB changes to a few corner pixels for deterministic uniqueness
// Deprecated: Use modifyImageLSBWithNonce for guaranteed uniqueness
func modifyImageLSB(data []byte, format string) ([]byte, error) {
	if len(data) == 0 {
		return data, fmt.Errorf("empty data")
	}

	// Only handle JPEG and PNG via stdlib to avoid adding deps
	if format != "jpeg" && format != "png" {
		return data, fmt.Errorf("format not supported for LSB modification: %s", format)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, fmt.Errorf("decode failed: %w", err)
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return data, fmt.Errorf("invalid dimensions")
	}

	// Choose up to 3 pixels near the center to avoid being removed by small crops
	cx := w / 2
	cy := h / 2
	coords := [][2]int{{cx, cy}, {cx + 1, cy}, {cx, cy + 1}}
	// Create an NRGBA image to edit pixels safely
	nimg := image.NewNRGBA(bounds)
	draw.Draw(nimg, bounds, img, bounds.Min, draw.Src)

	for _, c := range coords {
		x := c[0]
		y := c[1]
		if x >= w || y >= h {
			continue
		}
		off := nimg.PixOffset(x+bounds.Min.X, y+bounds.Min.Y)
		// Pix is [R,G,B,A]
		// Tweak each channel LSB by +/-1 randomly
		for i := 0; i < 3; i++ {
			v := nimg.Pix[off+i]
			if mathrand.Intn(2) == 0 {
				if v < 255 {
					v = v + 1
				}
			} else {
				if v > 0 {
					v = v - 1
				}
			}
			nimg.Pix[off+i] = v
		}
	}

	// Re-encode
	var buf bytes.Buffer
	if format == "jpeg" {
		if err := jpeg.Encode(&buf, nimg, &jpeg.Options{Quality: 95}); err != nil {
			return data, fmt.Errorf("jpeg encode failed: %w", err)
		}
	} else {
		if err := png.Encode(&buf, nimg); err != nil {
			return data, fmt.Errorf("png encode failed: %w", err)
		}
	}

	return buf.Bytes(), nil
}

func (ic *ImageConverter) ConvertWithScriptTechniques(ctx context.Context, inputData []byte, outputPath string) error {
	start := time.Now()

	if len(inputData) == 0 {
		return fmt.Errorf("empty input data")
	}

	// Generate unique nonce for this processing (guarantees uniqueness)
	nonce := GenerateNonce()
	
	// Create a local RNG seeded with nonce to ensure uniqueness
	localRand := mathrand.New(mathrand.NewSource(nonce.GetSeedForRand()))

	// Detect format
	inputFormat := ic.detectFormat(inputData)

	// Attempt LSB modification for formats we support
	// Pass nonce seed to ensure LSB modifications are unique
	if inputFormat == "jpeg" || inputFormat == "png" {
		if modified, err := modifyImageLSBWithNonce(inputData, inputFormat, nonce); err == nil {
			inputData = modified
		} else {
			// Log but continue with original data
			log.Printf("⚠️  LSB modification failed: %v", err)
		}
	}

	// Smart symmetric crop: 1-2 pixels (protected against tiny images)
	cropPixels := 1 + localRand.Intn(2) // 1 or 2
	
	// Add micro-variation from timestamp to ensure uniqueness
	cropVariation := int(nonce.Timestamp % 3) // 0-2
	cropPixels = (cropPixels + cropVariation) % 3
	if cropPixels == 0 {
		cropPixels = 1
	}
	
	// Use a safe min dimension constant to avoid cropping tiny images
	cropExprW := fmt.Sprintf("if(gt(iw\\,32)\\,iw-%d\\,iw)", cropPixels*2)
	cropExprH := fmt.Sprintf("if(gt(ih\\,32)\\,ih-%d\\,ih)", cropPixels*2)
	xExpr := "(iw-ow)/2"
	yExpr := "(ih-oh)/2"

	// MICRO-VARIATION DE GAMMA (0.995 - 1.005) for binary uniqueness
	gamma := 0.995 + localRand.Float64()*0.010
	
	// Add micro-variation from timestamp for absolute uniqueness
	gamma += float64(nonce.Timestamp%1000) / 1000000.0 // ±0.000999 additional variation
	if gamma > 1.005 {
		gamma = 1.005
	}
	
	vfilter := fmt.Sprintf("crop=w=%s:h=%s:x=%s:y=%s,eq=gamma=%.6f", cropExprW, cropExprH, xExpr, yExpr, gamma)

	// Use standard comment metadata field (more portable than custom tags) - includes nonce for guaranteed uniqueness
	uniqueComment := fmt.Sprintf("uid:%s", nonce.Nonce)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", "pipe:0",
		"-vf", vfilter,
		"-q:v", "2", // High quality for JPEG
		"-compression_level", "3",
		"-map_metadata", "-1",
		"-metadata", "comment="+uniqueComment,
		"-f", "image2",
		"-threads", "0",
		"pipe:1",
	)

	// Adjust for WebP: use -quality instead of -q:v
	if inputFormat == "webp" {
		newArgs := []string{}
		for i := 0; i < len(cmd.Args); i++ {
			arg := cmd.Args[i]
			// skip -q:v and its value if present
			if arg == "-q:v" {
				i++
				continue
			}
			newArgs = append(newArgs, arg)
		}
		cmd.Args = newArgs
		cmd.Args = append(cmd.Args, "-quality", "98")
	}

	cmd.Stdin = bytes.NewReader(inputData)
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &errorBuffer

	if err := cmd.Run(); err != nil {
		ic.recordFailure()
		return fmt.Errorf("ffmpeg error: %v, stderr: %s", err, errorBuffer.String())
	}

	output := outputBuffer.Bytes()
	if len(output) == 0 {
		ic.recordFailure()
		return fmt.Errorf("ffmpeg produced no output")
	}

	finalPath := ic.adjustOutputPath(outputPath, inputFormat)
	if err := os.WriteFile(finalPath, output, 0644); err != nil {
		ic.recordFailure()
		return fmt.Errorf("failed to write output file: %w", err)
	}

	ic.recordSuccess(time.Since(start))
	return nil
}

type imageParams struct {
	quality          int
	compressionLevel int
	jpegQScale       int
	addNoise         bool
	noiseStrength    int
	colorAdjust      bool
	brightness       float64
	contrast         float64
	addBlur          bool
	blurAmount       float64
}

func (ic *ImageConverter) getRandomizedParams(level string, format string) imageParams {
	params := imageParams{
		quality:          90,
		compressionLevel: 6,
		jpegQScale:       3,
	}

	// Adjust noise based on format (PNG is more sensitive)
	isPNG := (format == "png")

	switch level {
	case "basic":
		// Minimal randomization
		params.quality = 88 + mathrand.Intn(5)         // 88-92
		params.compressionLevel = 5 + mathrand.Intn(3) // 5-7
		params.jpegQScale = 3 + mathrand.Intn(2)       // 3-4

	case "moderate":
		// Moderate randomization (default, recommended)
		params.quality = 88 + mathrand.Intn(5)         // 88-92
		params.compressionLevel = 5 + mathrand.Intn(3) // 5-7
		params.jpegQScale = 3 + mathrand.Intn(2)       // 3-4
		params.addNoise = true
		if isPNG {
			params.noiseStrength = 1 + mathrand.Intn(2) // 1-2 (lower for PNG)
		} else {
			params.noiseStrength = 2 + mathrand.Intn(3) // 2-4
		}
		params.colorAdjust = true
		params.brightness = float64(mathrand.Intn(3)-1) / 1000.0   // ±0.001
		params.contrast = 1.0 + float64(mathrand.Intn(3)-1)/1000.0 // ±0.001

	case "paranoid":
		// Maximum randomization
		params.quality = 85 + mathrand.Intn(8)         // 85-92
		params.compressionLevel = 4 + mathrand.Intn(4) // 4-7
		params.jpegQScale = 2 + mathrand.Intn(3)       // 2-4
		params.addNoise = true
		if isPNG {
			params.noiseStrength = 1 + mathrand.Intn(3) // 1-3 (lower for PNG)
		} else {
			params.noiseStrength = 3 + mathrand.Intn(5) // 3-7
		}
		params.colorAdjust = true
		params.brightness = float64(mathrand.Intn(5)-2) / 1000.0   // ±0.002
		params.contrast = 1.0 + float64(mathrand.Intn(5)-2)/1000.0 // ±0.002
		params.addBlur = true
		params.blurAmount = 0.1 + float64(mathrand.Intn(5))/100.0 // 0.1-0.14

	default: // "none"
		params.quality = 90
		params.compressionLevel = 6
		params.jpegQScale = 3
	}

	return params
}

func (ic *ImageConverter) detectFormat(data []byte) string {
	if len(data) < 12 {
		return "unknown"
	}

	// PNG signature
	if bytes.Equal(data[0:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "png"
	}

	// JPEG signature
	if bytes.Equal(data[0:2], []byte{0xFF, 0xD8}) {
		return "jpeg"
	}

	// WebP signature
	if bytes.Equal(data[0:4], []byte{0x52, 0x49, 0x46, 0x46}) && bytes.Equal(data[8:12], []byte{0x57, 0x45, 0x42, 0x50}) {
		return "webp"
	}

	return "unknown"
}

func (ic *ImageConverter) adjustOutputPath(path, format string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)

	switch format {
	case "png":
		return base + ".png"
	case "webp":
		return base + ".webp"
	default:
		return base + ".jpg"
	}
}

func (ic *ImageConverter) recordSuccess(duration time.Duration) {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.stats.TotalConversions++
	ic.stats.AvgConversionTime = (ic.stats.AvgConversionTime*time.Duration(ic.stats.TotalConversions-1) + duration) / time.Duration(ic.stats.TotalConversions)
}

func (ic *ImageConverter) recordFailure() {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.stats.FailedConversions++
}

// GetStats returns current statistics
func (ic *ImageConverter) GetStats() ImageStats {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.stats
}

// GetOutputExtension returns the file extension for this converter
func (ic *ImageConverter) GetOutputExtension() string {
	return ".jpg" // Default, will be adjusted based on input format
}

// GenerateOutputPath creates a unique output path
func (ic *ImageConverter) GenerateOutputPath(cacheDir, deviceID, urlHash string) string {
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("%s_%s_%d%s", deviceID, urlHash[:8], timestamp, ic.GetOutputExtension())
	return filepath.Join(cacheDir, filename)
}
