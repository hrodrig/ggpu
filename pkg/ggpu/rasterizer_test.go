package ggpu

import (
	"image"
	"testing"
)

func TestClearAndSize(t *testing.T) {
	e := newEngine(4, 4, false, false, false)
	c := Color{R: 10, G: 20, B: 30, A: 40}
	e.clear(c)
	img := e.framebuffer()
	if img.Bounds() != image.Rect(0, 0, 4, 4) {
		t.Fatalf("bounds: %v", img.Bounds())
	}
	if img.Pix[0] != 10 || img.Pix[1] != 20 || img.Pix[2] != 30 || img.Pix[3] != 40 {
		t.Fatalf("first pixel: %v", img.Pix[:4])
	}
}

func TestDrawSingleTriangle(t *testing.T) {
	g := New(Config{Width: 32, Height: 32, DepthTest: true})
	vs := func(in VertexInput, u *UniformBuffer) VertexOutput {
		clip := Vec4Transform(u.ModelViewProjection, in.Position)
		return VertexOutput{ClipPos: clip, Color: in.Color}
	}
	fs := func(in FragmentInput, u *UniformBuffer) Vec4 {
		return in.Color
	}
	g.SetShaders(vs, fs)
	ortho := Ortho2D(0, 32, 32, 0, -1, 1)
	var u UniformBuffer
	u.ModelViewProjection = ortho
	g.SetUniforms(&u)
	g.Clear(Color{})

	verts := []VertexInput{
		{Position: Vec4{X: 4, Y: 4, Z: 0, W: 1}, Color: Vec4{X: 1, Y: 0, Z: 0, W: 1}},
		{Position: Vec4{X: 28, Y: 4, Z: 0, W: 1}, Color: Vec4{X: 0, Y: 1, Z: 0, W: 1}},
		{Position: Vec4{X: 16, Y: 28, Z: 0, W: 1}, Color: Vec4{X: 0, Y: 0, Z: 1, W: 1}},
	}
	g.DrawIndexed(verts, []uint16{0, 1, 2})

	img := g.Framebuffer()
	// (16, 12) lies inside the filled triangle (not on an edge).
	p := img.Pix[(16+12*32)*4 : (16+12*32)*4+4]
	if p[0] < 50 || p[1] < 50 || p[2] < 50 {
		t.Fatalf("expected non-dark blended color inside triangle, got %v", p)
	}
}

// With Ortho2D(0,32,32,0,-1,1) and the row-major Z row in Ortho2D, positive model Z
// maps to nearer depth (smaller buffer value) as verified below.
func TestDepthNearerWins(t *testing.T) {
	g := New(Config{Width: 32, Height: 32, DepthTest: true})
	vs := func(in VertexInput, u *UniformBuffer) VertexOutput {
		clip := Vec4Transform(u.ModelViewProjection, in.Position)
		return VertexOutput{ClipPos: clip, Color: in.Color}
	}
	fs := func(in FragmentInput, u *UniformBuffer) Vec4 { return in.Color }
	g.SetShaders(vs, fs)
	var u UniformBuffer
	u.ModelViewProjection = Ortho2D(0, 32, 32, 0, -1, 1)
	g.SetUniforms(&u)
	g.Clear(Color{})

	// Draw farther triangle first (red), then nearer (green); same screen footprint.
	backZ, frontZ := float32(-0.9), float32(0.9)
	red := []VertexInput{
		{Position: Vec4{X: 4, Y: 4, Z: backZ, W: 1}, Color: Vec4{X: 1, Y: 0, Z: 0, W: 1}},
		{Position: Vec4{X: 28, Y: 4, Z: backZ, W: 1}, Color: Vec4{X: 1, Y: 0, Z: 0, W: 1}},
		{Position: Vec4{X: 16, Y: 28, Z: backZ, W: 1}, Color: Vec4{X: 1, Y: 0, Z: 0, W: 1}},
	}
	green := []VertexInput{
		{Position: Vec4{X: 4, Y: 4, Z: frontZ, W: 1}, Color: Vec4{X: 0, Y: 1, Z: 0, W: 1}},
		{Position: Vec4{X: 28, Y: 4, Z: frontZ, W: 1}, Color: Vec4{X: 0, Y: 1, Z: 0, W: 1}},
		{Position: Vec4{X: 16, Y: 28, Z: frontZ, W: 1}, Color: Vec4{X: 0, Y: 1, Z: 0, W: 1}},
	}
	g.DrawIndexed(red, []uint16{0, 1, 2})
	g.DrawIndexed(green, []uint16{0, 1, 2})
	img := g.Framebuffer()
	p := img.Pix[(16+12*32)*4 : (16+12*32)*4+4]
	if p[1] < p[0] { // should be more green than red on the chosen pixel
		t.Fatalf("expected nearer (green) to win depth; got R=%d G=%d B=%d", p[0], p[1], p[2])
	}
}

