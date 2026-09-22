---
id: D001
title: Persona identity protocol — Nostr-based keys, peer attestation, social recovery
status: PLANNING
project: P001
created: 2026-09-21
updated: 2026-09-21
---

## Summary

Design for [[P001]], implementing [[T001]]: a persistent persona
identity built on Nostr's existing key and event primitives (not a new
protocol), a peer-attestation layer with a real answer to sybil
resistance (economic cost via L402/Lightning, not proof-of-personhood
or biometrics), and a key-loss/theft recovery mechanism that reuses the
same attestation primitive rather than being a bolted-on separate
system.

## Architecture

**Identity layer**: a persona is a secp256k1 keypair, using Nostr's
existing key format — not a custom scheme. This buys interoperability
with existing Nostr tooling/relays/libraries where useful, and avoids
inventing new cryptography, consistent with this whole plan family's
standing discipline (`superplan`'s own P003: "never invent new
cryptography, compose audited primitives"). Whether this project runs
its own relay-equivalent transport, reuses the public Nostr relay
network, or routes through `EphemNet`-hosted self-hosted relays is an
open question — see below.

**Why Nostr, in full — pros, cons, and what it actually gives us.**
Nostr ("Notes and Other Stuff Transmitted by Relays") is not a
blockchain and not owned by a company: identity is a secp256k1
keypair, content is signed JSON "events," and transport is relays —
simple WebSocket servers anyone can run, with no permission logic
required by the protocol itself. The spec is public domain, and NIPs
(Nostr Implementation Possibilities) are a lightweight, permissionless
extension process — anyone can define new event kinds without asking
anyone's approval, which is exactly the property this design leans on
(the attestation primitive below is one more event kind, not a fork).
Most client and relay implementations are open source, in multiple
languages, with no single company able to shut the ecosystem down.

- **Real, already-proven synergy with the payment rail already
  chosen**: NIP-57 ("zaps") bakes Lightning micropayments directly
  into the protocol and is genuinely widely used today — concrete,
  in-production proof that "identity plus Lightning payments" already
  works at real scale, not a novel combination this design would be
  first to attempt.
- **The honest weakness, and why it matters here specifically**:
  Nostr has essentially no protocol-level spam or sybil resistance —
  anyone can mint unlimited free keypairs and publish unlimited
  events. Deployed apps cope with ad hoc relay-side rate-limiting and
  client-side social-graph filtering, not a protocol-level answer.
  This is precisely why Nostr apps commonly feel noisy and full of
  low-quality threads — and the *same* weakness would undermine
  attestation credibility if adopted without a fix, since an attacker
  could mint personas to vouch for each other exactly like spam. The
  Sybil Resistance section below is this design's answer; Nostr itself
  doesn't provide one.
- **What's actually already built vs. what we'd still build
  ourselves**: the protocol gives identity and event transport. It
  does not give a trust-scoring layer, an attestation UI, or an
  identity-and-reputation product — that's real, still-greenfield work
  regardless of building on Nostr. The bulk of Nostr's existing
  tooling and mindshare is chat-app-shaped, even though the underlying
  protocol is general-purpose.
- **A structural parallel worth noting, not yet acted on**: a client
  has to know *which relays* to query to find someone's data — Nostr's
  own relay-discovery problem rhymes with `EphemNet`'s GUID→endpoint
  rendezvous problem. Worth revisiting whether `EphemNet`'s own
  mechanism could double as Nostr relay discovery down the line, once
  both are further along — not a decision to make now.

**Event/claim format**: Nostr-style signed JSON events, reusing
existing NIP patterns rather than inventing new wire formats:
- **NIP-98** (HTTP auth via signed Nostr events) is the base mechanism
  for sign-in — a site challenges, the persona's client signs, the
  site verifies. No custom auth protocol needed.
- **NIP-58** (badges) is the base pattern for the attestation
  primitive below — one persona issuing a signed claim about another
  already has real prior art in the Nostr ecosystem, not just a
  distant analogy.

**Attestation primitive**: a signed claim —
`(attester_key, subject_key, claim_type, claim_value, timestamp,
signature)` — published as an event. Cheap to issue, cheap to verify
(a signature check, no central lookup required). `claim_type` is
deliberately open-ended (this is the actual point of building on
Nostr's NIP-extension culture): skill/credential attestation, service-
ownership/provenance attestation (an owner persona representing a
site or service it operates, verifiable and portable across everything
that persona runs), and — critically — **recovery-guardian and
recovery-confirmation attestations**, which is how recovery (below)
reuses this same primitive instead of needing its own system.

**Trust model: relative, not global.** There is no single canonical
"reputation score." A verifier computes trust in a claim by weighting
it against *their own* vouched set (a web-of-trust graph, PGP-style),
not by querying one universal number anyone could game by controlling.
This is both the philosophically correct model (real-world reputation
works this way — trust is always relative to who's asking) and a
structural defense against a single point of manipulation.

**Sybil resistance: economic cost, not proof-of-personhood.** Nostr's
own well-known weakness (unlimited free keypairs, no protocol-level
spam resistance) would apply directly to attestations too if left
unaddressed: an attacker can mint unlimited personas to vouch for each
other for free. The fix, reusing infrastructure this whole product
group is already building rather than inventing a new one: an
attestation only counts toward a subject's weighted trust if it was
published with a small L402-metered payment attached, or backed by a
bonded stake that can be slashed if the claim is later disputed and
found false (the same dispute-resolution pattern used by real systems
like Kleros or Augur's reporting bonds). **Explicitly rejected**:
biometric proof-of-personhood (WorldCoin-style) — effective at sybil
resistance, but centralizing and coercive in a way that's in direct
tension with this project's own no-gatekeeper mission (`superplan`'s
T007). **Worth studying directly, not just citing**: BrightID's
decentralized social-graph-based uniqueness verification is the
closest existing project to what's wanted here philosophically.

**Recovery: the options actually considered, and why this combination
was chosen.** Key loss/theft is the hardest unresolved risk this whole
identity model carries. Four real options exist, evaluated against
each other rather than picking the first plausible one:

- **No recovery (pure bearer key)** — the honest baseline every
  alternative should be measured against: lose the key, lose the
  persona, permanently, the same as an unbacked Bitcoin wallet. Too
  brutal here specifically, since an identity carries years of
  accumulated reputation, not just a balance.
- **Social recovery via SSKR (Shamir's Secret Sharing)** — not a new
  idea: `superplan`'s own P003 vault research already validated this
  exact primitive. It's also proven outside this plan, in production,
  as the mechanism behind real social-recovery wallets — **Argent**
  and **Safe** (formerly Gnosis Safe) both let a threshold of trusted
  guardians jointly help recover access, with no single guardian able
  to steal it alone.
- **Delegated recovery via attestation** — the refinement adopted
  here: instead of SSKR guardianship being a separate system bolted on
  next to attestation, guardianship *is* an attestation. A persona
  designates recovery guardians via a `recovery_guardian` claim, and a
  threshold of `recovery_confirm` claims from those guardians
  authorizes recovery of a new key — the exact same primitive already
  built for reputation, reused rather than duplicated.
- **Threshold/multi-device signing (e.g. FROST)** — a different kind
  of answer: instead of recovering *after* loss, reduce how often
  recovery is needed at all, by never putting the whole key on one
  device in the first place. Genuinely valuable, but a materially
  bigger lift than SSKR-plus-delegation, so explicitly deferred past
  v1 rather than treated as required for a working answer today.

**Chosen**: SSKR-based key splitting, authorized via delegation
attestation, with multi-device threshold signing as a future hardening
layer, not a v1 requirement. **The residual risk, stated honestly
rather than glossed over**: any threshold-guardian scheme has its own
attack surface — coercing or colluding against a threshold of someone's
guardians is a real attack, not a hypothetical one. This mitigates the
key-loss risk; it does not make it disappear, and shouldn't be
marketed as though it does.

**Payment dependency**: this design does not implement its own
payment rail. Both the sybil-resistance mechanism above and sign-in
billing depend on `cinder`'s P004/L402 infrastructure. As of
2026-09-22, that infrastructure has a real, live-verified reference
architecture to follow (`cinder`'s D005) rather than being purely
notional — see the Payment Integration section below for how this
design actually plugs into it.

**Payment integration (added 2026-09-22, following `cinder`'s D005
pattern)**: `cinder`'s own L402 write path settled on fronting a
minimal backend with `lightninglabs/aperture` — a production L402
reverse proxy — rather than any service implementing macaroon minting,
invoicing, or preimage verification itself. This design reuses that
same pattern rather than inventing a second one, but the gating point
differs from `cinder`'s because persona has no single central write
server to sit Aperture in front of — attestation events can be
published to any relay.

- **The gate is a per-operator "attestation-cost gateway," not a
  relay.** Any operator — including a reference deployment this
  project itself could run — fronts a minimal backend service with
  Aperture, exactly like `cinder`'s internal paid listener, whose only
  job is: accept an already-L402-paid request, and mint a small signed
  receipt (timestamp + nonce, no macaroon/Lightning code of its own).
  The attester embeds that receipt in the attestation event's tags
  when publishing. No payment-verification code lives in persona's own
  client or relay software, same as `cinder` carries none — Aperture
  and the Lightning payment it gates own that entirely.
- **Consistent with the relative-trust model, not a canonical gate.**
  A verifier's weighting logic decides *which gateway operators'*
  receipts it trusts as evidence of real economic cost — there is no
  single blessed gateway anyone could capture by running it, the same
  way there's no single blessed reputation score. A verifier that
  trusts no gateway operator can still fall back to the bonded-stake
  alternative already described above, which needs no gateway at all.
- **Sign-in billing follows the identical shape**: a relying site that
  wants to charge for NIP-98 sign-in fronts its own login-verification
  endpoint with its own Aperture instance, registering one discrete
  pricing tier — the same "add one internal listener, no payment code
  of your own" shape as `cinder`'s paid-write listener, not a distinct
  mechanism.
- **Shared-gateway reuse, same as `cinder`/`EphemNet`**: one Aperture
  instance can front multiple backend services at once — nothing here
  requires a persona-specific gateway deployment; an operator already
  running Aperture for another purpose can register an
  attestation-receipt service alongside it.
- **Still open, not yet decided**: whether this project runs one
  reference gateway of its own (the way `cinder` runs its own paid
  listener) versus leaving every relying party to stand up its own —
  a positioning question, not an architectural blocker, and separate
  from the "where does persona run" open question below.

## Key Decisions

- **Adopt Nostr's identity/event format rather than a custom
  protocol**: avoids reinventing well-understood cryptography, and
  gets a real, already-standardized pattern for both sign-in (NIP-98)
  and attestation (NIP-58) instead of designing wire formats from
  scratch.
- **Trust is relative/per-viewer, not a global score**: avoids a
  single canonical authority anyone could capture, and matches how
  reputation actually works outside software.
- **Sybil resistance via economic cost (L402/Lightning), not
  biometrics**: consistent with the project's own mission; explicitly
  rejects the most effective *alternative* mechanism (proof-of-
  personhood) because it's centralizing, not because it doesn't work.
- **Recovery reuses the attestation primitive rather than being a
  separate system**: `recovery_guardian`/`recovery_confirm` are just
  claim types, not new infrastructure — the same design economy
  `cinder`'s own `owner_token` already favored (generalizing rather
  than adding a separate lease mechanism).
- **Multi-device threshold signing deferred past v1**: a real
  hardening layer, but not required for a working recovery story, and
  a materially bigger lift than SSKR-plus-delegation.
- **Payment integration reuses `cinder`'s Aperture-fronted L402
  pattern via per-operator "attestation-cost gateways," not a
  persona-specific payment implementation**: no macaroon/Lightning
  code in persona's own client or relay software, consistent with
  `cinder` carrying none either; which gateway operators' receipts
  count is a per-verifier trust decision, keeping this consistent with
  the relative-trust model rather than introducing a canonical gate.

## Open Questions / Unknowns

- **Where this actually runs**: own relay infrastructure, the
  existing public Nostr relay network (real network-effect upside,
  but inherits Nostr's existing spam/noise culture and userbase
  expectations), or `EphemNet`-routed self-hosted relays. `EphemNet`
  now has a concrete, buildable DNS-forwarding mechanism that makes
  self-hosting a real option rather than an aspiration — this doesn't
  pick a winner, but removes "self-hosting is too impractical" as a
  reason to rule it out. Still close to a product-positioning
  decision, not just a technical one.
- **Interop scope**: fully compatible with the existing Nostr network,
  or a deliberately separate, incompatible fork that doesn't inherit
  Nostr's existing baggage but also doesn't get its existing network
  effect? Not decided.
- **Bonding/slashing mechanics for disputed attestations**: named as a
  mechanism in Architecture above, but the actual dispute-resolution
  process (who adjudicates, how slashed stakes are handled, what
  prevents the adjudication step itself from being gamed) is genuinely
  hard mechanism design and needs its own dedicated pass — not
  resolved here.
- **`claim_type` governance**: who governs the namespace as it grows —
  Nostr has NIPs as a real (if informal) governance process for
  exactly this; this design doesn't have an equivalent yet.
- **Data-protection tension**: an append-only, public, peer-attested
  claim history sits in real tension with data-protection regimes that
  include a "right to erasure" (GDPR-style) — not addressed at all
  yet, and worth resolving before, not after, real personal data flows
  through this system, especially given how much this whole product
  group already leans on the EU's regulatory posture elsewhere
  (`superplan`'s T004).

## Related

- **Thesis**: [[T001]]
- **Project**: [[P001]]
- **Dependency**: `cinder`'s P004 (L402 payment rail)
- **Reference**: `superplan`'s own P003 (SSKR/Shamir's Secret Sharing
  precedent), `cinder`'s D003 (the `tlock`/mandatory-crypto-guarantee
  precedent this design's sybil-resistance mechanism follows the same
  spirit of — make the guarantee inherent, not optional), `cinder`'s
  D005 (the Aperture-fronted L402 write-path pattern this design's
  payment integration follows)
- **Origin**: seeded from `superplan`'s `plan/designs/D003-persona-
  identity-attestation-recovery.md`, where this spec was originally
  worked out

## Log

2026-09-21 — Design seeded from `superplan` (repo: superplan, D003),
carrying forward its full content unchanged at time of seeding. Future
revisions belong here, not in superplan's copy.

2026-09-21 — Checked `cinder`'s P004/D005 status: cinder's own side of
the L402 write path is built and verified live (internal paid
listener, shared-secret middleware, 30-day TTL ceiling), fronted by
`lightninglabs/aperture` rather than cinder rolling its own macaroon/
invoicing code. P004 itself moved to DEFERRED, but only its remaining
piece (Aperture deployment/config, a real hosted Lightning provider
account) — the part needed for cinder's *own* paid tier to go fully
live end-to-end, not the part this design needs. This design now has
a real, live-verified reference architecture (D005) to follow for its
own sybil-resistance/sign-in payment integration, rather than an
abstract "L402 will exist eventually" dependency.

2026-09-22 — Wrote the actual payment-integration design (new
Architecture subsection above), following `cinder`'s D005 pattern
rather than inventing a separate one. Key departure from `cinder`'s
own shape, worked through explicitly rather than copied blindly:
`cinder` gates a single central write server, but persona has no
single central point events flow through, so the gate is reframed as
a per-operator "attestation-cost gateway" issuing a signed receipt
embedded in the attestation event, with each verifier deciding which
gateway operators' receipts it trusts — kept consistent with the
existing relative-trust model instead of accidentally introducing a
canonical gate. Sign-in billing mapped onto the identical shape. Left
open: whether this project runs its own reference gateway or leaves
that entirely to relying parties.
