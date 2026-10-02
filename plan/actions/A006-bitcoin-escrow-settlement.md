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

**This section reflects the current, corrected design (2026-09-30). An
earlier escalation-based two-tier model went through several rounds of
real bugs found at implementation time; that full journey — including
the bug that ultimately motivated this section's redesign — is kept
verbatim in the Log below, not deleted, since it's the reasoning that
got here.**

- **Bitcoin-native, not a move to a smart-contract chain — decided
  2026-09-30, after real research, not asserted.** Before redesigning
  the script, the question was raised directly: doesn't this whole
  class of problem (a script needing to react to a party that doesn't
  exist yet) call for a smart-contract chain (Ethereum-style EVM,
  concretely evaluated: Hyperliquid's HyperEVM) instead of fighting
  Bitcoin Script's limits? Researched concretely rather than guessed:
  - Hyperliquid HyperEVM: 24 validators, BFT-tolerant to ~1/3
    malicious — a real, named tradeoff in its own documentation
    ("easier to coordinate, but also easier to influence, attack, or
    govern through a narrow group"). No confirmed core-protocol/bridge
    exploit yet, but real app-layer losses have already happened on
    top of it.
  - Getting BTC value onto *any* EVM chain (WBTC/custodial, HyperUnit/
    MPC-guardian, tBTC/large-threshold-set) always means trusting some
    off-chain signer group — never as trustless as a native UTXO.
    Bridge exploits are not hypothetical: >$2.8B stolen since 2022,
    >69% of all DeFi losses in that period (Ronin, Poly Network, BNB
    Bridge, Wormhole, Nomad — roughly split between key-compromise and
    contract-logic-bug root causes).
  - Kleros-style crowdsourced juror pools genuinely solve "unknown
    challenger at deploy time" elegantly — but that's *exactly* the
    global-crowdsourced-court model D001 already considered and
    rejected ("one canonical true/false verdict everyone inherits...
    the kind of single-point-of-capture authority this design's
    relative-trust principle exists to avoid"). Adopting it here would
    mean walking back a deliberate D001 decision, not swapping
    implementation platforms.
  - The Bitcoin-native 2-of-3-mediator pattern this design was already
    converging toward turns out to be a real, standardized (BIP-11),
    long-deployed pattern, whose one documented weakness — the
    mediator can simply refuse to act — is exactly the non-cooperation
    residual already accepted and resolved reputationally elsewhere in
    this design, not a new risk category.
  - **Conclusion**: a smart-contract chain's real advantage here is
    script *flexibility* (who can call what, when), not better
    arbitration of the dispute's substance — no chain's consensus,
    staked or not, can *compute* whether a scam actually happened; that
    still needs a human arbiter's judgment regardless of platform. The
    flexibility gap turned out to be closable on Bitcoin itself (below),
    at no bridge-custody or second-chain-trust cost.
- **The key insight that makes a single, fully upfront-configured
  Bitcoin script possible: the "unknown future party" isn't actually
  unknown for this dispute type.** D001/A005 scoped bonding/slashing to
  dispute type 1 only — "behavioral/quality disputes between two
  *identified* parties." The bond secures the attester's own
  attestation, and that attestation already names a `subject_key` —
  the very party who becomes the challenger if they disagree. Their
  pubkey is public the moment the attestation exists; it was never
  really a not-yet-existing stranger, the way an arbiter genuinely was.
  Only the arbiter piece is solved via a **standing**
  `net.persona.core.arbiter_commitment` claim — published once,
  independent of any specific bond, reusable across every future
  dispute where a persona is named as either attester or subject
  (self-referential, reuses A001's attestation primitive, same "X is
  just an attestation" precedent A003/A005 already set) — rather than
  a per-dispute selection that would reopen the original chicken-and-egg
  problem.
- **`UniversalScript`: a single, static, fully upfront-configured
  script — no escalation transaction, ever.** Always buildable, no
  dependency on the subject having published anything:
  ```
  IF        2-of-2 (attester, subject)          -- mutual settlement, any time
  ELSE IF   2-of-2 (attester, arbiter)           -- arbiter sides with attester
  ELSE IF   2-of-2 (subject, arbiter)            -- arbiter sides with subject
  ELSE      <sequence> CSV DROP attester CHECKSIG -- last resort: nobody engaged
  ```
  The mutual-settlement branch needs no timelock at all — attester and
  subject can settle directly, immediately, any time, which is what
  makes voluntary resolution actually possible *during* an active
  dispute (the superseded escalation design's single CSV-gated path
  blocked this entirely — see Log). The CSV-gated self-release is now a
  genuine last resort, reachable only if *nobody* — not subject, not
  arbiter — ever engaged at all; it no longer means "the dispute
  outcome is irrelevant once the clock runs out," which was the
  superseded design's real, structural flaw.
- **`ReinforcedScript`: `UniversalScript`'s branches plus one stronger
  branch, available only when the subject has independently published
  their own standing arbiter_commitment.**
  ```
  IF        2-of-2 (attester, subject)                     -- mutual settlement
  ELSE IF   2-of-2 (attesterArbiter, subjectArbiter)        -- both arbiters agree, no disputant needed
  ELSE IF   2-of-2 (attester, attesterArbiter)              -- arbiter sides with attester
  ELSE IF   2-of-2 (subject, subjectArbiter)                -- arbiter sides with subject
  ELSE      <sequence> CSV DROP attester CHECKSIG           -- last resort
  ```
  A real, bounded precondition, stated honestly rather than hidden: if
  the subject hasn't opted in with a standing commitment, only
  `UniversalScript` is buildable for a claim naming them — the stronger
  "neither disputant needed" property isn't available for free.
  - *Residual, stated honestly*: if the two independent arbiters
    disagree (attester's arbiter sides with attester via branch 3 while
    subject's arbiter separately sides with subject via branch 4),
    there is no further tie-break baked into this static script — under
    this design there's no live moment left to escalate to a third
    arbiter the way the superseded escalation design imagined.
    Whichever competing spend actually confirms first wins. Accepted as
    rare (both arbiters must genuinely disagree) rather than solved with
    more machinery, the same "an honestly-bounded limitation beats
    invented complexity" choice made elsewhere in this design (e.g.
    GDPR erasure).
- **PSBT (BIP174) for settlement construction — much smaller than
  originally scoped, since there's no escalation chain to construct
  anymore.** Settlement is always exactly one transaction spending the
  bond/stake output directly through whichever branch applies.
  `psbt.Finalize`'s own built-in multisig finalizer can't be reused
  as-is — it requires a script recognizable as a plain "M of N
  CHECKMULTISIG" (`checkIsMultiSigScript`), which neither this design's
  `OP_IF`-branching scripts nor the single-key CSV fallback branch are
  — so `FinalizeUniversal`/`FinalizeReinforced` (`settle.go`) do that
  assembly by hand, reusing the exact witness ordering already
  validated against the real consensus engine.
- **`EscrowRef` for these scripts is plain hex-encoded witness-script
  bytes, not a miniscript-shaped descriptor** — the earlier narrow
  BIP380-style descriptor (still kept, `descriptor.go`, for the
  single-key CSV shape alone) doesn't stretch to a genuinely custom,
  multi-branch script without a real compiler, which is out of scope.
  Unambiguous and exact (independently re-derivable via
  `WitnessScriptHash`), at the cost of not being human-readable.
- **Regtest/testnet only until a security review**, mainnet activation
  gated behind it explicitly, not left as an implicit "someday" — this
  is the fund-loss-risk piece of work D001 and A005 both flagged, and
  this action treats that flag as binding on itself, not just on its
  predecessors.

## Tasks

**This section reflects the current design (2026-09-30). The
superseded escalation-based two-tier model's task list — including the
non-escalation-detection and escalation-PSBT tasks it needed — no
longer applies; that history is kept in the Log, not here.**

### Scripts
- [x] `PreEscalationScript`/`SequenceForDays` — BIP68 time-based CSV
      fragment reused as the last-resort branch inside both
      `UniversalScript` and `ReinforcedScript` — `timelock.go`
- [x] `MultisigScript` — general M-of-N `OP_CHECKMULTISIG` builder,
      reused for every 2-of-2 branch — `multisig.go`
- [x] `UniversalScript` (4 branches) and `ReinforcedScript` (5
      branches) — `upfront.go`, every branch validated by actually
      signing and executing real spends (and confirming invalid
      cross-pairings fail) against `btcd`'s real `txscript` engine
- [x] `net.persona.core.arbiter_commitment` — standing, not
      per-bond, claim — `arbiter_commitment.go`, reuses A001's
      `ValidateClaimType`/`attestation.New`/`Verify` directly, additive
      to A005's protocol layer (not a change to A005's shipped claim
      types)
- [x] `TieBreakScript` (2-of-3) — kept as a general-purpose building
      block, though not one of `UniversalScript`/`ReinforcedScript`'s
      own branches under the current no-escalation design — `multisig.go`

### Funding and reference
- [x] `WitnessScriptHash`/`FundingAddress` — deriving the actual P2WSH
      scriptPubKey/bech32 address from any of this package's scripts —
      `timelock.go`
- [x] `EscrowRef` encoding: `FormatPreEscalationDescriptor` (narrow
      BIP380-style, single-key-CSV shape only) plus
      `FormatWitnessScriptRef`/`ParseWitnessScriptRef` (plain hex, for
      the genuinely custom multi-branch scripts) — `descriptor.go`
- [x] Verify a referenced escrow is actually funded on-chain —
      `verify.go` `VerifyFunded`, against a `ChainQuerier` interface.
      **Real backend now built**: `rpcquerier.go` `RPCQuerier`,
      backed by `btcd`'s own `rpcclient` package (already vendored as
      part of the same module — no new binary dependency), exercised
      against a real regtest `btcd` node built directly from that
      vendored source with `go install github.com/btcsuite/btcd@v0.24.2`
      — see the live regtest tests below

### Settlement (PSBT)
- [x] `NewSettlementPacket`/`SignSettlementInput` — build an unsigned
      PSBT for a single settlement transaction and let cooperating
      parties each attach their own partial signature independently —
      `settle.go`
- [x] `FinalizeUniversal`/`FinalizeReinforced` — hand-written
      finalizers (the standard library's own `psbt.Finalize` can't
      recognize these scripts as plain multisig), trying each branch
      in turn against whichever partial signatures are actually
      present, refusing the fallback branch unless the packet's own
      sequence already satisfies the CSV window
- [x] `ExtractSettlement` — thin wrapper over `psbt.Extract`
- [x] `dispute.IsSelfReleased` = true (nobody ever engaged): the
      fallback branch handles this with no new code beyond what's
      already in `UniversalScript`/`ReinforcedScript`

### Tests
- [x] Pre-escalation/fallback branch: owner-alone spend succeeds only
      after `window_days`, fails before it, fails for a non-owner
      signer — `TestPreEscalationScript_*`
- [x] `UniversalScript`: all four branches succeed with the right
      signers; arbiter alone satisfies nothing; attester alone can't
      take the mutual-settlement branch — `TestUniversalScript_AllFourBranches`
- [x] `ReinforcedScript`: all five branches succeed with the right
      signers; a cross-paired attempt (attester + the *other* side's
      arbiter) fails — `TestReinforcedScript_AllFiveBranches`
- [x] `TieBreakScript`: any 2 of 3 settle — `TestTieBreakScript_AnyTwoOfThreeSettle`
- [x] `arbiter_commitment` claim construction/verification round-trip
      and malformed-value rejection — `TestArbiterCommitmentClaim*`
- [x] `VerifyFunded` logic: funded/unfunded/underfunded/wrong-script/
      query-error cases, against a mock `ChainQuerier` — `TestVerifyFunded*`
- [x] Descriptor/witness-script-ref format/parse round-trips and
      malformed-value rejection — `TestDescriptorRoundTrip`,
      `TestWitnessScriptRefRoundTrip`, and their rejection counterparts
- [x] Full PSBT round trip (build → independently sign → finalize →
      extract → re-validate against the real engine, exactly as a
      relay/miner would) for `UniversalScript`'s mutual-settlement,
      arbiter-sides-with-subject, and fallback branches, plus
      `ReinforcedScript`'s both-arbiters-agree branch, plus a
      not-yet-finalizable case — `TestSettlement_*`
- [x] Live regtest tests against a real `btcd` node — `regtest_test.go`:
      **all of `UniversalScript`'s and `ReinforcedScript`'s non-
      fallback branches** (`UniversalScript`'s 3: mutual settlement,
      arbiter-sides-with-attester, arbiter-sides-with-subject;
      `ReinforcedScript`'s 4: mutual settlement, both-arbiters-agree,
      attester+own-arbiter, subject+own-arbiter) genuinely funded,
      broadcast, accepted by a real node's mempool, and confirmed —
      not just in-process engine validation. Gated behind a
      reachability probe (skips gracefully if no node is listening at
      `127.0.0.1:18443`, the same pattern A001's live-relay test
      already established), so this doesn't run in most environments —
      see the file's own doc comment for how to stand up a matching
      node. **The CSV fallback branch's live-node timing is *not*
      covered, for either script** — BIP68's time-based lock needs
      real median-time-past to advance ~14 days, which isn't practical
      to wait out in a test and `btcd` has no `setmocktime` RPC; that
      specific mechanism (comparing the script's required sequence
      against the input's actual sequence) is already exercised
      identically at the consensus-engine level in
      `TestPreEscalationScript_*`, which is the same code path a real
      node uses internally — a deliberate, stated scope boundary, not
      a silent gap. One real, generalizable test bug fixed along the
      way: funding a fixed absolute sat amount from a coinbase breaks
      once regtest's 150-block subsidy-halving decays the coinbase
      below that amount, which happens fast under repeated test runs
      against the same long-lived node — fixed by funding a fraction
      (half) of each coinbase's own freshly-fetched value instead of a
      constant

### Before mainnet
- [ ] Independent security review of every script and the finalizer
      logic (not self-certified) — hard gate, not a Task to check off
      solo; now tracked as its own action, [[A007]], filed 2026-09-30
      — this task is done once A007 records a pass
- [x] **Confirmation depth — implemented 2026-09-30.** Was a real,
      unresolved policy gap (`RPCQuerier.TxOut` hardcoded
      `result.Confirmations < 1` as its only threshold, no path to
      require more — not obviously safe for real mainnet value, since a
      1-confirmation reorg could un-fund an escrow a verifier already
      trusted). Fixed by moving the threshold from the querier to the
      caller: `ChainQuerier.TxOut` now returns the real confirmation
      count (`confirmations int64`) instead of collapsing "mempool
      only" and "confirmed" into one boolean; `VerifyFunded` gained a
      `minConfirmations int64` parameter, rejecting outright
      (`minConfirmations <= 0` is an error, not silently treated as
      "mempool is enough") rather than picking a number for every
      caller. `RPCQuerier.TxOut` simplified to just report the node's
      real `Confirmations` value, no threshold logic of its own left in
      the querier. Verified two ways: `verify_test.go`'s mock-based
      tests (insufficient-depth rejection, exact-boundary acceptance,
      invalid-`minConfirmations` rejection) and a new
      `TestRegtest_VerifyFundedEnforcesRealConfirmationDepth` — funds a
      real output, confirms a 3-confirmation requirement is correctly
      rejected at 1 real confirmation, accepted at exactly 1 when
      required, then mines 2 more blocks and confirms the same
      3-confirmation requirement now passes, against a real node's
      actual confirmation count, not a mock. Full repo build/vet/test
      clean; all 7 existing live-regtest branches re-verified against a
      freshly wiped chain (the long-lived node from earlier in this
      session had decayed into the documented subsidy-halving failure
      mode, unrelated to this change — confirmed by isolating non-live
      tests first, then re-running live tests against a fresh node).
      Still an open choice for whoever deploys this for real: *what*
      `minConfirmations` value to actually pass (fixed N vs. scaled to
      bond size) — this action makes that choice possible and explicit,
      it doesn't make it for every future caller.
- [x] **Fee/dust-limit gap — implemented 2026-10-02.** Was re-scoped
      with real numbers (2026-09-30), not just identified. `NewSettlementPacket` (`settle.go`) takes
      a fully caller-supplied `payouts []*wire.TxOut` with zero fee
      calculation of its own; the only fee value anywhere in the whole
      codebase is a hardcoded `const fee = int64(1000)` inside the
      **test** helpers (`regtest_test.go`), never exercised against
      real mainnet fee market conditions.
      - **This package is in an unusually strong position to solve this
        exactly, not just estimate it** — unlike a general-purpose
        wallet, every settlement's witness shape is fully known ahead
        of time (one of a small, fixed set of branches), so the real
        spending vsize can be *measured*, not guessed. Measured
        directly (throwaway test, not committed): `UniversalScript`'s
        own witness script is 263 bytes, `ReinforcedScript`'s 337 bytes
        (more pubkeys/branches); a 2-of-2 branch spend (mutual
        settlement, arbiter-sides-with-X, both-arbiters-agree) weighs
        in at vsize 199 (`UniversalScript`) / 218
        (`ReinforcedScript`); the single-sig CSV fallback branch at
        vsize 181 / 200 respectively (fewer witness elements,
        same-size embedded witness script dominates either way).
        `github.com/btcsuite/btcd/blockchain.GetTransactionWeight` —
        already effectively free, **zero new dependencies** (confirmed
        directly: importing it pulled in nothing new to `go.sum`,
        since it only needs `btcutil`/`txscript`/`wire`, all already
        used) — gives an exact weight/vsize for any real constructed
        witness, not just these four sample shapes.
      - **Fee-rate sourcing**: `rpcclient.Client.EstimateSmartFee` is
        already available on the same `*rpcclient.Client`
        `RPCQuerier` already wraps — zero new dependency, just a new
        method. Real caveat worth naming: `estimatesmartfee` needs
        real recent block/mempool history to return a useful estimate
        (a freshly-reset regtest node won't have it) — this needs
        testing against a node with real fee-paying transaction
        history, not just confirmed to exist as an RPC call.
      - **Dust-limit checking: deliberately NOT reusing
        `github.com/btcsuite/btcd/mempool.IsDust`/`GetDustThreshold`,
        checked concretely rather than assumed free.** Actually
        attempted the import: it drags in `github.com/aead/siphash`,
        `github.com/kkdai/bstream`, and `github.com/stretchr/testify`
        (plus its own `objx` dependency) — all unrelated to dust
        checking, pulled in because `mempool` also imports
        `btcutil/gcs` (compact block filters) for unrelated reasons.
        The exact same class of gotcha `hashicorp/vault/shamir` taught
        in [[A003]]: a small, wanted piece of a package dragging in an
        unrelated dependency tree. Also a **better fit to hand-roll
        anyway** — `mempool.GetDustThreshold`'s own formula assumes a
        generic "typical P2WKH spending input" (its own doc comment
        says so explicitly), when this package already knows its
        *exact* real spending cost per branch from the measurement
        above — reusing a generic heuristic would actually be less
        accurate than what's already achievable here for free.
      - **Resolved**: built inside `internal/escrow` (`fee.go`), not
        left to the caller — same reasoning the scoping pass already
        leaned toward (computing an exact fee for a script this
        package itself defines isn't a deployment/key-custody concern,
        it's the same kind of thing `WitnessScriptHash`/
        `FundingAddress` already do). `Branch` (8 named values matching
        `FinalizeUniversal`'s/`FinalizeReinforced`'s own branch order
        exactly, reusing their real `selTrue`/`selFalse` values rather
        than a second copy of that knowledge) + `EstimateSettlementVSize`/
        `EstimateSettlementFee` (worst-case witness, real
        `blockchain.GetTransactionWeight`, zero new dependencies) +
        `IsDustOutput`/`dustThreshold` (hand-rolled, matching btcd's own
        well-documented formula, generalized to take a feerate
        parameter instead of hardcoding the 1 sat/vByte default).
        `NewSettlementPacket` itself now rejects any payout below the
        standard dust threshold outright — closing the gap for real,
        not just offering an opt-in helper nobody's forced to call.
        `EstimateSmartFee` wrapping is **not** included here — still
        needs testing against a node with real fee history, named as
        future work, not silently assumed done.
- [ ] **`RPCConfig`/TLS gap, re-scoped with a real finding (2026-09-30):
      this is stronger than a hardening nicety — as shaped today, TLS
      effectively cannot be used at all against a typical self-hosted
      node.** Read `rpcclient`'s own dial code
      (`infrastructure.go`), not just the doc comment, to check this
      rather than assume: when `DisableTLS` is false, the client builds
      a `tls.Config` whose `RootCAs` pool comes *only* from
      `ConnConfig.Certificates` (a PEM cert chain) — if that's empty
      (as it always is today, since `RPCConfig` doesn't expose it at
      all), Go falls back to the system's public CA trust store. A
      self-hosted `btcd`/`bitcoind` node's default RPC cert is
      self-signed, not CA-issued — so a real connection attempt with
      `DisableTLS: false` against any node using default settings would
      fail TLS verification outright. In effect, today's `RPCConfig`
      only works at all with `DisableTLS: true`, regardless of intent —
      not just "credentials happen to be plaintext," the *connection
      itself* can't use TLS against a normal node without this fix.
      - **Fix, concretely scoped**: add `Certificates []byte` to
        `RPCConfig`, passed straight through to `ConnConfig.Certificates`
        — lets a caller supply their node's actual self-signed cert (or
        a real CA-issued one) and get real TLS verification instead of
        either failing outright or disabling TLS to work around it.
      - **Also concretely available, same `ConnConfig` already used**:
        `CookiePath` — bitcoind's/btcd's own standard production auth
        mechanism (a dynamically-generated, node-managed credential
        file, rotated on restart), used instead of `User`/`Pass` when
        set. Adding this as an alternative to plaintext `User`/`Pass`
        removes the need to embed real credentials in application
        config at all for a real deployment — the standard pattern, not
        an invented one.
      - **Lower-priority, noted but not scoped in depth**: `Proxy`/
        `ProxyUser`/`ProxyPass` (SOCKS5) already exist on `ConnConfig`
        too — relevant for reaching a remote or Tor-hidden node without
        exposing RPC on the open network directly, real but not as
        immediately blocking as the TLS gap above.
      - **Confirmed NOT an additional gap, checked rather than
        assumed**: no code anywhere in this package logs or prints
        `RPCConfig`/`ConnConfig` (`grep` for `%+v`/logging calls in
        `rpcquerier.go` found nothing) — the plaintext-`User`/`Pass`-
        as-Go-strings limitation (can't be securely zeroed after use,
        unlike `[]byte`) is real but minor and not fixable without a
        bigger API change; noted as an accepted residual rather than a
        task, the same "state the honest limit rather than invent
        complexity to fully solve it" move this design makes elsewhere.
      - Still adjacent to, not clearly inside, this action's own "no
        wallet/key-management infrastructure" exclusion (this is the
        *chain-query* connection, not fund custody) — worth an explicit
        decision on which action owns implementing it, same as the
        fee/dust-limit item above.
- [ ] **Confirmed NOT a gap, checked rather than assumed**: chain
      selection (mainnet vs. testnet vs. regtest) is already properly
      parameterized — `FundingAddress` (`timelock.go`) takes
      `*chaincfg.Params` as a real argument, nothing in the production
      package (only test files) hardcodes
      `chaincfg.RegressionNetParams`. Mainnet activation needs no
      script/address-derivation code change, just the caller passing
      `chaincfg.MainNetParams`.

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

2026-09-29 — Started implementation. New `internal/escrow` package:
`timelock.go` (BIP68 time-based `SequenceForDays`, `PreEscalationScript`,
`WitnessScriptHash`, `FundingAddress`), `multisig.go`
(`MultisigScript`, `SmallTierEscalatedScript`, `LargeTierEscalatedScript`,
`TieBreakScript`), `descriptor.go` (the narrow BIP380-shaped `EscrowRef`
format/parse this package's own fixed script template supports, not a
general descriptor engine), `verify.go` (`VerifyFunded` against a
`ChainQuerier` interface), `arbiter_commitment.go` (the new claim type
the corrected larger-bond design needs). Every script was validated
by actually executing it against `btcd`'s real consensus `txscript`
engine — the same one `btcd` uses to test its own opcodes — not just
checked for "builds without error": each spending path was signed and
run through `vm.Execute()`, and each *invalid* spend (wrong signer,
too early, only one of two required arbiters, arbiter alone
attempting the mutual-settlement branch) was confirmed to actually
fail. CSV timelock testing needed no live node or simulated passage of
time — BIP68's relative-locktime check compares the script's required
sequence directly against the spending input's own declared sequence
value, which a test can just set directly. 24 new tests, all passing;
full repo build/vet/test clean (104 tests total, `go mod tidy` added
only `btcd`/`btcutil`/`chaincfg`/`btclog`, no toolchain bump).

**Deliberately not marked DONE — substantial work remains**, listed
precisely in Tasks above rather than glossed over: (1) cooperative
PSBT-based transaction construction/signing/combining helpers a real
caller would use — this session validated the *scripts* directly
against hand-built transactions, which is sufficient to prove the
scripts themselves are correct, but not the same as a usable
construction API; (2) the non-escalation detection task; (3) a
concrete `ChainQuerier` implementation against a real node — no
regtest/bitcoind is reachable in this environment, so only the
interface and its logic (via a mock) could be tested; (4) the
mainnet-gating independent security review, not reached since the
surrounding plumbing isn't built yet. This is a large, multi-part,
explicitly fund-loss-risk action — treating "the script logic is
real and consensus-validated" as a legitimate, substantial slice
rather than claiming the whole action is finished.

2026-09-30 — Asked to build the PSBT construction helpers for the
escalation-based design above. Doing so surfaced a second, more
serious bug than the key-availability one already fixed: CSV blocks
*any* spend of the pre-escalation output before `window_days`,
including a legitimate cooperative escalation — meaning voluntary
escalation, as scripted, was actually impossible before the timelock
expired, and even once it did expire, the script only ever checked
the owner's own signature, tying it to no dispute outcome at all. The
bond as designed provided no real enforcement in either tier.

Considered and researched a genuine fork before patching further:
should the escrow layer move to a smart-contract chain (Ethereum-style
EVM, concretely Hyperliquid's HyperEVM) instead of continuing to fight
Bitcoin Script's limits? Real research (not assertion): Hyperliquid
runs 24 validators (BFT to ~1/3 malicious, a named tradeoff in its own
docs); every path to get BTC value onto an EVM chain (WBTC, HyperUnit,
tBTC) requires trusting some off-chain signer set, and bridge exploits
have cost >$2.8B since 2022 (>69% of all DeFi losses in that period) —
a real, demonstrated risk category, not a hypothetical one to weigh
against a hypothetical script bug. Kleros's juror-pool model does
solve "unknown challenger at deploy time" elegantly, but it's exactly
the global-crowdsourced-court model D001 already rejected for its own
stated reasons. Concluded: no chain's consensus can adjudicate the
dispute's *substance* (a human judgment call, not on-chain-computable
data) regardless of platform — the actual gap was script flexibility,
and that turned out to be closable on Bitcoin itself.

The closing insight: D001/A005 scoped bonding/slashing to disputes
between two *identified* parties — the attestation being bonded
already names its own `subject_key`, so the "future" counterparty was
never actually unknown, only the arbiter was. Redesigned around this:
`UniversalScript` (4 branches: mutual settlement any time, arbiter
sides with attester, arbiter sides with subject, CSV last resort) and
`ReinforcedScript` (adds a "both independently pre-committed arbiters
agree, no disputant needed" branch, available only if the subject has
published their own standing `arbiter_commitment`) — a single, static
script, fully configured at bond-creation time, no escalation
transaction ever. `arbiter_commitment` became a *standing* claim (not
per-bond), reusable across any future dispute. Removed
`SmallTierEscalatedScript`/`LargeTierEscalatedScript`
(superseded); kept `TieBreakScript` as a general building block, no
longer one of the two main scripts' own branches, since there's no
live moment left to escalate to a third arbiter under a static
design — a disagreement between the two independent arbiters is now an
accepted, rare residual (whichever competing spend confirms first
wins), not solved with more machinery.

Built the PSBT settlement helpers this was actually about, now much
smaller than originally scoped since there's no escalation chain:
`NewSettlementPacket`/`SignSettlementInput` (build, sign
independently), `FinalizeUniversal`/`FinalizeReinforced` (hand-written
— the standard library's own multisig finalizer doesn't recognize
`OP_IF`-branching scripts or the single-key CSV branch, confirmed by
reading `psbt`'s own `checkIsMultiSigScript`), `ExtractSettlement`.
Also replaced `EscrowRef`'s BIP380-descriptor framing with plain
hex-encoded witness-script bytes for these custom scripts (the earlier
narrow descriptor template only fit the single-key CSV shape, kept for
that case alone). Every branch of both new scripts, plus full PSBT
build→sign→finalize→extract→re-validate round trips for the common
paths, passes against the real consensus engine — including one
caught-and-fixed finalizer bug (the fallback finalizer initially
didn't check the packet's own sequence against the CSV requirement,
so it would have happily produced a witness real validation would
reject at broadcast time). 22 new/changed tests, full repo clean (110
tests total). Still not DONE: `ChainQuerier`'s real backend (still no
regtest node reachable here) and the mainnet security-review gate.

2026-09-30 — Filed [[A007]]: the mainnet security-review gate this
action's own Tasks list has always carried, now tracked as its own
action instead of an aspiration inside this one's task list. A007
also owns the regtest/testnet exercise (funding, spending, confirming
each branch for real) as its own prerequisite, ahead of the review
itself.

2026-09-30 — Found and used a real regtest node. `btcd`'s full source,
including the daemon's own `main` package and `rpcclient`, was already
present in this project's own Go module cache as a transitive
dependency of `txscript`/`psbt` — `go install
github.com/btcsuite/btcd@v0.24.2` built a real, working `btcd` binary
directly from it, no new binary dependency, no download beyond what
Go's own module system already needed. Started it in regtest mode
(`--miningaddr` pointing at a locally-generated key), mined past
segwit's real BIP9 activation threshold (regtest requires actually
signaling it, same as any other network — not active from genesis),
and built `rpcquerier.go`'s `RPCQuerier` (backed by `rpcclient`,
implementing the `ChainQuerier` interface `verify.go` left abstract)
plus `regtest_test.go`'s live tests. `UniversalScript`'s
mutual-settlement and both arbiter-assisted branches were genuinely
funded from a real matured coinbase, broadcast, accepted by the real
node's mempool, and confirmed — not just validated in-process.
Caught and fixed one real test bug along the way: regtest's coinbase
subsidy actually halves every 150 blocks
(`RegressionNetParams.SubsidyReductionInterval`), so reusing a
stale captured coinbase value across a halving boundary produced an
invalid signature amount; fixed by fetching each coinbase's actual
value fresh rather than reusing one across separately-mined blocks.
Branch 4 (the CSV fallback)'s live-node timing is explicitly not
covered — `btcd` has no `setmocktime`, and waiting out a real 14-day
BIP68 median-time-past lock isn't practical in a test; that specific
mechanism is already exercised at the consensus-engine level (the
same code path a real node uses) in the existing
`TestPreEscalationScript_*` tests. `ChainQuerier` now has a real
backend, not just a mock-tested interface. Tests remain gated behind
a reachability probe (skip gracefully, same pattern as A001's live-relay
test) since no node is reachable in most environments including this
one by default — this was a deliberately stood-up exception for this
session, not a standing environment change.

