package main

import (
	"bytes"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/hrodrig/ggpu/pkg/ggpu"
)

func TestLiveWebOutputBackend_listenAddressInUse(t *testing.T) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	g := ggpu.New(ggpu.Config{Width: 8, Height: 8, DepthTest: true})
	g.Clear(ggpu.Color{R: 0, G: 0, B: 0, A: 255})

	var stderr bytes.Buffer
	backend, err := selectOutputBackend("live-web", outputOptions{
		listenAddr: ln.Addr().String(),
		maxFPS:     30,
		renderFrame: func(_ float64) int {
			return 1
		},
		width:     8,
		height:    8,
		stopAfter: 50 * time.Millisecond,
		stderr:    &stderr,
	})
	if err != nil {
		t.Fatalf("select backend: %v", err)
	}

	if _, err := backend.Write(g); err == nil {
		t.Fatal("expected listen error when address is already in use")
	}
}

func TestLiveWebOutputBackend_servesFrameEndpoint(t *testing.T) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	g := ggpu.New(ggpu.Config{Width: 16, Height: 16, DepthTest: true})
	g.Clear(ggpu.Color{R: 16, G: 18, B: 28, A: 255})

	var stderr bytes.Buffer
	backend, err := selectOutputBackend("live-web", outputOptions{
		listenAddr: addr,
		maxFPS:     20,
		renderFrame: func(_ float64) int {
			return 1
		},
		width:     16,
		height:    16,
		stopAfter: 300 * time.Millisecond,
		stderr:    &stderr,
	})
	if err != nil {
		t.Fatalf("select backend: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, writeErr := backend.Write(g)
		done <- writeErr
	}()

	client := &http.Client{Timeout: 150 * time.Millisecond}
	var resp *http.Response
	for i := 0; i < 8; i++ {
		resp, err = client.Get("http://" + addr + "/frame.png")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("request frame endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if got := resp.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("want image/png content-type, got %q", got)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want status 200, got %d", resp.StatusCode)
	}

	select {
	case writeErr := <-done:
		if writeErr != nil {
			t.Fatalf("backend write returned error: %v", writeErr)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for backend shutdown")
	}
}
