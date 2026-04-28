package ggpu

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
)

// Config holds emulator creation options.
type Config struct {
	Width  int
	Height int
	// DepthTest enables per-fragment depth comparison (less-equal; smaller
	// depth wins for the current NDC-to-viewport mapping).
	DepthTest bool
	// SequentialTileRaster, when true, processes each 16×16 screen tile in the
	// caller’s goroutine instead of launching a new goroutine per tile. The
	// default (false) uses parallel tile workers for potentially faster
	// large-triangle coverage; sequential mode reduces goroutine overhead and
	// is easier to reason about under a debugger. See docs/ARCHITECTURE.md.
	SequentialTileRaster bool
	// DebugShowTileWorkload, when true, applies a small checkerboard color bias
	// on 16×16 tile boundaries so parallel tile boundaries are visible. Leave
	// false for bit-stable output from the rest of the pipeline. Teaching-only;
	// see docs/ARCHITECTURE.md.
	DebugShowTileWorkload bool
}

// GPU is the public facade over the software rasterizer. It is not safe
// for concurrent use from multiple goroutines; protect externally if needed.
type GPU struct {
	eng *engine
}

// New allocates a new GPU with the given dimensions. Width and height must be positive.
func New(cfg Config) *GPU {
	if cfg.Width < 1 {
		cfg.Width = 1
	}
	if cfg.Height < 1 {
		cfg.Height = 1
	}
	return &GPU{eng: newEngine(
		cfg.Width, cfg.Height, cfg.DepthTest,
		cfg.SequentialTileRaster, cfg.DebugShowTileWorkload,
	)}
}

// SetViewport sets the internal viewport to the full framebuffer (default).
// Reserved for future sub-viewport and scissor support.
func (g *GPU) SetViewport(x, y, w, h int) {
	g.eng.setViewport(x, y, w, h)
}

// Clear fills the color buffer with c. If depth testing is on, the depth
// buffer is set to 1.0. There is no separate stencil buffer in this package.
func (g *GPU) Clear(c Color) {
	g.eng.clear(c)
}

// SetShaders registers vertex and fragment stages for subsequent draws.
func (g *GPU) SetShaders(vs VertexShader, fs FragmentShader) {
	g.eng.setShaders(vs, fs)
}

// SetUniforms updates the uniform buffer for the next draw. If u is nil, the
// previous uniform values are left unchanged; call SetUniforms with a
// non-nil buffer before the first draw that needs explicit uniforms.
func (g *GPU) SetUniforms(u *UniformBuffer) {
	g.eng.setUniforms(u)
}

// DrawIndexed draws a triangle list from interleaved vertex attributes and
// 16-bit indices. indices length must be a multiple of 3. Panics on nil shaders.
func (g *GPU) DrawIndexed(verts []VertexInput, indices []uint16) {
	g.eng.drawIndexed(verts, indices)
}

// Framebuffer returns the color buffer as image.RGBA (top-left origin). The
// returned value aliases internal storage until the next Clear or DrawIndexed;
// copy the image if you need to retain it across those calls.
func (g *GPU) Framebuffer() *image.RGBA {
	return g.eng.framebuffer()
}

// WriteFramebufferPNG encodes the current color buffer to w as PNG.
func (g *GPU) WriteFramebufferPNG(w io.Writer) error {
	return png.Encode(w, g.Framebuffer())
}

// TotalTiles returns how many 16×16 screen tiles have been scheduled for
// raster work since engine creation. In parallel mode each tile runs in its
// own goroutine; in sequential mode (Config.SequentialTileRaster) the same
// count is incremented for each tile processed in the same goroutine. Useful
// for teaching only; not a real GPU tile counter.
func (g *GPU) TotalTiles() uint64 {
	return g.eng.totalTiles
}

// Ortho2D returns a row-major orthographic projection [left, right] x [bottom, top] x [near, far]
// in NDC-style space used by the emulator (Y down for screen: use top>bottom for typical UI).
func Ortho2D(left, right, bottom, top, near, far float32) [16]float32 {
	rl := 1 / (right - left)
	tb := 1 / (top - bottom)
	fn := 1 / (far - near)
	// Row-major: v' = v * M. Last column holds translations (X,Y,Z) and w scale in m15.
	// W must stay 1 for positions with w=1; translations belong on rows 0–2, not row 3 alone.
	return [16]float32{
		2 * rl, 0, 0, -(right + left) * rl,
		0, 2 * tb, 0, -(top + bottom) * tb,
		0, 0, -2 * fn, -(far + near) * fn,
		0, 0, 0, 1,
	}
}

// MulMat4 multiplies two row-major 4x4 matrices (A * B).
func MulMat4(a, b [16]float32) (out [16]float32) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			out[i*4+j] = a[i*4+0]*b[0*4+j] + a[i*4+1]*b[1*4+j] + a[i*4+2]*b[2*4+j] + a[i*4+3]*b[3*4+j]
		}
	}
	return out
}

// Vec4Transform applies a row-major matrix to a homogeneous vector.
func Vec4Transform(m [16]float32, v Vec4) Vec4 {
	return Vec4{
		X: m[0]*v.X + m[1]*v.Y + m[2]*v.Z + m[3]*v.W,
		Y: m[4]*v.X + m[5]*v.Y + m[6]*v.Z + m[7]*v.W,
		Z: m[8]*v.X + m[9]*v.Y + m[10]*v.Z + m[11]*v.W,
		W: m[12]*v.X + m[13]*v.Y + m[14]*v.Z + m[15]*v.W,
	}
}

// Saturate clamps components to [0,1] and is useful in fragment outputs.
func Saturate(v Vec4) Vec4 {
	return Vec4{
		X: float32(math.Min(1, math.Max(0, float64(v.X)))),
		Y: float32(math.Min(1, math.Max(0, float64(v.Y)))),
		Z: float32(math.Min(1, math.Max(0, float64(v.Z)))),
		W: float32(math.Min(1, math.Max(0, float64(v.W)))),
	}
}

// ColorToNRGBA8 converts 0..1 Vec4 to 8-bit NRGBA (straight alpha).
func ColorToNRGBA8(c Vec4) color.NRGBA {
	s := Saturate(c)
	return color.NRGBA{
		R: uint8(s.X * 255),
		G: uint8(s.Y * 255),
		B: uint8(s.Z * 255),
		A: uint8(s.W * 255),
	}
}
