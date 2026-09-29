---
id: A005
title: Bonding/slashing dispute resolution (dispute type 1 only)
status: DONE
design: D001
project: P001
created: 2026-09-29
updated: 2026-09-29
---

## Context

Implements D001's Bonding/slashing mechanism, scoped exactly as D001
scopes it: **dispute type 1 only** (behavioral/quality disputes
between two identified parties — "this persona scammed me"/"didn't
deliver as claimed"). D001 explicitly rules the other three dispute
types out of this mechanism entirely (sybil-ring vouching needs
graph-level heuristics; ownership/provenance is independently
checkable and doesn't need arbitration; recovery-guardian disputes
need a pre-finalization challenge window, not post-hoc slashing) — this
action doesn't reopen any of that, it only builds the one mechanism
D001 actually specified.

Builds directly on [[A001]]'s `internal/attestation` (`New`/`Verify`,
`claim_type` namespacing — reused for four new claim types below) and
follows the same claim-type-based precedent [[A003]] set for recovery
("guardianship is an attestation, not a separate system") — a bond, a
challenge, an arbiter panel, and a verdict are each just attestations
too, not a fifth persona-specific subsystem.

**Explicitly out of scope for this action — the protocol/decision
layer only, not the money-movement layer**:
- **Actual Bitcoin escrow script construction, funding, or broadcast.**
  D001 specifies "a simple n-of-m multisig" as the financial primitive
  but doesn't design the script itself, and getting a
  timelock+multisig spending script wrong is a fund-loss bug, not a
  logic bug — a materially different, higher-stakes piece of work than
  anything else built so far in this project. This action defines
  *who should be able to spend the escrow and when* (self-release
  timing, verdict-based winner) as pure data/logic; a follow-up action
  builds and audits the actual script. Same "defer the harder
  cross-domain slice" discipline [[A004]] applied to Lightning-payment
  execution and [[A003]] applied to FROST.
- **Verifying that a referenced escrow is actually funded on-chain.**
  This action's claim types *reference* an escrow (an address/
  descriptor and an amount), the same way [[A004]]'s payment receipts
  reference a hash rather than re-verifying a Lightning payment
  in-band — checking the chain is the not-yet-filed escrow-settlement
  action's job.
- **Off-protocol negotiation of who to pick as arbiter(s).** D001
  states arbiter choice is per-dispute, decided by whoever relies on
  the bond — this action represents the *result* of that choice as
  public data (so a verdict can be checked against who was actually
  authorized to arbitrate) but doesn't implement any
  discovery/negotiation flow, same posture [[A003]] took toward
  guardian contact.
- **Any UI/notification surface** for a subject learning they've been
  challenged, or an arbiter learning they've been asked to rule.

## Decisions made here (not in D001 — concrete choices this action owns)

- **Four new claim types, all `net.persona.core.*`**, each referencing
  a specific already-published attestation event by its Nostr `id`
  (no circularity concern here, unlike [[A004]]'s payment receipts —
  a dispute always concerns an *already-published* event with a fixed
  id, not content still being assembled):
  - `net.persona.core.bond` — attester = the original attestation's
    own attester, subject = themself (self-referential; the bond
    secures their own claim), `claim_value` =
    `"<disputed_event_id>:<escrow_ref>:<amount_sats>:<window_days>"`.
    Published alongside (not instead of) the attestation it bonds.
  - `net.persona.core.dispute_challenge` — attester = the challenger,
    subject = the disputed attestation's attester, `claim_value` =
    `"<disputed_event_id>:<challenger_escrow_ref>:<amount_sats>"` —
    the challenger's symmetric stake, per D001's anti-griefing
    requirement.
  - `net.persona.core.arbiter_panel` — attester = either the original
    attester or the challenger (each publishes their own copy),
    subject = the disputed attestation's attester, `claim_value` =
    `"<disputed_event_id>:<comma-separated arbiter pubkeys>"`. A panel
    is only confirmed once **both** sides have published a matching
    claim naming the identical panel for the identical dispute — the
    joint-selection requirement from D001, represented as two
    independent attestations that must agree, the same
    quorum-matching shape [[A003]] uses for guardian confirmation, not
    a new co-signing primitive.
  - `net.persona.core.dispute_verdict` — attester = one arbiter,
    subject = the disputed attestation's attester, `claim_value` =
    `"<disputed_event_id>:attester_wins"` or `"...:challenger_wins"`.
    One verdict claim per arbiter; resolution counts them (see below).
