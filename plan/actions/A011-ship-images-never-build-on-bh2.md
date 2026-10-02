---
id: A011
title: Ship locally built images; never build on bh2
status: PLANNING
project: P001
design: D001
created: 2026-10-02
updated: 2026-10-02
---

## Why

On 2026-10-02 bh2's shared 8.7 GB disk hit 99% (155 MB free) during a
deploy that compiled Go **on the server** (toolchain image pull, module
download, compile: ~2 GB of temporary space). Separately, server-side
builds leave their **toolchain base image** behind:
`golang:1.24-bookworm` (1.27 GB, pulled 2026-10-01 20:03 UTC by
cinder's deploy) took most of a day's 16% disk growth. `docker image
prune -f` never removes it, because it's tagged, and `docker builder
prune` only clears the cache. The user's decision: "All deploy scripts
should be doing this". **Build images on the dev machine and ship the
image; the live server never builds.**

## The pattern (reference: wisp `211a9cb`, `deploy/deploy.sh` + `deploy/wisp/Dockerfile`)

1. **Dockerfile:** build stage on the build machine's platform,
   cross-compiling natively. Go needs no emulation, and with CGO off the
   final layer is the only per-architecture part:
   ```dockerfile
   FROM --platform=$BUILDPLATFORM golang:<pinned> AS build
   ARG TARGETOS TARGETARCH
   …
   RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app
   FROM <runtime base>
   COPY --from=build /out/app /app
   ```
   The pinned toolchain stays in the Dockerfile, so builds don't depend
   on the dev machine's Go.
2. **docker-compose.yml:** replace each service's `build:` with
   `image: <name>:current`.
3. **deploy.sh**, after packaging the `git archive` snapshot it already
   makes:
   - **Build locally** for the server's platform (`ssh <target> uname -m`
     → `linux/amd64|arm64`):
     `docker build --platform $PLATFORM -f "$STAGING/…/Dockerfile" -t <name>:$GIT_REV "$STAGING"`.
   - **Ship all images in one stream:**
     `docker save <name1>:$GIT_REV <name2>:$GIT_REV … | gzip -1 | ssh <target> 'gunzip | docker load -q'`.
   - **On the server:** `docker tag <name>:$GIT_REV <name>:current`,
     then `docker compose … up -d`. Keep the two previous
     `<name>:<commit>` images for rollback (`docker tag <name>:<old>
     <name>:current && up -d`) and remove older ones.
   - **Cleanup in a trap:** `trap 'docker image prune -f' EXIT` on the
     remote side, so a failure cleans up too. Drop
     `docker compose build` and `docker builder prune` from the remote
     script: nothing builds there any more.
   - **Free-space guard:** refuse to start remotely when
     `df --output=avail -BM .` is under ~300 MB. bh2 is shared by every
     product.
4. **Measured on wisp:** a 14.7 MB image, a 6.3 MB transfer, and a
   13-second deploy with a warm local build cache. The server's disk
   doesn't move.

## persona specifics

- **Image to build locally:** `persona-web`
  (`deploy/persona-web/Dockerfile`, `golang:1.24`).
- `persona-caddy` stays `image: caddy:2`.
- **Also worth fixing in the same pass:** persona's pinned Docker
  subnet `172.32.x` is public address space (outside RFC 1918's
  172.16.0.0/12), noted in A010.

## Not done from wisp

This is a note, not a change: the code in this repo is untouched.

## Log

2026-10-02 — Filed from the wisp repo at the user's request ("All deploy
scripts should be doing this. Share with offgrid, cinder, ephemnet,
persona.") after the bh2 disk incident. Planning only; no code
changed.
