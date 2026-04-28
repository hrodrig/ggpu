---
marp: true
theme: default
paginate: true
header: 'ggpu: Understanding GPU-Style Parallelism'
footer: 'Computer architecture education'
---

[← Back to main README](../README.md)

# How does a GPU work?
### A look at wide parallelism with `ggpu` (fork and adapt this deck)

---

# CPU vs GPU?

| | CPU (Central Processing Unit) | GPU (Graphics Processing Unit) |
| :--- | :--- | :--- |
| **Cores** | Few (4–32), very fast. | Many thousands, small and efficient. |
| **Tuning** | Low latency (one thing fast). | High throughput (many things at once). |
| **Work** | Complex sequential work. | Massive parallel work (e.g. pixels). |

---

# The (logical) graphics pipeline

1. **Vertices** — 3D points
2. **Vertex shader** — Transforms (projection)
3. **Rasterization** — Shapes to pixels
4. **Fragment shader** — Per-pixel color
5. **Framebuffer** — The final image

---

# “Wide” parallelism in ggpu

In this project’s **default** raster path:

- **Tiled rasterization:** the screen is split into **16×16** blocks.
- **Goroutine per tile** (analogy only): each block is a unit of work; many blocks per triangle.
- **Scale:** a stress run can schedule **thousands** of tile goroutines for one frame.

(Use **`-sequential-tiles`** when you need a **single-threaded** tile loop for a simpler story or profiling.)

---

# Demo: stress mode

```bash
go run ./cmd/ggpu -stress 1000   # default PNG: work/demo.png
```

**You might see:**

- `Total tile work units: ~15,000` (order of magnitude; depends on triangle sizes)
- Elapsed time spread across your CPU’s cores

(Exact numbers vary with scene and hardware.)

---

# Why tiles?

- **Locality** — one tile (0,0) does not need to know about (100,100).
- **Scalability** — more physical cores (like a real GPU) can help process more **tiles in parallel** *in principle*; this is still a **CPU** model.
- **Visualization (opt-in):** use **`-debug-tiles`** to show a **subtle** color bias on tile edges for the whiteboard. **Default** output has **no** that pattern.

---

# Takeaways

- A GPU is not “smarter” than a CPU; it is **wider** for data-parallel work.
- Its strength is launching **huge** numbers of **lightweight** threads.
- `ggpu` makes that *structure* visible in **Go** (for teaching, not for timing accuracy).

---

# Thanks & questions
Adapt this deck for your course; see [`PRESENTER_GUIDE.md`](PRESENTER_GUIDE.md) for accurate Q&A.
