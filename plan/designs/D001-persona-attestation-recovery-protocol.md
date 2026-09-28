---
id: D001
title: Persona identity protocol — Nostr-based keys, peer attestation, social recovery
status: DONE
project: P001
created: 2026-09-21
updated: 2026-09-28
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
cryptography, compose audited primitives"). This project runs on the
public Nostr relay network rather than its own relay-equivalent
transport — see the Relay hosting and interop scope section below.

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

**Relay hosting and interop scope (resolved 2026-09-28).** Two
questions that turn out to be more decoupled than they first looked,
once the Key Decision above (Nostr's own key/event format, not a
custom protocol) is taken as already locked in: a true wire-format
*fork* was never really on the table, since new `claim_type`s are
additive NIP-style extensions, not a protocol break. What's actually
open is narrower — which relays carry the traffic, and how much effort
goes into staying legible to generic, non-persona-aware Nostr clients.

- **Where this runs: the public Nostr relay network, chosen over
  running a dedicated persona relay network.** A project-operated
  relay network was considered and rejected: if it became the de
  facto place personas publish, it would recreate exactly the kind of
  gatekeeper this whole product group's mission (`superplan`'s T007)
  exists to avoid, plus real ongoing operational burden (uptime, abuse
  handling) — the same "don't take on operational weight prematurely"
  reasoning already applied to choosing hosted Lightning and Aperture
  over self-hosting/rolling-your-own in `cinder`'s D005. The public
  network is real, already-proven infrastructure at scale, at zero
  build cost, and genuinely decentralized already (many independent
  operators, no single required one) — the same reuse-over-build
  instinct behind picking Aperture in the first place.
- **`EphemNet`-routed self-hosted relays stay a supported option, not
  a v1 investment.** Philosophically the most consistent choice (each
  person's own relay, no institutional dependency at all), but with no
  production precedent yet and the relay-discovery problem (which
  relay to even query for a given persona) still genuinely unresolved
  — a real option to keep open, not something worth designing further
  for v1 specifically.
- **Interop scope: full wire-format compatibility, not a fork —
  answered essentially for free by the relay choice above.** Since the
  format was already Nostr-compatible and the transport is now the
  public network itself, the remaining open question was only ever
  "how legible should this be to a generic client," not "compatible or
  not." Resolved as: degrade gracefully where cheap (e.g. a
  human-readable `content` summary on attestation events), but don't
  contort the schema chasing full semantic rendering in generic
  clients — the trust/verification computation is this project's own
  value-add regardless, same as Nostr itself not providing a
  trust-scoring layer (see above).

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

**`claim_type` namespace governance (resolved 2026-09-28).** NIPs
already govern Nostr's event *kinds* this way; this needs the same
model one layer down, for the `claim_type` vocabulary living inside
the attestation event kind itself. Two parts, working together:

- **A namespacing convention, so minting a new `claim_type` never
  needs anyone's permission.** Every `claim_type` carries a
  reverse-domain-style prefix (e.g. `org.solemn.skill.rust`,
  `com.example.recovery_guardian`) — this alone prevents collision
  with no registry required at all, the same purpose NIP-78's own
  app-data namespacing already serves elsewhere in the Nostr
  ecosystem.
- **An open, non-authoritative spec registry for convergence on common
  types** — a NIP-style document (public repo, PR-based, rough
  consensus, no formal approval gate) listing well-known `claim_type`s
  and their `claim_value` shape. Seeded from what this design already
  names, not designed in the abstract: `skill`/credential claims,
  service-ownership/provenance, `recovery_guardian`/`recovery_confirm`,
  and the dispute-related types from the bonding/slashing mechanism
  above.
