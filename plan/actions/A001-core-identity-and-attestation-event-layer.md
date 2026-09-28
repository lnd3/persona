---
id: A001
title: Core identity and attestation event layer
status: PLANNING
design: D001
project: P001
created: 2026-09-28
updated: 2026-09-28
---

## Context

D001 is now a complete spec (status DONE, 2026-09-28) covering
identity, the attestation primitive, trust, sybil resistance
(payment + bonding/slashing), recovery, relay hosting/interop,
`claim_type` governance, and GDPR handling. Nothing has been built
yet — this is the first action against it, and everything else in
D001 (payment integration, bonding/slashing escrow, SSKR recovery,
sign-in billing) depends on this substrate existing first. Deliberately
scoped tight: identity + the attestation event itself, publishable and
verifiable on the public Nostr relay network, nothing more.

**Explicitly deferred to later actions, not this one** (each maps to
an already-resolved piece of D001, just not built yet):
- Relative-trust computation (weighting claims against a verifier's
  own vouched set)
- Recovery (`recovery_guardian`/`recovery_confirm` claims, SSKR key
  splitting)
- Payment integration (attestation-cost gateway, Aperture-fronted
  L402 receipts)
- Bonding/slashing (escrow, arbiter selection, dispute flow)
- The open `claim_type` spec registry itself (the namespacing
  *convention* is implemented here; the community registry document
  is a separate, non-code deliverable)

## Tasks

### Decisions needed before/at implementation start
- [x] **Language/stack: Go — decided 2026-09-28.** Consistency with
      `cinder`/`EphemNet` (shared `deploy/` pattern, same operator
      running all of them) was the starting reason, but it holds up
      technically on its own merits too: mature secp256k1/Schnorr
      signing libraries already used elsewhere in this product family
      (btcec, the same primitive class `cinder` already depends on),
      solid websocket support for relay connections, and an existing,
      actively maintained Nostr client library
      (`nbd-wtf/go-nostr`) that covers NIP-01/NIP-19/NIP-98 directly —
      not starting from zero on relay/event plumbing.
- [ ] Pick and document the attestation event's Nostr `kind` number.
      D001 says NIP-58 is "the base pattern," not that this reuses
      NIP-58's own kind numbers (badge award/definition kinds don't
      match this design's generic `claim_type`/`claim_value` shape) —
      a new custom kind needs choosing from Nostr's unreserved range
      and documenting, the same way a NIP would.

### Identity layer
- [ ] Persona keypair generation (secp256k1, Nostr's own key format —
      no custom scheme, per D001's Key Decision)
- [ ] Nostr key encoding (`npub`/`nsec` bech32, per NIP-19) for
      human-facing display/copy-paste

### Attestation event
- [ ] Define and implement the event schema: `attester_key`,
      `subject_key`, `claim_type`, `claim_value`, `timestamp`,
      `signature`, per D001's Attestation Primitive section
- [ ] Enforce `claim_type` reverse-domain namespacing at the point of
      construction (D001's `claim_type` governance section) — reject
      or warn on a non-namespaced type
- [ ] Human-readable `content` summary field for graceful degradation
      in generic Nostr clients (D001's interop-scope resolution)
- [ ] Sign and publish an attestation event to the public Nostr relay
      network (multi-relay publish for redundancy, standard Nostr
      client practice)
- [ ] Fetch and verify attestation events referencing a given
      `subject_key` (signature check, no central lookup — per D001)

### Sign-in (NIP-98)
- [ ] Implement NIP-98 HTTP auth signing/verification (challenge →
      sign → verify), the base sign-in mechanism per D001 — no
      billing/payment gate yet, that's a separate later action

### Tests
- [ ] Round-trip test: generate persona, publish attestation, fetch
      and verify it back
- [ ] Reject a non-namespaced `claim_type`
- [ ] NIP-98 sign-in challenge/verify round trip

## Log

2026-09-28 — Action filed against D001 (now DONE) as the first,
foundational piece: identity + the attestation event layer itself,
nothing layered on top yet. Two real implementation decisions flagged
as open rather than silently assumed: language/stack choice, and the
attestation event's actual Nostr `kind` number.

2026-09-28 — Decided language/stack: Go. Started from product-family
consistency (`cinder`/`EphemNet` both Go, same operator, shared
`deploy/` pattern) but confirmed on independent technical merit before
committing — existing secp256k1/Schnorr signing libraries already used
elsewhere in this family, and `nbd-wtf/go-nostr` already covers
NIP-01/NIP-19/NIP-98, so relay/event plumbing isn't starting from
zero. `kind` number still open.
