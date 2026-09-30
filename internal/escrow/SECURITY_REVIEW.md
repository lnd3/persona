# Security review package — `internal/escrow`

Assembled for **A007**, `plan/actions/A007-bitcoin-escrow-security-review.md`
in this repo's own planning system — the independent security review
**A006**'s own Tasks list (`plan/actions/A006-bitcoin-escrow-settlement.md`)
has always required before any of this code touches a real key
holding real mainnet value.

**If you are the reviewer**: thank you for looking at this. Everything
you need should be in this document and the files it points to. Please
report findings against the pass/fail criteria in "What a pass means"
below, ranked by severity, with a file/line reference and a concrete
failure scenario for each — not just "this looks off."

## Scope

- **In scope**: `internal/escrow` at commit `33dc3d44078567009e780e2d7b6e64507ce61bf4`
  (branch `main`) — every `.go` file in this directory, both
  implementation and tests. (This review package itself, and a couple
  of stale doc-comment fixes with no logic changes, were committed
  after that hash — run `git log --oneline 33dc3d4..HEAD --
  internal/escrow` to see exactly what, if anything, has changed
  since.) The implementation files:
  - `timelock.go` — BIP68 time-based CSV sequence encoding, the
    reusable single-key CSV+CHECKSIG script fragment, P2WSH
    scriptPubKey/address derivation
  - `multisig.go` — bare M-of-N `OP_CHECKMULTISIG` script builder, plus
    a general-purpose (currently unused by the two main scripts) 2-of-3
    tie-break script
  - `upfront.go` — **the two scripts that actually matter**:
    `UniversalScript` (4 branches) and `ReinforcedScript` (5 branches)
  - `settle.go` — PSBT-based settlement transaction construction,
    signing, and hand-written finalization (the standard library's own
    `psbt.Finalize` cannot recognize either of this package's script
    shapes as plain multisig)
  - `arbiter_commitment.go` — a Nostr claim type (reused primitive,
    not new crypto) for a standing arbiter designation
  - `descriptor.go` — `EscrowRef` string encodings
  - `verify.go` / `rpcquerier.go` — on-chain funding verification, one
    abstract interface plus one real implementation backed by a
    btcd/bitcoind JSON-RPC connection
