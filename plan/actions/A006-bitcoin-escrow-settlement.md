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

- **Voluntary escalation into joint custody upon challenge, not a
  pre-designated mediator — decided 2026-09-29.** Filing this action
  surfaced a real structural tension D001's "simple n-of-m multisig"
  framing glossed over: the arbiter panel is chosen *after* a dispute
  arises (per D001 and A005's `ConfirmedPanel`), but a spendable
  Bitcoin script has to be constructed *before* anyone knows who will
  spend from it — a bond sits on-chain, potentially for a long time,
  before any challenge (and therefore any panel) exists.
  - **The pre-designated-mediator alternative was rejected, not just
    set aside.** The well-trodden real-world pattern (2-of-3: payer,
    payee, a pre-agreed mediator key) requires naming the mediator at
    bond-creation time — but the challenger doesn't exist yet then, so
    the attester alone would be choosing who gets to judge disputes
    against themselves. That's not a minor limitation to route around;
    it's a direct regression against D001's own founding principle
    (never let one party unilaterally pick the authority that
    adjudicates them) — the exact single-point-of-capture failure the
    whole relative-trust model exists to avoid. Rejected on principle,
    not just on inconvenience.
  - **How escalation actually works**: a bond starts as a plain
    timelocked output — attester spends alone after `window_days`, no
    multisig at all, since there's no counterparty yet. Symmetrically,
    a challenger's stake is *also* just their own timelocked
    self-custody output from the moment they challenge — not
    automatically joint with anything. Only once a panel is
    **confirmed** (`ConfirmedPanel`, both sides agreeing) do the
    attester and challenger cooperatively construct and fund a new
    n-of-m output naming that panel, moving both the bond and the
    stake into it before the attester's original timelock expires.
  - **This meaningfully improves the risk profile of the whole
    action, not just resolves the selection problem.** Before any
    escalation, both parties' funds sit in pure single-party
    self-custody — a bug there can at worst delay someone reclaiming
    their *own* money, never let the other party (or anyone else) take
    it. The genuinely dangerous multisig code — where a bug really
    could misdirect someone's funds — only gets constructed and
    exercised on actual disputes, not on every bond ever created. A
    materially smaller, later-triggered, more reviewable surface than
    "every bond has a live multisig from day one."
  - **Residual risk, stated honestly rather than glossed over**: an
    attester can refuse to cooperate with escalation and simply wait
    out their own timelock to reclaim their bond unilaterally — no
    script can force cooperation, since forcing it would require
    exactly the pre-committed keys the rejected alternative needed.
    This is **not resolved cryptographically, only reputationally**:
    a verifier checking a subject's bond status treats "challenged,
    never escalated within a reasonable window" as equivalent to (or
    worse than) losing the dispute outright, per D001's own
    relative-trust posture — the same move D001 already made honestly
    for GDPR erasure (crypto-shredding isn't literal erasure, but the
    best achievable given the real constraint, stated openly rather
    than marketed around). The challenger's own stake is never at risk
    from this defection — it just self-returns to them too, since it
    was never actually joint until escalation completed.
  - **Operational consequence**: `window_days` (A005's own per-bond
    field) must be set comfortably longer than the realistic
    real-world time to confirm a panel and construct a joint
    escalation transaction — otherwise "escalate before the timelock
    expires" becomes impractical rather than merely inconvenient. Not
    a new claim_value field; a deployment/product guidance note for
    whatever sets `window_days` when a bond is created.
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

### Pre-escalation output (single-party self-custody)
- [ ] Construct a plain CSV-timelocked single-sig output script:
      owner spends alone after `window_days` — used identically for
      both a bond (attester as owner) and a challenge stake
      (challenger as owner), per the Decision above
- [ ] Generate a funding address/descriptor for a new bond or
      challenge stake, to populate `EscrowRef` (BIP380 descriptor, per
      Decisions)
- [ ] Verify a referenced escrow is actually funded on-chain to the
      claimed `amount_sats` (watching for confirmation, not just an
      unconfirmed mempool entry)

### Escalation (joint custody, only once a panel is confirmed)
- [ ] Given a `ConfirmedPanel` result, cooperatively construct (via
      PSBT) a new n-of-m output naming the confirmed panel, spending
      both the bond's and the stake's pre-escalation outputs into it —
      requires both attester's and challenger's signatures, since
      each is spending their *own* pre-escalation output
- [ ] Detect non-escalation: a challenge exists, the panel is
      confirmed, but no escalation transaction has been broadcast
      within a reasonable margin of `window_days` — surfaced as data
      for a verifier's own reputational judgment (per the Decision's
      residual-risk handling), not enforced in-code

### Settlement (from the escalated joint-custody output only)
- [ ] Given `dispute.IsSelfReleased` = true (never escalated), the
      pre-escalation single-sig spend is just an ordinary timelocked
      self-spend — no new code beyond the pre-escalation output itself
- [ ] Given `dispute.Resolve` = decided, construct and (on
      regtest/testnet) broadcast the verdict-settlement spend from the
      escalated joint-custody output, paying the `Resolve`-determined
      winner, signed by the panel's own confirmed majority

### Tests
- [ ] Pre-escalation output: owner-alone spend succeeds only after
      `window_days`, fails before it — unit tests against `btcd`'s
      regtest/simnet tooling (no real funds)
- [ ] Escalation: cooperative n-of-m construction succeeds only with
      both original owners' signatures; a non-cooperating party blocks
      it (confirms the residual risk is real and structural, not just
      theoretical)
- [ ] Funding-verification tests against a regtest node: unfunded,
      underfunded, and correctly-funded cases
- [ ] Settlement tests: self-release spend succeeds only after the
      window when never escalated; verdict spend succeeds only with a
      genuinely decided `Resolve` result and the confirmed panel's
      actual majority signatures, from the escalated output only

### Before mainnet
- [ ] Independent security review of both the pre-escalation and
      escalated scripts (not self-certified) — hard gate, not a Task
      to check off solo

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
