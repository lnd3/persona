---
id: A002
title: Relative trust computation (web-of-trust weighting)
status: DONE
design: D001
project: P001
created: 2026-09-28
updated: 2026-09-29
---

## Context

Implements D001's Trust model: "no single canonical reputation
score... a verifier computes trust in a claim by weighting it against
*their own* vouched set (a web-of-trust graph, PGP-style)." D001
states the model's philosophy but doesn't specify an algorithm — that
concrete choice belongs in this action, the same way A001 made the
`kind`-number and language decisions without reopening D001.

Builds directly on [[A001]]'s `internal/attestation` package
(`FetchForSubject`, `Verify`) — this action scores/ranks the claims
A001 already knows how to fetch and verify; it doesn't change how
claims are published or verified.

**Explicitly out of scope for this action** (each is a separate,
still-unfiled or in-design piece of D001, not folded in here):
- Payment-receipt validity (attestation-cost gateway) as a trust
  signal — depends on the not-yet-built payment integration action
- Bonded-stake/dispute status as a trust signal — depends on the
  not-yet-built bonding/slashing action
- Any UI/config surface for a user to actually manage their own seed
  trust list — this action implements the computation only, given a
  seed set as input

## Decisions made here (not in D001 — algorithmic choices this action owns)

- **Trust edges are their own claim_type, distinct from content
  claims.** A verifier's "vouched set" is built from explicit trust
  edges (`net.persona.core.trust`, `claim_value` empty or a free-text
  note) — not inferred from *any* attestation someone happens to
  issue. Skill/ownership/recovery claims are content being weighted;
  trust edges are the graph being weighted against. Conflating the two
  would mean a skill attestation silently doubled as a trust vouch,
  which isn't what D001's PGP analogy describes.
- **Bounded-depth propagation with per-hop decay, not unbounded
  transitive trust.** A seed pubkey (direct, verifier-supplied) gets
  weight 1.0. Each additional hop out along `trust` edges halves the
  weight (hop 2 = 0.5, hop 3 = 0.25, ...), capped at a **max depth of
  3** — chosen as a starting default (unbounded propagation would let
  trust leak arbitrarily far from the verifier, which is exactly the
  "canonical score anyone can game" failure D001's model exists to
  avoid; depth 3 is a judgment call, not derived from anything, and
  genuinely revisitable).
- **The seed set is caller-supplied, not derived from the network.**
  Consistent with "relative to whoever's asking" — this action takes a
  `[]string` of directly-trusted pubkeys as input; where a real client
  gets that list from (local config, an onboarding flow) is a product
  concern, not this action's.
- **A claim's score is the weight of its most-trusted attesting path**,
  not a sum across multiple paths — avoids letting an attacker inflate
  a claim's score just by having many low-weight paths converge on it,
  which would reintroduce a gameable aggregate exactly like a global
  score would be.

## Tasks

### Trust graph
- [x] Define the `net.persona.core.trust` edge claim_type (reuses
      A001's namespacing enforcement — no new validation code needed)
      — `internal/trust/trust.go` (`EdgeClaimType`)
- [x] Fetch trust-edge events for a given pubkey, keyed by attester,
      not subject — `FetchTrustEdges`. Filters relay-side only on
      `Kind`/`Authors`; `claim_type` is filtered client-side in
      `trustedTo`, since NIP-01 only guarantees single-letter tags are
      relay-indexed and a live round trip against `nos.lol` came back
      empty until this was corrected (see Log)
- [x] Bounded BFS/graph walk from a seed set, computing per-pubkey
      weight (depth ≤ 3, weight halves per hop, best-path-wins on
      cycles/multiple paths) — `Compute`, parameterized over an
      `EdgeFetcher` so the walk itself is testable offline against a
      synthetic graph; `RelayEdgeFetcher` is the real network-backed
      instance

### Scoring
- [x] Given a subject's fetched attestations (via A001's
      `FetchForSubject`) and a computed trust-weight map, score each
      claim by its attester's weight (0 if attester isn't in the
      verifier's reachable set at all) — `Score`
- [x] Group/rank scored claims per `claim_type`, so a caller can ask
      "what's this subject's best-supported claim for X" rather than
      getting an undifferentiated list — `Score`'s
      `map[string][]ScoredClaim` return, sorted descending by weight
      within each group

### Tests
- [x] Direct trust (depth 1) scores 1.0 — `TestComputeDirectTrust`
- [x] Transitive trust decays correctly at depth 2/3 —
      `TestComputeTransitiveDecay`
- [x] Depth-4+ trust is unreachable (weight 0 / excluded) —
      `TestComputeTransitiveDecay`
- [x] A cycle in the trust graph doesn't loop forever or inflate weight
      — `TestComputeCycleDoesNotLoopOrInflate` (watchdog-timed)
- [x] An attester outside the reachable set scores 0, is excluded from
      ranked results — `TestComputeOutsideSeedSetUnreachable`,
      `TestScoreExcludesUnreachableAttester`
- [x] Multiple paths to the same attester use the best (highest-weight)
      path, not a sum — `TestComputeMultiPathUsesBestNotSum`
- [x] Live round trip: publish a real trust-edge attestation and fetch
      it back via `RelayEdgeFetcher` against a real public relay —
      `TestFetchTrustEdgesRoundTrip` (skips gracefully if unreachable)

## Log

2026-09-28 — Action filed against D001's Trust model. Made the
concrete algorithmic choices D001 deliberately left unspecified:
trust edges as their own claim_type, bounded depth-3 decay-by-half
propagation, caller-supplied seed set, best-path (not summed) scoring.
Explicitly scoped out payment-receipt and bond/dispute status as
additional trust signals — both depend on separate, not-yet-built
actions.

2026-09-29 — Implemented. `internal/trust/trust.go`: `Compute` walks
the graph via an injected `EdgeFetcher` interface rather than talking
to a relay directly, so the depth/decay/cycle/best-path logic has a
full offline unit-test suite against synthetic graphs, with only the
actual fetch (`RelayEdgeFetcher`/`FetchTrustEdges`) hitting the
network. Caught and fixed one real bug this way: the first cut of
`FetchTrustEdges` filtered on the `claim_type` tag server-side, which
came back empty against a live `nos.lol` query — NIP-01 only
guarantees single-letter tags (like `p`) are relay-indexed, so a
multi-character tag like `claim_type` isn't reliably queryable
server-side across public relays. Fixed by filtering `claim_type`
client-side (in `trustedTo`) after an unfiltered `Kind`+`Authors`
query, the same shape A001's own `p`-tag filtering already relied on
correctly. Live round trip (publish a trust-edge attestation, fetch it
back via `RelayEdgeFetcher`) passes against `wss://nos.lol`; all
offline algorithmic tests pass. Everything this action scoped is
done — moved to DONE.
