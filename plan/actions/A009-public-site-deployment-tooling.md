---
id: A009
title: Public site deployment tooling (solemn.network)
status: PLANNING
design: D001
project: P001
created: 2026-09-30
updated: 2026-09-30
---

## Context

[[P001]]'s 2026-09-22 Log entry flagged a "product site (landing page,
presentation, download page, origin/author/contact info)" as a known
but undesigned requirement. The landing page itself
(`site/index.html`) was built the same session this action was filed.
This action is the deployment tooling to actually publish it, copied
and adapted directly from `cinder`'s own `deploy/` (D007/P008 — the
original HTTP-01/shared-nginx-skeleton design) and `EphemNet`'s own
`deploy/` (D003 — the `<deploy-root>/<environment>` convention layered
on top, and the single-static-page Caddy shape this repo's own deploy
is closest to), not reinvented — same server (`bh2`), same convention,
same real incidents already paid for by those two repos.

## Decisions made here

- **Scope: one static page, no app binary.** persona has no `cmd/`
  daemon yet — every `internal/` package is a library with its own
  tests, not a running service. So this repo's `deploy/` is smaller
  than either cinder's (five products, real Go binaries) or EphemNet's
  (two products, one real Go binary plus one static-only) — one
  `persona-caddy` service (`image: caddy:2`, nothing to build) serving
  `site/index.html` as a static file tree, same shape EphemNet already
  used for its own two landing pages.
- **Domain: `solemn.network`'s domain root, at the operator's explicit
  direction (2026-09-30)** — not `persona.solemn.network`, the
  subdomain P001's "Naming" section otherwise documents as the
  longer-term home for persona's own page. Correction made the same
  day: this is an ordinary, traditionally-served GoDaddy domain,
  deliberately NOT an EphemNet-delegated apex domain the way
  `mera.network` is — the two are kept separate on EphemNet's own
  recommendation, per direct user correction. Recorded as a real
  decision, not an oversight — `deploy/.env.example`'s own comment
  cross-references this.
- **Ports/subnet: checked live against `bh2` itself (2026-09-30), not
  just inferred from other repos' docs.** SSH access to `bh2` turned
  out to be available this session (`~/.ssh/config`'s `Host bh2`
  entry) — used to actually run `docker ps`/`docker network ls`/
  `ss -tlnp` before finalizing anything. This caught a real collision:
  the original defaults (`172.31.1.0/24`, `9460`/`9200`) were already
  in live use by a fourth product on this server, `offgridapp` (not
  previously known to this repo's own plan). Corrected to the next
  free block: `172.32.1.0/24` (live) / `172.32.2.0/24` (dev), ports
  `9470`/`9210` (live) `9570`/`9310` (dev). Confirmed live ranges at
  check time: cinder `172.28.0.0/24`/`172.29.0.0/24`,
  `9180-9184`/`9443-9447` (live) `9280-9284`/`9543-9547` (dev);
  EphemNet `172.30.1.0/24`, `9190`/`9450`; `offgridapp`
  `172.31.1.0/24`, `9200`/`9460`.
- **Real DNS issue found live (2026-09-30): `solemn.network` currently
  resolves to three A records**, not one — `158.174.211.245` (`bh2`,
  correct) plus two GoDaddy parking/forwarding IPs
  (`3.33.130.190`, `15.197.148.33`) confirmed by direct `curl` to each
  (both serve a generic 114-byte GoDaddy parking page). With three
  round-robin A records, Let's Encrypt's own HTTP-01 validator has
  roughly a 2-in-3 chance of hitting the wrong IP, making certificate
  issuance flaky or fail outright — and risking a wasted attempt
  against Let's Encrypt's own rate limits. This is the operator's own
  registrar panel to fix (removing the two non-`bh2` A records), not
  something fixable from this session — `deploy/deploy.sh` (the step
  that triggers a real ACME request) is deliberately held pending that
  fix, even though everything else server-side is otherwise ready.

## Tasks

- [x] `deploy/environments.sh` — `live`/`dev` closed set, copied
      directly from cinder's/EphemNet's own (same incidents already
      paid for there: a free-form remote path, nginx env-tag, and
      Compose project name silently drifting apart or colliding across
      repos on the same server)
- [x] `deploy/docker-compose.yml` — one `persona-caddy` service,
      pinned subnet/ports (see Decisions above), no other services
- [x] `deploy/persona-caddy/Caddyfile` — HTTP-01 ACME (not
      TLS-ALPN-01, per cinder's own D007 lesson), PROXY-protocol
      listener wrapper scoped to `persona-edge`'s own subnet, static
      `file_server` over `/srv/site`
- [x] `deploy/nginx/stream-backends.d/persona.map.template` +
      `deploy/nginx/sites-enabled/persona-http01.conf.template` —
      persona's own per-repo nginx fragments, `.template` +
      `envsubst`, never touching the shared skeleton file
- [x] `deploy/configure-nginx.sh` — checks (never writes) the shared
      skeleton exists, installs persona's own fragments under the
      `persona-<environment>` env-tag
- [x] `deploy/deploy.sh` — `git archive` + `rsync --delete` (excluding
      `deploy/.env`) + build-info footer substitution into the staged
      copy of `site/index.html` + `docker compose up -d`; refuses to
      run on uncommitted changes (outside `plan/`) or the wrong branch
- [x] `deploy/ops.sh` — status/logs/start/stop/restart/down against an
      already-deployed stack
- [x] `deploy/.env.example` + `deploy/README.md` — one-time setup
      steps, port/subnet reasoning, dev-environment copy-paste block
- [x] `.gitignore`: `/deploy/.env` (never committed — real domain
      config lives only on the server)
- [x] Sanity-checked: `bash -n` on all three scripts, YAML-parsed
      `docker-compose.yml` — no live server access to verify further
      in this environment
- [ ] One-time server-side provisioning on `bh2` (the operator's own
      access) — not started, see Decisions above
- [ ] First real `deploy/configure-nginx.sh` + `deploy/deploy.sh` run
      against the live server — not started, depends on the above

## Log

2026-09-30 — Filed and the tooling built in the same pass, copying
cinder's/EphemNet's own `deploy/` conventions directly rather than
designing a new one — same server, same real incidents already found
and fixed by those two repos (bare "live"/"dev" collisions across
repos' nginx fragments and Compose project names; TLS-ALPN-01's
unresolved production failure; the Compose `.env`
`$`-in-bcrypt-hash interpolation gotcha, not applicable here since this
repo has no Basic-Auth-gated page yet). Scoped down from both existing
repos' patterns to match what persona actually has today: one static
page, no app binary, no `internal-services` network membership.
Domain, ports, and subnet all recorded as real decisions with their
own reasoning, including the one still-unverified piece (port/subnet
collision against the real server) named explicitly rather than
assumed clear.
