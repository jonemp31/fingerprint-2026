package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mathrand "math/rand"
	"time"
)

// Seed RNG once per process to ensure randomness across converters
func init() {
	mathrand.Seed(time.Now().UnixNano())
}

// ProcessingNonce represents a unique identifier for each processing operation
type ProcessingNonce struct {
	Timestamp int64
	Random    string // 16 bytes hex-encoded (32 chars)
	Nonce     string // Combined: timestamp_random (guaranteed unique)
}

// GenerateNonce creates a unique nonce for each processing operation
// Combines timestamp (nanoseconds) + crypto/rand (secure random)
// This guarantees uniqueness even for simultaneous processing
func GenerateNonce() *ProcessingNonce {
	now := time.Now().UnixNano()
	
	// Generate 16 random bytes using crypto/rand (secure, not predictable)
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to math/rand if crypto/rand fails (very rare)
		mathrand.Read(randomBytes)
	}
	randomHex := hex.EncodeToString(randomBytes)
	
	return &ProcessingNonce{
		Timestamp: now,
		Random:    randomHex,
		Nonce:     fmt.Sprintf("%d_%s", now, randomHex),
	}
}

// GetSeedForRand returns a seed value derived from the nonce
// This ensures that math/rand produces different values even if called at the same time
func (n *ProcessingNonce) GetSeedForRand() int64 {
	// Use nonce hash as additional seed
	hash := int64(0)
	for _, b := range []byte(n.Nonce) {
		hash = hash*31 + int64(b)
	}
	return n.Timestamp ^ hash
}
