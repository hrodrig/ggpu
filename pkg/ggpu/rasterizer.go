package ggpu

import (
	"image"
	"math"
	"sync"
	"sync/atomic"
)

// engine holds framebuffer, optional depth, shaders, and viewport. It is the
// software implementation of the graphics pipeline (unexported by design).
type engine struct {
	w, h int
	vp   viewport

	color []uint8
	depth []float32

	depthTest bool

	// When true, rasterize each 16×16 tile in the current goroutine (no per-tile goroutines).
	sequentialTileRaster bool
	// When true, apply optional checkerboard color bias in tiles (teaching aid).
	debugShowTileWorkload bool

	vs       VertexShader
	fs       FragmentShader
	uniforms UniformBuffer

	totalTiles uint64
}

type viewport struct {
	x0, y0, w, h int
}

func newEngine(w, h int, depthTest bool, sequentialTileRaster, debugShowTileWorkload bool) *engine {
	e := &engine{
		w:                     w,
		h:                     h,
		vp:                    viewport{x0: 0, y0: 0, w: w, h: h},
		color:                 make([]uint8, w*h*4),
		depth:                 make([]float32, w*h),
		depthTest:             depthTest,
		sequentialTileRaster:  sequentialTileRaster,
		debugShowTileWorkload: debugShowTileWorkload,
	}
	if depthTest {
		e.clearDepth(1)
	}
	return e
}

func (e *engine) setViewport(x, y, w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+w > e.w {
		w = e.w - x
	}
	if y+h > e.h {
		h = e.h - y
	}
	e.vp = viewport{x0: x, y0: y, w: w, h: h}
}

func (e *engine) clear(c Color) {
	for i := 0; i < e.w*e.h; i++ {
		o := i * 4
		e.color[o+0] = c.R
		e.color[o+1] = c.G
		e.color[o+2] = c.B
		e.color[o+3] = c.A
	}
	if e.depthTest {
		e.clearDepth(1)
	}
}

func (e *engine) clearDepth(z float32) {
	for i := range e.depth {
		e.depth[i] = z
	}
}

func (e *engine) setShaders(vs VertexShader, fs FragmentShader) {
	e.vs = vs
	e.fs = fs
}

// setUniforms copies *u when u is non-nil; a nil pointer leaves the previous
// uniform snapshot unchanged.
func (e *engine) setUniforms(u *UniformBuffer) {
	if u != nil {
		e.uniforms = *u
	}
}

func (e *engine) drawIndexed(verts []VertexInput, indices []uint16) {
	if e.vs == nil || e.fs == nil {
		panic("ggpu: SetShaders required before DrawIndexed")
	}
	if len(indices) < 3 || len(indices)%3 != 0 {
		return
	}
	for t := 0; t < len(indices); t += 3 {
		e.rasterFromIndices(verts, int(indices[t]), int(indices[t+1]), int(indices[t+2]))
	}
}

func (e *engine) rasterFromIndices(verts []VertexInput, i0, i1, i2 int) {
	if i0 < 0 || i1 < 0 || i2 < 0 || i0 >= len(verts) || i1 >= len(verts) || i2 >= len(verts) {
		return
	}
	tri := triangle{
		v0: e.vs(verts[i0], &e.uniforms),
		v1: e.vs(verts[i1], &e.uniforms),
		v2: e.vs(verts[i2], &e.uniforms),
	}
	if !clipValid(tri.v0.ClipPos) || !clipValid(tri.v1.ClipPos) || !clipValid(tri.v2.ClipPos) {
		return
	}
	e.rasterTriangle(tri)
}

func clipValid(c Vec4) bool {
	if c.W <= 1e-6 {
		return false
	}
	ndcX, ndcY, ndcZ := c.X/c.W, c.Y/c.W, c.Z/c.W
	if math.Abs(float64(ndcX)) > 1.1 || math.Abs(float64(ndcY)) > 1.1 || math.Abs(float64(ndcZ)) > 1.1 {
		return false
	}
	return true
}

