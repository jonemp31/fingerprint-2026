package services

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"testing"
	"time"
)

func md5SumBytes(b []byte) string {
	h := md5.New()
	_, _ = h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}
func md5File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func TestImageUniqueness(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available, skipping image uniqueness test")
	}

	ic := NewImageConverter(nil, nil)

	// generate a small PNG via ffmpeg from raw color data
	tmpRaw := os.TempDir() + string(os.PathSeparator) + "uniq_raw.png"
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "color=c=blue:s=50x50", "-vframes", "1", tmpRaw)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to generate raw image: %v", err)
	}
	defer os.Remove(tmpRaw)

	rawData, err := os.ReadFile(tmpRaw)
	if err != nil {
		t.Fatalf("read raw failed: %v", err)
	}

	// Run two conversions
	out1 := os.TempDir() + string(os.PathSeparator) + "uniq_out1.jpg"
	out2 := os.TempDir() + string(os.PathSeparator) + "uniq_out2.jpg"
	defer os.Remove(out1)
	defer os.Remove(out2)

	// Quick check: LSB modification alone produces different bytes
	if modified1, err := modifyImageLSB(rawData, "png"); err == nil {
		if modified2, err2 := modifyImageLSB(rawData, "png"); err2 == nil {
			if md1 := md5SumBytes(modified1); md1 == md5SumBytes(modified2) {
				t.Fatalf("LSB modification produced identical bytes, unexpected")
			}
		}
	}

	if err := ic.ConvertWithScriptTechniques(context.Background(), rawData, out1); err != nil {
		t.Fatalf("ConvertWithScriptTechniques failed 1: %v", err)
	}

	// small sleep to allow RNG differences
	time.Sleep(10 * time.Millisecond)

	if err := ic.ConvertWithScriptTechniques(context.Background(), rawData, out2); err != nil {
		t.Fatalf("ConvertWithScriptTechniques failed 2: %v", err)
	}

	// account for potential extension change by converter (e.g., .png)
	paths1 := []string{out1, out1[:len(out1)-4] + ".png"}
	paths2 := []string{out2, out2[:len(out2)-4] + ".png"}
	var real1, real2 string
	for _, p := range paths1 {
		if _, err := os.Stat(p); err == nil {
			real1 = p
			break
		}
	}
	for _, p := range paths2 {
		if _, err := os.Stat(p); err == nil {
			real2 = p
			break
		}
	}
	if real1 == "" || real2 == "" {
		t.Fatalf("expected output files to exist: %s, %s (tried defaults)", out1, out2)
	}

	md1, err := md5File(real1)
	if err != nil {
		t.Fatalf("md5 out1: %v", err)
	}
	md2, err := md5File(real2)
	if err != nil {
		t.Fatalf("md5 out2: %v", err)
	}

	if md1 == md2 {
		t.Fatalf("expected different MD5 for unique processing, got same: %s", md1)
	}
}

func TestAudioUniqueness(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available, skipping audio uniqueness test")
	}

	ac := NewAudioConverter(nil, nil)

	// generate 0.5s sine wave 16000Hz mono 16-bit PCM in WAV
	dur := 0.5
	sr := 16000
	ns := int(float64(sr) * dur)
	buf := make([]byte, 44+ns*2)
	// WAV header
	copy(buf[0:], []byte("RIFF"))
	binary.LittleEndian.PutUint32(buf[4:], uint32(36+ns*2))
	copy(buf[8:], []byte("WAVEfmt "))
	binary.LittleEndian.PutUint32(buf[16:], 16)
	binary.LittleEndian.PutUint16(buf[20:], 1) // PCM
	binary.LittleEndian.PutUint16(buf[22:], 1) // mono
	binary.LittleEndian.PutUint32(buf[24:], uint32(sr))
	binary.LittleEndian.PutUint32(buf[28:], uint32(sr*2))
	binary.LittleEndian.PutUint16(buf[32:], 2)
	binary.LittleEndian.PutUint16(buf[34:], 16)
	copy(buf[36:], []byte("data"))
	binary.LittleEndian.PutUint32(buf[40:], uint32(ns*2))

	// samples
	for i := 0; i < ns; i++ {
		s := int16(30000 * math.Sin(2*math.Pi*440*float64(i)/float64(sr)))
		binary.LittleEndian.PutUint16(buf[44+i*2:], uint16(s))
	}

	out1 := os.TempDir() + string(os.PathSeparator) + "uniq_audio1.opus"
	out2 := os.TempDir() + string(os.PathSeparator) + "uniq_audio2.opus"
	defer os.Remove(out1)
	defer os.Remove(out2)

	if err := ac.ConvertWithScriptTechniques(context.Background(), buf, out1, "wav"); err != nil {
		t.Fatalf("audio convert 1 failed: %v", err)
	}

	// small jitter
	time.Sleep(10 * time.Millisecond)

	if err := ac.ConvertWithScriptTechniques(context.Background(), buf, out2, "wav"); err != nil {
		t.Fatalf("audio convert 2 failed: %v", err)
	}

	md1, err := md5File(out1)
	if err != nil {
		t.Fatalf("md5 out1: %v", err)
	}
	md2, err := md5File(out2)
	if err != nil {
		t.Fatalf("md5 out2: %v", err)
	}

	if md1 == md2 {
		t.Fatalf("expected different MD5 for unique audio processing, got same: %s", md1)
	}
}
