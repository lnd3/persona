---
id: A006
title: Bitcoin escrow settlement (script construction, funding, on-chain verification)
status: PLANNING
design: D001
project: P001
created: 2026-09-29
updated: 2026-09-29
---

## Context

The last piece of D001's Bonding/slashing mechanism that [[A005]]
deliberately did not build: actually constructing the n-of-m multisig
escrow script D001 names as the financial primitive, funding it,
verifying it on-chain, and settling it (self-release or verdict-based
payout) once [[A005]]'s pure protocol-layer logic
(`IsSelfReleased`/`Resolve`) says who should be paid. A005 scoped this
out explicitly because getting a timelock+multisig spending script
wrong is a fund-loss bug, a categorically different risk class than
anything else built in this project (identity, attestation, trust,
recovery, payment receipts) — none of which can lose anyone's money if
the code has a bug, only produce a wrong *opinion*.

**This action started from a harder position than A001-A005 did**:
those actions each had a design (D001) that named the mechanism and
left only implementation-level choices open (which library, which
kind number, which claim_value shape). This one's starting design —
"a simple n-of-m multisig" — turned out, on closer inspection while
filing this action, to have a real structural problem underneath it:
resolved below (2026-09-29, before any script code), not left open.

**Builds on**: [[A005]]'s `internal/dispute` (`BondInfo`/
`ChallengeInfo`.`EscrowRef` — this action is what actually makes
`EscrowRef` refer to something real; `ConfirmedPanel`/`Resolve`/
`IsSelfReleased`, which this action's settlement logic calls but does
not reimplement).

**Explicitly out of scope for this action**:
- Reopening any of A005's protocol-layer decisions (claim types,
  panel confirmation, verdict resolution, self-release timing) — this
  action consumes those decisions, it doesn't revisit them
