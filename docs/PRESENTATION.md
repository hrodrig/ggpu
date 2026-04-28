[← Back to main README](../README.md)

# Classroom presentation (ggpu) — 15–20 minutes

Use this as a **talk script** for any audience. Adapt examples and time boxes to your course. The companion **[`PRESENTER_GUIDE.md`](PRESENTER_GUIDE.md)** has detailed Q&A; **[`slides.md`](slides.md)** is a **Marp** deck you can export to PDF or HTML.

---

## 1. Hook (~1 min)

- **Problem:** real GPUs are highly parallel black boxes; building intuition without opening the driver is hard.
- **Idea:** a **CPU** rasterizer that encodes a **minimal graphics pipeline**: **vertices → raster → fragments → framebuffer**, with **shaders as Go functions** so the control flow stays readable.

---

## 2. Learning goals (~1 min)

By the end, the audience should be able to:

1. Distinguish a **graphics pipeline** (triangles, interpolation, framebuffer) from a **compute GPU** (SIMT kernels, thread blocks, shared memory).
2. Name what is **true to the concept** here (perspective-correct attributes, optional depth) versus what is **pedagogy** (no ISA, no full clipper).

---

## 3. Live or recorded demo (~3 min)

Suggested command (writes a PNG). The **program** lives under **`cmd/ggpu`** (Go package path); the **binary name** is **`ggpu`** after `make build`:

```bash
go run ./cmd/ggpu -w 800 -h 600 -t 0   # default: work/demo.png
# after: make build
./bin/ggpu -w 800 -h 600 -t 0
```

Show the **gradient** triangle (per-vertex colors). Optionally change `-t` and explain that `TimeSeconds` is a **per-draw constant** (a uniform in real APIs).

**Key line:** “We are not running **shader machine code**; we are **mirroring the pipeline shape** with Go functions.”

---

## 4. Whiteboard or projection (~5 min)

Walk through:

1. **Input assembler:** `VertexInput` + indices.
2. **Vertex shader:** `MVP × position`, pass color.
3. **Clip / guard (not a full clipper):** drop triangles outside a reasonable NDC range.
4. **Viewport:** NDC → pixels (**Y down**, like bitmaps).
5. **Rasterizer:** barycentrics and **1/w** for attributes.
6. **Fragment shader:** final color.
7. **Optional depth test:** which fragment wins per pixel.

**Parallelism in this repo:** by default, each **16×16** screen tile can run in its **own goroutine**; for a “single-threaded raster” story, use the demo with **`-sequential-tiles`** or `Config.SequentialTileRaster`. **Say which mode you are using** so you never contradict the code.

**Figures:** see [`ARCHITECTURE.md`](ARCHITECTURE.md) (Mermaid). Durable answers: [`PRESENTER_GUIDE.md`](PRESENTER_GUIDE.md).

---

## 5. Optional HPC / prior-chat bridge (~3 min)

Use with care and clear scope:

- **SIMT / warps / goroutines:** a **teaching** metaphor. By default this project launches **one goroutine per 16×16 tile**, not one thread for the whole raster unless you enable **sequential** mode. Do not claim 1:1 real-time warps.
- **Memory hierarchy (global / shared / local):** mostly **CUDA**; the closest story here is **uniforms**, **per-vertex attributes**, **interpolated varyings**, **framebuffer**.

See [`TEACHING.md`](TEACHING.md) for scope and common misconceptions.

---

## 6. Honest limits (~2 min)

- No **textures**, **advanced blending**, **tessellation**, or GPU **ISA**.
- **No** frustum **clipper**: only a **guard** in clip/NDC; unstable boundary cases can **drop a whole triangle**; nothing is **split** into new triangles. Say *guard* vs *clip*; see `ARCHITECTURE.md`.
- **No stencil** buffer. Only color + optional depth. Do not say “stencil clear” for this project.
- **Debug tile** effect (`-debug-tiles` / `Config.DebugShowTileWorkload`) is **opt-in** and for the board; with `false`, output is not checker-tinted.
- **Security:** no third-party shader bytecode; Go functions in your process (`SECURITY.md`). `SetUniforms(nil)` **keeps** the last uniforms.

`PRESENTER_GUIDE.md` spells out the same for Q&A.

---

## 7. Wrap-up and questions (~2 min)

- **Question for the room:** “Where does **massive** parallelism live on a real GPU, versus what we do in **software** here?”
- **Short answer:** hardware runs huge parallel fragment work; this repo by default uses **many goroutines** per 16×16 **tiles**, or a **sequential** inner loop. The *idea* of many independent fragments is the analogy; the **scheduling** is for teaching, not performance modeling.

---

## 8. FAQ (cheat sheet)

| Question | Short answer |
|----------|---------------|
| Is this a CUDA emulator? | **No.** A minimal **graphics** pipeline. |
| Why Go? | Readability, `go test`, `go vet`, simple CI. |
| Where is parallelism? | On hardware, massive; in this repo, by default one **goroutine** per 16×16 **tile**, or a **single thread** in **sequential** mode. Not 1:1 with a real GPU. |

---

## More material

- Root **`README.md`** — setup, demo flags, Docker, CI, MIT `LICENSE`
- [`PRESENTER_GUIDE.md`](PRESENTER_GUIDE.md) — reference for anyone presenting
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — technical depth, guard, tiles, uniforms
- [`TEACHING.md`](TEACHING.md) — graphics vs compute
- [`SECURITY.md`](../SECURITY.md) — threat model
- [`slides.md`](slides.md) — Marp slides

---

Good luck with your talk.
