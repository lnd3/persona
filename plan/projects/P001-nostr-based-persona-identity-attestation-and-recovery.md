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

**Naming split decided (2026-09-30), resolving the tension above**:
persona is tightly coupled to Nostr's own attestation primitive, and
that primitive is bigger than just this one identity product — D001
already designed the `claim_type` namespace as an open, NIP-style
registry, general enough that something other than identity/recovery
could build on it later. So:
- **`solemn`** = the protocol layer — the Nostr-based attestation
  primitive itself (the `kind:3300` event format, `claim_type`
  open-registry governance), not identity-specific by nature. Lives at
  `solemn.network`. Not built yet — currently just the domain
  designated for it; the actual protocol-spec/registry page is
  unbuilt, separate future work, not assumed to happen automatically
  from this naming decision alone.
- **`persona`** = the product — this repo's own identity/recovery/
  trust/sybil-resistance system, one implementation built on the
  `solemn` protocol (the way a specific client relates to the Nostr
  protocol itself, or an app relates to HTTP). Repo/thesis/README name
  stays "persona," unchanged. Its own landing page (`site/index.html`,
  2026-09-30) is intended to live at `persona.solemn.network` —
  subdomain of the protocol, not the top-level domain itself — mirroring
  the `persona.cinderapps.org` subdomain pattern already in use for
  hosting.
- If a second, genuinely different product later wants to build on the
  same attestation primitive (not assumed, not currently planned), it
  would get its own subdomain of `solemn.network` alongside `persona`,
  the same way `EphemNet` and `persona` both sit under the `lnd3`
  umbrella today without needing to share a brand name.
- `persona.mera.network` and `sovranpersona.com` remain owned but are
  no longer the lead candidates for this repo's own domain.

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
- **Cross-repo watch item (noted 2026-09-30, not this repo's own
  work)**: `cinder`'s new `P017`/`D012` — a payment middle layer
  (opaque `PurchaseToken{token, expires_at}` contract, pluggable mock/
  real backends) intended to eventually become persona's own
  attestation-cost gateway's real backend. Cites persona's `A004`/
  `A008` (`internal/payment/l402.go`'s wire format, and the Stage 1
  simulated / Stage 2 deferred-real-regtest staging discipline) as
  prior art — correctly as *citation, not a dependency*: Go's own
  `internal/` package visibility means `cinder`/`EphemNet` couldn't
  import `persona/internal/payment` even if the design wanted them to.
  Asked directly whether this belongs in persona long-term: no — it
  has zero identity content, and folding it into persona would invert
  this repo's own stated "no runtime dependency on sibling repos"
  pattern (`README.md`), the same call this design's own history
  already reached once (caught mid-write in persona's plan, moved to
  `cinder` for having no persona-identity coupling). Staying a
  cross-repo pointer, not persona's own work, unless persona is
  explicitly asked to build against it later once it's real.
  Also confirmed (2026-09-30): D012 already names multi-backend support
  (a card processor alongside Lightning, say) as an open question,
  deliberately deferred until a second real backend is actually needed —
  the opaque contract is already shaped so a future backend swap is a
  new implementation, not a redesign. No action needed here; noted as
  confirmation the design already accommodates it.
  **Update, same day**: extracted for real —
  `github.com/lnd3/paylayer@v0.1.0` (private repo, `GOPRIVATE`+SSH git
  config needed to `go get` it). Contains the `PurchaseToken` contract,
  `Backend` interface, `RequireToken` middleware, `StaticBackend`, and
  the fault-injecting `Mock` (19 tests) — no real Lightning/L402
  backend yet. cinder's own `A018` paid listener already integrated
  against it (`StaticBackend`, zero behavior change); `EphemNet` has
  already adopted it for its own payment-gated registration work
  (`D010`). persona's own `internal/payment/l402.go` does real L402
  challenge/header work paylayer doesn't attempt yet, so no reason to
  adopt it now — but it's the natural place for persona's own
  Aperture/Lightning work to eventually plug in as paylayer's first
  real `Backend`, if/when that's ever wanted. Still a watch item, not
  persona's own work.
