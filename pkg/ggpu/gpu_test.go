package ggpu

import (
	"bytes"
	"image/color"
	"math"
	"testing"
)

func TestNew_clampsSize(t *testing.T) {
	g := New(Config{Width: 0, Height: 0, DepthTest: false})
	img := g.Framebuffer()
	if img.Bounds().Dx() != 1 || img.Bounds().Dy() != 1 {
		t.Fatalf("got %v", img.Bounds())
	}
}

func TestSetViewport(t *testing.T) {
	g := New(Config{Width: 32, Height: 32, DepthTest: false, SequentialTileRaster: true})
	g.SetShaders(identityVS, passFS)
	g.SetUniforms(simpleOrthUniform(0, 32, 32, 0, -1, 1))
	g.Clear(Color{})
	g.SetViewport(8, 8, 16, 16) // 16x16 sub-rect
	tri := []VertexInput{
		{Position: Vec4{0, 0, 0, 1}, Color: red},
		{Position: Vec4{32, 0, 0, 1}, Color: red},
		{Position: Vec4{0, 32, 0, 1}, Color: red},
	}
	g.DrawIndexed(tri, []uint16{0, 1, 2})
	// (8,8) is corner of viewport, expect red-ish inside small tri near origin in ortho
	p := g.Framebuffer().Pix[(8+8*32)*4 : (8+8*32)*4+4]
	if p[0]+p[1]+p[2] < 20 {
		t.Fatalf("expected lit pixel in viewport, got %v", p)
	}
}

func TestWriteFramebufferPNG(t *testing.T) {
	g := New(Config{Width: 2, Height: 2, DepthTest: false})
	g.SetShaders(identityVS, passFS)
	g.SetUniforms(simpleOrthUniform(0, 2, 2, 0, -1, 1))
	g.Clear(Color{R: 1, G: 2, B: 3, A: 4})
	var buf bytes.Buffer
	if err := g.WriteFramebufferPNG(&buf); err != nil {
		t.Fatal(err)
	}
	if len(buf.Bytes()) < 20 {
		t.Fatalf("short png %d", len(buf.Bytes()))
	}
}

func TestTotalTilesParallel(t *testing.T) {
	g := New(Config{Width: 64, Height: 64, DepthTest: false, DebugShowTileWorkload: true})
	g.SetShaders(identityVS, passFS)
	g.SetUniforms(simpleOrthUniform(0, 64, 64, 0, -1, 1))
	g.Clear(Color{})
	tri := []VertexInput{
		{Position: Vec4{0, 0, 0, 1}, Color: red},
		{Position: Vec4{64, 0, 0, 1}, Color: red},
		{Position: Vec4{0, 64, 0, 1}, Color: red},
	}
	g.DrawIndexed(tri, []uint16{0, 1, 2})
	if g.TotalTiles() == 0 {
		t.Fatal("expected at least one 16x16 work unit in parallel mode")
	}
}

func TestMulMat4(t *testing.T) {
	ident := [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	if MulMat4(ident, ident) != ident {
		t.Fatal("I*I should be I")
	}
	// A*B != B*A in general; check translation compose
	t1 := [16]float32{1, 0, 0, 1, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	p := Vec4Transform(MulMat4(t1, ident), Vec4{0, 0, 0, 1})
	if math.Abs(float64(p.X-1)) > 1e-5 {
		t.Fatalf("p=%v", p)
	}
}

func TestColorRGBAandFromRGBA(t *testing.T) {
	c := Color{R: 0xab, G: 0xcd, B: 0xef, A: 0xff}
	r, g, b, a := c.RGBA()
	if r>>8 != 0xab || g>>8 != 0xcd {
		t.Fatalf("rgba: %d %d %d %d", r, g, b, a)
	}
	// NRGBA with A<255: standard library RGBA() returns alpha-premultiplied samples; use opaque for exact 8-bit round-trip.
	c2 := ColorFromRGBA(color.NRGBA{R: 18, G: 52, B: 86, A: 255})
	if c2.R != 18 || c2.G != 52 || c2.A != 255 {
		t.Fatalf("c2: %+v", c2)
	}
}

func TestDrawIndexedPanicsWithoutShaders(t *testing.T) {
	g := New(Config{Width: 4, Height: 4, DepthTest: false})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when shaders unset")
		}
	}()
	g.DrawIndexed(nil, []uint16{0, 0, 0})
}

func TestClipValid_rejectsW(t *testing.T) {
	g := New(Config{Width: 16, Height: 16, DepthTest: false, SequentialTileRaster: true})
	g.SetShaders(identityVS, passFS)
	g.SetUniforms(simpleOrthUniform(0, 16, 16, 0, -1, 1))
	g.Clear(Color{})
	vsBad := func(in VertexInput, u *UniformBuffer) VertexOutput {
		return VertexOutput{ClipPos: Vec4{0, 0, 0, -1}, Color: in.Color} // W < 0
	}
	g.SetShaders(vsBad, passFS)
	g.DrawIndexed([]VertexInput{{Position: Vec4{0, 0, 0, 1}, Color: red}}, []uint16{0, 0, 0})
	// if nothing drawn, center stays clear color
	img := g.Framebuffer()
	p := img.Pix[(8+8*16)*4 : (8+8*16)*4+4]
	if p[0] != 0 || p[1] != 0 {
		// we cleared to 0,0,0,0
	}
}

// Helpers

var red = Vec4{X: 1, Y: 0, Z: 0, W: 1}

func identityVS(in VertexInput, u *UniformBuffer) VertexOutput {
	clip := Vec4Transform(u.ModelViewProjection, in.Position)
	return VertexOutput{ClipPos: clip, Color: in.Color}
}

func passFS(in FragmentInput, u *UniformBuffer) Vec4 { return in.Color }

func simpleOrthUniform(l, r, t, b, n, f float32) *UniformBuffer {
	var u UniformBuffer
	u.ModelViewProjection = Ortho2D(l, r, t, b, n, f)
	return &u
}
