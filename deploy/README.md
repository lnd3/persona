# Deployment — solemn.network

Copied and adapted from `cinder`'s own `deploy/` (D007/P008 — the
original design, HTTP-01-not-TLS-ALPN-01 ACME, the shared nginx
`stream{}` skeleton) and `EphemNet`'s own `deploy/` (D003 — the
`<deploy-root>/<environment>` convention on top of cinder's pattern).
See either repo's own `deploy/README.md` for the full architecture and
the real incidents that shaped it; this is just the how-to for
persona's own slice of it — one landing page (`site/index.html`), now
served by a real Go binary (`persona-web`, since A010/2026-10-01) so it
can report page views to wisp.

## Shape

```
                    :443 (real traffic, SNI passthrough)              :80 (ACME HTTP-01 only)
Internet ──▶ nginx (host, shared with cinder/EphemNet) ──────────────────┬──▶ nginx (host, Host-header vhost)
                │                                                        │
                ▼                                                        ▼
          persona-caddy                                            persona-caddy
          127.0.0.1:9470                                           127.0.0.1:9210
                │ reverse_proxy, header_up X-Real-IP
                ▼
          persona-web:8080 (persona-edge network, no host port)
                │
                ├──▶ site/index.html (static, read-only mount)
                └──▶ wisp.mera.network/v1/ingest (page views, server-to-server)
```

Two services since A010: `persona-web` (`cmd/persona-web`, a real Go
binary) and `persona-caddy` (`image: caddy:2`, nothing to build),
which still does all TLS termination and never talks to wisp itself.
No `internal-services` network membership (nothing here talks to
cinder's or EphemNet's own containers, and nothing needs to).

**`persona-web` is built on the dev machine and shipped as an image
(`docker save | ssh | docker load`), never built on `bh2`** — changed
2026-10-02, copied from `wisp`'s own identical fix: `bh2` is an 8.7G
disk shared by every product on it, and building there (the
`golang:1.24` toolchain pull plus compile, even for a small binary)
filled it during wisp's own deploy that day. `deploy/deploy.sh` needs
a local Docker daemon on the dev machine now — it cross-compiles for
the server's actual architecture (detected over SSH) using Docker's
own `--platform`/`$BUILDPLATFORM` support, so this works the same way
from an amd64 or arm64 dev machine either way, no QEMU emulation
needed (Go cross-compiles natively).

## One-time setup

Same "no GitHub access on the server, push from a dev machine instead"
model as cinder/EphemNet — see cinder's own `deploy/README.md` for why.

1. On the server: create the pre-deployment build folder for the
   `live` environment, e.g.
   `sudo mkdir -p /opt/persona/live && sudo chown $(whoami) /opt/persona/live`
   — an empty directory owned by whatever user you'll SSH in as;
   `deploy/deploy.sh` populates it via `rsync`, no `git` on the server
   at all.
2. On the server: confirm that user is already in the `docker` group
   (it should be already, from cinder's/EphemNet's own setup on this
   same server) — `deploy/deploy.sh`'s remote commands run
   `docker compose ...` with no `sudo` in front of them.
3. On the server:
   `mkdir -p /opt/persona/live/deploy && cd /opt/persona/live/deploy`
   and create `.env` by hand (there's nothing to copy from yet — the
   repo hasn't been synced there) with `PERSONA_DOMAIN` set (see
   `deploy/.env.example`). This file lives only on the server, is never
   part of what gets synced from the dev machine, and survives every
   future deploy untouched (see `deploy/deploy.sh`'s own
   `rsync --exclude`).
4. Point `solemn.network`'s domain root (not a subdomain, for now — see
   `deploy/.env.example`'s own comment; this is an ordinary,
   traditionally-served GoDaddy domain, deliberately NOT an
   EphemNet-delegated apex domain the way `mera.network` is — the two
   are kept separate on EphemNet's own recommendation) at this server's
   IPv4 address, `158.174.211.245`, with exactly **one** A record — add
   AAAA too only if the server has real IPv6 connectivity.
   **Checked live 2026-09-30: this domain currently has THREE A
   records** — `158.174.211.245` (correct) plus two GoDaddy
   parking/forwarding IPs (`3.33.130.190`, `15.197.148.33`) left over
   from before this domain pointed anywhere real. Remove those two in
   GoDaddy's DNS panel before step 8 — with three round-robin A
   records, Let's Encrypt's own HTTP-01 validator has roughly a 2-in-3
   chance of hitting GoDaddy's parking page instead of this server,
   which will make certificate issuance flaky or fail outright.
