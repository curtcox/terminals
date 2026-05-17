package usecasevalidation

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

func safeArtifactName(value string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "frame"
	}
	return out
}

func artifactsRoot() string {
	// Walk up from the test binary's working directory to find the repo root.
	// Fall back to the current directory if not found.
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			return filepath.Join(dir, "artifacts")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "artifacts"
}

func gitCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return ""
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	encErr := enc.Encode(v)
	closeErr := f.Close()
	if encErr != nil {
		return encErr
	}
	return closeErr
}

func writeJSONL(path string, records []any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func assertionsToAny(assertions []AssertionRecord) []any {
	out := make([]any, len(assertions))
	for i, a := range assertions {
		out[i] = a
	}
	return out
}

func interactionsToAny(interactions []InteractionRecord) []any {
	out := make([]any, len(interactions))
	for i, interaction := range interactions {
		out[i] = interaction
	}
	return out
}

// encodeWAV wraps raw PCM16-LE samples in a RIFF/WAV container.
func encodeWAV(pcm []byte, sampleRate, channels int) []byte {
	const bitsPerSample = 16
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := len(pcm)
	totalSize := 36 + dataSize

	var buf bytes.Buffer
	buf.Grow(8 + totalSize)
	mustWrite := func(v any) {
		if err := binary.Write(&buf, binary.LittleEndian, v); err != nil {
			panic(fmt.Sprintf("encodeWAV: binary.Write: %v", err))
		}
	}
	buf.WriteString("RIFF")
	mustWrite(uint32(totalSize))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	mustWrite(uint32(16)) // chunk size
	mustWrite(uint16(1))  // PCM
	mustWrite(uint16(channels))
	mustWrite(uint32(sampleRate))
	mustWrite(uint32(byteRate))
	mustWrite(uint16(blockAlign))
	mustWrite(uint16(bitsPerSample))
	buf.WriteString("data")
	mustWrite(uint32(dataSize))
	buf.Write(pcm)
	return buf.Bytes()
}