type triangle struct {
	v0, v1, v2 VertexOutput
}

type screenVert struct {
	sx, sy   float32
	invW     float32
	depth    float32
	color    Vec4
	colorMul Vec4
}

func (e *engine) toScreen(vo VertexOutput) screenVert {
	c := vo.ClipPos
	invW := 1 / c.W
	ndcX := c.X * invW
	ndcY := c.Y * invW
	ndcZ := c.Z * invW
	sx := (ndcX*0.5+0.5)*float32(e.vp.w) + float32(e.vp.x0)
	sy := (0.5-ndcY*0.5)*float32(e.vp.h) + float32(e.vp.y0)
	dep := ndcZ*0.5 + 0.5
	col := vo.Color
	return screenVert{
		sx:       sx,
		sy:       sy,
		invW:     invW,
		depth:    dep,
		color:    col,
		colorMul: Vec4{X: col.X * invW, Y: col.Y * invW, Z: col.Z * invW, W: col.W * invW},
	}
}

func (e *engine) rasterTriangle(tri triangle) {
	s0, s1, s2, x0, y0, x1, y1, x2, y2, area2, ok := e.triangleScreenSpace(tri)
	if !ok {
		return
	}
	minX, maxX, minY, maxY, ok2 := e.clampTriangleBoundsToViewport(x0, y0, x1, y1, x2, y2)
	if !ok2 {
		return
	}

	const tileSize = 16
	launch := func(tx, ty int) {
		e.rasterTileBlock(tx, ty, tileSize, maxX, maxY, x0, y0, x1, y1, x2, y2, s0, s1, s2, area2)
	}
	e.forEachScreenTile(minX, maxX, minY, maxY, tileSize, launch)
}

// triangleScreenSpace returns screen vertices, 2D positions, 2x signed area, and false if degenerate.
func (e *engine) triangleScreenSpace(tri triangle) (s0, s1, s2 screenVert, x0, y0, x1, y1, x2, y2, area2 float32, ok bool) {
	s0 = e.toScreen(tri.v0)
	s1 = e.toScreen(tri.v1)
	s2 = e.toScreen(tri.v2)
	x0, y0 = s0.sx, s0.sy
	x1, y1 = s1.sx, s1.sy
	x2, y2 = s2.sx, s2.sy
	area2 = edge(x0, y0, x1, y1, x2, y2)
	if area2 == 0 {
		return
	}
	// Y-down: negative signed area is common; swap one edge for consistent barycentrics.
	if area2 < 0 {
		s1, s2 = s2, s1
		x0, y0 = s0.sx, s0.sy
		x1, y1 = s1.sx, s1.sy
		x2, y2 = s2.sx, s2.sy
		area2 = edge(x0, y0, x1, y1, x2, y2)
		if area2 == 0 {
			return
		}
	}
	ok = true
	return
}

func (e *engine) clampTriangleBoundsToViewport(x0, y0, x1, y1, x2, y2 float32) (minX, maxX, minY, maxY int, ok bool) {
	minX = int(math.Floor(float64(min3(x0, x1, x2))))
	maxX = int(math.Ceil(float64(max3(x0, x1, x2))))
	minY = int(math.Floor(float64(min3(y0, y1, y2))))
	maxY = int(math.Ceil(float64(max3(y0, y1, y2))))
	if minX < e.vp.x0 {
		minX = e.vp.x0
	}
	if minY < e.vp.y0 {
		minY = e.vp.y0
	}
	if maxX > e.vp.x0+e.vp.w-1 {
		maxX = e.vp.x0 + e.vp.w - 1
	}
	if maxY > e.vp.y0+e.vp.h-1 {
		maxY = e.vp.y0 + e.vp.h - 1
	}
	if minX > maxX || minY > maxY {
		return
	}
	return minX, maxX, minY, maxY, true
}

