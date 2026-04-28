# [← Main README](README.md)
.DEFAULT_GOAL := help

# Single source of truth; read by make, CI, and Docker build-args.
VERSION := $(shell cat VERSION 2>/dev/null | tr -d ' \n\r' || echo 0.0.0)
# GNU make $(shell) collapses newlines to spaces — use head -1 and explicit empty check.
GIT_COMMIT := $(firstword $(shell git rev-parse --short HEAD 2>/dev/null | head -1))
ifeq ($(strip $(GIT_COMMIT)),)
  GIT_COMMIT := unknown
endif
GIT_BRANCH := $(firstword $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null | head -1))
ifeq ($(strip $(GIT_BRANCH)),)
  GIT_BRANCH := unknown
endif

# Injected into cmd/ggpu (package main) at link time.
LDFLAGS := -s -w -X 'main.version=$(VERSION)' -X 'main.commit=$(GIT_COMMIT)' -X 'main.branch=$(GIT_BRANCH)'

# install(1): PREFIX/BINDIR/DESTDIR (e.g. make install DESTDIR=/tmp/stage PREFIX=/usr).
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

# Release knobs: optional strict image scan, Grype policy, dist output (see release-check).
STRICT_RELEASE ?= 0
GRYPE_FAIL_ON ?= high
DIST ?= dist
TAG := v$(VERSION)
# Grype/Syft exclusion globs must start with ./, */, or **/ (see anchore/grype catalog rules).
GRYPE_DIR_EXCLUDES := --exclude './bin/**' --exclude './work/**' --exclude './dist/**'

.PHONY: help all fmt lint lint-fix vet test gocyclo cover build build-ggmat build-linux-amd64 build-linux-arm64 demo docker clean govulncheck vulncheck security ci grype docker-scan docker-build docker-buildx docker-run install ggpu ggmat release-check snapshot test-release release

help:
	@echo "ggpu — GNU make targets (VERSION=$(VERSION) from file VERSION)"
	@echo ""
	@echo "  make all         format, vet, test, gocyclo, cover, and build → bin/ggpu"
	@echo "  make fmt         gofmt -w . (no simplify; use lint-fix for gofmt -s)"
	@echo "  make lint        Check gofmt (like CI: gofmt -l) and go vet"
	@echo "  make lint-fix    gofmt -s -w . (simplify with gofmt -s)"
	@echo "  make vet         go vet ./..."
	@echo "  make test        go test -race -count=1 ./..."
	@echo "  make gocyclo     max cyclomatic complexity < 15 per function (gocyclo -over 14)"
	@echo "  make cover       go test with coverage (writes coverage.out)"
	@echo "  make build       link bin/ggpu with version=$(VERSION), commit, branch from git"
	@echo "  make build-ggmat  build bin/ggmat (dense matrix CLI: add, mul-ew, matmul, identity, dot)"
	@echo "  make install [ggpu|ggmat]  install(1) into DESTDIR+BINDIR (default BINDIR=$(BINDIR); both binaries if no extra goals)"
	@echo "  make build-linux-amd64    static Linux/amd64 → bin/ggpu-linux-amd64"
	@echo "  make build-linux-arm64     static Linux/arm64  → bin/ggpu-linux-arm64"
	@echo "  make demo        build, then run ./bin/ggpu (default PNG: work/demo.png)"
	@echo "  make clean       remove bin/ and generated work/demo.png (and legacy demo.png)"
	@echo "  make ci          lint, test, gocyclo, cover (no binary build; matches CI checks)"
	@echo "  release-check      Validate semver, tooling, lint, test and security"
	@echo "  make govulncheck  govulncheck via go run (no install)"
	@echo "  make vulncheck   alias for govulncheck"
	@echo "  make security    govulncheck + gocyclo + grype (dir scan; Docker fallback if grype missing)"
	@echo "  make grype       Grype directory scan (excludes bin/work/dist); Docker fallback if grype missing"
	@echo "  make docker-scan docker build ggpu:local + Grype image (optional: STRICT_RELEASE=1 in release-check)"
	@echo "  make docker-build    docker build (passes VERSION + git metadata)"
	@echo "  make docker-buildx   multi-arch image linux/amd64,linux/arm64 (needs buildx)"
	@echo "  make docker-run      run ggpu:local (mounts ./work to /work; PNG per image CMD)"
	@echo ""
	@echo "Release (GoReleaser; see .goreleaser.yaml):"
	@echo "  make snapshot      goreleaser build --snapshot --clean (after release-check)"
	@echo "  make test-release   goreleaser release --snapshot --skip=publish --clean"
	@echo "  make release       goreleaser release --clean (branch must be main)"
	@echo ""
	@echo "  ./bin/ggpu -version  → prints VERSION, commit hash, branch"
	@echo "With no target, this help is shown."

all: fmt vet test gocyclo cover build

fmt:
	gofmt -w .

# Read-only format check + vet (CI uses gofmt -l without -s).
lint:
	@echo "Checking gofmt -l..."
	@unformatted=$$(gofmt -l .); [ -z "$$unformatted" ] || { echo "Files not formatted (run: make fmt or make lint-fix):"; echo "$$unformatted"; exit 1; }
	@echo "Running go vet..."
	@go vet ./...

lint-fix:
	gofmt -s -w .

