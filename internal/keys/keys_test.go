package keys

import (
	"bytes"
	"strings"
	"testing"
)

func TestMaxUnbiasedByteMatchesCharset(t *testing.T) {
	want := 256 / len(Charset) * len(Charset)
	if MaxUnbiasedByte != want {
		t.Fatalf("MaxUnbiasedByte = %d, want %d (floor(256/%d)*%d)", MaxUnbiasedByte, want, len(Charset), len(Charset))
	}
	if MaxUnbiasedByte%len(Charset) != 0 {
		t.Fatalf("MaxUnbiasedByte %d is not a multiple of the charset size %d", MaxUnbiasedByte, len(Charset))
	}
}

func TestGenerate_RejectedBytesDoNotConsumeOutputSlots(t *testing.T) {
	// Eight rejectable bytes (>= 248), then the bytes that map onto the first 48
	// charset entries, then filler for the generator's 10% over-read.
	source := make([]byte, 0, 128)
	for value := MaxUnbiasedByte; value <= 255; value++ {
		source = append(source, byte(value))
	}
	for index := range RandomLength {
		source = append(source, byte(index))
	}
	source = append(source, make([]byte, 128-len(source))...)

	generated, err := NewGeneratorWithSource(bytes.NewReader(source)).Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	want := Prefix + Charset[:RandomLength]
	if generated != want {
		t.Fatalf("Generate() = %q, want %q", generated, want)
	}
}

func TestGenerate_WrapsOnCharsetBoundary(t *testing.T) {
	source := make([]byte, 0, 128)
	source = append(source, byte(len(Charset)), byte(MaxUnbiasedByte-1))
	source = append(source, make([]byte, 128-len(source))...)

	generated, err := NewGeneratorWithSource(bytes.NewReader(source)).Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated[3] != Charset[0] {
		t.Fatalf("byte %d mapped to %q, want charset index 0 (%q)", len(Charset), generated[3], Charset[0])
	}
	if generated[4] != Charset[len(Charset)-1] {
		t.Fatalf("byte %d mapped to %q, want charset index %d", MaxUnbiasedByte-1, generated[4], len(Charset)-1)
	}
}

func TestGenerate_ExhaustedSourceFails(t *testing.T) {
	rejected := bytes.Repeat([]byte{255}, 64)
	if _, err := NewGeneratorWithSource(bytes.NewReader(rejected)).Generate(); err == nil {
		t.Fatal("Generate() error = nil, want an error when the source only yields rejected bytes")
	}
}

func TestGenerate_ShapeMatchesOfficialFormat(t *testing.T) {
	generator := NewGenerator()

	first, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.HasPrefix(first, Prefix) {
		t.Fatalf("key %q does not start with %q", first, Prefix)
	}
	if got := len(first); got != len(Prefix)+RandomLength {
		t.Fatalf("key length = %d, want %d", got, len(Prefix)+RandomLength)
	}
	for _, char := range strings.TrimPrefix(first, Prefix) {
		if !strings.ContainsRune(Charset, char) {
			t.Fatalf("key %q contains %q which is outside the official charset", first, char)
		}
	}

	second, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if first == second {
		t.Fatalf("two generated keys are identical: %q", first)
	}
}

func TestGenerate_NilGeneratorFails(t *testing.T) {
	var generator *Generator
	if _, err := generator.Generate(); err == nil {
		t.Fatal("Generate() on a nil generator returned no error")
	}
}
