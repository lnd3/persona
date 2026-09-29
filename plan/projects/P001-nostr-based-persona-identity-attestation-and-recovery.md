---
id: P001
title: Nostr-based persona identity, attestation, and recovery
status: IN_PROGRESS
priority: MEDIUM
priority_drivers:
  - strategic_edge
created: 2026-09-21
updated: 2026-09-28
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
- Implementation now underway (2026-09-28), starting with [[A001]]:
  the core identity and attestation event layer. No longer purely a
  design-stage project.
- Not in scope: this project's own payment rail — depends on `cinder`'s
  P004/L402, sequenced after it per `superplan`'s M002 build order.
- Resolved (2026-09-28): runs on the public Nostr relay network, full
  wire-format compatibility, not a fork; `EphemNet`-routed self-hosting
  stays a supported option, not a v1 investment. `claim_type`
  namespace governance also resolved (2026-09-28): reverse-domain
  namespacing plus an open, non-authoritative NIP-style spec registry.
  Resolved (2026-09-27): bonding/slashing mechanics, scoped to one
  dispute type. See D001 for all three.

## Linked

- **Thesis**: [[T001]]
- **Design**: [[D001]]
- **Dependency**: `cinder`'s P004 (L402 payment rail)
- **Related repo**: `EphemNet` — its DNS-forwarding capability is what
  keeps self-hosted relays a real, supported (though not v1) option
  now that "where this runs" has resolved to the public Nostr relay
  network by default
- **Idea worth carrying over to `EphemNet` (noted 2026-09-28, not
  designed there yet)**: D001's `claim_type` namespace-governance
  pattern — reverse-domain namespacing for permission-free minting,
  plus an open, non-authoritative registry for convergence, with
  disputes resolved by per-verifier choice of whose authority to
  trust rather than one canonical arbiter — looks structurally like a
  good fit for `EphemNet`'s own domain-naming/dispute problem (checked
  2026-09-28: nothing under this in `EphemNet`'s own `plan/` yet).
  Deliberately left as a pointer here, not started in `EphemNet`'s
  plan — that's real design work belonging in that repo's own
  project/design structure, not a drive-by from this repo's context.
- **Origin**: this repo's design was previously tracked only in
  `superplan` (theses T010/T011, project P008, design D003) — see that
  repo for the earlier design-formation history; this repo's own plan
  is now the source of truth for anything persona-identity-specific
  going forward

## Tasks

- [x] Consolidate the design-formation conversation into a real
      thesis and project (this action)
- [x] Resolve where this runs and Nostr interop scope — done
      2026-09-28: public Nostr relay network, not a dedicated persona
      relay network; full wire-format compatibility, not a fork — see
      D001
- [x] Resolve `claim_type` namespace governance — done 2026-09-28:
      reverse-domain namespacing plus an open, non-authoritative
      NIP-style spec registry — see D001
- [x] Design payment integration against `cinder`'s D005 pattern
      (Aperture-fronted attestation-cost gateways) — done 2026-09-22,
      see D001
- [x] Resolve bonding/slashing dispute mechanics — done 2026-09-27,
      scoped to one dispute type only (behavioral/quality disputes
      between two identified parties); sybil-ring vouching, ownership/
      provenance, and recovery disputes explicitly need separate,
      still-undesigned mechanisms — see D001
- [x] Resolve GDPR right-to-erasure tension — done 2026-09-28:
      architectural avoidance first, crypto-shredding fallback,
      per-operator controller responsibility — see D001. **All five of
      D001's original open questions are now resolved; D001 moved to
      DONE.**
- [x] File the first implementation action against the now-complete
      D001 — done 2026-09-28, see [[A001]]: core identity and
      attestation event layer, scoped tight (identity + the
      attestation event itself), everything else in D001 explicitly
      deferred to later actions
- [x] Implement A001 — done 2026-09-28, moved to DONE. Go module
      `github.com/lnd3/persona`, `internal/identity`,
      `internal/attestation`, `internal/nip98`; all tests pass
      including a live publish/fetch round trip against a real public
      relay. This repo now has working code, not just a plan.
- [x] File [[A002]]: relative-trust computation — done 2026-09-28,
      builds on A001, scoped to the web-of-trust graph-weighting
      algorithm only (payment/bond signals explicitly deferred)
- [x] Implement A002 — done 2026-09-29, moved to DONE. New
      `internal/trust` package: bounded depth-3 decay-by-half graph
      walk (testable offline via an injected `EdgeFetcher`), best-path
      scoring, claim scoring/ranking. Caught and fixed a real bug along
      the way — relay-side filtering on the multi-character
      `claim_type` tag isn't reliably indexed by public relays per
      NIP-01, so that filter moved client-side. All tests pass,
      including a live trust-edge publish/fetch round trip against
      `wss://nos.lol`.
- [x] File [[A003]]: recovery (guardian attestation + SSKR key
      splitting) — done 2026-09-29, builds on A001, scoped to
      split/combine, guardian designation, and recovery confirmation
      primitives only (guardian-contact workflow and verifier-side
      recovery policy explicitly deferred)
- [x] File [[A004]]: payment integration (attestation-cost gateway,
      L402/Aperture) — done 2026-09-29, reuses `cinder`'s D005
      reference architecture; scoped to receipt mint/embed/verify and
      the minimal gateway backend, not to running Aperture, executing
      a Lightning payment, or choosing which gateways a verifier
      trusts
- [ ] File the remaining action: bonding/slashing still unfiled
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