vet:
	go vet ./...

test:
	go test -race -count=1 ./...

# Merged report across all packages (matches CI). Fails if total statement coverage is below 80%.
cover:
	go test -count=1 -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...
	@P=$$(go tool cover -func=coverage.out | tail -1 | sed 's/^.*[[:space:]]\([0-9.]*\)%.*/\1/'); \
		echo "total (merged) statement coverage: $$P% (minimum 80%)"; \
		if [ "$$(echo "$$P < 80" | bc)" -eq 1 ]; then \
			echo "coverage below 80% — add tests (or change threshold in this Makefile target)"; \
			exit 1; \
		fi

# Fail if any function has cyclomatic complexity >= 15 (gocyclo: complexity > 14).
gocyclo:
	go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -over 14 .

build:
	go build -ldflags "$(LDFLAGS)" -o bin/ggpu ./cmd/ggpu

build-ggmat:
	go build -o bin/ggmat ./cmd/ggmat

# Optional goals ggpu / ggmat after install pick binaries (default: both). Uses install(1).
ggpu ggmat:

install: ggpu ggmat
	@tgts="$(filter ggpu ggmat,$(MAKECMDGOALS))"; \
	if [ -z "$$tgts" ]; then tgts="ggpu ggmat"; fi; \
	set -e; \
	for t in $$tgts; do \
		case $$t in \
			ggpu) $(MAKE) build; install -d "$(DESTDIR)$(BINDIR)"; install -m 755 bin/ggpu "$(DESTDIR)$(BINDIR)/ggpu" ;; \
			ggmat) $(MAKE) build-ggmat; install -d "$(DESTDIR)$(BINDIR)"; install -m 755 bin/ggmat "$(DESTDIR)$(BINDIR)/ggmat" ;; \
			*) echo "make install [ggpu] [ggmat] — unknown target in list"; exit 2 ;; \
		esac; \
	done

# Static Linux binaries (no CGO). Same VERSION/git metadata as build.
build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/ggpu-linux-amd64 ./cmd/ggpu

build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/ggpu-linux-arm64 ./cmd/ggpu

demo: build
	./bin/ggpu

govulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

vulncheck: govulncheck

ci: lint test gocyclo cover
	@echo "OK: lint, test, gocyclo, cover"

security: govulncheck gocyclo grype

# Semver + goreleaser check + lint + test + security (+ optional docker-scan when STRICT_RELEASE=1).
release-check:
	@test -f VERSION || { echo "VERSION file is required"; exit 1; }
	@echo "Release version: $(VERSION) (tag: $(TAG))"
	@echo "$(VERSION)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "VERSION must be semantic version (e.g. 0.1.0)"; exit 1; }
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser is required. Install from https://goreleaser.com/install/"; exit 1; }
	goreleaser check
	@$(MAKE) lint
	@$(MAKE) test
	@$(MAKE) security
	@if [ "$(STRICT_RELEASE)" = "1" ]; then \
		echo "STRICT_RELEASE=1 -> running docker-scan"; \
		$(MAKE) docker-scan; \
	else \
		echo "STRICT_RELEASE=0 -> skipping docker-scan"; \
	fi
	@echo "All release checks passed."

snapshot: release-check
	goreleaser build --snapshot --clean

test-release: release-check
	goreleaser release --snapshot --skip=publish --clean

release: release-check
	@branch=$$(git branch --show-current 2>/dev/null); \
	if [ "$$branch" != "main" ]; then \
		echo "Error: release only from main (current: $$branch)."; \
		exit 1; \
	fi
	goreleaser release --clean

# Directory scan with path exclusions; Docker fallback via anchore/grype image if grype is not installed.
grype:
	@if command -v grype >/dev/null 2>&1; then \
		grype dir:. $(GRYPE_DIR_EXCLUDES) --fail-on $(GRYPE_FAIL_ON); \
	else \
		echo "grype not found locally, using container image..."; \
		docker run --rm --pull=always -v "$(PWD):/workspace" anchore/grype:latest \
			dir:/workspace $(GRYPE_DIR_EXCLUDES) --fail-on $(GRYPE_FAIL_ON); \
	fi

# Image scan after docker-build; --pull=always avoids a stale local anchore/grype:latest cache.
docker-scan: docker-build
	@if command -v grype >/dev/null 2>&1; then \
		grype ggpu:local --fail-on $(GRYPE_FAIL_ON); \
	else \
		echo "grype not found locally, using container image..."; \
		docker run --rm --pull=always -v /var/run/docker.sock:/var/run/docker.sock anchore/grype:latest \
			ggpu:local --fail-on $(GRYPE_FAIL_ON); \
	fi

clean:
	rm -rf bin/ $(DIST) work/demo.png demo.png

docker-build:
	docker build -t ggpu:local -f Dockerfile \
		--build-arg APP_VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_BRANCH=$(GIT_BRANCH) \
		.

# Multi-arch image. Pass the same build-args so version strings match your tree.
docker-buildx:
	docker buildx build --platform linux/amd64,linux/arm64 -t ggpu:local -f Dockerfile \
		--build-arg APP_VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_BRANCH=$(GIT_BRANCH) \
		.

docker-run:
	docker run --rm -v "$$PWD/work:/work" ggpu:local
