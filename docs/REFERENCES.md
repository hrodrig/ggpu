[← Back to main README](../README.md)

# References and related work (for students)

## Official and standards (background reading)

- **OpenGL / SPIR-V / Vulkan** specifications (Khronos): show how *real* graphics pipelines are specified. Compare with the **intentionally tiny** API in `pkg/ggpu`.
- **CUDA Programming Guide** (NVIDIA): SIMT, grids, blocks, shared memory—useful if you extend the course with a **compute** toy.

## Go ecosystem (analogies)

- **Gorgonia** (`github.com/gorgonia/gorgonia`): tensor graphs and execution—*not* a GPU emulator, but a good example of scheduling dataflow in Go.
- **CUDA bindings** (e.g. `github.com/gorgonia/cu`): how host code allocates and copies device memory—**interoperability** patterns, not emulation.
- **Tiny CPU emulators** in Go (search “chip-8 go”, “6502 emulator go”): **fetch–decode–execute** loops; useful if you later design a **bytecode** shader ISA for a follow-up project.

## Graphics fundamentals (books & notes)

- *Fundamentals of Computer Graphics* (Shirley & Marschner) — transformations, rasterization, perspective.
- MIT 6.837 / similar courses—public notes on **pipeline** and **homogeneous coordinates**.

## This repository

- `docs/ARCHITECTURE.md` — pipeline diagram and module map.
- `docs/TEACHING.md` — misconceptions (SIMT vs raster), exercise ideas.
