---
id: T001
title: A persistent, keypair-anchored persona identity — built on Nostr's existing primitives, with economic sybil-resistance and attestation-based recovery — needs no institutional countersignature
status: HELD
conviction: 6
created: 2026-09-21
updated: 2026-09-21
---

## The Belief

A persistent persona — "I am this entity, consistently, over time,
with these specific properties" — bound to a keypair rather than any
state- or employer-issued document, whose properties are established
by consistency (the same key, acting coherently, over time) and peer
attestation (others who have dealt with this persona vouch for
specific claims), needs no institutional countersignature to be
useful. This sidesteps, rather than solves, the harder problem
self-issued *state-recognized* credentials carry (see `superplan`'s
T004): a verifier who cares about "is this the same entity I dealt
with before, and do people I trust vouch for it" never needed an
institution in the first place — money doesn't either, which is why
Bitcoin adoption never required a bank's permission and this shouldn't
require a government's.

Built on Nostr's existing identity/event primitives (secp256k1 keys,
signed JSON events, NIP-extension culture) rather than a custom
protocol, with two hard problems solved deliberately rather than
assumed away: sybil resistance (economic cost via L402/Lightning, not
biometrics) and key loss/theft recovery (SSKR-based social recovery,
authorized via the same attestation primitive built for reputation,
not a bolted-on second system).

## Why This Could Be True

- **Real working precedent, not an unproven idea.** Nostr demonstrates
  persistent, portable, keypair-based identity with no platform
  lock-in and real, growing usage today. PGP's web-of-trust proved the
  cryptographic model works, decades ago, even though its UX never
  reached the mainstream. NIP-98 (HTTP auth via signed events) and
  NIP-58 (badges) are real, already-standardized patterns for sign-in
  and attestation respectively — not wire formats this project has to
  invent.
- **NIP-57 ("zaps") already proves identity plus Lightning payments
  works in production at real scale** — direct evidence the
  sybil-resistance mechanism this design depends on (L402-metered
  attestation cost) isn't a novel combination.
- **The honest weakness is well understood, not hidden.** Nostr has no
  protocol-level spam/sybil resistance — anyone can mint unlimited free
  keypairs — which is exactly why deployed Nostr apps feel noisy. The
  same weakness would undermine attestation credibility if adopted
  without a fix; this design's sybil-resistance mechanism (below) is
  that fix, not an afterthought.
