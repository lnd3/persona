# Deployment — solemn.network

Copied and adapted from `cinder`'s own `deploy/` (D007/P008 — the
original design, HTTP-01-not-TLS-ALPN-01 ACME, the shared nginx
`stream{}` skeleton) and `EphemNet`'s own `deploy/` (D003 — the
`<deploy-root>/<environment>` convention on top of cinder's pattern,
and the single-static-page Caddy shape this repo's own deploy is
closest to). See either repo's own `deploy/README.md` for the full
architecture and the real incidents that shaped it; this is just the
how-to for persona's own, much smaller slice of it — one static
landing page (`site/index.html`), no app binary of its own yet.

## Shape

```
                    :443 (real traffic, SNI passthrough)              :80 (ACME HTTP-01 only)
Internet ──▶ nginx (host, shared with cinder/EphemNet) ──────────────────┬──▶ nginx (host, Host-header vhost)
                │                                                        │
                ▼                                                        ▼
          persona-caddy                                            persona-caddy
          127.0.0.1:9470                                           127.0.0.1:9210
                │
                ▼
          site/index.html (static, read-only mount)
```

One product, one container (`persona-caddy`, `image: caddy:2` —
nothing to build), no `internal-services` network membership (nothing
here talks to cinder's or EphemNet's own containers, and nothing needs
to).

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
repo's live defaults are `172.32.1.0/24` and `9470`/`9210` — see
`deploy/docker-compose.yml`'s own comment. An earlier draft of this
tooling picked `172.31.1.0/24`/`9460`/`9200`, which turned out to
already be in live use by a fourth product on this server,
`offgridapp` (not previously known to this repo's own plan) — caught
by actually running `docker ps`/`docker network ls`/`ss -tlnp` against
`bh2` before deploying, not assumed clear from documentation alone.

## Redeploying (`deploy/deploy.sh`, run from the dev machine)

```bash
deploy/deploy.sh <user>@<server> /opt/persona live
```

Packages the current commit on `main` with `git archive` (only tracked,
committed content), bakes the build-info footer into the staged copy of
`site/index.html` (never the committed file — see `deploy/deploy.sh`'s
own comment), `rsync --delete`s it to the server's build folder
(excluding `deploy/.env`), then starts/restarts `persona-caddy`.
Refuses to run if the dev machine's working tree has uncommitted
changes outside `plan/`, if it's not actually on `main`, or if the
server's `deploy/.env` is missing.

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

## Adding a second environment (`dev`)

Same shape as cinder's/EphemNet's own second-environment setup — its
own `<deploy-root>/dev` path, its own `deploy/.env` with a distinct
`PERSONA_DOMAIN` and non-colliding ports/subnet (see
`deploy/.env.example`'s own "Second environment" section):

```bash
deploy/configure-nginx.sh <user>@<server> /opt/persona dev
deploy/deploy.sh <user>@<server> /opt/persona dev
```
