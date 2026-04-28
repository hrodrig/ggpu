package ggpu

// UniformBuffer holds user-defined shader constants (set once per draw call).
// The emulator copies this by value; keep it small and POD-like.
type UniformBuffer struct {
	// ModelViewProjection is a 4x4 row-major transform: row i = elements [i*4..i*4+3].
	ModelViewProjection [16]float32
	// TimeSeconds can drive animations in demos.
	TimeSeconds float32
}

// VertexShader transforms one vertex. Must be non-nil for DrawIndexed.
type VertexShader func(in VertexInput, u *UniformBuffer) VertexOutput

// FragmentShader produces one output color (premultiplied or straight RGBA in 0..1).
type FragmentShader func(in FragmentInput, u *UniformBuffer) Vec4
