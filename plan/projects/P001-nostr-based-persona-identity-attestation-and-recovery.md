---
id: P001
title: Nostr-based persona identity, attestation, and recovery
status: PLANNING
priority: MEDIUM
priority_drivers:
  - strategic_edge
created: 2026-09-21
updated: 2026-09-21
depends: []
external_dependencies:
  - cinder P004 / L402 (Lightning payment rail) — required for
    sybil-resistance payments and sign-in billing; cinder's own side
    (D005, Aperture-fronted) is built and verified live, but cinder's
    paid tier is not fully deployed end-to-end (Aperture deployment,
    real Lightning provider account still outstanding on cinder's
    side, deferred there) — see P001's 2026-09-21 log entry
enables: []
---

## Naming

Domain chosen: `sovranpersona.com` (2026-09-22) — keeps "persona" (the
repo/thesis name throughout this plan) while adding a sovereignty/
freedom connotation via "sovran" (archaic spelling of "sovereign,"
chosen over the plain word to avoid colliding with **Sovrin**, an
existing self-sovereign-identity project). Repo/project name stays
"persona"; this is the product-facing domain/brand name, not a repo
rename.

Second domain now owned (2026-09-28): `solemn.network` — not yet
assigned a role (product brand vs. e.g. the attestation-cost-gateway
reference deployment vs. something else). Worth noting: "solemn" (a
solemn oath/vow) is thematically a strong fit specifically for the
attestation primitive, arguably a tighter match than "sovran" is for
the identity/recovery side — not decided which domain ends up
fronting what, or whether both stay in use for different pieces.

Hosting already available (2026-09-27), separate from the above:
`persona.cinderapps.org` (live) and `dev4637.persona.cinderapps.org`
(dev) — subdomains of `cinder`'s existing `cinderapps.org` multi-
product deployment (see `cinder`'s `deploy/README.md`; `cinderapps`
is `cinder`'s own landing-site product, and its Caddy sidecar derives
product subdomains from `CINDERAPPS_BASE_DOMAIN` alone). `cinder`'s
`deploy/` scripts (`deploy.sh`, `configure-nginx.sh`, `ops.sh`,
`environments.sh`, per-product `Caddyfile`/`Dockerfile` under
`deploy/<product>/`) are the pattern to copy into this repo once
there's something real to publish here — not before.

**Incident to carry forward when that copy happens** (`cinder` commit
8689652, 2026-09-27): `configure-nginx.sh`'s nginx fragment filenames
(`/etc/nginx/stream-backends.d/<tag>.map`,
`sites-enabled/<tag>-http01.conf`) were bare `<environment>`
("live"/"dev") — fine for the Compose project name and the
`<deploy-root>/<environment>` directory, both already isolated by
their own structure, but those two nginx directories are **shared
across every repo running its own equivalent of this script on the
same server** (`cinder`, `EphemNet`, this repo). Bare "live"/"dev"
would let one repo's script silently overwrite another's nginx
routing. Fixed in `cinder` by prefixing with its own repo name
(`cinder-live`, `cinder-dev`). When this repo copies `cinder`'s
`deploy/` scripts, its own `ENV_TAG` must be prefixed the same way
(`persona-live`, `persona-dev`), not left bare — copy the fixed
version of `configure-nginx.sh`/`environments.sh`, not a pre-8689652
one.

## Goal

Build [[T001]]: a persistent, keypair-anchored persona identity on
Nostr's existing primitives, with a real, designed answer to the two
hardest problems that determine whether this actually works — sybil
resistance for attestation, and key loss/theft recovery. Full design
in [[D001]].

## Scope

- In scope: the identity/event format (Nostr keys and events, NIP-98
  for sign-in, NIP-58-pattern attestations), the attestation primitive
  and its trust/sybil-resistance model, and the recovery mechanism
  (SSKR-based social recovery, authorized via the same attestation
  primitive).
- In scope, standing requirement: any network service this project
  stands up (e.g. an attestation-cost gateway backend, a relay) gets a
  web UI with API reference, documentation, and status sections —
  matching `cinder`'s own pattern (its API reference page). Applies
  once there's an actual service to build it for.
- In scope, standing requirement: a client application holding the
  user's keys across their different personas (one person, multiple
  context-specific personas per the naming discussion), plus the
  product site around it — landing page, a presentation, and a
  download page for that application. Site should carry origin/author
  and contact info. Not yet designed — recorded as a known
  requirement, not a spec.
- Not in scope yet: implementation. Design-stage project, matching
  where `cinder` was between its founding thesis and its own first
  design docs.
- Not in scope: this project's own payment rail — depends on `cinder`'s
  P004/L402, sequenced after it per `superplan`'s M002 build order.
- Explicitly not yet decided: where this actually runs (own relays,
  the existing public Nostr network, or `EphemNet`-routed self-hosted
  relays — now concretely possible, not yet chosen), Nostr interop
  scope, and the bonding/slashing mechanics for disputed attestations.