2026-09-30 — Extended live regtest coverage to `ReinforcedScript`'s
four non-fallback branches, refactoring the shared mining/funding/
settling plumbing into reusable helpers (`regtestMiner`,
`regtestFundFreshOutput`, `regtestSettleAndConfirm`) rather than a
third near-copy of the same test body. Found and fixed a real,
generalizable bug in the process: the original tests funded a fixed
absolute sat amount from a fresh coinbase, which works fine at first
but breaks once regtest's 150-block subsidy-halving decays a
long-lived node's coinbases below that fixed amount — confirmed this
concretely by re-running the full suite against the same node
repeatedly and watching it start failing with "inputs less than
amount spent," then "insufficient priority," then "negative output
value" as the chain got mined deeper. Fixed by funding half of each
coinbase's own freshly-fetched value instead of a constant — verified
stable across repeated `-count=1` reruns against the same
increasingly-mined chain. Documented (in `regtest_test.go`'s own doc
comment) that this still isn't unbounded forever — a genuinely
long-lived, never-reset node will eventually decay far enough to trip
the same failure modes again, which is regtest's own subsidy design
doing what it's supposed to, not a code bug; the fix is periodically
wiping the node's datadir, not chasing an ever-smaller fraction.
All 7 live branches (3 `UniversalScript` + 4 `ReinforcedScript`) pass
against a real node; full repo clean aside from one confirmed
unrelated, pre-existing flaky failure (`relay.damus.io` returning 503
on `internal/attestation`'s own live test, an external service issue
untouched by this session's work).

2026-09-30 — **Scoped the mainnet-gate follow-up beyond A007's code
review**, per direct request, by actually reading the production code
(`rpcquerier.go`, `settle.go`, `timelock.go`) rather than assuming
"Before mainnet" was already complete. Found three real, unresolved
policy gaps a correctness-focused code review wouldn't necessarily
flag as *wrong* (the code does exactly what it currently says): (1)
confirmation depth is hardcoded to 1, not reorg-safe for real value,
and nobody has actually chosen a real number; (2) zero fee-estimation
or dust-limit logic exists anywhere in the package — the only fee
value in the whole codebase is a `1000`-sat constant inside test
helpers; (3) `RPCConfig` already self-documents that it "makes no
attempt to be safe for a mainnet node's credentials" (plaintext
user/pass, optional TLS disable) — a real comment left unaddressed as
a task. Also explicitly confirmed one thing is *not* a gap, rather
than assumed: chain-param selection (mainnet vs. regtest) is already
properly parameterized via `*chaincfg.Params`, not hardcoded anywhere
in production code. All recorded as distinct Tasks above, not folded
into A007's own review scope — A007 reviews script/finalizer
*correctness*; these are separate policy decisions this action's own
"Before mainnet" gate should also require an explicit answer to.

