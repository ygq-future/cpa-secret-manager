// Package keys generates proxy API keys with the same algorithm as the official
// CLIProxyAPI management center: an "sk-" prefix followed by 48 characters drawn
// uniformly from [A-Za-z0-9] through rejection sampling.
package keys

import (
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"io"
)

const (
	// Prefix is the leading marker shared with the official generator.
	Prefix = "sk-"
	// RandomLength is the number of random characters after the prefix.
	RandomLength = 48
	// Charset is the official character set, in the official order.
	Charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	// MaxUnbiasedByte is the first byte value that would introduce modulo bias
	// when reduced onto Charset: floor(256 / 62) * 62 = 248. Bytes at or above
	// it are rejected so every output character stays equally likely. The
	// relationship is locked by TestMaxUnbiasedByteMatchesCharset.
	MaxUnbiasedByte = 248
)

// Generator produces proxy API keys from a random byte source.
type Generator struct {
	source io.Reader
}

// NewGenerator returns a generator backed by crypto/rand.
func NewGenerator() *Generator {
	return &Generator{source: cryptorand.Reader}
}

// NewGeneratorWithSource returns a generator reading from a custom source. It is
// used by tests to verify rejection sampling deterministically.
func NewGeneratorWithSource(source io.Reader) *Generator {
	return &Generator{source: source}
}

// Generate returns one API key.
func (g *Generator) Generate() (string, error) {
	if g == nil || g.source == nil {
		return "", errors.New("keys: generator has no random source")
	}

	out := make([]byte, 0, RandomLength)
	for len(out) < RandomLength {
		remaining := RandomLength - len(out)
		// Match the official request size: a small over-read keeps the loop
		// short while rejected bytes are rare (8 in 256).
		chunk := make([]byte, (remaining*11+9)/10)
		if _, err := io.ReadFull(g.source, chunk); err != nil {
			return "", fmt.Errorf("keys: read random bytes: %w", err)
		}
		for _, value := range chunk {
			if value >= MaxUnbiasedByte {
				continue
			}
			out = append(out, Charset[value%byte(len(Charset))])
			if len(out) == RandomLength {
				break
			}
		}
	}
	return Prefix + string(out), nil
}
