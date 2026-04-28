package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/hrodrig/ggpu/pkg/ggpu"
)

type outputBackend interface {
	Write(*ggpu.GPU) (string, error)
}

type pngOutputBackend struct {
	path string
}

func (b pngOutputBackend) Write(g *ggpu.GPU) (string, error) {
	if d := filepath.Dir(filepath.Clean(b.path)); d != "." {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return "", err
		}
	}
	f, err := os.Create(b.path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	if err := g.WriteFramebufferPNG(f); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wrote %s", b.path), nil
}

type outputOptions struct {
	outPath     string
	listenAddr  string
	maxFPS      int
	openBrowser bool
	renderFrame func(seconds float64) int
	width       int
	height      int
	stopAfter   time.Duration
	stderr      io.Writer
}

type liveWebOutputBackend struct {
	listenAddr  string
	maxFPS      int
	openBrowser bool
	renderFrame func(seconds float64) int
	width       int
	height      int
	stopAfter   time.Duration
}

func (b liveWebOutputBackend) Write(g *ggpu.GPU) (string, error) {
	store := newFrameStore()
	if _, err := store.writeCurrentFrame(g); err != nil {
		return "", err
	}
	refreshMS := refreshIntervalMS(b.maxFPS)
	srv := b.newLiveWebServer(refreshMS, store)
	errCh := startHTTPServer(srv)
	stopRender := make(chan struct{})
	renderErrCh := b.startRenderLoop(g, refreshMS, store, stopRender)
	announceLiveURL(b.listenAddr, b.openBrowser)
	return b.waitForStop(srv, errCh, renderErrCh, stopRender)
}

type frameStore struct {
	mu    sync.RWMutex
	bytes []byte
}

func newFrameStore() *frameStore {
	return &frameStore{}
}

func (s *frameStore) writeCurrentFrame(g *ggpu.GPU) (time.Duration, error) {
	encStart := time.Now()
	var frame bytes.Buffer
	if err := g.WriteFramebufferPNG(&frame); err != nil {
		return 0, err
	}
	s.mu.Lock()
	s.bytes = append(s.bytes[:0], frame.Bytes()...)
	s.mu.Unlock()
	return time.Since(encStart), nil
}

func (s *frameStore) snapshot() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]byte(nil), s.bytes...)
}

func refreshIntervalMS(maxFPS int) int {
	refreshMS := 1000 / maxFPS
	if refreshMS < 1 {
		return 1
	}
	return refreshMS
}

func (b liveWebOutputBackend) newLiveWebServer(refreshMS int, store *frameStore) *http.Server {
	html := []byte(fmt.Sprintf(liveWebHTMLTemplate, refreshMS, refreshMS))
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(html)
	})
	mux.HandleFunc("/frame.png", func(w http.ResponseWriter, _ *http.Request) {
		data := store.snapshot()
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(data)
	})
	return &http.Server{
		Addr:              b.listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func startHTTPServer(srv *http.Server) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	return errCh
}

func (b liveWebOutputBackend) startRenderLoop(g *ggpu.GPU, refreshMS int, store *frameStore, stopRender <-chan struct{}) <-chan error {
	renderErrCh := make(chan error, 1)
	go func() {
		if b.renderFrame == nil {
			renderErrCh <- nil
			return
		}
		start := time.Now()
		ticker := time.NewTicker(time.Duration(refreshMS) * time.Millisecond)
		metricsTicker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		defer metricsTicker.Stop()
		var (
			frameCount      uint64
			trianglesSecond uint64
			renderAccum     time.Duration
			encodeAccum     time.Duration
			lastTiles       = g.TotalTiles()
		)
		for {
			select {
			case <-stopRender:
				renderErrCh <- nil
				return
			case <-ticker.C:
				renderStart := time.Now()
				triangles := b.renderFrame(time.Since(start).Seconds())
				renderDur := time.Since(renderStart)
				encodeDur, err := store.writeCurrentFrame(g)
				if err != nil {
					renderErrCh <- err
					return
				}
				frameCount++
				trianglesSecond += uint64(triangles)
				renderAccum += renderDur
				encodeAccum += encodeDur
			case <-metricsTicker.C:
				if frameCount == 0 {
					continue
				}
				avgRenderMS := float64(renderAccum.Microseconds()) / 1000.0 / float64(frameCount)
				avgEncodeMS := float64(encodeAccum.Microseconds()) / 1000.0 / float64(frameCount)
				tileNow := g.TotalTiles()
				tileDelta := tileNow - lastTiles
				lastTiles = tileNow
				fmt.Printf(
					"[metrics] fps=%d triangles/s=%d frame_px=%d tile_work=%d avg_render_ms=%.2f avg_encode_ms=%.2f\n",
					frameCount,
					trianglesSecond,
					b.width*b.height,
					tileDelta,
					avgRenderMS,
					avgEncodeMS,
				)
				frameCount = 0
				trianglesSecond = 0
				renderAccum = 0
				encodeAccum = 0
			}
		}
	}()
	return renderErrCh
}

