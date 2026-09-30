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

## Outreach message (draft, 2026-09-30)

Drafted for whoever the project operator identifies as a candidate
reviewer — sharpened once already, specifically to avoid implying
this is untested code being thrown over a wall (it isn't: rigorous
self-testing is what makes asking for review worth someone's time, not
a reason to skip it). Placeholders (`[...]`) are for the operator to
fill in — repo access, timeline, compensation — none of which is this
action's own call to make.

> **Subject: Independent review request — Bitcoin escrow script
> (thoroughly tested, needs a second opinion)**
>
> Hi [name],
>
> I'm working on a small identity/attestation protocol built on Nostr
> (persona), and one piece of it — a bonding/slashing escrow mechanism
> — moves real Bitcoin through some custom Script (`OP_IF`-branching
> multisig plus a CSV timelock fallback) and hand-rolled PSBT
> finalization logic.
>
> To be clear about where this stands: it's already been tested
> rigorously on my end — every spending path validated against
> `btcd`'s real consensus script engine, and every branch but one
> genuinely funded, broadcast, and confirmed against a live regtest
> node. I'm not asking you to find the bugs my own testing missed
> through carelessness; I'm asking because **no amount of self-testing
> substitutes for an independent review on fund-custody code** — the
> person who designs a script and the person testing it share the same
> blind spots by construction, and this is exactly the kind of code
> where "I tested it myself and it passed" isn't a credible enough bar
> before real money moves through it.
>
> **What I'm asking for**: a review of one self-contained Go package
> (~700 lines of implementation, roughly matched in tests). I've put
> together a review package that should get you oriented without
> back-and-forth: scope, design rationale, exactly what's been tested
> and how (including the one thing that *isn't* covered, and why),
> known accepted tradeoffs, and concrete pass/fail criteria — so you're
> evaluating against a stated bar, not guessing what "looks safe" means
> to me.
>
> Link/location: `internal/escrow/SECURITY_REVIEW.md` in the repo,
> pinned to commit `33dc3d44078567009e780e2d7b6e64507ce61bf4`. [repo
> access: fill in]
>
> **Why this needed real design work, briefly**: the interesting part
> is that a Bitcoin script can't reference a counterparty who doesn't
> exist yet at funding time, which killed an earlier design outright —
> the review package's own "why this needed its own design pass"
> section has the full story, including why I stayed Bitcoin-native
> instead of moving to a smart-contract chain.
>
> **What I need back**: findings ranked by severity, each with a
> concrete failure scenario — not "this looks off," but "here's the
> input that breaks it."
>
> [Timeline: ...]
> [Compensation: ...]
>
> Happy to hop on a call first if that's easier than reading cold. Let
> me know if you're interested or if this isn't the right fit for you.
>
> Thanks,
> [your name]

## Tasks

### Prerequisite: regtest/testnet exercise
- [x] Stand up or gain access to a regtest node (bitcoind/btcd) — done
      2026-09-30: `btcd`'s full source (including the daemon's own
      `main` package) was already present in this project's Go module
      cache as a transitive dependency of `txscript`/`psbt`, so `go
      install github.com/btcsuite/btcd@v0.24.2` builds a real, working
      node directly, no new binary dependency. **Not a standing
      environment fixture** — stood up for this session's exercise,
      not guaranteed reachable in any future session; `regtest_test.go`
      documents exactly how to reproduce it
- [x] Build the real `ChainQuerier` backend A006 left as an interface,
      against that node — done 2026-09-30, `internal/escrow/
      rpcquerier.go` `RPCQuerier`, backed by `btcd`'s own `rpcclient`
      (same module, no new dependency)
- [x] Actually fund, spend, and confirm every non-fallback branch of
      both `UniversalScript` (3: mutual settlement, arbiter-sides-
      with-attester, arbiter-sides-with-subject) and `ReinforcedScript`
      (4: mutual settlement, both-arbiters-agree, attester+own-arbiter,
      subject+own-arbiter) on a real regtest node — done 2026-09-30,
      `regtest_test.go`, genuine broadcast + real mempool acceptance +
      confirmation for all 7, not just in-process engine validation.
      **Only the CSV fallback branch (shared by both scripts) remains
      untested live** — a deliberate, stated scope boundary (see A006's
      own Log: no `setmocktime` in `btcd`, and the underlying sequence-
      comparison mechanism is already proven at the consensus-engine
      level), not an oversight

