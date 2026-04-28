[← Back to main README](../README.md)

# Presenter’s guide (ggpu) — talk track and answers

A **shared reference** for **anyone** who clones the repo: design trade-offs, real limits, configuration flags, and how to **explain** each point in a classroom or viva. It stays consistent with the code and with [`ARCHITECTURE.md`](ARCHITECTURE.md).

---

## 1. What it is and is not (30 seconds)

| Claim | Truth |
|--------|--------|
| “A CUDA / SIMT GPU emulator” | **No.** No blocks, warps, or shared memory. A minimal **graphics** CPU pipeline. |
| “Shaders are GPU assembly” | **No.** They are **Go functions** (vertex + fragment) you write in the same binary. |
| “Same as OpenGL / Vulkan” | **Only roughly:** vertex → (guard) → viewport → raster → fragment → framebuffer. No full clipper, real blending, texturing, etc. |

For contrast with compute, read [`TEACHING.md`](TEACHING.md).

---

## 2. Buffers: color, depth, stencil

- **Color:** `Clear` fills an 8-bit RGBA framebuffer.
- **Depth (optional):** when `Config.DepthTest` is `true`, clear also resets depth to **1.0** (far). The **smaller** stored depth value **wins** the test with the viewport mapping in this engine (see `ARCHITECTURE.md`).
- **Stencil:** **not** implemented. There is no `ClearStencil`. If asked about “OpenGL stencil,” say: “Out of scope; this package does not implement it.”

---

## 3. “Clip” in this repository (important to say clearly)

Talks often say “clip / guard.” In code:

- There is **no** polygon **clip** to the frustum border.
- There is a **validation** in clip/NDC: if *W* ≤ 0 or *x, y, z* / NDC go outside a loose band, the **entire triangle** is **dropped** (*cull*), not **split** for partial visibility.

**Suggested phrasing:** “This is a **defensive guard**, not a clipper; a triangle that crosses a logical clip plane is discarded as a whole—we do not fan it into new triangles.”

---

## 4. Parallelism: goroutines, tiles, “GPU threads”

- **Default:** each **16×16** screen tile in a triangle’s bounding box can run in a **new** goroutine; the barrier waits for all tiles of that triangle.
- **Teaching counter `GPU.TotalTiles()`:** counts **16×16 work units** scheduled, not the number of draw calls. In **sequential** mode the counter still increments, but without **per-tile** goroutines.

**Closing line for students:** “Real hardware has millions of logical threads; we stress **many pixel packets** with goroutines. That is a metaphor, not a timing model.”

### Sequential mode (`Config.SequentialTileRaster` or `go run ... -sequential-tiles`)

- All tiles of one triangle run in **nested loops** in the **caller's** goroutine.
- **When to mention it:** debugging, scheduler cost, avoiding huge goroutine counts on large scenes, single-threaded profiling.

### Visible tile pattern (`Config.DebugShowTileWorkload` or `go run ... -debug-tiles`)

- Adds a **subtle** color bias at tile edges **before** the fragment shader, for teaching only.
- **Say out loud:** “This muddies **raw** color fidelity—leave it **false** for clean output or fixed assignments.”

---

## 5. Uniforms: `SetUniforms(nil)` and the first draw

- Every **non-nil** `SetUniforms` **copies** the `UniformBuffer` into the engine.
- `SetUniforms(nil)` **does not clear** uniforms; the **last** copy remains.
- On the **first** draw, if you never set uniforms and the **zero** struct is wrong for you, set a **non-nil** `UniformBuffer` first.

---

## 6. `DrawIndexed` and indices

- `indices` length not a multiple of **3:** the loop does not emit triangles (early exit or empty work).
- **Out-of-range** index vs. `verts`: the triangle is **skipped** (no panic). `SECURITY.md` and package tests document this. Course assignments can require explicit validation in student code.

---

## 7. License and reuse

- The root **`LICENSE`** is **MIT**. Cite it when reusing the code in other courses or repos. Fork maintainers can adjust the **copyright** line to their name or institution.

---

## 8. How documents fit together

| File | Use |
|------|-----|
| `README.md` (root) | Install, flags, CI, Docker |
| `ARCHITECTURE.md` | Diagrams, guard, tile modes, uniforms |
| `PRESENTATION.md` | 15–20 min talk script |
| `STUDENT_QUICKSTART.md` | Short student how-to |
| `slides.md` | Marp → PDF/HTML |

**Prep order (suggested):** 1) this file, 2) `PRESENTATION.md`, 3) `ARCHITECTURE.md` (limitations) for hard technical Q&A.

---

## 9. Flash questions

- **Why not a fixed worker pool?** *Not implemented; a possible advanced exercise is a bounded `worker pool` with identical pixels.*
- **1:1 with GPU “threads”?** *No. We use 16×16 units and goroutines as analogy; real hardware is different in scale and memory behavior.*
- **Same depth as OpenGL?** *Only the idea of “test before color.” Exact formula and range are in **this** code and `ARCHITECTURE.md`—do not attribute unverified OpenGL behavior to ggpu.*

If you add a feature, update `ARCHITECTURE.md` and this file in the same change.
