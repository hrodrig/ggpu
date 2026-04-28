# syntax=docker/dockerfile:1
# Local / CI image: compile inside Docker (e.g. make docker-build, docker compose).
# Release images: GoReleaser builds static binaries, then Dockerfile.release packages them.
# Multi-arch: docker buildx build --platform linux/amd64,linux/arm64 -t ggpu:local .
# Makefile passes APP_VERSION, GIT_COMMIT, GIT_BRANCH from ./VERSION and git.
FROM --platform=$BUILDPLATFORM golang:1.26.2-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG APP_VERSION=0.0.0
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
	-ldflags="-s -w -X main.version=${APP_VERSION} -X main.commit=${GIT_COMMIT} -X main.branch=${GIT_BRANCH}" \
	-o /out/ggpu ./cmd/ggpu \
	&& CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
	-ldflags="-s -w" \
	-o /out/ggmat ./cmd/ggmat

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /work
COPY --from=builder /out/ggpu /usr/local/bin/ggpu
COPY --from=builder /out/ggmat /usr/local/bin/ggmat
ENTRYPOINT ["/usr/local/bin/ggpu"]
CMD ["-out", "/work/demo.png", "-w", "800", "-h", "600"]