- **Cross-repo watch item (noted 2026-09-30, not this repo's own
  work)**: `cinder`'s and `EphemNet`'s own `deploy/cinderapps/Caddyfile`
  and `deploy/ephemnet-caddy/Caddyfile` both have the same real gap
  [[A009]] found and fixed in persona's own copy — `auto_https
  disable_redirects` with no explicit `http://` → `https://` redirect
  block, meaning a real visitor hitting either domain's bare `http://`
  gets a 502 from nginx instead of a redirect (see A009's own Log for
  the fix and why it's safe alongside the ACME HTTP-01 responder).
  Worth those repos' own attention since it affects real visitor
  traffic, not just an edge case — not fixed here, since it's their own
  files to edit, not a drive-by from this repo's context.
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
- [x] Implement A003 — done 2026-09-29, moved to DONE. New
      `internal/recovery` package: GF(256) Shamir split/combine,
      NIP-44 share encryption plus a `kind: 3301` delivery-event
      wrapper, `recovery_guardian`/`recovery_confirm` claim
      construction/parsing, distinct-guardian threshold counting. Two
      corrections made along the way: swapped `hashicorp/vault/shamir`
      for a direct implementation after discovering it isn't
      independently modularized (would have pulled the entire
      `hashicorp/vault` repo and forced a Go toolchain bump); and fixed
      a real bug in A001's `ValidateClaimType` regex, which rejected
      D001's own established underscore-containing claim types
      (`recovery_guardian`/`recovery_confirm`) before this fix. All
      tests pass, full repo build/vet/test clean.
- [x] File [[A004]]: payment integration (attestation-cost gateway,
      L402/Aperture) — done 2026-09-29, reuses `cinder`'s D005
      reference architecture; scoped to receipt mint/embed/verify and
      the minimal gateway backend, not to running Aperture, executing
      a Lightning payment, or choosing which gateways a verifier
      trusts
- [x] Implement A004 — done 2026-09-29, moved to DONE. New
      `internal/payment` package: `Receipt` type + sign/verify (via
      `btcec/schnorr`, the same primitive go-nostr's own event signing
      already uses — no new dependency), `PendingClaim`/`Finalize` for
      embedding an optional receipt in an attestation event without a
      package import cycle, a reference `POST /receipt` gateway
      handler, and 402-challenge/`Authorization`-header L402 client
      plumbing. The L402-client-library question flagged as open
      turned out to need no library at all for this action's actual
      scope (challenge parsing is plain `net/http`); paying an invoice
      remains genuinely out of scope and still has none chosen. 16 new
      tests, full repo build/vet/test clean.
- [x] File [[A005]]: bonding/slashing dispute resolution (dispute type
      1 only, per D001's own scoping) — done 2026-09-29, protocol/
      decision layer only (claim types, panel confirmation, verdict
      resolution, self-release timing); actual Bitcoin escrow script
      construction/funding/verification deliberately left to a
      separate, not-yet-filed follow-up action
- [x] Implement A005 — done 2026-09-29, moved to DONE. New
      `internal/dispute` package: the four claim types
      (`bond`/`dispute_challenge`/`arbiter_panel`/`dispute_verdict`),
      `ConfirmedPanel` (two-sided matching-claim agreement),
      `Resolve` (majority-vote, distinct-arbiter), `IsSelfReleased`
      (per-bond window arithmetic). One real parsing gap found and
      closed: `escrow_ref` can itself contain colons (Bitcoin
      descriptor key-origin syntax does), which would have broken the
      original colon-delimited `claim_value` format — fixed with
      anchored regexes using a greedy middle capture against
      digits-only trailing fields. 24 new tests, full repo
      build/vet/test clean.
- [x] **A001-A005 are now all DONE and implemented.** The only D001
      piece still unbuilt is the Bitcoin-escrow settlement follow-up
      A005 deliberately deferred (script construction, funding,
      on-chain verification — a fund-loss-risk piece of work kept
      separate on purpose), which remains unfiled.
- [x] File [[A006]]: Bitcoin escrow settlement — done 2026-09-29.
      Filing it surfaced a real unresolved structural problem, not
      just implementation choices: D001/A005's "arbiter panel chosen
      per-dispute" is in tension with a Bitcoin script needing to name
      its spending conditions before anyone knows who the arbiters
      will be. Documented two candidate directions (pre-designated
      mediator vs. voluntary escalation into joint custody) as open
      questions rather than guessing an answer while filing; that
      decision is A006's own first task, before any script code.
- [ ] Implement A006 — started 2026-09-29, redesigned 2026-09-30,
      **not yet DONE** (large, multi-part, explicitly fund-loss-risk
      action). Current `internal/escrow` package: `UniversalScript`/
      `ReinforcedScript` (single, static, upfront-configured scripts,
      no escalation transaction), the standing `arbiter_commitment`
      claim type, PSBT settlement helpers (`settle.go`), funding
      verification logic — every script and full PSBT round trip
      validated against `btcd`'s real consensus engine. Still missing:
      a real `ChainQuerier` node backend (no regtest node reachable in
      this environment) and the mainnet-gating security review, now
      tracked as its own action, [[A007]].
- [x] File [[A007]]: independent security review of A006's Bitcoin
      escrow scripts — done 2026-09-30, tracked as its own action
      rather than an aspiration inside A006's task list. Scoped to
      arranging and acting on the review, not performing it (an
      AI-assisted self-review isn't independent) or building A006's
      still-missing pieces. Regtest/testnet exercise is this action's
      own prerequisite, ahead of the review itself — cheaper bugs
      should be caught cheaply before spending a reviewer's time.
- [x] File [[A008]]: Aperture/L402 integration template — done
      2026-09-30, staged deliberately: Stage 1 (a simulated L402 gate
      fronting the real gateway handler, proving persona's own code
      end to end at no infrastructure cost) implemented in the same
      pass; Stage 2 (real `lnd`+`aperture` regtest infrastructure)
      deferred, its actual requirements (two funded Lightning nodes, a
      real channel, real Aperture config) named concretely rather than
      left vague, same "defer the harder cross-domain slice"
      discipline as A006's own escrow-settlement follow-up
- [ ] Wait on `cinder` actually deploying its own paid tier live
      (Aperture config, a real Lightning provider account) only if/when
      this project wants to stand up its own reference gateway;
      otherwise no longer a hard blocker to this project's own design
      or implementation work
- [x] File and implement [[A009]]: public site deployment tooling —
      done 2026-09-30, copied `cinder`'s/`EphemNet`'s own `deploy/`
      conventions directly (same server, `bh2`), scoped to what persona
      actually has today (one static page, no app binary). SSH access
      to `bh2` turned out available this session — found and fixed a
      real port/subnet collision (a fourth product, `offgridapp`, not
      previously known to this plan) before deploying, and a real DNS
      issue (two GoDaddy Domain Forwarding A records, separate from
      "Parking" — operator fixed both, verified clean before
      proceeding). **`https://solemn.network` is live**: real Let's
      Encrypt cert obtained on the first attempt, verified with a real
      `curl` (200, clean TLS, correct build-info footer). A009 moved to
      DONE.

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

2026-09-29 — Filed [[A005]]: bonding/slashing dispute resolution,
scoped exactly to dispute type 1 as D001 itself scopes it. Split the
same way A004 split payment integration: this action owns the
protocol/decision layer (four new claim types — `bond`,
`dispute_challenge`, `arbiter_panel`, `dispute_verdict` — panel
confirmation via matching two-sided attestations, majority-vote
verdict resolution, self-release timing arithmetic), leaving actual
Bitcoin escrow script construction/funding/verification to a separate,
not-yet-filed follow-up — a fund-loss-risk piece of work deserving its
own focused action. Two judgment calls made explicitly: panel size
must be 1 or odd-≥3 (D001 never addresses even-panel ties), and a
14-day self-release window as a revisitable per-bond default. **All
four of D001's implementation pieces now have an action filed** —
A001 (DONE), A002 (DONE), A003, A004, A005 (all PLANNING, not yet
implemented).

2026-09-29 — Implemented A003, moved to DONE. New `internal/recovery`
package: GF(256) Shamir split/combine (direct implementation, see
below), NIP-44 share encryption plus a `kind: 3301` delivery-event
wrapper, `recovery_guardian`/`recovery_confirm` claim construction and
parsing, distinct-guardian threshold counting. Two corrections worth
carrying forward: (1) `hashicorp/vault/shamir` isn't independently
modularized — importing it pulled in the entire `hashicorp/vault` repo
and forced a Go toolchain bump (1.24.1 → 1.25.3) for one small,
dependency-free algorithm, so it was swapped for a direct ~180-line
implementation instead, no new dependencies; worth remembering before
reaching for any hashicorp/vault subpackage again in this project.
(2) A001's `ValidateClaimType` regex only allowed hyphens within a
label, which would have rejected D001's own established underscore-
containing claim types (`recovery_guardian`/`recovery_confirm`) —
fixed to accept both, a fix that affects every future claim_type
choice, not just this action's two. All tests pass, full repo
build/vet/test clean.

2026-09-29 — Implemented A004, moved to DONE. New `internal/payment`
package: `Receipt` sign/verify via `btcec/schnorr` (already an
indirect dependency, just promoted to direct — no new signature
scheme), `PendingClaim`/`Finalize` embedding an optional receipt tag
into an attestation event, a reference `POST /receipt` gateway
handler, and 402-challenge/`Authorization`-header L402 client
plumbing. Two things corrected from the original plan: the event
integration couldn't literally extend `attestation.New` (the payment
package already imports attestation, so the reverse import would
cycle) — `PendingClaim` fixes a claim's content/timestamp once so the
receipt's hash and the final event's timestamp can't drift apart
between requesting payment and signing; and `ContentHash` was changed
to length-prefix each field before hashing, closing a field-boundary
collision a bare `|`-join would have left open. The L402-client-
library choice flagged as open in A004's filing turned out to be moot
for this action's actual scope — challenge parsing needs nothing
beyond `net/http`/`regexp`; paying an invoice remains genuinely
deferred with no library chosen. 16 new tests, full repo
build/vet/test clean. **A001-A004 are now all DONE and implemented —
only A005 (bonding/slashing) remains to implement**, plus its own
deferred Bitcoin-escrow follow-up once reached.

2026-09-29 — Implemented A005, moved to DONE. New `internal/dispute`
package: the four claim types (`bond`, `dispute_challenge`,
`arbiter_panel`, `dispute_verdict`), `ConfirmedPanel` (agreement
between both sides' independently-published panel claims),
`Resolve` (distinct-arbiter majority voting), `IsSelfReleased`
(per-bond self-release window arithmetic, blocked by any matching
challenge regardless of resolution). One real gap found and closed:
`escrow_ref` can contain colons (Bitcoin descriptor key-origin syntax
does), which would have broken a naive colon-split of the
`claim_value` format — fixed with anchored regexes using a greedy
middle capture against digits-only trailing fields, confirmed with a
dedicated test. 24 new tests, full repo build/vet/test clean.
**A001-A005 are now all DONE and implemented.** The only D001 piece
still unbuilt is the Bitcoin-escrow settlement follow-up A005
deliberately deferred throughout (script construction, funding,
on-chain verification — a fund-loss-risk piece of work kept separate
on purpose), which remains unfiled.

2026-09-29 — Filed [[A006]]: Bitcoin escrow settlement, the follow-up
A005 named throughout. Unlike every action filed so far, this one
surfaced a real structural problem rather than just missing
implementation detail: D001/A005 decided arbiter panels are chosen
*per-dispute*, jointly by attester and challenger, but a Bitcoin
script has to name its spending conditions *before* a dispute (and
therefore an arbiter panel) exists — a bond can sit on-chain
unchallenged for a long time first. Documented two candidate
directions honestly as open questions rather than picking one while
filing: a pre-designated mediator chosen at bond-creation time
(simple, well-precedented, but contradicts the per-dispute joint
selection D001 already decided), or voluntary escalation into joint
custody once challenged (preserves per-dispute selection, but needs a
real consequence for an attester who refuses to cooperate, or they
can just wait out their original timelock and keep their bond
regardless of the dispute). Resolving this is A006's own first task,
before any script-construction code — deliberately not resolved by
guessing during filing. Also gated mainnet activation behind an
explicit independent-security-review task, given the fund-loss stakes
this action carries that nothing else in this project does.

2026-09-29 — Resolved A006's open design question: voluntary
escalation into joint custody, not a pre-designated mediator. The
mediator alternative was rejected on principle (it would make the
attester unilaterally pick their own judge), not just on
inconvenience. Escalation also improves the action's own risk profile:
pre-escalation, both a bond and a challenge stake are pure
single-party self-custody, so the dangerous multisig code only gets
built and exercised on actual disputes, not on every bond. Residual
risk — an attester can refuse to escalate and wait out their own
timelock — is resolved reputationally, not cryptographically, the same
honest move D001 already made for GDPR erasure. A006's Tasks rewritten
around this two-stage (self-custody, then escalated joint-custody)
structure; script construction can now actually start.

2026-09-29 — Refined A006's escrow structure into two tiers, matching
D001's own small/larger-bond distinction. Small bonds keep cooperative
escalation but now settle via 2-of-2 mutual agreement first, arbiter
fallback second — most disputes resolve by direct agreement, so
third-party adjudication should be the exception, not the default path
(D001 itself treats a single mutually-agreed arbiter as sufficient for
this tier). Larger bonds needed a different answer: cooperative
escalation alone can't support a genuinely *requirable* right to
arbitration, since whoever expects to lose can just refuse to
cooperate. Resolved by having each side independently pre-commit their
own arbiter key into their own output at funding time — real
unilateral leverage without reintroducing the earlier-rejected
single-shared-judge pattern. Left the two-appointed-arbiters-disagree
case as a genuinely open tie-breaking question, deliberately not
guessed.

2026-09-29 — Resolved A006's tie-breaking question: the two
independently-appointed arbiters jointly escalate to a third, using
the same joint signing power their arbitration branch already grants
them, rather than pre-committing a third key upfront (which would
reopen the same chicken-and-egg problem this tier already solved
once). Confirmed this terminates in exactly one escalation step and
never recurses further, since binary verdicts mean a third arbiter's
vote always immediately produces a 2-of-3 majority. The only remaining
failure mode — the two arbiters refusing to escalate at all — is a
cooperation failure covered by the same pre-escalation timelock
fallback already used elsewhere in this plan, not a new residual risk.
A006's structural design is now fully resolved; next is actually
building it.

2026-09-29 — Started implementing A006. New `internal/escrow` package:
BIP68 time-based CSV timelock (`SequenceForDays`, `PreEscalationScript`),
the small-bond and larger-bond tiers' escalated scripts
(`SmallTierEscalatedScript`, `LargeTierEscalatedScript`), the tie-break
2-of-3 script, a new `net.persona.core.arbiter_commitment` claim type
(additive to A005, not a change to its shipped claim types), a narrow
BIP380-shaped descriptor for `EscrowRef`, and funding-verification
logic against an abstract `ChainQuerier` interface. Every script was
validated by actually signing and executing real spends (and
attempted-invalid spends) against `btcd`'s own consensus `txscript`
engine, the strongest testing available without a live node — CSV
timelock testing needed no simulated passage of time, since BIP68
compares the script's required sequence directly against the input's
declared value. 24 new tests, full repo clean (104 tests total).
Explicitly **not** marked DONE: PSBT-based transaction construction
helpers, a real `ChainQuerier` node backend (no regtest node reachable
here), non-escalation detection, and the mainnet security-review gate
all remain, named precisely in A006's own Tasks list rather than
glossed over — this is the fund-loss-risk action D001 and A005 both
flagged, and it's being treated with the caution that implies.

2026-09-30 — Building A006's PSBT helpers surfaced a second, more
serious bug: the escalation-based design's CSV timelock blocked *any*
early spend, including legitimate cooperative escalation, and never
actually tied a post-timelock spend to any dispute outcome — the bond
provided no real enforcement. Seriously considered moving the escrow
layer to a smart-contract chain (researched Hyperliquid's HyperEVM
concretely: 24 validators, real bridge-risk numbers — >$2.8B stolen
since 2022) before concluding no chain's consensus can adjudicate the
dispute's actual substance anyway, and Kleros's juror-pool model is
exactly the crowdsourced-court pattern D001 already rejected. Redesigned
on Bitcoin instead, around the insight that the disputed attestation
already names its own subject_key — the "future" counterparty was
never actually unknown, only the arbiter was. New `UniversalScript`/
`ReinforcedScript` (upfront.go): a single static script per bond, no
escalation transaction ever, with a *standing* (not per-bond)
`arbiter_commitment` claim closing the one genuinely-unknown-upfront
piece. Built the PSBT settlement helpers this was originally about,
now much smaller since there's no escalation chain to construct. Full
build→sign→finalize→extract→re-validate round trips pass against the
real consensus engine. 110 tests total, full repo clean. Still not
DONE: a real `ChainQuerier` backend and the mainnet security-review
gate.

2026-09-30 — Filed [[A007]]: the mainnet security-review gate A006's
own Tasks list has always carried, tracked as its own action rather
than left as an aspiration inside A006's task list. Deliberately
scoped to arranging and acting on an independent review, not
performing one — the same logic that makes this action necessary at
all (an AI-assisted self-review isn't independent) also means this
action can't substitute for the real thing. Decided regtest/testnet
exercise (actually funding, spending, and confirming each script
branch on a real node) is this action's own prerequisite, ahead of
the reviewer's time — no regtest node is reachable in this environment
yet, so this blocks on that becoming available. Stated concrete pass
criteria rather than "looks safe": no valid spend without a real
matching signature, no branch satisfiable with fewer signatures than
designed, no fallback spend before its CSV window has genuinely
elapsed.

2026-09-30 — Found a real regtest node opportunistically: `btcd`'s
full daemon source was already sitting in this project's own Go
module cache (a transitive dependency of `txscript`/`psbt`), so
`go install github.com/btcsuite/btcd@v0.24.2` built a real, working
node directly — no new binary dependency. Built the real `ChainQuerier`
backend (`RPCQuerier`, `internal/escrow/rpcquerier.go`) A006 had left
as an interface, and genuinely funded, broadcast, and confirmed three
of `UniversalScript`'s four branches against it — real mempool
acceptance and confirmation, not just in-process engine validation.
Not a standing environment fixture — stood up for this session, not
guaranteed reachable later, though `regtest_test.go` documents exactly
how to reproduce it.

2026-09-30 — Extended live regtest coverage to `ReinforcedScript`'s
four non-fallback branches — all 7 live-testable branches across both
scripts now pass against a real node. Found and fixed a real bug
along the way: funding a fixed sat amount from a fresh coinbase
breaks once regtest's 150-block subsidy-halving decays a long-lived
node's coinbases below that amount, confirmed by watching the test
suite fail progressively worse across repeated reruns against the
same node. Fixed by funding a fraction of each coinbase's own value
instead. Only the CSV fallback branch's live timing (a deliberate
scope boundary, not an oversight) and the actual independent review
itself remain.

2026-09-30 — Started arranging A007's review: assembled the actual
review package (`internal/escrow/SECURITY_REVIEW.md` — scope, design
rationale, categorized test coverage, known residuals, pass criteria,
reproduction commands). Found and fixed stale doc comments while
doing so (the package's own top-level comment still described the
superseded escalation design). Identifying and engaging an actual
reviewer is explicitly not something to fabricate — surfaced back to
the project's own operator as a real decision, not simulated.

2026-09-30 — Filed and implemented A008's Stage 1: a simulated L402
gate fronting A004's real gateway handler, proving persona's own
challenge-parsing/header-construction/receipt-signing/independent-
verification code is wired correctly end to end, at no infrastructure
cost. Checked Stage 2 (real `lnd`+`aperture` regtest) concretely
before deferring it — both resolve as real Go modules, but `lnd`
requires Go ≥1.25.13, the same toolchain-bump risk pattern A003's
`hashicorp/vault/shamir` lesson already taught, mitigated the same way
`btcd` was in A006/A007 (standalone external binaries, never imported
into persona's own module). Named Stage 2's real requirements
concretely (two funded Lightning nodes, a real channel, real Aperture
config, orchestration code) rather than leaving them vague, and
deferred it deliberately as a separate, much larger undertaking.

2026-09-30 — Asked to watch `cinder`'s new payment-middle-layer work
(`P017`/`D012`) and flag anything worth steering. Read both files:
well-designed, self-correcting (already caught and fixed its own early
mistake of putting L402/Aperture vocabulary in the contract, and of
briefly landing in persona's own plan where it didn't belong), and
accurately cites persona's actual `A004`/`A008` code. Asked directly
whether the eventual extraction target should be a standalone repo or
persona itself: recommended standalone, for a concrete reason beyond
"feels out of scope" — Go's own `internal/` visibility means
`cinder`/`EphemNet` can't import `persona/internal/payment` regardless,
so there's no real code-reuse case for co-locating it in persona, and
doing so would invert persona's own stated no-runtime-dependency-on-
siblings pattern. Recorded as a cross-repo pointer in `Linked` above,
not persona's own work — no plan/code changes made in `cinder` itself,
since another session is actively building there.

2026-09-30 — Follow-up: asked whether the payment layer should
optimally also interface toward other payment systems if requested.
Checked D012 directly rather than assuming: already listed under its
own Open Questions ("Multiple real backends... probably premature
until a second real backend is actually needed") — the opaque
`PurchaseToken` contract is already shaped so a future second backend
is a new implementation satisfying the same contract, not a redesign.
Confirmed as already-accommodated design, not a gap; no action taken,
here or in `cinder`.

2026-09-30 — Built `site/index.html`, a comprehensive public landing
page (identity/attestation, relative trust, economic sybil resistance,
recovery with an honest key-loss-risk caveat, the Bitcoin bonding/
slashing escrow design, L402 payment, and an explicit independent-
review call-to-action for Bitcoin devs). Alongside it, resolved the
`solemn.network` naming ambiguity flagged earlier: `solemn` becomes the
home for the Nostr-based attestation *protocol* itself (the
`kind:3300` event format, the open `claim_type` registry D001 already
designed but never placed anywhere) — not a rename of persona.
`persona` stays the product name, intended at `persona.solemn.network`.
See "Naming" above for the full split and reasoning.

2026-09-30 — Repo pushed to `github.com/lnd3/persona` and made public
(the user's own action, confirmed reachable via the public GitHub API).
MIT `LICENSE` was already in place from earlier this session. A006's
Bitcoin escrow code is real, tested against a live regtest node, and
now publicly visible — but still pending A007's independent review
before being treated as mainnet-ready; the landing page states this
status honestly rather than implying it's already cleared.
