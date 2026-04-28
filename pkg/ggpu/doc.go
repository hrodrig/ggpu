// Package ggpu implements a small, educational software graphics pipeline in
// pure Go: programmable vertex and fragment stages, indexed triangle lists,
// optional depth testing, and RGBA framebuffer output (including PNG export).
//
// # Scope
//
// It is not a hardware-accurate GPU, a CUDA/SIMT simulator, or a sandbox for
// untrusted shader bytecode; “shaders” are Go functions provided by your
// application.
//
// # Configuration
//
// Use [Config] with [New]. Important fields (the repository’s
// “docs/ARCHITECTURE.md” file goes into more detail):
//
//   - [Config.DepthTest] — per-fragment depth buffer (less-equal, smaller value wins
//     for the current NDC-to-viewport mapping).
//
//   - [Config.SequentialTileRaster] — if true, each 16×16 tile is filled in the
//     current goroutine; if false (default), each tile is scheduled on its own
//     goroutine. Sequential mode can reduce scheduling overhead for small scenes
//     and avoids goroutine-based surprises under the debugger. Parallel mode
//     illustrates “many work items” in teaching but is not a real GPU.
//
//   - [Config.DebugShowTileWorkload] — optional checkerboard color bias on tile
//     edges for screenshots; leave off for bit-stable output. Teaching aid only.
//
// # Buffers
//
// [GPU.Clear] fills the color buffer; if depth is enabled, depth is set to 1.0.
// This package has no separate stencil buffer (despite the classic GPU trio name).
//
// [GPU.SetUniforms] with a nil pointer does not change the previous uniform block;
// pass a non-nil [UniformBuffer] to replace it.
//
// # Documentation
//
// The repository root and docs/ contain teaching notes, presentation scripts,
// and security notes. Start with the root README and docs/ARCHITECTURE.md.
package ggpu
