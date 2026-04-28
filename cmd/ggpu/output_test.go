package main

import (
	"bytes"
	"testing"
)

func TestSelectOutputBackend_png(t *testing.T) {
	t.Helper()
	var stderr bytes.Buffer
	b, err := selectOutputBackend("png", outputOptions{
		outPath: "work/demo.png",
		stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b == nil {
		t.Fatal("backend is nil")
	}
}

func TestSelectOutputBackend_liveWeb(t *testing.T) {
	t.Helper()
	var stderr bytes.Buffer
	b, err := selectOutputBackend("live-web", outputOptions{
		listenAddr: "127.0.0.1:8080",
		maxFPS:     30,
		stderr:     &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b == nil {
		t.Fatal("backend is nil")
	}
}

func TestSelectOutputBackend_liveWebInvalidFPS(t *testing.T) {
	t.Helper()
	var stderr bytes.Buffer
	b, err := selectOutputBackend("live-web", outputOptions{
		listenAddr: "127.0.0.1:8080",
		maxFPS:     0,
		stderr:     &stderr,
	})
	if err == nil {
		t.Fatal("expected error for invalid max-fps")
	}
	if err != errInvalidArgs {
		t.Fatalf("want errInvalidArgs, got %v", err)
	}
	if b != nil {
		t.Fatal("backend should be nil for invalid config")
	}
	if stderr.Len() == 0 {
		t.Fatal("expected error message on stderr")
	}
}

func TestSelectOutputBackend_invalidMode(t *testing.T) {
	t.Helper()
	var stderr bytes.Buffer
	b, err := selectOutputBackend("invalid", outputOptions{
		outPath: "work/demo.png",
		stderr:  &stderr,
	})
	if err == nil {
		t.Fatal("expected error for unsupported output mode")
	}
	if err != errInvalidArgs {
		t.Fatalf("want errInvalidArgs, got %v", err)
	}
	if b != nil {
		t.Fatal("backend should be nil for unsupported mode")
	}
	if stderr.Len() == 0 {
		t.Fatal("expected error message on stderr")
	}
}