2026-09-30 — **Implemented the confirmation-depth fix**, picked from
the three scoped gaps as the most self-contained (no new external
dependency, fully testable with the existing mock plus the real
node). `ChainQuerier.TxOut` now returns a real `confirmations int64`
instead of baking a threshold into the query; `VerifyFunded` gained a
`minConfirmations` parameter and rejects `<= 0` outright rather than
silently accepting mempool-only as funded. `RPCQuerier` simplified —
no threshold logic left in the querier at all, just reports what the
node says. No callers outside `internal/escrow` itself, so the
interface change was safe to make directly (checked first via grep,
not assumed). Verified against both the mock (5 tests, 3 new) and a
real node (1 new live test funding a real output and checking
rejection/acceptance at different real confirmation depths). Caught
the long-lived regtest node from earlier in this session had decayed
past the documented subsidy-halving threshold mid-verification —
isolated by running non-live tests first (all passed), confirmed it
was the known pre-existing issue rather than this change, then wiped
the node's datadir and re-verified all 7 existing live branches plus
the new test against a fresh chain. Full repo build/vet/test clean.
Doesn't decide *what* `minConfirmations` value real callers should use
(fixed vs. scaled to bond size) — that's still an open operational
choice this change makes possible to express, not one it makes for
every future caller.

