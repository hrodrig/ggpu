# Changelog

All notable changes to this project are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.8] - 2026-04-27

### Added

- **`make release-check`** (semver, `goreleaser check`, lint, test, security; **`STRICT_RELEASE=1`** opcional para **`docker-scan`**), **`make snapshot` / `test-release` / `release`**, **`make lint`** / **`lint-fix`**, **`make security`**, **`govulncheck`** vía `go run`, **`grype`** en directorio con exclusión `bin`/`work`/`dist`, y **`make docker-scan`** para la imagen.
- **CI** — job **`lint`** (`make lint`, `make gocyclo`); **`go-linux-amd64`** con **`go-version-file: go.mod`**, cobertura fusionada, subida opcional a **Codecov**, builds nativos y **cross-compile de `ggpu` y `ggmat`**, **`go-linux-arm64`** con build de **ambos** binarios.
- **Plantillas GitHub** — PR, issues, **`release.yml`**, **`security.yml`**, **CodeQL**; enlace de seguridad al repo **ggpu**.

### Changed

- **`cmd/demo` → `cmd/ggpu`** (rutas de build, Docker, GoReleaser, docs).
- **`make ci`** — **`lint`** + test + gocyclo + cover (sin `gofmt -w`).
- **`.goreleaser.yaml`** — `-trimpath`, **`archives.ids`** (sin `builds` deprecado), filtros de changelog **`ci:`** / **`.github:`**, **`release.prerelease: auto`**.

## [0.1.0] - 2026-04-28

### Added

- **`ggpu`** — educational **CPU** software rasterizer (`pkg/ggpu`, `cmd/ggpu`): vertex/fragment shaders as Go functions, perspective-correct interpolation, optional depth test, PNG export; Makefile **`make build`** → `bin/ggpu` with `VERSION` + git metadata via `-ldflags`.
- **`ggmat`** — dense **CSV** matrix CLI (`pkg/ggmat`, `cmd/ggmat`): `add`, `mul-ew`, `matmul`, `identity`, `dot`, optional parallelism via `-vcores`; **`make build-ggmat`** → `bin/ggmat`.
- **Quality & supply chain** — `gofmt`, `go vet`, **`go test -race`**, merged **statement coverage ≥ 80%** (`make cover`), **`gocyclo`** (every function under complexity 15), **`govulncheck`**, **Grype** on the CI-built image (fail on High+).
- **Containers** — multi-stage **`Dockerfile`** (static `ggpu` + `ggmat`, distroless `nonroot` runtime); **`Dockerfile.release`** for **GoReleaser `dockers_v2`** (pre-built Linux binaries, `COPY ${TARGETPLATFORM}/…`).
- **Release tooling** — **`.goreleaser.yaml`**: archives (tar.gz / zip on Windows), checksums, **`ghcr.io/hrodrig/ggpu`** image tags, GitHub release metadata.
- **Docs** — architecture, teaching notes, student quickstart, slides under `docs/`.

### Changed

- **`.gitignore`** — whitelist layout so `pkg/`, `docs/`, and project roots are trackable (deny-all default).
- **`.dockerignore`** — allow only Go module + `cmd/**` + `pkg/**` for reproducible image builds.

[Unreleased]: https://github.com/hrodrig/ggpu/compare/v0.1.8...HEAD
[0.1.8]: https://github.com/hrodrig/ggpu/releases/tag/v0.1.8
[0.1.0]: https://github.com/hrodrig/ggpu/releases/tag/v0.1.0
