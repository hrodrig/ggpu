package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRun_version(t *testing.T) {
	code := run([]string{"-version"})
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestRun_invalidDimensions(t *testing.T) {
	if code := run([]string{"-w", "0", "-h", "1"}); code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
}

func TestRun_invalidOutputMode(t *testing.T) {
	if code := run([]string{"-output", "invalid-mode"}); code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
}

func TestRun_invalidMaxFPS(t *testing.T) {
	if code := run([]string{"-output", "live-web", "-max-fps", "0"}); code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
}

func TestRun_stressPath(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "s.png")
	if code := run([]string{"-w", "64", "-h", "64", "-stress", "2", "-out", out}); code != 0 {
		t.Fatalf("exit %d", code)
	}
}

func TestRun_rendersPNG(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.png")
	if code := run([]string{"-w", "32", "-h", "32", "-out", out}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// PNG signature
	if len(b) < 8 || !bytes.Equal(b[:8], []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}) {
		nb := 8
		if len(b) < nb {
			nb = len(b)
		}
		t.Fatalf("not a png: %v", b[:nb])
	}
}
