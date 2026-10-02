# The persona-web image (cmd/persona-web). Built by deploy/deploy.sh
# on the DEV machine and shipped to the server as an image (docker
# save | ssh | docker load), so the server never compiles anything —
# copied from wisp's own deploy/wisp/Dockerfile, adopted here
# 2026-10-02 after building on `bh2` (shared by every product on an
# 8.7G disk) filled it during wisp's own deploy.
#
# The build stage runs on the build machine's own platform and
# cross-compiles for the target (Go needs no emulation for this); only
# the tiny final layer is per-architecture. golang:1.24 pins the
# toolchain, matching go.mod's own `go 1.24.1`, so the result doesn't
# depend on whichever Go the dev machine has installed.
#
#   docker build --platform linux/amd64 -f deploy/persona-web/Dockerfile -t persona-web:test .
FROM --platform=$BUILDPLATFORM golang:1.24 AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/persona-web ./cmd/persona-web

# persona-web dials OUT to wisp's ingest endpoint over real HTTPS, so
# (outside dev runs with WISP_ENDPOINT unset, where the hook is a
# no-op) it needs to verify a real certificate chain —
# distroless/static includes the CA bundle, unlike scratch. Same
# reasoning as EphemNet's own ephemnet-agent/Dockerfile.
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/persona-web /persona-web
ENTRYPOINT ["/persona-web"]