- Any Lightning-based mechanism — D001 already ruled this out for
  bonding specifically ("the wrong shape for 'hold funds forfeitable
  pending a dispute window'"); this action is Bitcoin-only
- Running any wallet infrastructure, key-management UI, or hosted
  signing service — this action defines the scripts and transaction
  construction logic; actually holding private keys safely is a
  deployment/product concern, same posture [[A004]] took toward
  Lightning wallets

## Decisions made here (not in D001 — concrete choices this action owns)

- **The escrow structure is two-tiered, matching D001's own small/
  larger-bond distinction (A005 already left the exact sats threshold
  as a caller policy call) — decided 2026-09-29, refined across three
  passes below.** Filing this action first surfaced a structural
  tension D001's "simple n-of-m multisig" framing glossed over: an
  arbiter panel is chosen *after* a dispute arises, but a spendable
  Bitcoin script has to be constructed *before* anyone knows who will
  spend from it. Working through that produced two genuinely different
  answers for the two cases D001 already distinguishes, not one
  answer that fits both:

  **Small bonds — cooperative escalation, mutual settlement first.**
  A bond starts as a plain timelocked output: attester spends alone
  after `window_days`, no multisig at all, since there's no
  counterparty yet. A challenger's stake is symmetrically just their
  own timelocked self-custody output. Only on a challenge do the two
  parties cooperatively escalate into a joint output — but that joint
  output's *first* spending path is a plain **2-of-2 mutual
  settlement** (any split either side actually agrees to), available
  immediately, no panel needed: most disputes between two people who
  both have real money on the line resolve by direct agreement once a
  challenge forces the conversation, and D001 itself says a single
  mutually-agreed arbiter is "enough" for this tier — third-party
  adjudication should be the fallback for the minority that can't
  agree, not the default path. Only if 2-of-2 settlement fails does a
  *second* cooperative move (agreeing "let an arbiter decide" is a much
  lower-friction ask than agreeing on a number) hand it to a
  mutually-picked single arbiter, per `ConfirmedPanel`.
  - *Rejected for this tier*: a pre-designated mediator chosen at
    bond-creation time. The well-trodden real-world pattern (2-of-3:
    payer, payee, a pre-agreed mediator) requires naming the mediator
    before the challenger exists — meaning the attester alone would be
    choosing who judges disputes against themselves. Rejected on
    principle, not inconvenience: a direct regression against D001's
    founding "never let one party unilaterally pick their own judge"
    rule, the exact single-point-of-capture failure the relative-trust
    model exists to avoid.
  - *Residual risk, stated honestly*: an attester can refuse to
    cooperate with escalation at all and simply wait out their own
    timelock to reclaim their bond unilaterally — no script can force
    cooperation without reintroducing the pre-committed-authority
    problem just rejected. **Not resolved cryptographically, only
    reputationally**: a verifier treats "challenged, never escalated"
    as at least as bad as losing the dispute outright, the same
    technical-limit-to-social-enforcement move D001 already made for
    GDPR erasure. The challenger's own stake is never at risk from this
    — it just self-returns to them too, since it was never actually
    joint until escalation completed.

  **Larger bonds — each side independently pre-commits their own
  arbiter's *identity*, upfront; the escalation *mechanics* still need
  one cooperative step — corrected 2026-09-29, see Log.** The original
  version of this decision claimed each party's own output "already
  commits to an arbitration path from the start," baked into the
  script at funding time. **That was wrong, caught while actually
  implementing it**: a Bitcoin script's public keys are fixed the
  moment an output is created and can never be added later. The
  attester's bond is created *first*, before any challenger (and
  therefore any second arbiter) exists — so the bond's own output
  structurally cannot name a key that doesn't exist yet, the same
  chicken-and-egg problem this whole action started from, one level
  down. The obvious patch — let each side's *own* appointed arbiter
  alone move that side's output — was checked and rejected too: it's
  insecure, not just simpler, since an owner and their own hand-picked
  arbiter could then always collude to return funds to the owner
  regardless of the actual dispute, exactly the check requiring *both*
  arbiters to agree exists to prevent.
  - **What's actually achievable, split honestly into two parts**:
    - *Pre-committed, immutable arbiter identity* — each side still
      independently picks their own arbiter, at funding time, and that
      choice can never be swapped or haggled over later. This part
      genuinely needs no cooperation and is published as its own
      claim (see Decisions below) rather than baked into the escrow
      script itself, since the script can't hold a key that doesn't
      exist yet regardless.
    - *Escalation into joint arbiter custody still needs one
      cooperative step* — moving self-custodied funds into a jointly-
      controlled output needs each owner's own signature on their own
      output, exactly the same shape (and the same residual
      non-cooperation risk) as the small-bond tier's escalation.
      Bitcoin's own constraints make this unavoidable; no clever
      scripting removes it.
  - **What this tier still actually buys, stated honestly rather than
    reasserting the original overclaim**: once *both* sides have
    escalated, resolution power sits *entirely* with the two
    pre-committed arbiters — unlike the small-bond tier, there is no
    2-of-2 mutual-settlement path back to the disputants themselves,
    so once escalated, neither party can further stall or renegotiate
    the outcome. The value is a decisive, un-renegotiable process and
    an arbiter choice that can't be haggled over after the fact — not
    "arbitration with zero cooperation ever required," which isn't
    actually achievable here.
  - **Not the same thing as the rejected pre-designated mediator**,
    despite also being chosen "upfront": the earlier rejection was
    specifically about *one shared judge*, unilaterally imposed by
    whichever party existed first. This is each side independently
    appointing their *own* representative over their *own* funds only
    — structurally more like each side hiring their own lawyer than
    one side picking a shared arbitrator for both.
  - **New claim type: `net.persona.core.arbiter_commitment`** — reuses
    A001's attestation primitive again rather than inventing a
    separate durable-record mechanism, consistent with A003/A005's own
    "X is just an attestation" precedent. Attester = the bond or
    challenge owner, subject = themself, `claim_value` =
    `"<bond_or_challenge_event_id>:<arbiter_pubkey>"`, published
    alongside the bond/challenge event. This is an addition to A005's
    protocol layer, not a change to any of A005's already-shipped
    claim types (`bond`/`dispute_challenge` keep their existing
    `claim_value` shape unchanged) — a deliberate choice to avoid
    reopening a DONE action's format for something that can just as
    well be a companion claim.
  - **Tie-break when the two appointed arbiters disagree — decided
    2026-09-29: they jointly escalate to a third arbiter, using the
    same joint signing power they already have.** No new key needs to
    be pre-committed upfront to make this work: once escalation has
    happened, `arbiter_A` and `arbiter_B` already jointly control the
    resulting joint output (it takes both of them to move the funds at
    all), so instead of using that joint power to decide the dispute's
    substance, they can use
    the identical power procedurally — cooperatively constructing (via
    PSBT, same tool already chosen) a transaction that moves both
    outputs into a fresh 2-of-3 output naming themselves plus a
    jointly-nominated third arbiter. Settlement then just needs that
    third arbiter to agree with whichever of the original two they
    find more persuasive (2-of-3, not the third arbiter alone) —
    preserving some accountability rather than handing a stranger
    unilateral power. This is the same shape as the small-bond tier's
    own "agreeing to let someone decide is lower-friction than
    agreeing on the decision itself" move, one level up: the two
    arbiters are far more likely to cooperate on *escalating* than the
    original disputants were, since they have professional/
    reputational incentive to actually resolve the case, not a direct
    financial stake in the outcome.
    - **This terminates in exactly one escalation step, never
      recurses further.** Verdicts are binary (A005's `Verdict` type:
      `attester_wins`/`challenger_wins`, nothing else) — so the moment
      a third arbiter casts any vote at all, they necessarily side
      with one of the original two, and 2-of-3 is immediately reached.
      There is no way for three binary voters to produce a fresh tie
      once all three have actually weighed in, so no fourth or fifth
      arbiter is ever needed. The only way this step fails to resolve
      anything is the two original arbiters never agreeing to involve
      a third at all — a cooperation failure, not a repeating tie, and
      already covered by the residual below.
    - *Rejected*: pre-committing a third tie-breaking key at
      bond-creation time (reopens the exact chicken-and-egg problem
      this tier already solved once — nobody exists yet to jointly
      name a tie-breaker at funding time either), and giving per-side
      small panels instead of single arbiters (doesn't actually
      eliminate ties, just moves the same problem to a larger even
      split, e.g. 3-vs-3, without a structural fix).
    - *Residual, stated honestly*: if the two original arbiters can't
      even agree to escalate (rarer and one step further removed than
      the disputants themselves failing to cooperate, but not
      impossible), nothing forces it — each side's own pre-escalation
      timelock fallback stands as the bounded worst case, same
      "reverts to each party keeping their own original funds, nobody
      else's money is ever at risk" property the small-bond tier's own
      non-cooperation residual already relies on.
  - **This tier still improves on a naive shared-multisig-from-day-one
    approach**: each side's arbitration branch only activates on
    *their own* output, so a bug in one side's script can't misdirect
    the other side's funds — the blast radius of a mistake stays
    contained to whichever side's own arbiter-branch code is wrong.

  **Operational consequence shared by both tiers**: `window_days`
  (A005's own per-bond field) must be set comfortably longer than the
  realistic real-world time needed to actually resolve a dispute
  through whichever tier's process applies — a deployment/product
  guidance note, not a new claim_value field.
- **Testnet-only until a real security review happens.** Given the
  fund-loss stakes, this action's own Tasks list below treats "get it
  reviewed before it ever touches mainnet sats" as a hard gate, not a
  nice-to-have — consistent with how seriously `cinder`'s own D005
  treated Lightning infrastructure risk, but a strictly higher bar
  here since a script bug is unrecoverable in a way a Lightning
  provider outage isn't.

- **PSBT (BIP174), not a custom transaction-serialization format**,
  for any multi-party transaction construction this action ends up
  needing (whichever direction above is chosen, cooperative signing
  between at least two parties is unavoidable) — a standardized,
  wallet-interoperable format, not something worth reinventing.
- **Output descriptors (BIP380), not a free-text `escrow_ref`
  convention**, for what `EscrowRef` in A005's `BondInfo`/
  `ChallengeInfo` actually contains once this action defines it
  concretely — standard, tool-supported, and already what A005's own
  colon-parsing fix anticipated (descriptors can contain colons,
  which is exactly the case A005's regex fix was written to handle).
- **Regtest/testnet only until a security review**, mainnet
  activation gated behind it explicitly, not left as an implicit
  "someday" — this is the fund-loss-risk piece of work D001 and A005
  both flagged, and this action treats that flag as binding on itself,
  not just on its predecessors.

## Tasks

### Shared: pre-escalation output (both tiers, single-party self-custody)
- [ ] Construct a plain CSV-timelocked single-sig output script:
      owner spends alone after `window_days` — used identically for
      both a bond (attester as owner) and a challenge stake
      (challenger as owner)
- [ ] Generate a funding address/descriptor for a new bond or
      challenge stake, to populate `EscrowRef` (BIP380 descriptor, per
      Decisions)
- [ ] Verify a referenced escrow is actually funded on-chain to the
      claimed `amount_sats` (watching for confirmation, not just an
      unconfirmed mempool entry)

### Small-bond tier: cooperative escalation, mutual settlement first
- [ ] On challenge, cooperatively construct (via PSBT) a joint output
      with two paths: 2-of-2 mutual settlement (any split, no panel
      needed), and a fallback spendable once `ConfirmedPanel` confirms
      a single mutually-agreed arbiter
- [ ] Detect non-escalation: a challenge exists but no escalation
      transaction has been broadcast within a reasonable margin of
      `window_days` — surfaced as data for a verifier's own
      reputational judgment, not enforced in-code
- [ ] Settlement: 2-of-2 mutual-settlement spend (either side proposes
      a split, the other co-signs); arbiter-decided spend once
      `Resolve` reports a decision from the single confirmed arbiter

### Larger-bond tier: independent upfront arbiter commitments
- [ ] Define and validate the `net.persona.core.arbiter_commitment`
      claim_type (reuses A001's `ValidateClaimType` — no new
      namespacing code needed), published by each side at their own
      funding time, immutable once published
- [ ] On challenge, cooperatively construct (via PSBT) a fresh joint
      output naming both sides' already-committed arbiters, 2-of-2 —
      the same cooperative-escalation shape as the small-bond tier,
      just with the arbiter identities pre-fixed rather than
      negotiated at escalation time, and no direct 2-of-2
      mutual-settlement path back to the disputants themselves
- [ ] Settlement: spend the escalated 2-of-2 output once both
      pre-committed arbiters agree on a split
- [ ] Tie-break: given the two appointed arbiters disagree, construct
      (via PSBT, cooperatively between just the two arbiters) the
      escalation transaction moving the joint output into a fresh
      2-of-3 output naming both original arbiters plus a
      jointly-nominated third; settle once any 2-of-3 agree

### Settlement (shared)
- [ ] Given `dispute.IsSelfReleased` = true (never escalated, either
      tier), the pre-escalation single-sig spend is an ordinary
      timelocked self-spend — no new code beyond the shared
      pre-escalation output itself

### Tests
- [ ] Pre-escalation output: owner-alone spend succeeds only after
      `window_days`, fails before it — unit tests against `btcd`'s
      regtest/simnet tooling (no real funds)
- [ ] Small-bond tier: 2-of-2 mutual settlement succeeds with both
      signatures at any time; single-arbiter fallback succeeds only
      once `ConfirmedPanel`/`Resolve` actually agree; a non-cooperating
      party blocks escalation entirely (confirms the residual risk is
      real and structural, not just theoretical)
- [ ] Larger-bond tier: `arbiter_commitment` claim construction/
      verification round-trip; the escalated 2-of-2 output spends only
      with *both* pre-committed arbiters' signatures — a lone appointed
      arbiter cannot move it alone (confirms the collusion concern the
      corrected design exists to close); a non-cooperating disputant
      blocks escalation entirely, same residual as the small-bond tier
- [ ] Larger-bond tier tie-break: the two appointed arbiters
      disagreeing blocks direct settlement of the escalated output; the
      further escalation to a 2-of-3 with a third arbiter succeeds only
      with both original arbiters' cooperation; once escalated, any 2
      of the 3 (in any combination) can settle — confirming no fourth
      arbiter is ever structurally needed; the two arbiters refusing to
      escalate at all leaves each side's pre-escalation timelock
      fallback as the only path
- [ ] Funding-verification tests against a regtest node: unfunded,
      underfunded, and correctly-funded cases

### Before mainnet
- [ ] Independent security review of every script variant across both
      tiers (not self-certified) — hard gate, not a Task to check off
      solo

## Log

2026-09-29 — Action filed as the deliberately-deferred follow-up
[[A005]] named throughout its own filing and implementation: actual
Bitcoin escrow script construction, funding, and on-chain verification
for D001's bonding/slashing mechanism. Unlike A001-A005, filing this
one surfaced a real structural problem rather than just implementation
choices: D001/A005's "arbiter panel chosen per-dispute" decision is in
real tension with a Bitcoin script needing to name its spending
conditions before anyone knows who the arbiters will be. Documented
two candidate directions (pre-designated mediator vs. voluntary
escalation into joint custody) and their respective tensions, without
picking one — that decision needs its own focused pass, not a
guess made while filing. Made only the implementation-format choices
that don't depend on resolving it (PSBT, output descriptors,
regtest/testnet-first with a hard mainnet security-review gate).

2026-09-29 — Decided: voluntary escalation into joint custody, not a
pre-designated mediator. The mediator alternative was rejected on
principle, not just inconvenience — it would require the attester to
unilaterally pick their own judge before a challenger exists, the
exact single-point-of-capture failure D001's relative-trust model
exists to avoid, not a neutral implementation tradeoff. Escalation
also turned out to improve the action's own risk profile beyond just
resolving the selection problem: pre-escalation, both a bond and a
challenge stake are pure single-party timelocked self-custody — a bug
there can at worst delay someone reclaiming their own money, never let
the other party take it. The dangerous multisig code only gets built
and exercised on actual disputes, not on every bond. Residual risk
stated honestly: an attester can refuse to escalate and just wait out
their own timelock — no script can force cooperation without
reintroducing the pre-committed-authority problem the rejected
alternative had. Resolved reputationally, not cryptographically: a
verifier treats an unescalated, challenged bond as at least as bad as
losing the dispute, the same honest technical-limit-to-social-
enforcement move D001 already made for GDPR erasure. Noted one
operational consequence: `window_days` must be set comfortably longer
than realistic panel-confirmation-plus-escalation time, or the
requirement becomes impractical rather than just inconvenient. Tasks
rewritten to reflect the two-stage (pre-escalation self-custody,
then escalated joint-custody) structure this decision implies.

2026-09-29 — Refined into a two-tier structure, matching D001's own
small/larger-bond distinction rather than one mechanism for both.
Small bonds: kept the cooperative-escalation model above, but added a
2-of-2 mutual-settlement path as the *first* resort within the
escalated output, available before any arbiter is even involved — most
disputes between two people with real money at stake resolve by direct
agreement once forced to the table, and D001 itself treats a single
mutually-agreed arbiter as sufficient for this tier, so third-party
adjudication should be the fallback, not the default. Larger bonds:
the cooperative model isn't actually sufficient here, since whoever
expects to lose an arbitration can just refuse to cooperate with any
escalation step, defeating a "requirement." Resolved by having each
side independently pre-commit their *own* arbiter key into their
*own* output at funding time (attester when bonding, challenger when
challenging) — this gives either side a real unilateral right to
invoke arbitration on their own funds, without depending on the other
side's cooperation, while still not reintroducing the rejected
single-shared-judge pattern (each side only controls their own
arbitration branch; a real settlement needs both appointed arbiters to
agree). Left one thing genuinely open rather than guessed: what
happens when the two independently-appointed arbiters disagree — a
tie-breaking mechanism deferred to its own decision before that code
gets written. Tasks rewritten around both tiers.

2026-09-29 — Resolved the larger-bond tier's tie-breaking question:
the two appointed arbiters jointly escalate to a third, using the
exact same joint signing power their arbitration branch already grants
them, rather than pre-committing a third key at funding time (which
would just reopen the chicken-and-egg problem this tier already solved
once). Cooperatively constructing an escalation transaction is a much
lower-friction ask for two professionally-incentivized arbiters than
it was for the original disputants, the same "agreeing to let someone
decide beats agreeing on the decision" move the small-bond tier
already uses, applied one level up. Confirmed this terminates in
exactly one step and never recurses to a fourth or fifth arbiter:
verdicts are binary, so once a third arbiter casts any vote at all
they necessarily side with one of the original two, reaching 2-of-3
immediately — three binary voters can't produce a fresh tie once all
three have weighed in. The only way this stalls is the two original
arbiters refusing to even agree to escalate, which isn't a repeating
tie but a cooperation failure — already covered by the same
pre-escalation timelock fallback as every other non-cooperation case
in this plan, so no new residual risk was introduced, just extended
the existing one. Rejected pre-committing a third key upfront and
per-side small panels (neither actually eliminates ties structurally).
Tasks and Tests updated to reflect the concrete escalation mechanism.

2026-09-29 — Corrected a real bug in the larger-bond tier's design,
caught while actually implementing the pre-escalation output rather
than left to surface later. The committed decision claimed each
side's output "already commits to an arbitration path from the start"
at funding time — false: a Bitcoin script's keys are fixed the moment
an output is created and can never be added later, and the attester's
bond is created before any challenger (or their arbiter) exists, so it
structurally cannot name a key that doesn't exist yet — the same
chicken-and-egg problem this action started from, one level down.
Checked the obvious patch (let each side's own arbiter move that
side's output alone) and rejected it too: it's insecure, since an
owner and their own hand-picked arbiter could then collude to return
funds to the owner regardless of the actual dispute, exactly what
requiring both arbiters to agree exists to prevent. Split the decision
honestly into what's actually achievable: pre-committed, immutable
arbiter *identity* (published via a new `net.persona.core.
arbiter_commitment` claim, an addition to A005's protocol layer, not
a change to A005's already-shipped claim types) needs no cooperation
and is achievable upfront; actually escalating into joint arbiter
custody still needs one cooperative step, the same shape and the same
residual non-cooperation risk as the small-bond tier's escalation —
Bitcoin's own constraints make this unavoidable, no clever scripting
removes it. Restated honestly what this tier actually buys: once
escalated, resolution sits entirely with the two pre-committed
arbiters with no path back to the disputants, a decisive process and
an arbiter choice that can't be haggled over later — not "zero
cooperation ever required," which was the overclaim. Tasks and Tests
updated to match; the shared pre-escalation output section already
described the correct (single-path, identical-across-tiers) script and
needed no change.