func announceLiveURL(listenAddr string, open bool) {
	url := "http://" + listenAddr
	fmt.Printf("Live preview at %s (Ctrl+C to stop)\n", url)
	if open {
		_ = openBrowser(url)
	}
}

func (b liveWebOutputBackend) waitForStop(srv *http.Server, errCh, renderErrCh <-chan error, stopRender chan struct{}) (string, error) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	var stopTimer <-chan time.Time
	if b.stopAfter > 0 {
		timer := time.NewTimer(b.stopAfter)
		defer timer.Stop()
		stopTimer = timer.C
	}

	select {
	case err := <-errCh:
		b.stopRenderLoop(stopRender, renderErrCh)
		if err == http.ErrServerClosed {
			return "Live preview stopped", nil
		}
		return "", err
	case err := <-renderErrCh:
		_ = stopHTTPServer(srv)
		if err != nil {
			return "", err
		}
		return "Live preview stopped", nil
	case <-sigCh:
		b.stopRenderLoop(stopRender, renderErrCh)
		if err := stopHTTPServer(srv); err != nil {
			return "", err
		}
		return "Live preview stopped", nil
	case <-stopTimer:
		b.stopRenderLoop(stopRender, renderErrCh)
		if err := stopHTTPServer(srv); err != nil {
			return "", err
		}
		return "Live preview stopped", nil
	}
}

func (b liveWebOutputBackend) stopRenderLoop(stopRender chan struct{}, renderErrCh <-chan error) {
	close(stopRender)
	_ = <-renderErrCh
}

func stopHTTPServer(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

func selectOutputBackend(mode string, opts outputOptions) (outputBackend, error) {
	switch mode {
	case "png":
		return pngOutputBackend{path: opts.outPath}, nil
	case "live-web":
		if opts.listenAddr == "" {
			fmt.Fprintln(opts.stderr, "listen address cannot be empty")
			return nil, errInvalidArgs
		}
		if opts.maxFPS < 1 {
			fmt.Fprintln(opts.stderr, "max-fps must be positive")
			return nil, errInvalidArgs
		}
		return liveWebOutputBackend{
			listenAddr:  opts.listenAddr,
			maxFPS:      opts.maxFPS,
			openBrowser: opts.openBrowser,
			renderFrame: opts.renderFrame,
			width:       opts.width,
			height:      opts.height,
			stopAfter:   opts.stopAfter,
		}, nil
	default:
		fmt.Fprintf(opts.stderr, "unsupported output mode %q (supported: png, live-web)\n", mode)
		return nil, errInvalidArgs
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

const liveWebHTMLTemplate = `<!doctype html>
<html>
  <head>
    <meta charset="utf-8">
    <title>ggpu live preview</title>
    <style>
      body { margin: 0; background: #101010; color: #e5e7eb; font-family: sans-serif; }
      .wrap { display: grid; place-items: center; min-height: 100vh; gap: 8px; }
      canvas { image-rendering: pixelated; border: 1px solid #2f3640; }
    </style>
  </head>
  <body>
    <div class="wrap">
      <canvas id="cv"></canvas>
      <small>Refreshing every ~%d ms. Press Ctrl+C in terminal to stop.</small>
    </div>
    <script>
      const c = document.getElementById("cv");
      const ctx = c.getContext("2d");
      const img = new Image();
      function fitCanvasToViewport() {
        if (!c.width || !c.height) return;
        const maxW = window.innerWidth * 0.96;
        const maxH = window.innerHeight * 0.88;
        const scale = Math.min(maxW / c.width, maxH / c.height);
        if (!Number.isFinite(scale) || scale <= 0) return;
        c.style.width = Math.floor(c.width * scale) + "px";
        c.style.height = Math.floor(c.height * scale) + "px";
      }
      async function tick() {
        img.src = "/frame.png?t=" + Date.now();
      }
      img.onload = () => {
        const srcW = img.naturalWidth || img.width;
        const srcH = img.naturalHeight || img.height;
        if (srcW && srcH && (!c.width || !c.height || c.width !== srcW || c.height !== srcH)) {
          c.width = srcW;
          c.height = srcH;
          fitCanvasToViewport();
        }
        fitCanvasToViewport();
        ctx.drawImage(img, 0, 0);
      };
      window.addEventListener("resize", fitCanvasToViewport);
      setInterval(tick, %d);
      tick();
    </script>
  </body>
</html>`
