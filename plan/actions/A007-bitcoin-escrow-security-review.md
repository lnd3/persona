---
id: A007
title: Independent security review of the Bitcoin escrow scripts
status: PLANNING
design: D001
project: P001
created: 2026-09-30
updated: 2026-09-30
---

## Context

[[A006]]'s own Tasks list has always carried a "before mainnet" gate:
"Independent security review of every script and the finalizer logic
(not self-certified)." That gate can't be satisfied by A006 itself —
by definition, a review isn't independent if the same party who wrote
the code performs it. This action exists to actually arrange and
track that review as a distinct, real deliverable, rather than leaving
"someday, before mainnet" as an unfiled aspiration inside another
action's task list.

This is the highest-stakes review this project will ever need. Every
other action built so far (identity, attestation, trust, recovery,
payment receipts, dispute protocol) can be wrong without losing
anyone's money — a bug there produces a wrong *opinion*, at worst.
A006's scripts, if wrong, can lose real Bitcoin, permanently and
unrecoverably. This action's whole purpose is making sure that
specific risk gets a second, genuinely independent set of eyes before
any of A006's code ever touches a real key holding real value.

**Builds on**: [[A006]]'s `internal/escrow` package as it currently
stands (`UniversalScript`/`ReinforcedScript`, the finalizers in
`settle.go`, `arbiter_commitment.go`) — this action reviews that code,
it doesn't change the design. If the review finds a real bug, fixing
it happens back in A006, not here; this action tracks and re-verifies
the fix, it doesn't own the redesign.

**Explicitly out of scope for this action**:
- Performing the review myself — an AI-assisted review by the same
  system that wrote the code under review isn't independent, by the
  same logic that makes this action necessary in the first place. This
  action's job is arranging a genuinely independent review and acting
  on its findings, not substituting for one.
- Building the still-missing pieces A006 itself left open (a real
  `ChainQuerier` node backend, a resolved threshold for "small" vs.
  "larger" bonds) — see Decisions below on why those come first, but
  building them is A006's job, not this action's.
- Any mainnet deployment/activation work itself — this action's
  output is a pass/fail verdict and a punch list, not a launch.

## Decisions made here (not in D001/A006 — concrete choices this action owns)

- **Regtest/testnet validation happens first, the independent review
  second — not in parallel, not reversed.** A006's own scripts are
  currently validated only against btcd's in-process consensus engine
  (no live node reachable in this environment) — real, but narrower
  than actually broadcasting funded transactions on regtest/testnet
  and watching them confirm. Spending a reviewer's time on issues a
  regtest run would surface immediately (a malformed PSBT, a fee
  miscalculation, an address-encoding mismatch) wastes the one
  resource this action can't get more of cheaply. Regtest/testnet
  exercise is this action's own prerequisite, not the reviewer's job.
- **"Independent" means: not this AI, not solely the same human
  operator who's been directing this project.** A genuine second set
  of eyes — ideally someone with real Bitcoin Script/consensus
  experience, since this project's own script-writing here (however
  carefully tested against the real engine) is still a first attempt
  by people/systems without a production Bitcoin-script security
  track record. Doesn't have to be a paid audit firm to start — a
  knowledgeable third party reviewing the actual script logic and
  finalizer code is the bar, not a specific vendor.
- **Review package, not just "here's the repo"**: the reviewer gets a
  scoped, self-contained bundle — the exact commit, the script
  construction code (`upfront.go`, `multisig.go`, `timelock.go`), the
  finalizer logic (`settle.go`), the full test suite and what it does
  and doesn't cover, and A006's own plan file (Decisions + Log) as the
  design rationale and known-residuals record, so the reviewer isn't
  reverse-engineering intent from code alone.
- **Pass criteria, stated concretely rather than left vague**: no
  finding that allows (a) a party to move funds without a valid
  signature actually matching the intended branch, (b) any branch to
  be satisfied with fewer real signatures than its design requires, or
  (c) the fallback branch to be spendable before its CSV window has
  genuinely elapsed on a real chain. Findings outside this bar
  (style, gas/fee efficiency, defense-in-depth suggestions) get
  tracked but don't block a pass.
- **A "no mainnet" gate lives in the plan, not in code, for now** — no
  build tag or runtime check currently distinguishes network params
  (`FundingAddress` already takes a `*chaincfg.Params` the caller
  chooses); this action doesn't add one, since a determined caller
  could bypass a soft code-level gate just as easily as a plan-file
  one. The actual protection is operational: nobody funds one of these
  scripts with real mainnet sats before this action records a pass.

## Tasks

### Prerequisite: regtest/testnet exercise
- [ ] Stand up or gain access to a regtest node (bitcoind/btcd) — not
      available in this environment as of A006's own implementation
      pass; blocks everything below until resolved
- [ ] Build the real `ChainQuerier` backend A006 left as an interface,
      against that node
- [ ] Actually fund, spend, and confirm each of `UniversalScript`'s
      four branches and `ReinforcedScript`'s five on regtest —
      real broadcast and confirmation, not just in-process engine
      validation

### Arranging the review
- [ ] Identify a reviewer meeting the "independent" bar above
- [ ] Assemble the review package (commit, code, tests, A006's plan
      file as design rationale)
- [ ] Hand off and track turnaround

### Acting on findings
- [ ] Triage findings against the stated pass criteria
- [ ] For any finding that fails the pass bar: file/track the fix back
      in A006, re-verify (regtest + engine tests), do not treat as
      resolved until re-reviewed
- [ ] Record the final verdict here, and update A006's own "before
      mainnet" task to point at this action's outcome

## Log

2026-09-30 — Filed as A006's own long-standing "before mainnet"
gate, tracked as its own action rather than left as an aspiration
inside A006's task list. Scoped tightly: this action arranges and
acts on an independent review, it doesn't perform one itself (an
AI-assisted self-review isn't independent, by the same logic that
makes this action necessary at all) and doesn't build A006's still-
missing pieces. Decided regtest/testnet exercise must happen first,
before the reviewer's time gets spent — cheaper bugs should be caught
cheaply. Stated concrete pass criteria (no valid spend without a real
matching signature, no branch satisfiable with fewer signatures than
designed, no fallback spend before its CSV window has genuinely
elapsed) rather than leaving "looks safe" as the bar.
