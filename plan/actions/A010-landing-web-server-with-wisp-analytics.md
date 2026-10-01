---
id: A010
title: Report persona's web traffic to wisp analytics
status: IN_PROGRESS
design: D001
project: P001
created: 2026-10-01
updated: 2026-10-01
---

## Context

`wisp` (`github.com/lnd3/wisp`, live at `https://wisp.mera.network`)
is the product group's own cookieless analytics. persona reports to it
as of 2026-10-01 — reversing the same day's earlier deferral, at the
user's explicit direction ("Yes, build it") after being asked directly
whether today's situation was actually different from the reasoning
that paused this the first time. The original deferral's own
reasoning (below) is kept verbatim, not deleted — it's why this needed
a real Go server to begin with, not a config flag.

**Why deferred.** wisp's `hook` must run inside the Go code handling
the visitor's request. persona's only public piece is
`site/index.html`, served by `persona-caddy`'s `file_server` (A009),
so no persona code sees a request. Adding a Go server only to feed
analytics was considered and rejected: a service with no purpose of
its own.

**When to pick this up:** when persona gets a real web-facing Go
service. That might be the relay/payment backend once cinder's L402
unblocks the build (see CLAUDE.md's handoff note), or a landing page
that needs server-side behaviour anyway. The user also wants persona
eventually published at `persona.solemn.network`, which needs an A
record at GoDaddy and a hostname in persona-caddy's Caddyfile and
nginx fragments. That can happen independently of analytics.

**wisp's rules for any integration** (wisp's CLAUDE.md and C001 are
authoritative):
- The visitor's browser never contacts wisp. The hook forwards
  per-visitor daily counts server-to-server, every 5 minutes and only
  when something is pending.
- The raw IP is used only inside the hook, to derive a day-scoped
  HMAC key under an in-memory salt discarded at UTC midnight. It is
  never logged, stored or sent.
- No cookies, no storage.

## Integration steps — implemented 2026-10-01

1. **Dependency:** `github.com/lnd3/wisp`, the `hook` package only
   (standard library only). Pinned to commit `bbb773f` — the latest
   actually pushed to `github.com/lnd3/wisp` at the time (the local
   wisp checkout on this machine had three further unpushed commits;
   checked with `git ls-remote` rather than assumed, and none of the
   three touch the `hook` package per their own commit messages, but
   worth knowing a local-vs-remote gap exists there). No tag yet.
2. **Started** in `cmd/persona-web/main.go` with
   `hook.Start(hook.Config{Endpoint: os.Getenv("WISP_ENDPOINT"),
   ProductKey: "persona", Token: os.Getenv("WISP_TOKEN"), ClientIP:
   trustedProxyClientIP(...)})`. Empty `WISP_ENDPOINT` makes it a
   no-op (confirmed locally — dev/local runs need no token).
   `w.Close(ctx)` runs on `SIGTERM`/`SIGINT`, after `srv.Shutdown`.
3. **Counted**: `w.View(r, "/")` after a real `200` from
   `http.ServeFile` — wrapped in a small `statusRecorder` so a `404`/
   `304` doesn't count as a view. `"/"` is already a route template
   (persona has exactly one real page); no `Download` calls yet, since
   there's nothing else to download.