func TestPerspectiveVaryingWInterpolates(t *testing.T) {
	// One vertex with W=2 and distinct per-vertex colors: pixel-center attributes
	// are perspective-correct; sequential raster so sampling is explainable in docs.
	g := New(Config{Width: 64, Height: 64, DepthTest: false, SequentialTileRaster: true})
	vs := func(in VertexInput, u *UniformBuffer) VertexOutput {
		clip := Vec4Transform(u.ModelViewProjection, in.Position)
		return VertexOutput{ClipPos: clip, Color: in.Color}
	}
	fs := func(in FragmentInput, u *UniformBuffer) Vec4 { return in.Color }
	g.SetShaders(vs, fs)
	var u UniformBuffer
	u.ModelViewProjection = Ortho2D(0, 64, 64, 0, -1, 1)
	g.SetUniforms(&u)
	g.Clear(Color{})

	verts := []VertexInput{
		{Position: Vec4{X: 0, Y: 0, Z: 0, W: 1}, Color: Vec4{X: 1, Y: 0, Z: 0, W: 1}},
		{Position: Vec4{X: 64, Y: 0, Z: 0, W: 1}, Color: Vec4{X: 0, Y: 1, Z: 0, W: 1}},
		{Position: Vec4{X: 32, Y: 64, Z: 0, W: 2}, Color: Vec4{X: 0, Y: 0, Z: 1, W: 1}},
	}
	g.DrawIndexed(verts, []uint16{0, 1, 2})
	img := g.Framebuffer()
	// (32, 20) is inside the triangle: expect a true blend, not a solid primary.
	p := img.Pix[(32+20*64)*4 : (32+20*64)*4+4]
	if p[0] == 0 || p[1] == 0 || p[2] == 0 {
		t.Fatalf("expected blended non-zero RGB at (32,20), got %v", p)
	}
	if p[0] > 250 && p[1] < 5 && p[2] < 5 {
		t.Fatalf("interior should not match solid red; got %v (affine vs persp check)", p)
	}
}

func TestOutOfRangeIndicesIgnored(t *testing.T) {
	g := New(Config{Width: 8, Height: 8, DepthTest: false})
	vs := func(in VertexInput, u *UniformBuffer) VertexOutput {
		return VertexOutput{ClipPos: Vec4{X: 0, Y: 0, Z: 0, W: 1}, Color: in.Color}
	}
	g.SetShaders(vs, func(FragmentInput, *UniformBuffer) Vec4 { return Vec4{W: 1} })
	tri := []VertexInput{
		{Position: Vec4{W: 1}, Color: Vec4{W: 1}},
	}
	// index 0 out of range for len(verts)==0 — should not panic
	g.DrawIndexed([]VertexInput{}, []uint16{0, 0, 0})
	// one vertex, still invalid
	g.SetUniforms(&UniformBuffer{ModelViewProjection: Ortho2D(0, 8, 8, 0, -1, 1)})
	g.DrawIndexed(tri, []uint16{0, 1, 2})
}

func TestSetUniformsNilKeepsPrevious(t *testing.T) {
	g := New(Config{Width: 4, Height: 4, DepthTest: false, SequentialTileRaster: true})
	ortho := Ortho2D(0, 4, 4, 0, -1, 1)
	var u1 UniformBuffer
	u1.ModelViewProjection = ortho
	g.SetShaders(
		func(in VertexInput, u *UniformBuffer) VertexOutput {
			clip := Vec4Transform(u.ModelViewProjection, in.Position)
			return VertexOutput{ClipPos: clip, Color: in.Color}
		},
		func(FragmentInput, *UniformBuffer) Vec4 { return Vec4{X: 0.9, Y: 0.1, Z: 0.1, W: 1} },
	)
	g.SetUniforms(&u1)
	g.SetUniforms(nil) // no-op: keep u1
	g.Clear(Color{})
	verts := []VertexInput{
		{Position: Vec4{0, 0, 0, 1}, Color: Vec4{W: 1}},
		{Position: Vec4{4, 0, 0, 1}, Color: Vec4{W: 1}},
		{Position: Vec4{2, 4, 0, 1}, Color: Vec4{W: 1}},
	}
	g.DrawIndexed(verts, []uint16{0, 1, 2})
	// (2,2) is inside: something must have been drawn; black would mean MVP was not applied
	img := g.Framebuffer()
	p := img.Pix[(2+2*4)*4 : (2+2*4)*4+4]
	if p[0] == 0 && p[1] == 0 && p[2] == 0 {
		t.Fatalf("expected lit fragment after SetUniforms(nil), got black pixel %v", p)
	}
}
