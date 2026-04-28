[← Back to main README](README.md)

# Security

## Threat model

`ggpu` is a **local, in-process** software rasterizer intended for **education and offline rendering**. It does not accept network input, does not parse untrusted shader bytecode, and does not execute foreign code: shaders are **Go function values** that you provide.

- **Not applicable**: RCE from GPU drivers, web GPU sandbox escapes, or CUDA injection—this is not a real driver.

## Input handling

- `DrawIndexed` expects caller-supplied `[]VertexInput` and `[]uint16` indices. Out-of-range indices are **silently skipped** (the triangle is not drawn; there is no panic). This is documented in `docs/ARCHITECTURE.md` and covered by tests. For security-sensitive code, validate indices before the call.
- `SetUniforms(nil)` does **not** clear uniforms; it leaves the last copied `UniformBuffer` in place. See the `GPU` doc comment in `pkg/ggpu` and `docs/ARCHITECTURE.md`.
- `WriteFramebufferPNG` writes to an `io.Writer` you control; use safe paths and permissions in your app.

## Dependencies

- Run `govulncheck ./...` regularly (see `Makefile` and CI) to detect known vulnerabilities in the **Go standard library and toolchain** used by the module.
- The module has **no third-party Go dependencies** at the time of writing (`go.mod` only pins the Go version).

## Docker

- Images are built as **static, non-root** (distroless: see `README.md` and `Dockerfile`). Mount only the volumes you need when running the demo.
- CI runs **[Grype](https://github.com/anchore/grype)** on the built image and **fails** the pipeline when findings of **High** or **Critical** severity are present (`--fail-on high`). Rebuild or update base images when fixes are published; this is distinct from `govulncheck`, which scans **Go** code usage, not the container layers.
- A separate **`go test -cover...`** run (without the race detector) is used in CI; see the `README` for why race and cover are not combined in a single `go test` invocation.

## Reporting

For issues found in this repository, open a private discussion with the course maintainer or use the issue tracker as directed by your institution.