- **Panel size and quorum, not left fully open**: a panel is either
  exactly **one** arbiter (small bonds, per D001) or an **odd number
  ≥ 3** (D001's "panel... for larger bonds"), rejecting even sizes at
  construction — an even panel can tie, which D001 never addresses and
  this action isn't the place to invent tie-breaking rules for. What
  counts as "small" vs. "larger" (a specific sats threshold) is a
  policy call D001 leaves as a judgment call too, so it's left to the
  caller here, not hardcoded — this action takes the panel as given
  and only enforces the odd-size-or-one shape.
- **Verdict resolution is majority-of-panel, not first-verdict-wins**:
  `Resolve` counts one verdict per distinct arbiter pubkey (duplicates
  from the same arbiter don't count twice — mirrors [[A002]]'s
  distinct-attester counting for trust edges) and only returns a
  winner once strictly more than half the confirmed panel has voted
  the same way; otherwise reports "undecided."
- **Self-release timing is pure arithmetic, not an on-chain check**:
  `IsSelfReleased(bond, now, challenges)` returns true only if
  `now > bond.timestamp + window_days` **and** no
  `dispute_challenge` referencing the same `disputed_event_id` exists
  — computing "was this bond ever actually challenged," not "has the
  money actually moved," which stays out of scope per above.
- **Default self-release window: 14 days** — a judgment call, not
  derived from anything (D001 says "a fixed window" without a number);
  chosen as a starting default, encoded per-bond in `claim_value`
  rather than hardcoded globally, so it's revisitable per-attester
  without a protocol change.
- **`escrow_ref` parsing found and closed at implementation time**: a
  colon-delimited `claim_value` breaks if `escrow_ref` itself contains
  a colon — plausible, since Bitcoin output descriptors can (e.g. key
  origin info like `[fingerprint/path]xpub...`). Fixed with an anchored
  regex whose middle capture is greedy (`(.+)`) while the *trailing*
  fields are constrained to digits-only (`amount_sats`, `window_days`)
  — this lets `escrow_ref` contain colons freely, since only the
  digit-only suffix pattern anchors where it ends. Not a design
  change, just closing a parsing gap the Decisions section's example
  format didn't anticipate.

## Tasks

### Claim types
- [x] Define and validate the four claim_type strings (reuses A001's
      `ValidateClaimType`) — `internal/dispute/claims.go`
- [x] `claim_value` parse/format helpers for each of the four shapes,
      with clear errors on malformed values — `New*Claim`/`Parse*Claim`
      pairs, anchored regexes with a greedy middle capture for
      `escrow_ref` (see Decisions correction)

### Panel and quorum
- [x] Construct/validate an arbiter panel: size 1, or odd ≥ 3 —
      `internal/dispute/panel.go` `ValidatePanelSize`
- [x] Detect a *confirmed* panel: matching `arbiter_panel` claims from
      both the original attester and the challenger for the same
      `disputed_event_id` — `ConfirmedPanel` (most-recent-by-timestamp
      per side, in case of redesignation)
- [x] `Resolve(verdicts, disputedEventID, confirmedPanel) (winner
      Verdict, decided bool, error)` — `internal/dispute/resolve.go`,
      distinct-arbiter counting, strict-majority threshold

### Self-release
- [x] `IsSelfReleased(bond Claim, now time.Time, challenges []Claim)
      (bool, error)` — `internal/dispute/release.go`
- [x] Parses the bond's `window_days` per-bond via `ParseBondClaim`,
      not a hardcoded constant

### Tests
- [x] Claim construction/verification round-trips for all four types
      — `TestBondClaimRoundTrip`, `TestChallengeClaimRoundTrip`,
      `TestPanelClaimRoundTrip`, `TestVerdictClaimRoundTrip`
- [x] Malformed `claim_value` rejected with a clear error, for each
      type — the four `Test*RejectsMalformedValue` tests, plus
      `TestBondClaimEscrowRefMayContainColons` for the colon-parsing
      correction
- [x] Panel construction rejects size-0 and even sizes ≥ 2 —
      `TestValidatePanelSize`, `TestNewPanelClaimRejectsInvalidSize`
- [x] Panel confirmed only when both sides' claims name the identical
      arbiter set for the identical dispute — `TestConfirmedPanel*`
      (both-agree, mismatched, one-sided)
- [x] `Resolve`: unanimous 1-arbiter panel decides immediately;
      3-arbiter panel needs 2 matching votes, not decided at 1-1;
      duplicate verdicts from the same arbiter don't count twice;
      a verdict from outside the confirmed panel doesn't count at all
      — `TestResolveSingleArbiterDecidesImmediately`,
      `TestResolveThreeArbiterPanelNeedsTwoVotes`,
      `TestResolveDuplicateVerdictFromSameArbiterDoesNotCountTwice`,
      `TestResolveIgnoresVerdictFromOutsidePanel`
- [x] `IsSelfReleased`: true after the window with no challenge, false
      before the window, false at any time once a matching challenge
      exists regardless of resolution — `TestIsSelfReleasedTrue*`,
      `TestIsSelfReleasedFalse*` (5 tests covering window timing,
      challenged/unresolved, and unrelated-event isolation)

## Log

2026-09-29 — Action filed against D001's Bonding/slashing mechanism,
scoped exactly to dispute type 1 as D001 itself scopes it. Split the
mechanism the same way [[A004]] split payment integration: this action
owns the protocol/decision layer (four new claim types, panel
confirmation via matching two-sided attestations, majority-vote
verdict resolution, self-release timing arithmetic) entirely in Go,
fully testable offline; actual Bitcoin escrow script construction,
funding, and on-chain verification are deliberately left to a
separate, not-yet-filed follow-up — getting a timelock+multisig
spending script wrong is a fund-loss bug, a different risk class than
anything else built in this project so far, and deserves its own
focused action rather than being bundled in here. Two judgment calls
made explicitly rather than left silently assumed: panel size must be
1 or odd-≥3 (D001 never addresses even-panel tie-breaking, so this
action refuses to invent one), and a 14-day self-release window as a
starting default, encoded per-bond so it's revisitable without a
protocol change.

2026-09-29 — Implemented. New `internal/dispute` package, four files:
`claims.go` (the four claim types, `New*Claim`/`Parse*Claim` pairs),
`panel.go` (`ValidatePanelSize`, `ConfirmedPanel`), `resolve.go`
(`Resolve`), `release.go` (`IsSelfReleased`). One real parsing gap
found and closed: the Decisions section's colon-delimited
`claim_value` format breaks if `escrow_ref` itself contains a colon —
plausible for Bitcoin descriptors (e.g. `[fingerprint/path]xpub...`
key-origin syntax). Fixed with anchored regexes whose middle capture
is greedy while the *trailing* fields are constrained to digits-only,
so `escrow_ref` can safely contain colons — confirmed directly with
`TestBondClaimEscrowRefMayContainColons`. No corrections needed
elsewhere; `ConfirmedPanel` additionally picks the most-recently-
timestamped claim per side in case of redesignation, a small
robustness detail beyond the letter of the original Tasks list. 24 new
tests, full repo build/vet/test clean. Everything this action scoped
is done — moved to DONE. **A001-A005 are now all DONE and
implemented** — the only work D001 still names as unbuilt is the
Bitcoin-escrow settlement follow-up this action deliberately deferred
(script construction/funding/on-chain verification), still unfiled.