- **Out of scope**: A005's protocol/claim layer
  (`plan/actions/A005-bonding-slashing-dispute-resolution.md`,
  `internal/dispute` — the claim types, panel confirmation, and
  verdict-resolution logic this package's scripts ultimately act on),
  the Nostr/attestation layer generally, and any not-yet-written
  wallet/key-management/UI code — this package builds and validates
  scripts and transactions, it does not hold anyone's private keys in
  a real deployment.

## What this package actually does, in one paragraph

D001/A005 scope the bonding/slashing mechanism to disputes between two
*identified* parties, meaning the attestation being bonded already
names its own `subject_key` — so the counterparty in a dispute is
never actually an unknown future party, only the arbiter is. Given
that, `UniversalScript` builds a single, static Bitcoin script, fully
determined at bond-creation time, with four spending paths: (1)
attester and subject cosign a mutual settlement, any split, any time,
no timelock; (2) the attester's own pre-designated arbiter cosigns
with the attester; (3) that arbiter cosigns with the subject instead;
(4) failing all of that, the attester alone reclaims after a CSV
timelock. `ReinforcedScript` adds a fifth, stronger path — both sides'
*independently* pre-committed arbiters agreeing with each other
directly, no disputant signature needed at all — available only when
the subject has separately published their own standing arbiter
commitment. Neither script involves an escalation transaction; nothing
about them changes after the funding output is created.

## Why this needed its own design pass (read before assuming the shape is arbitrary)

An earlier design used a two-step "escalate into joint custody" model.
Building the PSBT construction helpers for it surfaced a real,
disqualifying bug: its CSV timelock blocked *any* spend of the
pre-escalation output before the window elapsed — including a
legitimate cooperative escalation — and even once the window did
elapse, the script only ever checked the owner's own signature,
tying no spend to any actual dispute outcome. The bond provided no
real enforcement in either tier. That full history, including a
seriously considered and rejected move to a smart-contract chain
(concretely researched: Hyperliquid's HyperEVM, BTC-bridging trust
models, real bridge-exploit loss figures), is preserved verbatim in
A006's own Log (`plan/actions/A006-bitcoin-escrow-settlement.md`) —
worth reading if "why not just use a contract here" is your first
reaction; it was ours too, and here is the reasoning for staying
Bitcoin-native.

## Test coverage — what's actually been verified, and how

Three distinct levels of rigor were used, each proving something
different. Read the specific test names, not just the pass/fail
count — that's what tells you what's actually been exercised.

### 1. Consensus-engine-level (`engine_test.go`, `settle_test.go`)

Every script is validated by actually signing and executing spends
against `btcd`'s own `txscript.Engine` — the identical code a real
node uses to validate a block. This runs with no live node and proves
the *scripts themselves* are correct Bitcoin Script, independent of
transaction/PSBT plumbing:

- `TestPreEscalationScript_OwnerSpendsAfterWindow` /
  `_FailsBeforeWindow` / `_WrongSignerFails` — the shared CSV fallback
  fragment
- `TestUniversalScript_AllFourBranches` — all four branches succeed
  with correct signers; explicitly confirms the arbiter alone can't
  satisfy anything, and the attester alone can't take the
  mutual-settlement branch
- `TestReinforcedScript_AllFiveBranches` — all five branches, plus a
  cross-paired attempt (attester + the *other* side's arbiter) failing
- `TestTieBreakScript_AnyTwoOfThreeSettle` — the general-purpose
  building block, not currently used by either main script
- `TestSettlement_*` — full PSBT round trips (build → independently
  sign → finalize → extract → **re-validate against the real engine**)
  for the mutual-settlement, arbiter-sides-with-subject, and fallback
  branches of `UniversalScript`, plus `ReinforcedScript`'s
  both-arbiters-agree branch, plus a not-yet-finalizable case

### 2. Real regtest-node-level (`regtest_test.go`)

All **7 non-fallback branches across both scripts** were genuinely
funded from a real matured coinbase, broadcast, accepted by a real
node's mempool, and confirmed — not simulated:

- `TestRegtest_UniversalScriptBranches` — all 3 of `UniversalScript`'s
  cooperative branches (mutual settlement, arbiter-sides-with-attester,
  arbiter-sides-with-subject)
- `TestRegtest_ReinforcedScriptBranches` — all 4 of `ReinforcedScript`'s
  non-fallback branches (mutual settlement, both-arbiters-agree,
  attester+own-arbiter, subject+own-arbiter)

These require a real `btcd`/`bitcoind` regtest node reachable at
`127.0.0.1:18443` and skip gracefully otherwise (see `regtest_test.go`'s
own doc comment for exact setup — `btcd`'s full daemon source is
already in this project's Go module cache as a transitive dependency
of `txscript`/`psbt`, so `go install github.com/btcsuite/btcd@v0.24.2`
builds a working node directly, no new binary dependency). **A live
node was stood up and these tests were run and passed during this
package's own development** (see A006's own Log,
`plan/actions/A006-bitcoin-escrow-settlement.md`, for the exact date).
It is not a standing fixture and may not be running when you read
this — stand it up again to re-verify, or take the recorded pass on
faith pending your own re-run.

### 3. Not covered — a stated boundary, not a silent gap

**Neither script's CSV fallback branch has been exercised against a
real node's actual elapsed time.** BIP68's time-based relative lock
needs real median-time-past to advance the requested window (14 days
in every test here); `btcd` has no `setmocktime` RPC, and waiting out
a real multi-day window isn't practical in a test. The specific
mechanism this would additionally prove — comparing the script's
required sequence against the spending input's actual sequence — is
already exercised at the consensus-engine level
(`TestPreEscalationScript_*`), the same code path a real node uses
internally to reject an early spend. **If you have a way to test this
more directly (a testnet with patience, a modified node build with
mocktime, or a from-first-principles argument that the engine-level
test already covers what matters), that would close the one deliberate
gap in this package's own test coverage.**

## Known, already-accepted design residuals

Stated here so you don't have to rediscover them independently —
these are accepted tradeoffs with reasoning already on record in
A006 (`plan/actions/A006-bitcoin-escrow-settlement.md`), not things
this review needs to re-litigate unless you disagree with the
reasoning itself:

- **Non-cooperation is resolved reputationally, not cryptographically.**
  Neither script can force any party to sign. If nobody ever engages,
  the attester's own CSV fallback lets them reclaim their own bond
  unilaterally after the window — by design, this is meant to be
  treated by verifiers as at least as bad as losing the dispute
  outright, not prevented at the script level.
- **`ReinforcedScript`'s two independently-appointed arbiters can
  disagree with no tie-break.** If attester's arbiter sides with the
  attester (branch 3) while subject's arbiter separately sides with
  the subject (branch 4), both are independently valid spends of the
  same UTXO — whichever gets mined first wins, a double-spend race,
  not a clean resolution. Accepted as rare (requires the two arbiters
  to genuinely disagree) rather than solved with a baked-in
  tie-breaker, since no third party can be pre-named any more easily
  than the original arbiter chicken-and-egg problem allowed.
- **`EscrowRef` for these scripts is plain hex-encoded witness-script
  bytes**, not a compiled descriptor — deliberately, since these
  scripts are genuinely custom (`OP_IF`-branching) and a real
  miniscript compiler is out of scope. Anyone can independently
  re-derive the P2WSH address from it via `WitnessScriptHash`, but it
  isn't human-readable.

## What a pass means (from A007's own stated criteria)

No finding that allows:

1. A party to move funds without a valid signature actually matching
   the intended branch
2. Any branch to be satisfied with fewer real signatures than its
   design requires
3. The fallback branch to be spendable before its CSV window has
   genuinely elapsed on a real chain

Findings outside this bar (style, gas/fee efficiency, defense-in-depth
suggestions beyond the stated scope) are welcome and will be tracked,
but don't block a pass on their own.

## How to actually run everything yourself

```sh
# Consensus-engine-level tests (no live node needed):
go test ./internal/escrow/... -run 'TestPreEscalationScript|TestUniversalScript|TestReinforcedScript|TestTieBreakScript|TestSettlement' -v

# Everything else (claim/descriptor/verify unit tests, no live node needed):
go test ./internal/escrow/... -short -v

# Live regtest tests (needs a real node — see regtest_test.go's own
# doc comment for exact setup):
go test ./internal/escrow/... -run TestRegtest -v
```
