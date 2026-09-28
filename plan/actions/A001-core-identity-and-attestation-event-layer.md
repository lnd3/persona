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
- [x] **Nostr `kind`: 3300 — decided 2026-09-28.** Must sit in the
      *regular* event range (1000–9999, relay-stored, not
      replaceable) rather than the parameterized-replaceable range
      (30000+) — an attestation is a permanent, independent record;
      later ones from the same attester about the same subject don't
      overwrite earlier ones the way "latest wins" replaceable events
      do. Checked the current NIPs kind registry (regular-event range)
      and picked 3300: it sits in the middle of the largest confirmed-
      open gap (~2023–4549), clear of the Label/AI-embeddings cluster
      ending ~1987, the Torrent cluster (2003–2022), Community Post
      Approval (4550), and NIP-90's DVM job-kind range starting ~5000
      — isolated enough to stay clear even as neighboring ranges fill
      in. **Caveat**: checked via a summarized fetch of the NIPs
      index, not an exhaustive parse of the live registry — worth a
      final re-check against `nostr-protocol/nips` at actual
      implementation time before shipping. Documenting this choice
      openly (here, and eventually in this repo's own docs) is this
      project's version of a NIP submission — no central body approves
      a kind number, but documenting it is what actually reduces
      collision risk for everyone, the same "coordination, not
      permission" logic already applied to `claim_type` governance in
      D001.

### Identity layer
- [ ] Persona keypair generation (secp256k1, Nostr's own key format —
      no custom scheme, per D001's Key Decision)
- [ ] Nostr key encoding (`npub`/`nsec` bech32, per NIP-19) for
      human-facing display/copy-paste

### Attestation event
- [ ] Define and implement the event schema at `kind: 3300`:
      `attester_key`, `subject_key`, `claim_type`, `claim_value`,
      `timestamp`, `signature`, per D001's Attestation Primitive
      section
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

2026-09-28 — Decided the attestation event's Nostr `kind`: 3300, in
the regular-event range (1000–9999), not the parameterized-replaceable
range — attestations are permanent independent records, not
"latest wins." Checked the current NIPs kind registry and picked a
number in the middle of the largest open gap, clear of neighboring
clusters (Label/AI-embeddings, Torrent, Community Post Approval,
NIP-90's DVM job kinds). Flagged one caveat: checked via a summarized
fetch, not an exhaustive registry parse — needs a final re-check
against the live `nostr-protocol/nips` repo before shipping. Both of
A001's implementation-decision tasks are now resolved.