func (e *engine) forEachScreenTile(minX, maxX, minY, maxY, tileSize int, launch func(tx, ty int)) {
	if e.sequentialTileRaster {
		for ty := minY; ty <= maxY; ty += tileSize {
			for tx := minX; tx <= maxX; tx += tileSize {
				e.totalTiles++
				launch(tx, ty)
			}
		}
		return
	}
	var wg sync.WaitGroup
	for ty := minY; ty <= maxY; ty += tileSize {
		for tx := minX; tx <= maxX; tx += tileSize {
			wg.Add(1)
			atomic.AddUint64(&e.totalTiles, 1)
			tx, ty := tx, ty
			go func() {
				defer wg.Done()
				launch(tx, ty)
			}()
		}
	}
	wg.Wait()
}

// rasterTileBlock covers a 16×16-aligned block starting at (tx,ty), clipped
// to x <= maxX and y <= maxY, using the same barycentric coverage as the
// parent triangle. Vertex coordinates (x0..y2) are the 2D screen positions
// after winding normalization.
func (e *engine) rasterTileBlock(
	tx, ty, tileSize int,
	maxX, maxY int,
	x0, y0, x1, y1, x2, y2 float32,
	s0, s1, s2 screenVert,
	area2 float32,
) {
	for y := ty; y < ty+tileSize && y <= maxY; y++ {
		for x := tx; x < tx+tileSize && x <= maxX; x++ {
			px := float32(x) + 0.5
			py := float32(y) + 0.5
			w0 := edge(x1, y1, x2, y2, px, py)
			w1 := edge(x2, y2, x0, y0, px, py)
			w2 := edge(x0, y0, x1, y1, px, py)
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}
			b0 := w0 / area2
			b1 := w1 / area2
			b2 := w2 / area2

			sumInvW := b0*s0.invW + b1*s1.invW + b2*s2.invW
			if sumInvW <= 0 {
				continue
			}
			invSum := 1 / sumInvW
			interpC := Vec4{
				X: (b0*s0.colorMul.X + b1*s1.colorMul.X + b2*s2.colorMul.X) * invSum,
				Y: (b0*s0.colorMul.Y + b1*s1.colorMul.Y + b2*s2.colorMul.Y) * invSum,
				Z: (b0*s0.colorMul.Z + b1*s1.colorMul.Z + b2*s2.colorMul.Z) * invSum,
				W: (b0*s0.colorMul.W + b1*s1.colorMul.W + b2*s2.colorMul.W) * invSum,
			}
			if e.debugShowTileWorkload {
				// Subtle checkerboard on tile coordinates (teaching: visible tile boundaries).
				tileTint := float32((tx/tileSize+ty/tileSize)%2) * 0.05
				interpC.X += tileTint
				interpC.Y += tileTint
			}

			interpZ := b0*s0.depth + b1*s1.depth + b2*s2.depth

			xi, yi := x, y
			if e.depthTest {
				idx := xi + yi*e.w
				if interpZ > e.depth[idx] {
					continue
				}
			}
			fin := e.fs(FragmentInput{
				PixelX: xi, PixelY: yi,
				Color: interpC,
				Depth: interpZ,
			}, &e.uniforms)
			n := ColorToNRGBA8(fin)
			if e.depthTest {
				e.depth[xi+yi*e.w] = interpZ
			}
			p := (xi + yi*e.w) * 4
			e.color[p+0] = n.R
			e.color[p+1] = n.G
			e.color[p+2] = n.B
			e.color[p+3] = n.A
		}
	}
}

func edge(x0, y0, x1, y1, px, py float32) float32 {
	return (px-x0)*(y1-y0) - (py-y0)*(x1-x0)
}

func min3(a, b, c float32) float32 {
	return float32(math.Min(float64(a), math.Min(float64(b), float64(c))))
}

func max3(a, b, c float32) float32 {
	return float32(math.Max(float64(a), math.Max(float64(b), float64(c))))
}

func (e *engine) framebuffer() *image.RGBA {
	return &image.RGBA{
		Pix:    e.color,
		Stride: 4 * e.w,
		Rect:   image.Rect(0, 0, e.w, e.h),
	}
}
