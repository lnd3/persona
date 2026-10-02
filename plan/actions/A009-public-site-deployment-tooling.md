---
id: A009
title: Public site deployment tooling (solemn.network)
status: DONE
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
- [x] One-time server-side provisioning on `bh2` — done 2026-09-30.
      SSH access to `bh2` turned out to be available this session
      (`~/.ssh/config`'s `Host bh2` entry); `/opt/persona/live/deploy/.env`
      written, `docker`-group membership already in place from prior
      setup
- [x] `deploy/configure-nginx.sh bh2 /opt/persona live` — run
      2026-09-30, installed persona's own nginx fragments
      (`persona-live.map`, `persona-live-http01.conf`), `nginx -t`
      passed, reloaded cleanly
- [x] DNS fixed — `solemn.network` had three A records (the real
      server plus two GoDaddy Domain Forwarding IPs, a different
      GoDaddy feature from "Parking" — same underlying AWS Global
      Accelerator host for both, so disabling Forwarding removed both
      at once); operator disabled both Parking and Forwarding, verified
      clean against both a public resolver and GoDaddy's own
      authoritative nameserver before deploying
- [x] `deploy/deploy.sh bh2 /opt/persona live` — run 2026-09-30,
      `persona-caddy` started, a real Let's Encrypt certificate for
      `solemn.network` obtained on the first attempt (HTTP-01, no
      retries needed — DNS was clean by the time this ran).
      **Verified live**: `https://solemn.network` returns 200, TLS
      verifies clean, build-info footer confirms the exact deployed
      commit

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

2026-09-30 — **Live on `bh2` — action complete.** Found and fixed the
flagged port/subnet collision before deploying (real `docker
ps`/`network ls`/`ss` check caught `offgridapp` already on the original
defaults). Found and the operator fixed a real DNS issue (`solemn.network`
had two extra GoDaddy Domain Forwarding A records, a different
GoDaddy feature from "Parking" the operator had already disabled;
verified clean via both a public resolver and GoDaddy's own
authoritative nameserver before deploying, rather than assuming the fix
worked). `configure-nginx.sh` (touches shared nginx config, held for
explicit go-ahead per this session's own risk-confirmation discipline)
and `deploy.sh` (triggers a real Let's Encrypt request, held until DNS
was confirmed clean) both run successfully — a real certificate was
obtained on the first attempt, no wasted ACME attempts. Verified with a
real `curl` against the live HTTPS endpoint, not just trusting the
deploy script's own "Done" output: 200, clean TLS, build-info footer
matching the exact deployed commit. Status moved to `DONE`.

2026-09-30 — **Real gap found post-deploy, same day: no `http://` →
`https://` redirect.** `curl -I http://solemn.network` returned a bare
502 from nginx, not a redirect — `auto_https disable_redirects` (in
the Caddyfile from the start, copied from cinder's/EphemNet's own
pattern) only disables Caddy's *automatic* redirect listener, which
cinder's own D007 Log calls "meaningless" there because that
architecture's real traffic only ever arrives via nginx's `:443` SNI
passthrough — not a safe assumption for a landing page real visitors
might reach by typing a bare domain or following a `http://` link.
**Same gap confirmed present, unfixed, in both cinder's and EphemNet's
own Caddyfiles** — worth a cross-repo pointer (see P001's `Linked`
section) rather than editing either repo directly. Fixed here with an
explicit `http://{$PERSONA_DOMAIN} { redir https://{host}{uri}
permanent }` block — safe alongside the ACME HTTP-01 responder per
D007's own confirmation that Caddy's internal challenge handler stays
active independent of site-block routes. Validated against the real
`caddy:2` image (`caddy validate`) before deploying, then confirmed
live: `curl -I http://solemn.network` now returns a clean `301` to
`https://solemn.network/`, HTTPS unaffected.

2026-10-01 — **Real bug in this action's own subnet pick, caught by
the user, not this session.** `PERSONA_EDGE_SUBNET`'s live default,
`172.32.1.0/24` (picked during the port/subnet collision check above),
is not actually RFC 1918 private space — `172.16.0.0/12` covers only
`172.16.0.0`-`172.31.255.255`; `172.32.x.x` is real, publicly-routable
IPv4 address space. Not a direct traffic leak (still Docker
bridge/NAT-isolated), but a real route-table collision risk: this
host or a container could wrongly prefer the local bridge route over
the real internet route for any genuine public host inside that
block. Corrected to `172.21.1.0/24` (live) / `172.21.2.0/24` (dev) —
confirmed free at the original 2026-09-30 check (`172.21.0.0/16`
through `172.27.0.0/16` were entirely unused on `bh2`) and correctly
inside the private range this time. Updated
`docker-compose.yml`/`.env.example`/`README.md`; redeploying to
actually apply it on `bh2` is this entry's own next step.

2026-10-02 — **Hardened `deploy.sh` against two real gaps cinder's own
`deploy.sh` already found and fixed** (pointed at directly: "Look at
cinder deploy, they clear build cache after every product build").
(1) Build cache is now pruned (`docker builder prune -a -f`, plus
`docker image prune -f`) right after every build, not just once at the
end — cinder's own comment names the exact reason: `bh2` is an
8.7G-disk server shared by every product on it, and Docker's cache is
never pruned on its own. **Live and urgent while this was being
applied**: `bh2` actually hit 75MB free / 100% used mid-session, from
a different session's (`wisp`) in-progress build compiling
`modernc.org/sqlite` inside `golang:1.24` — not caused by this change,
but a real-time demonstration of exactly the failure mode this fix
guards against. Flagged to that session directly rather than pruning
anything while their build was still in flight (their cache showed
`RECLAIMABLE: false` via `docker buildx du`, confirming a prune
wouldn't have been safe regardless); they killed their own build and
reclaimed it themselves, disk settled at 1.8G free (80%) before this
fix was actually committed/run. (2) Added an unconditional `docker
compose restart persona-caddy` after every `up -d` — Compose's own
change detection doesn't notice a bind-mounted Caddyfile's *content*
changing, only the service definition itself, so a Caddyfile-only
edit would silently never take effect. This exact symptom already
hit this repo once (the http->https redirect fix needed a manual
`ops.sh ... restart` to actually apply, in A009's own earlier Log
entry) — cinder's `deploy.sh` had already independently hit and fixed
the identical thing, confirming it as a real, recurring gap worth
closing permanently rather than working around by hand each time.

2026-10-02 (later, same day) — **Superseded building on `bh2`
entirely, copying `wisp`'s own follow-up fix** ("Wisp build docker
snapshot now instead of building on bh2. Copy it."). `wisp` resolved
its own disk emergency (above) by never building on `bh2` again —
cross-compiling `persona-web`'s image on the dev machine and shipping
it with `docker save | gzip | ssh | gunzip | docker load`, matching
`wisp`'s own `deploy/wisp/Dockerfile`/`deploy/deploy.sh` pattern
exactly: `deploy/persona-web/Dockerfile` now uses `FROM
--platform=$BUILDPLATFORM golang:1.24` with `ARG TARGETOS TARGETARCH`
so the build stage runs natively on the dev machine while
cross-compiling for the server's real architecture (detected via `ssh
... uname -m`) — no QEMU emulation needed, since Go cross-compiles
natively and the binary has no CGO dependency.
`deploy/docker-compose.yml`'s `persona-web` service changed from
`build:` to `image: persona-web:current`; `deploy.sh` now builds
locally, ships the image, refuses to start if the server has under
300MB free (copied from `wisp`'s own guard, same reasoning: `bh2` is
shared), tags the loaded image `persona-web:current`, and keeps the
two most recent commit-tagged images for rollback. The per-build
`docker builder prune -a -f` added earlier this same day is now
largely moot for `persona-web` specifically (nothing builds on the
server anymore to leave cache behind) but left in place — harmless,
and still correct if this repo ever adds a second server-side-built
service. Verified before touching the server: local cross-build
(`docker build --platform linux/amd64`) succeeds, the image runs
standalone and serves the real site correctly (`200` on `/`, `404`
elsewhere). `bh2`'s own disk was rechecked clean (3.4G free, 62% used)
before deploying for real.