5. Confirm the host firewall already allows `443`/`80` (it should,
   from cinder's/EphemNet's own setup) — see cinder's own
   `deploy/README.md` "Firewall" section if not.
6. Confirm nginx + the `stream` module + the one-time `nginx.conf`
   include line are already installed (they should be, from cinder's
   own one-time setup — see that repo's `deploy/README.md`,
   "One-time setup" steps 6-7). This repo's own
   `deploy/configure-nginx.sh` checks for the shared skeleton file and
   fails with clear instructions if it's missing, rather than trying
   to install it itself.
7. On the **dev machine**, from a clone of this repo:
   `deploy/configure-nginx.sh <user>@<server> /opt/persona live` —
   installs this repo's own nginx site config, reading `PERSONA_DOMAIN`
   straight from the server's own `.env` (step 3 above) over SSH.
8. On the **dev machine**:
   `deploy/deploy.sh <user>@<server> /opt/persona live` — first real
   sync and start.

**Port/subnet collision already checked live (2026-09-30)**: this
repo's live defaults are `172.21.1.0/24` and `9470`/`9210` — see
`deploy/docker-compose.yml`'s own comment. An earlier draft of this
tooling picked `172.31.1.0/24`/`9460`/`9200`, which turned out to
already be in live use by a fourth product on this server,
`offgridapp` (not previously known to this repo's own plan) — caught
by actually running `docker ps`/`docker network ls`/`ss -tlnp` against
`bh2` before deploying, not assumed clear from documentation alone.
**Corrected again 2026-10-01**: the subnet picked at that check,
`172.32.1.0/24`, turned out to be outside RFC 1918's private range
entirely (172.16.0.0/12 only covers up to 172.31.255.255) — caught by
the user, not this session. Moved to `172.21.1.0/24`, confirmed free
at the same 2026-09-30 check and correctly private.

## Redeploying (`deploy/deploy.sh`, run from the dev machine)

```bash
deploy/deploy.sh <user>@<server> /opt/persona live
```

Packages the current commit on `main` with `git archive` (only tracked,
committed content), bakes the build-info footer into the staged copy of
`site/index.html` (never the committed file — see `deploy/deploy.sh`'s
own comment), builds the `persona-web` image **locally** (cross-
compiled for the server's architecture), ships it with `docker save |
ssh | docker load`, `rsync --delete`s everything else to the server's
build folder (excluding `deploy/.env`), then starts/restarts both
services — refusing to start at all if the server has less than 300MB
free (`bh2` is shared by every product on it). Refuses to run if the
dev machine's working tree has uncommitted changes outside `plan/`, if
it's not actually on `main`, if there's no local Docker daemon, or if
the server's `deploy/.env` is missing.

```bash
deploy/deploy.sh <user>@<server> /opt/persona live --branch=some-other-branch
```

## Day-to-day operations (`deploy/ops.sh`, run from the dev machine)

```bash
deploy/ops.sh <user>@<server> /opt/persona live status
deploy/ops.sh <user>@<server> /opt/persona live logs
deploy/ops.sh <user>@<server> /opt/persona live restart
deploy/ops.sh <user>@<server> /opt/persona live stop
deploy/ops.sh <user>@<server> /opt/persona live down
```

## Verifying it actually works, once deployed

- `docker exec live-persona-caddy-1 find /data/caddy/certificates -type f`
  — a real cert actually landed (direct proof HTTP-01 succeeded).
- `curl -v https://solemn.network` — real cert, real page.
- Confirm the build-info footer at the bottom of the page shows the
  commit you actually just deployed, not a stale one.
- `deploy/ops.sh <user>@<server> /opt/persona live logs persona-web` —
  confirm it's listening and not erroring on startup (a missing/wrong
  `WISP_TOKEN` or malformed `WISP_ENDPOINT` fails loud at `hook.Start`,
  not silently).
- wisp's own dashboard shows the first closed day for `persona` at
  02:00 UTC the next day, plus up to 10 minutes (see A010's own
  integration notes, step 7).

## Adding a second environment (`dev`)

Same shape as cinder's/EphemNet's own second-environment setup — its
own `<deploy-root>/dev` path, its own `deploy/.env` with a distinct
`PERSONA_DOMAIN` and non-colliding ports/subnet (see
`deploy/.env.example`'s own "Second environment" section):

```bash
deploy/configure-nginx.sh <user>@<server> /opt/persona dev
deploy/deploy.sh <user>@<server> /opt/persona dev
```