- **Recovery is unified with attestation, not bolted on.** A persona
  designates recovery guardians via a `recovery_guardian` attestation
  claim; a threshold of `recovery_confirm` claims from those guardians
  authorizes recovery — the same primitive built for reputation, reused
  rather than duplicated. The underlying key-splitting math is SSKR
  (Shamir's Secret Sharing), already validated in `superplan`'s own P003
  vault research and proven in production by real social-recovery
  wallets (Argent, Safe/Gnosis Safe).

## What Would Change My Mind

- If real usage shows people and institutions overwhelmingly still
  demand institutional attestation even where peer/consistency-based
  trust would technically suffice, the addressable use-case set is
  much smaller than assumed — proving the mechanism works is not the
  same as proving people will accept it in place of what they're used
  to.
- If peer-attestation proves trivially gameable at real scale (sybil
  attacks — cheaply minting many personas to vouch for each other)
  without a fix that doesn't reintroduce central gatekeeping, the trust
  model is structurally weaker than claimed, not just slower to adopt.
- If most of the *valuable* real-world identity use cases turn out to
  specifically require institutional backing (professional licensing,
  legal capacity, regulated-goods age verification), this remains a
  real, working idea with a smaller footprint than state-recognized
  credentials — not a replacement for that harder problem.
- If self-hosting a relay (see Relationship to `EphemNet` below) proves
  too operationally heavy for ordinary users despite `EphemNet` making
  it concretely possible, the "no platform, no gatekeeper" promise
  would depend on infrastructure most users never actually run
  themselves — worth testing against real adoption, not assumed.

## Entropic Constraints

- **Decay mechanism**: sybil resistance is the load-bearing risk — any
  attestation system without a real cost to creating new personas
  degrades toward meaninglessness at scale, the same failure mode that
  limits every reputation system that doesn't solve it.
- **Horizon**: not time-bound in the market-timing sense — Nostr, PGP,
  and ENS already show this is a live, ongoing space, not a closing
  window.
- **Early warning signs**: real sybil or spam problems at scale in
  this or comparable systems (Nostr's own NIP-based trust extensions,
  EAS-based projects) — the concrete signal to watch, since that's the
  exact mechanism this design depends on holding up.
- **What comes after**: if peer-attestation alone proves insufficient
  for the highest-stakes use cases, the natural next step is a hybrid —
  persona-based identity for low-stakes contexts, falling back to
  institutional countersignature (superplan's T004) only where the
  stakes genuinely require it.

## Sybil Resistance — the Real Mechanism, Not Just the Principle

Economic cost, not proof-of-personhood: an attestation only counts
toward a subject's weighted trust if published with a small
L402-metered payment attached, or backed by a bonded stake slashable
if the claim is later disputed and found false (the same
dispute-resolution pattern real systems like Kleros or Augur's
reporting bonds already use). **Explicitly rejected**: biometric
proof-of-personhood (WorldCoin-style) — effective, but centralizing
and coercive, in direct tension with this whole product group's
no-gatekeeper mission. **Worth studying directly**: BrightID's
decentralized social-graph-based uniqueness verification is the
closest existing project to this philosophically. The actual
dispute-resolution mechanics (who adjudicates, how a slashed stake is
handled, what stops the adjudication step itself from being gamed)
remain genuinely hard, unresolved mechanism design — not solved by
naming the pattern, still needs its own dedicated pass.

## Trust Model

Relative, not global — there is no single canonical reputation score.
A verifier weighs a claim against *their own* vouched set (a
web-of-trust graph, PGP-style), not a universal number anyone could
capture by controlling it. Philosophically correct (real-world
reputation works this way) and structurally resistant to a single
point of manipulation.

## Relationship to `cinder`

Same relationship as `EphemNet`'s to `cinder`: shares the architectural
philosophy (capability-based access, no accounts, macaroon-authenticated
ownership, L402 payment) with no functional runtime dependency on
`cinder`'s relay or code. That's why this lives in its own repo rather
than as a `cinder` companion (the test, per `EphemNet`'s own precedent:
does it actually call `cinder`'s API to operate, or does it just
resemble `cinder` in style? — this fails that test the same way
`EphemNet` does, so it gets the same treatment).

## Relationship to `EphemNet`

`EphemNet`'s DNS-forwarding capability turns "where does this run" from
a hand-wavy option into something concretely buildable: a self-hosted
relay (Nostr-compatible or custom) behind a home NAT can now get a
real, DDoS-protected, TLS-covered domain via `EphemNet`, making
self-hosting a genuine option rather than an aspiration — this is
`superplan`'s M002 "Full Self-Hosted Stack" milestone made real. This
doesn't resolve the deeper "own relay vs. public Nostr network vs.
self-hosted-via-EphemNet" question by itself (see Open Questions), but
it removes the practical objection that self-hosting was previously
too hard to be a real option.

## Master Plans Seeded by This

- None yet — this repo does not have its own master plan tier
  populated. See `superplan`'s M002 (sovereignty product group) for
  the cross-repo strategic view this project is scoped within.

## Log

2026-09-21 — Thesis formed, seeding this repo's own plan for the first
time — this design was previously tracked only in `superplan`
(theses T010/T011, project P008, design D003), which this repo's plan
now supersedes for anything persona-identity-specific going forward,
the same handoff pattern already used for `cinder` and `EphemNet`. Name
chosen: "persona" (Loom was considered and rejected — an existing
cryptocurrency by that name, not something worth associating with).
