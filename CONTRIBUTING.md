# Contributing to ggpu

Thank you for your interest in contributing.

## How to contribute

- **Bug reports and feature requests:** Open an [issue](https://github.com/hrodrig/ggpu/issues) and use the template that fits best.
- **Code changes:** Open a pull request from a branch (for example `fix/description` or `feat/short-name`).
- **Branch policy:** CI runs on **`main`** and **`master`**; open PRs against the default branch of this repository (usually **`main`**).
- **Scope:** Keep PRs focused and small when possible.

## Code style

- Format Go code with `gofmt -s` (or run **`make lint-fix`**).
- Run checks locally before submitting:
  - `make lint`
  - `make test`
  - `make security`
- For release-related changes, run:
  - `make release-check`

## Release flow

- Version is read from **`VERSION`** (semantic version without `v`, for example `0.1.8`).
- Release tag format is **`v<version>`** (for example **`v0.1.8`**).
- **`make release`** expects branch **`main`** (see `Makefile`).
- Useful commands:
  - `make snapshot`
  - `make test-release`
  - `make release`

## Questions

If you are unsure, open an issue and describe your proposal before implementing large changes.