2026-09-28 — Resolved D001's two remaining coupled open questions
(see D001's own log): runs on the public Nostr relay network rather
than a dedicated persona relay network, for the same reasoning
`cinder` used to choose hosted Lightning/Aperture over self-hosting —
avoids recreating a gatekeeper and avoids operational burden not worth
taking on prematurely. `EphemNet`-routed self-hosting stays a
supported, non-required option. Interop scope followed from that:
full wire-format compatibility, not a fork, with graceful degradation
in generic Nostr clients where cheap.

2026-09-28 — Resolved `claim_type` namespace governance in D001:
reverse-domain namespacing (permission-free minting) plus an open,
non-authoritative NIP-style spec registry for convergence on common
types. This is the third open question resolved by the same recurring
shape of answer (per-verifier choice of authority instead of one
canonical authority) — flagged in D001 as a standing design instinct
worth carrying into whatever's resolved next, not just noted after
each instance.

2026-09-28 — Noted the `claim_type` governance pattern looks like a
good structural fit for `EphemNet`'s own domain-naming/dispute
problem — recorded as a cross-repo pointer in Linked above,
deliberately not started in `EphemNet`'s own plan yet (that's separate
design work belonging there, not a drive-by from here).

2026-09-28 — Resolved the last open question, GDPR right-to-erasure:
architectural avoidance first (keep personal data off the public
attestation layer by default), cryptographic erasure/crypto-shredding
as the fallback (subject-controlled key destruction, not byte
removal), GDPR-controller responsibility distributed per-operator
rather than solved once at the protocol level — the fourth instance
of the same no-canonical-authority pattern. **All five of D001's
original open questions are now resolved.** Moved D001's own status to
DONE per lplan's own schema (design complete; implementation tracked
in actions). This project (P001) stays PLANNING — no action files
exist yet, and "not in scope yet: implementation" in Scope above still
holds until the first one is filed.

2026-09-28 — Filed [[A001]]: core identity and attestation event
layer, the first action against D001. Scoped tight to identity +
the attestation event itself (generation, schema, `claim_type`
namespacing enforcement, publish/fetch/verify on the public Nostr
network, NIP-98 sign-in) — everything else in D001 (trust computation,
recovery, payment integration, bonding/slashing) explicitly deferred
to later actions rather than bundled in. Flagged two real
implementation decisions as still open, not silently assumed:
language/stack (Go is the likely default, matching `cinder`/
`EphemNet`, but not yet actually decided for this repo) and the
attestation event's Nostr `kind` number.

2026-09-28 — Implemented A001. This project moved from PLANNING to
IN_PROGRESS — it now has working code, not just a resolved design.
Go module `github.com/lnd3/persona` using `github.com/nbd-wtf/go-nostr`
for base primitives, three `internal/` packages (identity, attestation,
nip98), full test suite passing including a live round-trip
publish/fetch against a real public relay. See A001's own log for the
one correction made along the way: go-nostr does not actually cover
NIP-98 as the earlier language decision assumed, so it's hand-
implemented instead. Next up: none of trust computation, recovery,
payment integration, or bonding/slashing have their own action filed
yet.

2026-09-28 — Filed [[A002]]: relative-trust computation, building on
A001's fetch/verify primitives. D001's Trust model states the
"weight against your own vouched set" philosophy but no algorithm, so
A002 makes the concrete choices itself: a dedicated `trust` edge
claim_type separate from content claims, bounded depth-3 propagation
with per-hop weight halving, caller-supplied seed set, best-path (not
summed) scoring. Payment-receipt and bond/dispute status as additional
trust signals explicitly deferred to their own not-yet-filed actions.

2026-09-29 — Implemented A002, moved to DONE. New `internal/trust`
package. Worth noting for future actions: the graph-walk (`Compute`)
takes its edge lookup as an injected `EdgeFetcher` function rather than
calling a relay directly, so the depth/decay/cycle/best-path logic got
a full offline unit-test suite against synthetic graphs — only the
real fetch implementation touches the network, and only one dedicated
live test exercises it. That pattern caught a real bug before it
shipped quietly: the first cut of the relay query filtered on the
`claim_type` tag server-side and came back empty against a live
`nos.lol` query, because NIP-01 only guarantees single-letter tags
(like `p`) are relay-indexed — a multi-character tag isn't reliably
filterable server-side across public relays. Fixed by moving that
filter client-side, after an unfiltered `Kind`+`Authors` relay query.
Worth remembering for any future relay-side filtering on a
`persona`-specific tag: single-letter tags only, everything else
client-side.

2026-09-29 — Filed [[A003]]: recovery, building on A001's attestation
primitive per D001's "SSKR-based key splitting, authorized via
delegation attestation" resolution. Made the choices D001 left open:
plain Shamir (`hashicorp/vault/shamir`) over bech32-SSKR framing since
shares here are transport-only, not human-transcribed; two new claim
types (`recovery_guardian`, `recovery_confirm`) with a `group_id`/
`recovery_nonce` scheme to disambiguate guardian sets and recovery
attempts; NIP-44-encrypted DM events as the only channel shares travel
over. Guardian-contact workflow and verifier-side recovery policy
explicitly deferred, same posture as A002 took toward trust-weighting
policy.

2026-09-29 — Filed [[A004]]: payment integration, reusing `cinder`'s
D005 reference architecture (Aperture-fronted gateway, discrete
pricing tier, metered prepaid-bundle anti-replay) rather than
inventing a second L402 pattern. Made the choices D001 left as
architecture-only: receipts bind to a deterministic hash of a claim's
own content fields (not the final event id, which would be circular),
travel as one opaque base64-JSON tag, and are checked only if a given
verifier chooses to require them — kept orthogonal to
`attestation.Verify` itself. Left the concrete Go L402-client library
choice genuinely open for a reuse-vs-build pass at implementation
time, same honesty A001 applied to its own `kind`-number research.
