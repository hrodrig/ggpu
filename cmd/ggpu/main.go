// The ggpu command exercises the ggpu software rasterizer: a perspective-correct
// gradient triangle is rendered to a PNG. Default output is work/demo.png (see -out).
// Binary name: ggpu (see Makefile; -version shows VERSION + git metadata from ldflags).
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/hrodrig/ggpu/pkg/ggpu"
)

// Injected at link time (make, CI, Docker). Defaults support go run without ldflags.
var (
	version = "0.0.0-dev"
	commit  = "unknown"
	branch  = "unknown"
)

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
	out := fs.String("out", "work/demo.png", "output PNG path (default under ./work)")
	timeSec := fs.Float64("t", 0, "optional time (seconds) uniform for fragment tint demo")
	stress := fs.Int("stress", 0, "number of random triangles to render for performance test")
	sequentialTiles := fs.Bool("sequential-tiles", false, "raster 16x16 tiles on one goroutine (no per-tile goroutines)")
	debugTiles := fs.Bool("debug-tiles", false, "show subtle checkerboard on tile edges (teaching; Config.DebugShowTileWorkload)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Printf("ggpu %s (commit: %s, branch: %s)\n", version, commit, branch)
		return 0
	}

	if *width < 1 || *height < 1 {
		fmt.Fprintln(os.Stderr, "width and height must be positive")
		return 2
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
	var u ggpu.UniformBuffer
	u.ModelViewProjection = ortho
	u.TimeSeconds = float32(*timeSec)
	g.SetUniforms(&u)
	g.Clear(ggpu.Color{R: 16, G: 18, B: 28, A: 255})

	start := time.Now()

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
	} else {
		// Counter-clockwise in screen space (Y down): gradient corners.
		verts := []ggpu.VertexInput{
			{Position: ggpu.Vec4{X: wf * 0.1, Y: hf * 0.12, Z: 0, W: 1},
				Color: ggpu.Vec4{X: 0.95, Y: 0.2, Z: 0.25, W: 1}},
			{Position: ggpu.Vec4{X: wf * 0.9, Y: hf * 0.18, Z: 0, W: 1},
				Color: ggpu.Vec4{X: 0.2, Y: 0.95, Z: 0.3, W: 1}},
			{Position: ggpu.Vec4{X: wf * 0.5, Y: hf * 0.92, Z: 0, W: 1},
				Color: ggpu.Vec4{X: 0.25, Y: 0.35, Z: 1, W: 1}},
		}
		g.DrawIndexed(verts, []uint16{0, 1, 2})
	}

	elapsed := time.Since(start)
	fmt.Printf("Rendered in %v\n", elapsed)
	fmt.Printf("Total 16x16 tile work units scheduled: %d\n", g.TotalTiles())

	if d := filepath.Dir(filepath.Clean(*out)); d != "." {
		if err := os.MkdirAll(d, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = f.Close() }()
	if err := g.WriteFramebufferPNG(f); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("Wrote", *out, "at", *width, "x", *height)
	return 0
}
