package apppackage

import (
	"errors"
	"testing"
)

func TestVerifyTapRejectsUnknownTopLevel(t *testing.T) {
	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/manifest.toml", body: "name='kitchen_timer'"},
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/secrets/key.txt", body: "shh"},
	})

	if _, err := VerifyTap(tap); err == nil {
		t.Fatalf("expected unknown top-level rejection")
	}
}

func TestVerifyTapRejectsPathTraversal(t *testing.T) {
	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/manifest.toml", body: "name='kitchen_timer'"},
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/../escape.txt", body: "oops"},
	})

	if _, err := VerifyTap(tap); err == nil {
		t.Fatalf("expected unsafe path rejection")
	}
}

func TestVerifyTapRejectsMissingMain(t *testing.T) {
	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/manifest.toml", body: "name='kitchen_timer'"},
	})

	if _, err := VerifyTap(tap); err != ErrMissingMainTAL {
		t.Fatalf("expected missing main.tal, got %v", err)
	}
}

func TestVerifyTapRejectsZstdChecksumFlag(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	mutated := append([]byte(nil), tapBytes...)
	mutated[4] |= 0x04

	if _, err := VerifyTap(mutated); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsZstdDictionaryIDFlag(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	mutated := append([]byte(nil), tapBytes...)
	mutated[4] = (mutated[4] &^ 0x03) | 0x01

	if _, err := VerifyTap(mutated); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsZstdMissingContentSizeFlag(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	mutated := append([]byte(nil), tapBytes...)
	mutated[4] = (mutated[4] &^ 0xC0) | 0x20

	if _, err := VerifyTap(mutated); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsZstdTrailingBytes(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	mutated := append(append([]byte(nil), tapBytes...), 0x00)

	if _, err := VerifyTap(mutated); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsZstdMultiframe(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	mutated := append(append([]byte(nil), tapBytes...), tapBytes...)

	if _, err := VerifyTap(mutated); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsSkippableFrameMagic(t *testing.T) {
	skippable := []byte{0x50, 0x2A, 0x4D, 0x18, 0x00, 0x00, 0x00, 0x00}

	if _, err := VerifyTap(skippable); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsZstdWindowTooLarge(t *testing.T) {
	tapBytes := []byte{
		0x28, 0xB5, 0x2F, 0xFD, // zstd frame magic
		0x40,       // FCS flag=1, single segment=0, checksum=0, dict ID=0
		0x70,       // window descriptor => window log 24 (>23)
		0x00, 0x01, // frame content size field (2 bytes for FCS flag=1)
		0x01, 0x00, 0x00, // last raw block, size 0
	}

	if _, err := VerifyTap(tapBytes); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}
