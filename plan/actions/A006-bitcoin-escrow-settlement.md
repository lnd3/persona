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

**This action starts from a harder position than A001-A005 did**:
those actions each had a design (D001) that named the mechanism and
left only implementation-level choices open (which library, which
kind number, which claim_value shape). This one's starting design —
"a simple n-of-m multisig" — turns out, on closer inspection while
filing this action, to have a real unresolved structural problem
underneath it, not just missing implementation detail. See Open
Design Questions below before assuming this is a straightforward
build.

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

## Open Design Questions (not yet resolved — read before implementing)

Filing this action surfaced a real structural tension D001's "simple
n-of-m multisig" framing glossed over, honestly flagged here rather
than papered over with a premature Decision:

- **The arbiter panel is chosen *after* a dispute arises (per D001 and
  A005's `ConfirmedPanel`), but a spendable Bitcoin script has to be
  constructed *before* anyone knows who will spend from it.** A bond
  is created and sits on-chain, potentially for a long time, before
  any challenge exists — so the escrow output funded at bond-creation
  time cannot name arbiter keys that don't exist yet. Two real
  directions, neither obviously right:
  1. **Pre-designated mediator, chosen at bond-creation time** — the
     well-trodden pattern real Bitcoin escrow services already use
     (2-of-3: payer, payee, a pre-agreed mediator key), simple and
     provably spendable from day one. **Tension**: this contradicts
     D001/A005's own "arbiter choice is per-dispute, jointly selected
     by attester and challenger" decision — the mediator would have to
     be chosen unilaterally by the attester alone, before a challenger
     even exists to jointly agree on anyone.
  2. **Voluntary escalation into joint custody upon challenge** — the
     bond starts as a plain timelocked output (attester spends alone
     after the window, no multisig needed since there's no
     counterparty yet); when challenged, the attester must
     cooperatively move their bonded funds into a new, freshly
     constructed n-of-m output naming the just-confirmed panel,
     *before* their original timelock expires. **Tension**: this is a
     required voluntary action, not something a script alone enforces
     — an attester who refuses to cooperate could simply wait out the
     original timelock and reclaim their bond unilaterally, defeating
     the entire point of bonding. Would need a real consequence for
     non-cooperation (e.g. verifiers treating an unescalated,
     challenged bond as automatic forfeit — itself a new policy
     decision, not free).
  - **Not resolved here.** Whichever direction is chosen changes what
    "funding" and "the escrow script" even mean for this action's
    other tasks — this needs its own focused decision (with the same
    honesty D001 applies to its other hard tradeoffs) before any
    script-construction code is written, not assumed away by filing
    this action.
- **Testnet-only until a real security review happens.** Given the
  fund-loss stakes, this action's own Tasks list below treats "get it
  reviewed before it ever touches mainnet sats" as a hard gate, not a
  nice-to-have — consistent with how seriously `cinder`'s own D005
  treated Lightning infrastructure risk, but a strictly higher bar
  here since a script bug is unrecoverable in a way a Lightning
  provider outage isn't.

## Decisions made here (the few not blocked on the open question above)

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

### Resolve the open design question first
- [ ] Decide between pre-designated-mediator and voluntary-escalation
      (or a third option this action's own research turns up) —
      **this must happen before any of the tasks below**, the same
      "decisions before implementation" discipline A001 applied to its
      own language/kind-number choices
- [ ] Document the chosen direction's own honestly-stated residual
      risk (e.g. non-cooperation griefing, or upfront-mediator
      centralization) — D001's own standard for every hard tradeoff in
      this design

### Script construction (shape depends on the resolved question above)
- [ ] Construct the escrow output script for the chosen direction
- [ ] Build the self-release spending path (attester alone, after
      `window_days`, when never escalated/never disputed)
- [ ] Build the verdict-settlement spending path (confirmed panel
      pays the `Resolve`-determined winner)

### Funding and verification
- [ ] Generate a funding address/descriptor for a new bond or
      challenge stake, to populate `EscrowRef`
- [ ] Verify a referenced escrow is actually funded on-chain to the
      claimed `amount_sats` (watching for confirmation, not just an
      unconfirmed mempool entry)

### Settlement
- [ ] Given `dispute.IsSelfReleased` = true, construct and (on
      regtest/testnet) broadcast the self-release spend
- [ ] Given `dispute.Resolve` = decided, construct and (on
      regtest/testnet) broadcast the verdict-settlement spend

### Tests
- [ ] Script construction unit tests against `btcd`'s regtest/simnet
      tooling (no real funds) for both spending paths
- [ ] Funding-verification tests against a regtest node: unfunded,
      underfunded, and correctly-funded cases
- [ ] Settlement tests: self-release spend succeeds only after the
      window with no escalation; verdict spend succeeds only with a
      genuinely decided `Resolve` result and the confirmed panel's
      actual signatures

### Before mainnet
- [ ] Independent security review of the chosen script (not
      self-certified) — hard gate, not a Task to check off solo

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
