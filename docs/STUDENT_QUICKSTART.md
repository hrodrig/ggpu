[← Back to main README](../README.md)

# Quick start (students)

## What is `ggpu`?

A Go library that **rasterizes triangles in memory** using a small pipeline like **3D graphics** (vertex → raster → fragment) **without** OpenGL, Vulkan, or Metal. Use it to learn transforms, interpolation, and optional depth.

## Run the demo

```bash
cd /path/to/clone
go run ./cmd/ggpu -out work/output.png -w 800 -h 600
```

Useful flags:

- `-w`, `-h` — size in pixels
- `-out` — output PNG path (default **`work/demo.png`** if omitted)
- `-t` — `TimeSeconds` uniform (small fragment tint in the sample shader)
- `-sequential-tiles` — no per-tile goroutine (single-threaded tile loop; good for debugging)
- `-debug-tiles` — subtle tile-edge pattern in color (teaching only; **not** the default “clean” output)
- `-stress N` — draw `N` random triangles (rough performance smoke test)

## Run tests

```bash
go test ./...
```

(For CI parity, use `make test` with the race detector.)

## Makefile targets

Run **`make`** with no arguments (or `make help`) to print the list of targets.

| Target | What it does |
|--------|----------------|
| `make fmt` | `gofmt -w .` |
| `make vet` | `go vet ./...` |
| `make test` | Tests with `-race` |
| `make gocyclo` | Fail if any function has cyclomatic complexity ≥ 15 |
| `make cover` | Merged coverage across packages; **fails below 80%** (needs `bc`) |
| `make build` | Build `bin/ggpu` (reads `VERSION` + git; run `./bin/ggpu -version`) |
| `make build-linux-amd64` / `make build-linux-arm64` | Static Linux binaries (cross-compile) |
| `make demo` | Build `bin/ggpu`, then run it (default **`work/demo.png`**) |

## Docker

```bash
mkdir -p work
docker compose up --build
```

The PNG is written to `work/demo.png` (see `docker-compose.yml`: `./work` is mounted at `/work` in the container).

## Read more

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — diagrams and pipeline walkthrough
- [`PRESENTATION.md`](PRESENTATION.md) — sample talk script if you (or your instructor) present the project
- [`PRESENTER_GUIDE.md`](PRESENTER_GUIDE.md) — Q&A, limits, flags, `SetUniforms(nil)` behavior
- [`slides.md`](slides.md) — [Marp](https://marp.app/) deck: use [Marp for VS Code](https://marketplace.visualstudio.com/items?itemName=marp-team.marp-vscode) or `npx @marp-team/marp-cli docs/slides.md -o ggpu.pdf`
