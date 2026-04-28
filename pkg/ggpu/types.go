// Package ggpu provides a software GPU emulator API: programmable vertex and
// fragment stages, triangle rasterization, depth testing, and framebuffer output.
package ggpu

import "image/color"

// Vec2 is a two-component vector used for screen-space and texture coordinates.
type Vec2 struct {
	X, Y float32
}

// Vec3 is a three-component vector (e.g. normals, barycentric weights).
type Vec3 struct {
	X, Y, Z float32
}

// Vec4 is a four-component vector (homogeneous positions, RGBA).
type Vec4 struct {
	X, Y, Z, W float32
}

// Color is an 8-bit RGBA color for clear and framebuffer packing.
type Color struct {
	R, G, B, A uint8
}

// RGBA returns a standard library color.Color.
func (c Color) RGBA() (r, g, b, a uint32) {
	return uint32(c.R) * 0x101, uint32(c.G) * 0x101, uint32(c.B) * 0x101, uint32(c.A) * 0x101
}

// ColorFromRGBA converts color.Color to Color (lossy 8-bit).
func ColorFromRGBA(c color.Color) Color {
	r16, g16, b16, a16 := c.RGBA()
	return Color{
		R: uint8(r16 >> 8),
		G: uint8(g16 >> 8),
		B: uint8(b16 >> 8),
		A: uint8(a16 >> 8),
	}
}

// VertexInput is one vertex supplied to the input assembler.
type VertexInput struct {
	Position Vec4
	Color    Vec4
	// TexCoord is reserved for future texture sampling in the fragment stage.
	TexCoord Vec2
}

// VertexOutput is the result of the vertex shader (clip-space position + varyings).
type VertexOutput struct {
	ClipPos Vec4
	// Varyings interpolated per-fragment (here: primary color).
	Color Vec4
}

// FragmentInput is passed to the fragment shader after rasterizer interpolation.
type FragmentInput struct {
	// Screen pixel coordinates (integer grid, top-left origin).
	PixelX, PixelY int
	// Interpolated vertex color (perspective-correct if W differs per vertex).
	Color Vec4
	// Depth in [0,1] after viewport (for depth test).
	Depth float32
}
