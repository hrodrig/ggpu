// The ggpu command exercises the ggpu software rasterizer: a perspective-correct
// gradient triangle is rendered to a PNG. Default output is work/demo.png (see -out).
// Binary name: ggpu (see Makefile; -version shows VERSION + git metadata from ldflags).
package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/hrodrig/ggpu/pkg/ggpu"
)

// Injected at link time (make, CI, Docker). Defaults support go run without ldflags.
var (
	version = "0.0.0-dev"
	commit  = "unknown"
	branch  = "unknown"
)

const (
	exitOK         = 0
	exitIOError    = 1
	exitInvalidArg = 2
)

var errInvalidArgs = fmt.Errorf("invalid arguments")

func main() {
	os.Exit(run(os.Args[1:]))
}

// run parses CLI args (not including argv[0]). It returns a process exit code: 0 ok, 1 I/O error, 2 bad args.
func run(args []string) int {
	fs := flag.NewFlagSet("ggpu", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	width := fs.Int("w", 800, "framebuffer width in pixels")
	height := fs.Int("h", 600, "framebuffer height in pixels")
	output := fs.String("output", "png", "output mode (supported: png, live-web)")
	out := fs.String("out", "work/demo.png", "output PNG path (default under ./work)")
	listen := fs.String("listen", "127.0.0.1:8080", "listen address for live-web output")
	maxFPS := fs.Int("max-fps", 30, "maximum preview refresh rate for live-web output")
	openBrowser := fs.Bool("open-browser", false, "open live-web preview in default browser")
	timeSec := fs.Float64("t", 0, "optional time (seconds) uniform for fragment tint demo")
	stress := fs.Int("stress", 0, "number of random triangles to render for performance test")
	sequentialTiles := fs.Bool("sequential-tiles", false, "raster 16x16 tiles on one goroutine (no per-tile goroutines)")
	debugTiles := fs.Bool("debug-tiles", false, "show subtle checkerboard on tile edges (teaching; Config.DebugShowTileWorkload)")
	if err := fs.Parse(args); err != nil {
		return exitInvalidArg
	}
	if *showVersion {
		fmt.Printf("ggpu %s (commit: %s, branch: %s)\n", version, commit, branch)
		return exitOK
	}

	if *width < 1 || *height < 1 {
		fmt.Fprintln(os.Stderr, "width and height must be positive")
		return exitInvalidArg
	}

	g := ggpu.New(ggpu.Config{
		Width: *width, Height: *height, DepthTest: true,
		SequentialTileRaster:  *sequentialTiles,
		DebugShowTileWorkload: *debugTiles,
	})

	vs := func(in ggpu.VertexInput, u *ggpu.UniformBuffer) ggpu.VertexOutput {
		clip := ggpu.Vec4Transform(u.ModelViewProjection, in.Position)
		return ggpu.VertexOutput{ClipPos: clip, Color: in.Color}
	}
	fsShader := func(in ggpu.FragmentInput, u *ggpu.UniformBuffer) ggpu.Vec4 {
		return ggpu.Saturate(ggpu.Vec4{
			X: in.Color.X + 0.05*u.TimeSeconds,
			Y: in.Color.Y,
			Z: in.Color.Z,
			W: in.Color.W,
		})
	}
	g.SetShaders(vs, fsShader)

	wf, hf := float32(*width), float32(*height)
	ortho := ggpu.Ortho2D(0, wf, hf, 0, -1, 1)
	renderFrame := func(seconds float64) int {
		var u ggpu.UniformBuffer
		u.ModelViewProjection = ortho
		u.TimeSeconds = float32(seconds)
		g.SetUniforms(&u)
		g.Clear(ggpu.Color{R: 16, G: 18, B: 28, A: 255})

		if *stress > 0 {
			for i := 0; i < *stress; i++ {
				x, y := rand.Float32()*wf, rand.Float32()*hf
				sz := rand.Float32()*100 + 10
				v := []ggpu.VertexInput{
					{Position: ggpu.Vec4{X: x, Y: y, Z: rand.Float32(), W: 1}, Color: ggpu.Vec4{X: rand.Float32(), Y: rand.Float32(), Z: rand.Float32(), W: 1}},
					{Position: ggpu.Vec4{X: x + sz, Y: y, Z: rand.Float32(), W: 1}, Color: ggpu.Vec4{X: rand.Float32(), Y: rand.Float32(), Z: rand.Float32(), W: 1}},
					{Position: ggpu.Vec4{X: x + sz/2, Y: y + sz, Z: rand.Float32(), W: 1}, Color: ggpu.Vec4{X: rand.Float32(), Y: rand.Float32(), Z: rand.Float32(), W: 1}},
				}
				g.DrawIndexed(v, []uint16{0, 1, 2})
			}
			return *stress
		}

		cx, cy := wf*0.5, hf*0.5
		rot := float32(seconds * 0.8)
		cosR, sinR := float32(math.Cos(float64(rot))), float32(math.Sin(float64(rot)))
		rotate := func(x, y float32) (float32, float32) {
			rx := x*cosR - y*sinR
			ry := x*sinR + y*cosR
			return rx + cx, ry + cy
		}
		ax, ay := rotate(-wf*0.35, -hf*0.22)
		bx, by := rotate(wf*0.35, -hf*0.16)
		cxv, cyv := rotate(0, hf*0.34)

		verts := []ggpu.VertexInput{
			{Position: ggpu.Vec4{X: ax, Y: ay, Z: 0, W: 1},
				Color: ggpu.Vec4{X: 0.95, Y: 0.2, Z: 0.25, W: 1}},
			{Position: ggpu.Vec4{X: bx, Y: by, Z: 0, W: 1},
				Color: ggpu.Vec4{X: 0.2, Y: 0.95, Z: 0.3, W: 1}},
			{Position: ggpu.Vec4{X: cxv, Y: cyv, Z: 0, W: 1},
				Color: ggpu.Vec4{X: 0.25, Y: 0.35, Z: 1, W: 1}},
		}
		g.DrawIndexed(verts, []uint16{0, 1, 2})
		return 1
	}

	start := time.Now()
	trianglesPerFrame := renderFrame(*timeSec)

	elapsed := time.Since(start)
	fmt.Printf("Rendered in %v\n", elapsed)
	fmt.Printf("Total 16x16 tile work units scheduled: %d\n", g.TotalTiles())
	fmt.Printf("Triangles processed: %d\n", trianglesPerFrame)
	fmt.Printf("Framebuffer pixels: %d\n", (*width)*(*height))

	backend, err := selectOutputBackend(*output, outputOptions{
		outPath:     *out,
		listenAddr:  *listen,
		maxFPS:      *maxFPS,
		openBrowser: *openBrowser,
		renderFrame: renderFrame,
		width:       *width,
		height:      *height,
		stderr:      os.Stderr,
	})
	if err != nil {
		if err == errInvalidArgs {
			return exitInvalidArg
		}
		fmt.Fprintln(os.Stderr, err)
		return exitIOError
	}
	msg, err := backend.Write(g)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitIOError
	}
	if msg != "" {
		fmt.Printf("%s at %d x %d\n", msg, *width, *height)
	}
	return exitOK
}