- **The registry documents; it doesn't govern.** Same as a NIP itself
  carries no enforcement power, an entry here is coordination, not
  permission — nothing stops anyone from using an undocumented or
  differently-defined `claim_type`. If two registries (or two
  communities) disagree about what a given type means, that's the same
  "pick whose authority you trust" relativity already used for gateway
  operators (payment integration, above) and arbiters (dispute types,
  above) — a pattern recurring often enough in this design that it's
  worth naming explicitly rather than treating each instance as a
  one-off: **wherever this design would otherwise need one canonical
  authority, it instead makes the choice of whose authority to trust a
  per-verifier decision.**

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

**Dispute types, and where bonding/slashing actually applies (added
2026-09-27).** The bonded-stake alternative above named a mechanism
without scoping it — "disputed and found false" covers wildly
different situations depending on what `claim_type` is being disputed,
and a single arbiter-adjudicated bond-slashing process is a good fit
for only one of them. Four real dispute types, evaluated separately
rather than assumed to be one problem:

- **1. Behavioral/quality disputes between two identified parties**
  ("this persona scammed me," "didn't deliver as claimed") — the
  single most common dispute in practice, by direct analogy to every
  existing reputation system (eBay, Airbnb, Yelp all handle exactly
  this). Bounded, factual/quality disagreement between two known
  parties. **Bonding/slashing with a per-dispute arbiter is the right
  tool here, and this is the only dispute type v1's mechanism is
  scoped to.** Kleros's own flagship real-world use case is this exact
  pattern, not prediction-market-style objective-outcome disputes.
- **2. Sybil-ring vouching** — the structural threat the bonded-stake
  mechanism was originally motivated by (attackers minting personas to
  vouch for each other), but a poor fit for per-claim arbitration:
  "was *this one* attestation between A and B genuine" is nearly
  unanswerable in isolation. Sybil rings are a **graph-level pattern**
  (clusters of freshly-minted keys vouching for each other, no
  independent history) — closer to fraud/pattern detection than
  fact-checking a single claim. **Not suitable for arbiter-adjudicated
  bonding/slashing at all.** Belongs to heuristic/graph analysis that
  flags suspicious clusters for verifiers to independently discount —
  a separate mechanism, not designed here yet.
- **3. Ownership/provenance claims** ("this persona operates domain
  X") — usually independently, objectively checkable (a DNS record, an
  HTTP challenge, cryptographic proof-of-control), not a social
  judgment call. **Not suitable for bonding/slashing** — a bond+arbiter
  process would be strictly worse than direct verification for
  anything actually checkable this way; reserve bonding for claims that
  aren't independently verifiable.
- **4. Recovery-guardian/recovery-confirm disputes** (a coerced or
  colluding guardian wrongly confirming recovery) — categorically
  different: the harm is losing the identity itself, not a financial
  loss a slashed bond could make whole after the fact. **Not suitable
  for post-hoc bonding/slashing at all** — this needs a challenge
  window *before* recovery finalizes, not a bond to slash afterward.
  Stays out of scope for the bonding/slashing mechanism entirely; see
  the Recovery section below, which already carries its own honestly-
  stated residual risk for exactly this scenario.

**Bonding/slashing mechanism (v1, scoped to dispute type 1 only):**

- **Escrow, not a payment.** This is the key structural point: L402/
  Lightning (used for the payment path above) settles instantly — the
  right shape for "pay once, get a receipt," the wrong shape for
  "hold funds forfeitable pending a dispute window." A bonded stake
  needs actual escrow, a different financial primitive sharing only
  the same non-custodial spirit, not "L402 but bigger." V1 uses a
  simple n-of-m multisig (no new cryptographic primitive — deferred:
  DLC/oracle-based escrow, same "defer the bigger lift" spirit as
  deferring FROST for recovery).