2026-09-30 — **Scoped the fee/dust-limit gap with real measurements**,
not just the earlier identification. Key finding: this package can
solve this *exactly* rather than estimate it, since every settlement's
witness shape is one of a small, known set — measured real vsize per
branch with a throwaway test (deleted after use, not committed):
`UniversalScript` witness script 263 bytes (2-of-2 branch vsize 199,
fallback vsize 181), `ReinforcedScript` 337 bytes (2-of-2 branch vsize
218, fallback vsize 200). `blockchain.GetTransactionWeight` (zero new
dependencies — verified directly, nothing new landed in `go.sum`) did
the measuring. Fee-rate sourcing is already available for free too:
`rpcclient.Client.EstimateSmartFee`, same client `RPCQuerier` already
wraps — caveat, needs testing against a node with real fee history,
not just confirmed to exist as an RPC call. Deliberately rejected
reusing `btcd/mempool.IsDust`/`GetDustThreshold` after actually
attempting the import: drags in `aead/siphash`/`kkdai/bstream`/
`stretchr/testify` for unrelated reasons (the package also imports
`btcutil/gcs` for compact block filters), the same class of gotcha
`hashicorp/vault/shamir` taught in A003 — and a worse fit anyway, since
its own formula assumes a generic P2WKH spend where this package
already knows its *exact* real cost. Left genuinely open: whether the
fee/dust helper belongs inside `internal/escrow` (leaning yes — this
isn't a deployment/key-custody concern, it's the same kind of thing
`WitnessScriptHash` already does) or stays the caller's job — a real
decision to make before implementing, not decided here.

