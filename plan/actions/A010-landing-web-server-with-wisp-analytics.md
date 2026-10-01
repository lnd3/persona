---
id: A010
title: Report persona's web traffic to wisp analytics (when persona has a web server)
status: DEFERRED
design: D001
project: P001
created: 2026-10-01
updated: 2026-10-01
---

## Context

`wisp` (`github.com/lnd3/wisp`, live at `https://wisp.mera.network`)
is the product group's own cookieless analytics. persona is meant to
report to it, but **not yet**. The user's decision on 2026-10-01: "If
persona is not yet a full webserver, only html, we should hold off."

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

## Integration steps (once a Go handler serves persona's pages)

1. **Dependency:** `github.com/lnd3/wisp`, the `hook` package only
   (standard library only). Pin a commit, or a tag once wisp has one.
2. **Start it** with `hook.Start(hook.Config{Endpoint:
   os.Getenv("WISP_ENDPOINT"), ProductKey: "persona", Token:
   os.Getenv("WISP_TOKEN"), ClientIP: …})`. An empty endpoint makes it
   a no-op for dev. Call `Close(ctx)` on SIGTERM.
3. **Count:** `View(r, routeTemplate)` after a successful page
   response, and `Download(r, …)` for served files. Always a route
   template, never `r.URL.Path`, since raw paths can carry
   identifiers.
4. **Client IP:** persona-caddy sends
   `header_up X-Real-IP {http.request.remote.host}`. The handler
   trusts that header only when `RemoteAddr` is inside
   `PERSONA_EDGE_SUBNET` — now `172.21.1.0/24`, fixed 2026-10-01 (the
   user caught that the original `172.32.1.0/24` was outside RFC
   1918's private range; see A009's own Log).
5. **Config:** pass `WISP_ENDPOINT`/`WISP_TOKEN` from `deploy/.env`.
   The serving container needs outbound HTTPS to `wisp.mera.network`.
6. **Register at wisp** (operator), from a wisp checkout:
   ```bash
   TOK=$(openssl rand -hex 32)          # into persona's deploy/.env as WISP_TOKEN (never commit it)
   echo "$TOK" | deploy/ops.sh bh2 /opt/wisp live register persona
   deploy/ops.sh bh2 /opt/wisp live registry   # confirm persona is listed
   ```
   `register` hashes the token locally (only the SHA-256 is sent),
   **merges** it into wisp's shared `products.json`, backs up the old
   file and restarts wisp. **Never hand-edit or rewrite
   `products.json`**: every product registers there, and on
   2026-10-01 a hand-written replacement wiped another product's entry.
7. **Verify:** the first closed day appears on wisp's dashboard at
   02:00 UTC the next day, plus up to 10 minutes.

## Log

2026-10-01 — Filed as IN_PROGRESS to add a Go landing server
(`cmd/persona-web`) at `persona.solemn.network` with the hook. The
user paused it before any code was written, and asked why persona
would need another web service. Rescoped the same day to DEFERRED,
with integration notes only. No persona code or deployment changed.

2026-10-01 (later) — Step 6 updated from the wisp repo: register through wisp's new merge-only `deploy/ops.sh … register`, never by editing `products.json` by hand.
