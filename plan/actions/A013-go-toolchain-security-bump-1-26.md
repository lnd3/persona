---
id: A013
title: Go toolchain security bump (1.24 -> 1.26.9)
status: DONE
design: D001
project: P001
created: 2026-10-22
updated: 2026-10-22
---

## Context

`EphemNet` flagged this via a real cross-repo notice —
[`plan/inbox/I001`](../inbox/I001-ephemnet-go-toolchain-security-bump.md),
persona's own first `plan/inbox/` message, which `EphemNet` created
the directory for since it didn't exist here yet. After finding its
own `go.mod` unpatched since Go 1.26 shipped (2026-02-11 — Go only
backports security fixes to the latest two major branches), `EphemNet`
checked persona's own `go.mod`/Dockerfile too and found the same
exposure: `go 1.24.1`, `golang:1.24` in `deploy/persona-web/
Dockerfile`. Verified for real before acting, not taken on faith:
confirmed both files matched exactly what the notice claimed.

Real CVEs this closes, per `EphemNet`'s own research (not re-verified
independently here — same product family, same source, no reason to
re-derive): Go toolchain checksum-bypass (`CVE-2026-94447`),
`crypto/tls` ECH handling and an HTTP/2 server crash in `net/http`
(`CVE-2026-97031`/`97032`, fixed in `go1.26.9`), plus an August round
(`crypto/tls` KeyUpdate DoS, an HTTP/2 preface timeout bypass, and an
`html/template` XSS — `CVE-2026-56862`/`56853`/`56858`).

## Decisions made here

- **Matched `EphemNet`'s own fix exactly** (`go 1.26.9`, `golang:1.26`
  in the Dockerfile) rather than going further to Go 1.27 — same
  product family, no reason for persona to diverge and carry a
  different toolchain version than its siblings for the same fix.
- **Dockerfile pinned to the floating `golang:1.26` tag, not the
  exact `golang:1.26.9` patch** — confirmed this matches `EphemNet`'s
  own `ephemnet-site`/`ephemnetd`/`ephemnet-agent` Dockerfiles
  directly (not assumed), which already made this same choice. The
  floating minor-version tag picks up future patch releases
  automatically on the next rebuild, which is the point of a Docker
  base-image tag for a toolchain that gets security patches.
- **`golang.org/x/crypto` bumped to `v0.58.0`** (via `go get` +
  `go mod tidy`, not hand-edited) — the same version `EphemNet`
  bumped to, and the one actually pulled in by persona's own
  dependency graph as `// indirect` (via `github.com/nbd-wtf/go-nostr`
  or one of its own dependencies — not something persona imports
  directly, but still in the real build).
- **Did not edit `I001` itself** — it's `EphemNet`'s own filed record
  of what it sent; this action references it rather than modifying
  someone else's inbox message.

## Tasks

- [x] Verify the claimed exposure for real — confirmed `go.mod`'s
      `go 1.24.1` and the Dockerfile's `golang:1.24` directly, not
      taken on the inbox message's word alone
- [x] Bump `go.mod` to `go 1.26.9` — `go build ./...` triggered a real
      toolchain fetch (`GOTOOLCHAIN=auto`) and succeeded
- [x] Bump `golang.org/x/crypto` to `v0.58.0` via `go get` + `go mod
      tidy` (also pulled `golang.org/x/sys` v0.31.0 -> v0.49.0 as a
      transitive consequence)
- [x] Full `go build ./... && go vet ./... && go test ./...` clean
- [x] Repin `deploy/persona-web/Dockerfile` from `golang:1.24` to
      `golang:1.26`, matching `EphemNet`'s own convention exactly
- [x] Real verification, not just a clean `go build`: built the actual
      Docker image (`docker build --platform linux/amd64 -f deploy/
      persona-web/Dockerfile`), ran the resulting container, and
      confirmed a real `200` serving the real site before cleaning up
      the test image/container
- [x] Deploy the bumped image to `bh2` — **done 2026-10-22**, at the
      operator's explicit go-ahead. `deploy/deploy.sh bh2 /opt/persona
      live` built `persona-web:9d3438c` locally, shipped it
      (`docker save | ssh | docker load`), and restarted the live
      containers. Verified live, not just a clean script exit:
      `https://solemn.network/` returns a real `200` with a real
      cert, and its own build-info footer reads `build 9d3438c ·
      2026-10-22T11:09:24Z` — the exact commit with this fix, not a
      stale image.

## Log

2026-10-22 — Filed and fixed in the same pass, following `EphemNet`'s
own real cross-repo notice (`I001`). Verified the exposure directly
before acting, matched the sibling repo's exact fix (`go1.26.9`,
`golang:1.26`, `x/crypto@v0.58.0`) rather than picking an independent
version, and proved the result with a real Docker build + run, not
just a clean compile. Deployment to `bh2` intentionally left as a
separate, operator-approved step — this action's own scope is the
code fix, verified working, not a live production push.

2026-10-22 (later) — Operator gave the go-ahead to deploy. Ran
`deploy/deploy.sh bh2 /opt/persona live`: built `persona-web:9d3438c`
locally, shipped it to `bh2`, restarted `persona-live-persona-web-1`/
`persona-live-persona-caddy-1`. Verified live rather than trusting
the script's own clean exit: `curl -s -o /dev/null -w "%{http_code}"
https://solemn.network/` → `200`, and the live page's own build-info
footer reads `build 9d3438c · 2026-10-22T11:09:24Z` — confirms the
running container is genuinely serving this exact fix, not a cached
or stale image. No further action needed on this gap.