2026-09-30 — **Scoped the RPCConfig/TLS gap, and it's a stronger
finding than expected.** Read `rpcclient`'s own dial code
(`infrastructure.go`) directly rather than trusting the existing doc
comment: `RPCConfig` as currently shaped can only actually connect
with `DisableTLS: true` — enabling TLS with no `Certificates` supplied
falls back to the system CA trust store, which a self-hosted node's
default self-signed cert will never satisfy, so a real TLS connection
attempt against a normal node would just fail. Not just "credentials
are plaintext," the *connection itself* doesn't work as shaped.
Scoped two concrete, already-available fixes on the same
`rpcclient.ConnConfig` this code already wraps: `Certificates []byte`
(lets a caller supply their node's real cert, fixing the TLS gap
directly) and `CookiePath` (bitcoind's/btcd's own standard production
auth mechanism, avoids embedding plaintext credentials in application
config at all). Also checked and confirmed NOT an additional gap:
nothing in this package logs or prints the config today. Noted
SOCKS5 proxy support as available but lower-priority. Same open
question as the fee/dust-limit item: whether this belongs inside
`internal/escrow`'s own scope or is a separate deployment concern —
not decided, scoping only.

2026-10-02 — **Implemented the fee/dust-limit fix**, the second of the
three scoped mainnet-gate gaps (after confirmation depth). New
`internal/escrow/fee.go`: `Branch` (8 named values mirroring
`FinalizeUniversal`'s/`FinalizeReinforced`'s own branch order and real
`selTrue`/`selFalse` selector values — deliberately the single source
of truth, not a second copy), `EstimateSettlementVSize`/
`EstimateSettlementFee` (worst-case 72-byte placeholder signatures,
measured via `blockchain.GetTransactionWeight`, zero new
dependencies), `IsDustOutput`/`dustThreshold` (hand-rolled, matching
btcd's own `mempool.GetDustThreshold` formula without importing the
package that drags in `aead/siphash`/`kkdai/bstream`/`stretchr/
testify`). `NewSettlementPacket` now rejects any payout below the
standard dust threshold outright, closing the actual gap rather than
just adding an opt-in helper. Caught and fixed one real bug in the
process: the original scoping pass's throwaway measurement used
`{0x01}` as a placeholder for every selector including `selFalse`
ones (which are actually empty, zero bytes) — one byte too generous
per `selFalse`, enough to flip `ReinforcedScript`'s fallback branch
from the originally-quoted vsize 200 to the now-correctly-measured
199. Fixed in the test, not the implementation, since the
implementation always used the real selector values; the earlier
scoping note's number was the one that was slightly wrong. 9 new
tests (4 pinned against the real measured vsize values, including
the corrected one; 1 broad all-branches sanity sweep; dust-threshold
boundary cases including the unconditional OP_RETURN-is-always-dust
rule; feerate scaling; `NewSettlementPacket`'s own new rejection
behavior). Full repo build/vet/test clean — existing tests
unaffected, since every payout amount they use is many orders of
magnitude above the dust threshold. Did not re-run the live regtest
suite for this change specifically (no node was running, and the new
check is provably a no-op for the live tests' own amounts — tens of
millions of sats against a threshold in the low hundreds); noted
here rather than silently skipped. **Not done**: wrapping
`EstimateSmartFee` into a real feerate source — still needs testing
against a node with real fee-paying history, named as remaining
future work, not assumed covered by this pass.
