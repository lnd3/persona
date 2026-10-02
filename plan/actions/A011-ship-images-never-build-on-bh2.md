---
id: A011
title: Ship locally built images; never build on bh2
status: DONE
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
  (`deploy/persona-web/Dockerfile`, `golang:1.24`). **Done** — see
  [[A009]]'s own Log for the implementation (commit `9eda269`) and
  live verification on `bh2`.
- `persona-caddy` stays `image: caddy:2`. Unchanged, as this note
  anticipated.
- **Also worth fixing in the same pass:** persona's pinned Docker
  subnet `172.32.x` is public address space (outside RFC 1918's
  172.16.0.0/12), noted in A010. **Already fixed** by this repo's own
  session before this note was filed — see A009's 2026-10-01 Log entry
  (corrected to `172.21.1.0/24`), not something this pass needed to
  do.

## Not done from wisp

This was a note, not a change, when filed: the code in this repo was
untouched at that point. persona's own session picked up the pattern
independently the same day (prompted directly: "Wisp build docker
snapshot now instead of building on bh2. Copy it.") and implemented it
under [[A009]] before reconciling with this note — see A009's own Log
for the real work (Dockerfile, `docker-compose.yml`, `deploy.sh`,
verified live on `bh2`: 8.46MB image, seconds to ship, disk unchanged
before/after).

## Log

2026-10-02 — Filed from the wisp repo at the user's request ("All deploy
scripts should be doing this. Share with offgrid, cinder, ephemnet,
persona.") after the bh2 disk incident. Planning only; no code
changed.

2026-10-02 (later, same day) — Reconciled with [[A009]]: persona's own
session implemented this pattern independently (same prompt, routed
through this repo directly rather than this note) before noticing this
action existed. No conflict in approach — both describe the exact same
`wisp 211a9cb` reference pattern. Status moved to `DONE`; the real
implementation Log lives in A009, not duplicated here.