## Linked

- **Thesis**: [[T001]]
- **Design**: [[D001]]
- **Dependency**: `cinder`'s P004 (L402 payment rail)
- **Related repo**: `EphemNet` — its DNS-forwarding capability makes
  self-hosted relays a real option for this project's own "where does
  this run" question
- **Origin**: this repo's design was previously tracked only in
  `superplan` (theses T010/T011, project P008, design D003) — see that
  repo for the earlier design-formation history; this repo's own plan
  is now the source of truth for anything persona-identity-specific
  going forward

## Tasks

- [x] Consolidate the design-formation conversation into a real
      thesis and project (this action)
- [ ] Resolve remaining open questions: where this runs, Nostr interop
      scope
- [x] Design payment integration against `cinder`'s D005 pattern
      (Aperture-fronted attestation-cost gateways) — done 2026-09-22,
      see D001
- [x] Resolve bonding/slashing dispute mechanics — done 2026-09-27,
      scoped to one dispute type only (behavioral/quality disputes
      between two identified parties); sybil-ring vouching, ownership/
      provenance, and recovery disputes explicitly need separate,
      still-undesigned mechanisms — see D001
- [ ] Wait on `cinder` actually deploying its own paid tier live
      (Aperture config, a real Lightning provider account) only if/when
      this project wants to stand up its own reference gateway;
      otherwise no longer a hard blocker to this project's own design
      or implementation work

## Log

2026-09-21 — Project seeded properly for the first time — this repo
previously didn't exist at all; the design lived only in `superplan`.
Named "persona" (Loom considered and rejected due to an existing,
unrelated cryptocurrency by that name).

2026-09-21 — Checked `cinder`'s P004/L402 status: moved to DEFERRED
the same day, but not stalled — cinder's own side of the L402 write
path (internal paid listener, shared-secret middleware, 30-day TTL
ceiling, 9 tests) is built and verified live, and D005 was revised to
front payment through `lightninglabs/aperture` (a production L402
reverse proxy) rather than cinder implementing macaroons/invoicing
itself. Deferred per the user's explicit call: the goal was always "a
concrete pattern to follow" for `EphemNet` and this project, not
cinder's own paid tier fully live — that bar is now met. Still
outstanding on cinder's side, separately: Aperture deployment/config
and a real hosted Lightning provider account, needed only for cinder's
*own* paid tier to go fully live end-to-end. This project is no longer
blocked on that remaining piece — D005 gives a real, live-verified
reference architecture to design against now.

2026-09-22 — Payment integration designed in D001 against that
reference: per-operator "attestation-cost gateway" (Aperture-fronted,
same as `cinder`'s paid listener) issuing signed receipts embedded in
attestation events, with each verifier deciding which gateway
operators it trusts — reusing D005's mechanism while keeping the
relative-trust model intact. Sign-in billing mapped onto the same
shape. This closes the design gap that was this project's main
remaining hard dependency on `cinder`.

2026-09-22 — Added a standing scope requirement: any network service
this project creates gets a web UI with API reference, documentation,
and status sections.

2026-09-22 — Chose `sovranpersona.com` as the product domain/brand
name, after a naming brainstorm (candidates considered included
Mantle, Covenant, Shard, Freehold, Agora — see chat log for full
reasoning). Repo/project name unchanged ("persona").

2026-09-22 — Recorded a known-but-undesigned requirement: a client
application holding the user's keys across their personas, plus a
product site (landing page, presentation, download page, origin/
author/contact info). Not yet designed.

2026-09-27 — Recorded existing hosting access: `persona.cinderapps.org`
(live) and `dev4637.persona.cinderapps.org` (dev), subdomains of
`cinder`'s own multi-product `cinderapps.org` deployment. Decided to
copy `cinder`'s `deploy/` scripts into this repo once there's
something real to publish, rather than building a deployment setup
from scratch.

2026-09-27 — Checked `cinder`'s latest deploy/ changes: a critical fix
(commit 8689652) product-qualified `configure-nginx.sh`'s shared
nginx fragment filenames (previously bare "live"/"dev", collidable
across every repo sharing that server's nginx directories). This
repo has no `deploy/` of its own yet, so nothing to patch here today —
recorded the requirement above so the eventual copy starts from the
fixed version with this repo's own `persona-` prefix, not a
pre-8689652 one.

2026-09-27 — Resolved D001's bonding/slashing open question (see
D001's own log for the full reasoning): split disputes into four
types by what's actually being claimed, and found arbiter-adjudicated
bonding/slashing is a good fit for exactly one of them (behavioral/
quality disputes between two identified parties) and a poor-to-wrong
fit for the other three (sybil-ring vouching, ownership/provenance,
recovery disputes), which each need a different, still-undesigned
mechanism instead of being forced through one.

2026-09-28 — Second domain acquired: `solemn.network`, role not yet
assigned relative to `sovranpersona.com`.