- **Arbiter choice is per-dispute, not protocol-wide.** Whoever relies
  on a bond (the subject, or a verifier) picks which arbiter(s) they
  trust to adjudicate that specific dispute — consistent with this
  design's own relative-trust model, not a single blessed court. A
  global crowdsourced court (Kleros's own actual model) was considered
  and rejected for exactly this reason: it produces one canonical
  true/false verdict everyone inherits, which is itself the kind of
  single-point-of-capture authority this design's relative-trust
  principle exists to avoid.
- **Arbiter panel scales with stake size.** A single mutually-agreed
  arbiter is enough for small bonds — the dispute is itself a tiny
  loss either way. For larger bonds, a panel selected jointly by the
  attester and the challenger (rather than either side unilaterally
  picking someone favorable) is the v1 answer for higher-stakes
  disputes.
- **Self-releasing by default.** An unchallenged bond returns to the
  attester automatically after a fixed window — no arbiter action for
  the common, non-disputed case, only for actual disputes.
- **Symmetric skin-in-the-game against frivolous disputes.** A
  challenger must also stake; if the challenge fails, the
  *challenger's* stake is what gets slashed, not just the attester's —
  guards against dispute-spam/griefing, the same logic Augur's own
  reporting-bond design uses.
- **Residual risk, stated honestly rather than glossed over**: arbiter
  collusion or capture over time is real, the same honesty this design
  already applies to guardian collusion in the Recovery section below.
  Mitigated, not eliminated, by keeping arbiter choice relative and
  per-dispute rather than protocol-wide.

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

**Data protection / GDPR right-to-erasure (resolved 2026-09-28).**
Nostr relays are independently operated
and copy data freely once published — there is no single deletable
copy, and NIP-09 "deletion" events are only advisory (a well-behaved
relay hides the referenced event; nothing stops another relay from
ignoring the request, or a client that already fetched it from keeping
a copy). Protocol-level guaranteed erasure isn't achievable on this
transport, full stop — the honest answer routes around that rather
than pretending otherwise, the same discipline this design already
applies to recovery's residual guardian-collusion risk.

- **Architectural avoidance, as the primary strategy.** Claims
  reference pseudonymous keys and abstract `claim_type`/`claim_value`s
  by default — nothing in a public attestation event should actually
  constitute "personal data" under GDPR's definition unless someone
  deliberately puts their own real-world PII into a claim they
  control. No personal data on the wire means no erasure obligation to
  satisfy in the first place.
- **Cryptographic erasure ("crypto-shredding") as the fallback**, for
  anything that genuinely must reference personal content: encrypt
  `claim_value` with a key the *subject* — not the attester — controls,
  so erasure means the subject destroying their own key. The ciphertext
  persists on relays permanently, but becomes permanently unreadable.
  This is the accepted pattern other immutable-ledger projects use to
  reconcile immutability with erasure obligations. **Stated honestly,
  not marketed around**: metadata, the fact an attestation existed at
  all, and the ciphertext itself all still persist — this satisfies
  the spirit of erasure for the underlying personal content, not a
  literal removal of bytes from every relay that ever held them.
- **"Who's the controller" resolves the same way the other open
  questions have.** There is no single global controller across an
  independently-operated relay network (reinforced by this design's
  own choice to run on the public Nostr network rather than a
  project-operated one), so GDPR responsibility attaches per-operator:
  whoever's software actually stores personal data — a client, an
  attestation-cost gateway operator — is the controller for what *they*
  store. This design's job is minimizing what any operator needs to
  store at all, not solving erasure for the whole network — the fourth
  instance of the recurring pattern named in the `claim_type`
  governance section above: substitute distributed, per-operator
  responsibility wherever a single canonical authority would otherwise
  be needed.
- **Still genuinely open**: the exact threshold for what counts as
  "personal data" triggering the encrypted-`claim_value` path by
  default (a policy/legal judgment call, not resolved here), and
  whether this project needs its own plain-language data-protection
  notice given `superplan`'s existing EU-regulatory-posture leaning
  (T004) — a product/legal task, not an architectural one.

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
- **Bonding/slashing is scoped to one dispute type only — behavioral/
  quality disputes between two identified parties**: sybil-ring
  vouching, ownership/provenance claims, and recovery disputes each
  need a different mechanism (graph heuristics, direct cryptographic
  verification, and a pre-finalization challenge window, respectively)
  and are explicitly out of scope for arbiter-adjudicated bonding.
- **Arbiter choice is per-dispute, not a global court**: rejects
  Kleros's own crowdsourced-court model specifically because it would
  produce one canonical verdict every verifier inherits, in tension
  with this design's relative-trust principle; a single arbiter for
  small bonds, a panel jointly selected by both parties for larger
  ones.
- **Bonding uses escrow (an n-of-m multisig, v1), not a Lightning
  payment**: L402 settles instantly and isn't the right primitive for
  funds that must stay forfeitable pending a dispute window; DLC/
  oracle-based escrow deferred past v1, same spirit as deferring FROST.
- **Runs on the public Nostr relay network, not a dedicated persona
  relay network**: a project-operated relay would recreate the kind of
  gatekeeper this project's own mission rejects, plus real operational
  burden not worth taking on prematurely — the same reasoning already
  applied to choosing hosted Lightning/Aperture over self-hosting in
  `cinder`. `EphemNet`-routed self-hosting stays a supported option,
  not a v1 investment.
- **Interop scope: full wire-format compatibility, not a fork** —
  answered by the relay choice above once the Key Decision to reuse
  Nostr's own format is taken as already locked in; degrade gracefully
  in generic clients where cheap, without contorting the schema to
  chase full semantic rendering there.
- **`claim_type` governance mirrors NIPs one layer down**: a
  reverse-domain namespacing convention so minting a type never needs
  permission, plus an open, non-authoritative spec registry for
  convergence on common types — coordination, not enforcement, same as
  a NIP itself. Names explicitly, as a recurring pattern across this
  design: wherever a canonical authority would otherwise be needed,
  substitute a per-verifier choice of whose authority to trust
  instead.
- **GDPR erasure handled by architectural avoidance first, crypto-
  shredding as fallback, responsibility distributed per-operator**:
  keep personal data off the public attestation layer by default;
  where it must appear, encrypt under a subject-controlled key so
  erasure means key destruction, not byte removal; no single global
  controller across an independently-operated relay network, so
  responsibility attaches to whoever's software actually stores
  something — the fourth instance of this design's recurring
  no-canonical-authority pattern.

## Open Questions / Unknowns

- **Where this actually runs — resolved 2026-09-28**: the public
  Nostr relay network, not a dedicated persona relay network. See the
  new "Relay hosting and interop scope" Architecture subsection above
  for the full reasoning. `EphemNet`-routed self-hosting remains a
  supported option for anyone wanting zero reliance on public
  operators, but isn't a v1 design investment.
- **Interop scope — resolved 2026-09-28**: full wire-format
  compatibility, not a fork; graceful degradation in generic clients
  where cheap, no further chase beyond that. See the same subsection.
- **Bonding/slashing mechanics — resolved 2026-09-27 for one dispute
  type, deliberately not the other three.** See the new "Dispute
  types" Architecture subsection above: scoped to behavioral/quality
  disputes between two identified parties only (escrow via n-of-m
  multisig, per-dispute arbiter choice, panel-for-larger-stakes,
  symmetric challenger staking). Sybil-ring vouching, ownership/
  provenance claims, and recovery disputes are explicitly named as
  needing separate, still-undesigned mechanisms — not folded into this
  one. Still genuinely open within the scoped mechanism itself: exact
  bond-size thresholds for single-arbiter vs. panel, and how an
  arbiter panel is actually selected jointly (a fair joint-selection
  protocol isn't specified yet, just the requirement that it be
  joint).
- **`claim_type` governance — resolved 2026-09-28**: a reverse-domain
  namespacing convention (permission-free minting) plus an open,
  non-authoritative NIP-style spec registry for convergence on common
  types. See the new "`claim_type` namespace governance" Architecture
  subsection above.
- **Data-protection tension — resolved 2026-09-28**: architectural
  avoidance first (no personal data on the public attestation layer by
  default), cryptographic erasure as the fallback for anything that
  must reference personal content, GDPR-controller responsibility
  distributed per-operator rather than solved at the protocol level.
  See the new "Data protection / GDPR right-to-erasure" Architecture
  subsection above. Still genuinely open within that: the exact
  "personal data" threshold that triggers encryption by default, and
  whether this project needs its own data-protection notice — policy/
  legal tasks, not architectural ones.

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

2026-09-27 — Resolved the bonding/slashing open question, but only
for one dispute type, deliberately. Started from "what kind of
disputes would actually be most common" rather than designing the
mechanism first: split disputes into four types (behavioral/quality
between two parties, sybil-ring vouching, ownership/provenance,
recovery-guardian) and found only the first is actually a good fit for
arbiter-adjudicated bonding — the other three need graph heuristics,
direct cryptographic verification, and a pre-finalization challenge
window respectively, and would be badly served by forcing them through
one mechanism. For the scoped mechanism: rejected a Kleros-style
global crowdsourced court explicitly, since one canonical verdict
every verifier inherits is itself the kind of capturable central
authority this design's relative-trust principle exists to avoid — per-
dispute arbiter choice instead (single arbiter for small bonds, a
panel jointly picked by both parties for larger ones, per the user's
own suggestion). Also surfaced that bonding needs real escrow, not a
Lightning payment — a different financial primitive than the payment-
integration path above, sharing only the non-custodial spirit, not the
mechanism itself.

2026-09-28 — Resolved the two remaining coupled open questions: where
this runs, and interop scope. Recognized they were more decoupled than
they first looked once the existing Key Decision to reuse Nostr's own
format is taken as already locked in — a true wire-format fork was
never really on the table, so the actual open question was narrower
than "compatible vs. fork." Chose the public Nostr relay network over
a dedicated persona relay network, for the same reason `cinder` chose
hosted Lightning/Aperture over self-hosting: a project-operated relay
would recreate a gatekeeper and take on operational burden this
project doesn't need. `EphemNet`-routed self-hosting stays a
supported, not required, option. Interop scope followed from that
choice almost for free: full wire-format compatibility, graceful
degradation in generic clients where cheap, no further investment
chasing full semantic rendering there.

2026-09-28 — Resolved `claim_type` namespace governance: mirrored
NIPs one layer down rather than inventing new governance machinery —
a reverse-domain namespacing convention so minting a type needs no
permission, plus an open, non-authoritative spec registry (PR-based,
rough consensus) for convergence on common types, seeded from types
this design already names. Named explicitly, prompted by noticing
this is the third time the same shape of answer has come up: wherever
this design would otherwise need one canonical authority (a trust
score, a payment gateway, a dispute arbiter, now a namespace
governor), it substitutes a per-verifier choice of whose authority to
trust instead. Worth keeping in mind as a standing design instinct for
whatever open question comes next, not just documented after the
fact each time.

2026-09-28 — Resolved the last open question: GDPR right-to-erasure.
Started from the honest limit rather than looking for a way around it
— Nostr relays are independently operated and copy freely once
published, so protocol-level guaranteed erasure genuinely isn't
achievable, the same "state the residual risk plainly" discipline
already applied to recovery's guardian-collusion risk. Landed on
architectural avoidance as the primary fix (keep personal data off the
public layer by default, so there's usually nothing to erase),
cryptographic erasure/crypto-shredding as the fallback for anything
that must reference personal content (subject-controlled key
destruction, not byte removal), and — recognizing the fourth instance
of the same recurring shape — GDPR-controller responsibility
distributed per-operator rather than solved once at the protocol
level, consistent with there being no single canonical authority
anywhere else in this design. All five of D001's original open
questions are now resolved; nothing left unaddressed in this design.