### Arranging the review
- [ ] Identify a reviewer meeting the "independent" bar above — **the
      one piece of this action that isn't mine to do**: finding and
      engaging an actual human reviewer (or firm) is the user's own
      call, not something to fabricate or simulate. An outreach
      message is drafted above, ready to send once a candidate is
      identified — sharpened once already so it doesn't undersell the
      testing already done (self-testing is what makes asking for
      review worth someone's time, not a reason to skip it)
- [x] Assemble the review package — done 2026-09-30,
      `internal/escrow/SECURITY_REVIEW.md`: scope, design rationale,
      a categorized test-coverage summary (consensus-engine-level /
      real-regtest-node-level / not-covered-and-why), the known
      already-accepted residuals, concrete pass criteria, and exact
      commands to reproduce every result. Caught and fixed one real
      issue while assembling it: the package's own top-level doc
      comment (and `PreEscalationScript`'s) still described the
      *superseded* escalation-based design verbatim — precisely the
      kind of stale documentation that would mislead an external
      reviewer's first orientation to the code, fixed before handing
      anything to anyone
- [ ] Hand off and track turnaround — blocked on identifying a
      reviewer above

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

2026-09-30 — Made real progress on this action's own prerequisite,
found opportunistically while working on A006: `btcd`'s full daemon
source was already present in this project's Go module cache (a
transitive dependency of `txscript`/`psbt`), so a real regtest node
could be built and run directly, no new binary dependency. Built the
real `ChainQuerier` backend (`RPCQuerier`) and genuinely funded,
broadcast, and confirmed three of `UniversalScript`'s four branches
against it. **Not fully done**: this node isn't a standing fixture —
it was stood up for this session and isn't guaranteed reachable in a
future one (though `regtest_test.go` documents exactly how to
reproduce it); branch 4's live timing and all of `ReinforcedScript`'s
branches remain untested live. The review itself (identifying a
reviewer, assembling the package, acting on findings) hasn't started.

2026-09-30 — Extended live coverage to `ReinforcedScript`'s four
non-fallback branches, closing the gap the previous entry left open.
All 7 live-testable branches across both scripts now pass against a
real node. Found and fixed a real bug specific to this action's own
prerequisite while doing so: funding a fixed sat amount from a fresh
coinbase breaks once regtest's 150-block subsidy-halving decays a
long-lived node's coinbases below that amount — confirmed concretely
by watching the same test suite fail progressively worse
("insufficient inputs" → "insufficient priority" → "negative output
value") across repeated reruns against the same node. Fixed by
funding a fraction of each coinbase's own value rather than a
constant; documented that this still isn't unbounded and a
sufficiently long-lived, never-reset node will eventually need its
datadir wiped regardless — regtest's own design, not a persona bug.
**Still not started**: the CSV fallback branch's live timing (a
deliberate, stated scope boundary, not an oversight — see A006's own
Log) and the actual review itself.

2026-09-30 — Started arranging the review. Assembled the actual
review package (`internal/escrow/SECURITY_REVIEW.md`) — the one piece
of "arranging" genuinely within scope to do directly: scope, design
rationale (including a pointer to the seriously-considered-and-
rejected EVM pivot, since "why not a smart contract" is a fair first
question), a categorized test-coverage summary, the known
already-accepted residuals, concrete pass criteria, and exact
reproduction commands. Assembling it surfaced a real, worth-fixing
issue on its own: the package's own top-level doc comment (`go doc
./internal/escrow`) and `PreEscalationScript`'s doc comment still
described the superseded escalation-based design verbatim, months
out of date relative to the actual code — exactly the kind of stale
documentation that would mislead an external reviewer's first
orientation. Fixed both before handing anything to anyone. **Not
done**: identifying and actually engaging a reviewer is explicitly
not something this action can do solo — that's a real business/
logistics decision for the project's own operator, surfaced back to
them rather than simulated. "Hand off and track turnaround" and
"acting on findings" both remain blocked on that.

2026-09-30 — Drafted the actual outreach message for the operator to
send once they identify a candidate reviewer (recorded above under
"Outreach message"). First draft was checked against a real risk:
does it accidentally read as "untested code, please find the bugs,"
which would both undersell the work already done and set the wrong
expectation for a reviewer? Revised to lead with what's already been
verified (consensus-engine validation, live regtest confirmation for
all but one branch) and state directly why independent review is
still asked for anyway — self-testing can't catch a design flaw the
author never thought to test for in the first place, and "I tested it
myself" isn't a credible enough bar for fund-custody code regardless
of how true it is. Still not this action's call to make: who actually
receives it, and when.
