---
id: A002
title: Relative trust computation (web-of-trust weighting)
status: PLANNING
design: D001
project: P001
created: 2026-09-28
updated: 2026-09-28
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
- [ ] Define the `net.persona.core.trust` edge claim_type (reuses
      A001's namespacing enforcement — no new validation code needed)
- [ ] Fetch trust-edge events for a given pubkey (reuses
      `attestation.FetchForSubject`-style querying, but keyed by
      attester, not subject — trust edges are queried outward from a
      person, not looked up by who they're about)
- [ ] Bounded BFS/graph walk from a seed set, computing per-pubkey
      weight (depth ≤ 3, weight halves per hop, best-path-wins on
      cycles/multiple paths)

### Scoring
- [ ] Given a subject's fetched attestations (via A001's
      `FetchForSubject`) and a computed trust-weight map, score each
      claim by its attester's weight (0 if attester isn't in the
      verifier's reachable set at all)
- [ ] Group/rank scored claims per `claim_type`, so a caller can ask
      "what's this subject's best-supported claim for X" rather than
      getting an undifferentiated list

### Tests
- [ ] Direct trust (depth 1) scores 1.0
- [ ] Transitive trust decays correctly at depth 2/3
- [ ] Depth-4+ trust is unreachable (weight 0 / excluded)
- [ ] A cycle in the trust graph doesn't loop forever or inflate weight
- [ ] An attester outside the reachable set scores 0, is excluded from
      ranked results
- [ ] Multiple paths to the same attester use the best (highest-weight)
      path, not a sum

## Log

2026-09-28 — Action filed against D001's Trust model. Made the
concrete algorithmic choices D001 deliberately left unspecified:
trust edges as their own claim_type, bounded depth-3 decay-by-half
propagation, caller-supplied seed set, best-path (not summed) scoring.
Explicitly scoped out payment-receipt and bond/dispute status as
additional trust signals — both depend on separate, not-yet-built
actions.
