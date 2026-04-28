[← Back to main README](../README.md)

# Teaching notes (graphics vs compute, and common misconceptions)

## Two “GPU” families students confuse

| Aspect | *Graphics* pipeline (this project) | *Compute* (CUDA / OpenCL style) |
|--------|-------------------------------------|----------------------------------|
| Work unit | Primitives (triangles) + *fragments* | *Threads* in a grid of blocks |
| Program form | Usually vertex + fragment stages (here: Go functions) | *Kernel* |
| Memory story | Interpolated *varyings*, framebuffer, depth | Global / shared / local / registers |
| Parallelism model | *Massively parallel* over pixels (SIMD-like) | *SIMT* (lock-step warps) |

`ggpu` is a **rasterization teaching tool**. It is **not** a CUDA simulator. If the syllabus needs a separate **dense linear-algebra** angle, use the small **`ggmat`** CLI in this repo (`cmd/ggmat`) or add your own lab package—keep the *graphics* story in `pkg/ggpu`.

## What we simplified (on purpose)

- **No** instruction-level ISA: shaders are Go closures (easy to read, not representative of real GPU assembly).
- **No** frustum clipping of triangles: vertices outside a loose NDC box are *culled* as a block (good enough for demos; a full clipper is a separate lesson).
- **No** blending / MSAA: opaque overwrites; alpha is written as 8-bit straight color.
- **No** texturing, tessellation, or compute: extensions for future assignments.

## Perspective-correct interpolation

The rasterizer interpolates *color* using **1/w** weighting in screen space so that, when per-vertex *W* differs, attributes remain approximately correct. This is the same *idea* as in fixed-function hardware, implemented in a minimal way for teaching.

## Why “goroutine per tile” is only a story

A real GPU schedules **thousands** of *lightweight* threads in warps. In `ggpu`, the **default** raster path launches **one goroutine per 16×16 screen tile** (per triangle, bounded by the 2D bounding box). `Config.SequentialTileRaster` (or the demo flag `-sequential-tiles`) switches to a **single-threaded** inner loop for the same math—handy to contrast scheduling overhead, not to model hardware timing. The **Go scheduler** is still unrelated to GPU hardware schedulers; use the split only for *pedagogy* and for **small** test scenes, not to claim cycle-accurate GPU modeling.

## Good classroom exercises (incremental)

1. **Change the fragment shader** to implement a 2D function (e.g. Mandelbrot escape time per pixel) *without* the triangle: still draw a full-screen quad in clip space to drive pixels.
2. **Depth ordering**: add two draws with different vertex Z; explain why **less depth** keeps the “closer” fragment (compare with the depth test in the code).
3. **“VRAM” indirection**: store vertex/index data in a byte slice; decode in the vertex shader (still Go code, but the indirection is visible).
4. **External exercise**: implement **Sobel** on CPU, then a toy **block-parallel** version with goroutines (separate from `ggpu`) to discuss races and `sync` primitives.

## Security / sandbox note

In real drivers, *compiled* GPU code is a security surface. Here, *you* supply Go functions, so the security model is the same as the rest of your process—suitable for learning, not for untrusted user shaders.