4. **Client IP:** `persona-caddy`'s Caddyfile sends `header_up
   X-Real-IP {http.request.remote.host}`; `persona-web`'s
   `trustedProxyClientIP` trusts it only when the request's own
   `RemoteAddr` is inside `PERSONA_EDGE_SUBNET` (`172.21.1.0/24`,
   fixed 2026-10-01 — see A009's own Log). `persona-web` and
   `persona-caddy` share that one Docker network (`persona-edge`) and
   nothing else is on it, so `RemoteAddr` there is always
   `persona-caddy`'s own container IP — nothing else can forge it.
5. **Config:** `WISP_ENDPOINT`/`WISP_TOKEN` read from `deploy/.env` on
   the server, passed through as container env vars (never committed —
   `deploy/.env` is gitignored). `persona-web` has outbound HTTPS to
   `wisp.mera.network` (`gcr.io/distroless/static-debian12` base image
   carries the CA bundle needed to verify that TLS connection).
6. **Registered at wisp** (operator, from a local `wisp` checkout —
   `wisp` was already confirmed live on `bh2` before this step, not
   assumed):
   ```bash
   TOK=$(openssl rand -hex 32)
   echo "$TOK" | deploy/ops.sh bh2 /opt/wisp live register persona
   deploy/ops.sh bh2 /opt/wisp live registry   # confirmed: persona now listed alongside cinderapps/wisp/offgridapp/eph-network
   ```
   Merged cleanly (not a replacement) — existing products' entries
   confirmed still present after. `TOK` written straight into
   `/opt/persona/live/deploy/.env` on the server over SSH, then the
   local plaintext copy shredded — never committed, never left on this
   dev machine either.
7. **Deployed and verified**: `deploy/deploy.sh bh2 /opt/persona live`
   — built `persona-web`'s image on the server (first build pulled
   `golang:1.24`/`distroless/static-debian12` fresh, ~3 minutes), both
   containers started clean. Verified immediately: `https://
   solemn.network` returns `200`, `http://` still redirects, `docker
   logs persona-live-persona-web-1` shows a clean startup with no
   config errors (a bad `WISP_TOKEN`/`WISP_ENDPOINT` would have failed
   loud at `hook.Start`), several real page loads (including a `404`
   on a nonexistent path, confirmed *not* counted) produced no errors.
   wisp's own ingest log confirms persona's registration (`registry: 5
   products`, restarted at the right time). **Not yet confirmed**: an
   actual batch arriving at wisp (the hook only sends every 5 minutes
   and only when something is pending) or appearing on wisp's own
   dashboard, which needs the next UTC day boundary — that's wisp's
   own already-documented timing, not a gap in this verification.

## Log

2026-10-01 — Filed as IN_PROGRESS to add a Go landing server
(`cmd/persona-web`) at `persona.solemn.network` with the hook. The
user paused it before any code was written, and asked why persona
would need another web service. Rescoped the same day to DEFERRED,
with integration notes only. No persona code or deployment changed.

2026-10-01 (later) — Step 6 updated from the wisp repo: register through wisp's new merge-only `deploy/ops.sh … register`, never by editing `products.json` by hand.

2026-10-01 (later still) — **Reversed the deferral, at the user's
explicit direction, and implemented.** Asked directly whether today's
situation differed from the reasoning that paused this the first
time; the user confirmed and said to build it. Built `cmd/persona-web`
(see Integration steps above for the real implementation, not just a
plan), `deploy/persona-web/Dockerfile` (`gcr.io/distroless/static-
debian12`, needs the CA bundle for outbound HTTPS to wisp — same
reasoning as EphemNet's own `ephemnet-agent/Dockerfile`), updated
`deploy/docker-compose.yml` (new `persona-web` service, no host-
published port, shares `persona-edge` with `persona-caddy`),
`deploy/persona-caddy/Caddyfile` (now `reverse_proxy persona-web:8080`
instead of `file_server`, plus the `X-Real-IP` header), and
`deploy/deploy.sh`/`ops.sh`/`.env.example`/`README.md` to match.
Added `github.com/lnd3/wisp` to `go.mod`, pinned to the latest commit
actually on GitHub (`bbb773f` — the local wisp checkout had further
unpushed commits, checked via `git ls-remote` rather than assumed);
confirmed zero transitive dependencies landed (the `hook` package is
genuinely stdlib-only, as documented). Verified locally before
touching the server: `go build`/`vet`/`test ./...` clean; `go run
./cmd/persona-web` against the real `site/` directory serves `200` on
`/` and `404` on anything else; the Caddyfile validates against the
real `caddy:2` image; the Docker image builds and, run standalone,
serves the page correctly. Registered `persona` at wisp (merged
cleanly, confirmed via `registry`), wrote `WISP_ENDPOINT`/`WISP_TOKEN`
into the server's `deploy/.env` over SSH, shredded the local plaintext
token copy. Deployed with `deploy/deploy.sh` and verified live — see
step 7 above for the exact checks. Status moved to `IN_PROGRESS` (not
`DONE`): the integration is live and erroring nowhere, but a real
batch actually reaching wisp's dashboard hasn't been confirmed yet —
check back after the next UTC day boundary.
