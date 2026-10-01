# Builds persona-web as a static binary and ships it in a minimal
# image. Same layout convention as EphemNet's own
# deploy/ephemnet-agent/Dockerfile — build from the repo root:
#   docker build -f deploy/persona-web/Dockerfile -t persona-web .
#
# Pinned to golang:1.24 to match go.mod's own `go 1.24.1` exactly —
# same convention as cinder's/EphemNet's own Dockerfiles.
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/persona-web ./cmd/persona-web

# persona-web dials OUT to wisp's ingest endpoint over real HTTPS, so
# (outside dev runs with WISP_ENDPOINT unset, where the hook is a
# no-op) it needs to verify a real certificate chain —
# distroless/static includes the CA bundle, unlike scratch. Same
# reasoning as EphemNet's own ephemnet-agent/Dockerfile.
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/persona-web /persona-web
ENTRYPOINT ["/persona-web"]
