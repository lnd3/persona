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
      scope, bonding/slashing dispute mechanics
- [ ] Wait on `cinder`'s P004/L402 before real sybil-resistance
      payments or sign-in billing can be built

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
