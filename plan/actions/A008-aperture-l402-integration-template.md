---
id: A008
title: Aperture/L402 integration template (simulated), plus deferred real Lightning-regtest integration
status: PLANNING
design: D001
project: P001
created: 2026-09-30
updated: 2026-09-30
---

## Context

[[A004]] built the attestation-cost gateway's individual pieces
(`Receipt` sign/verify, `PendingClaim`/`Finalize`, the reference
`POST /receipt` gateway handler, L402 challenge parsing/header
construction) with isolated unit tests for each. This action assembles
them into one cohesive, documented template use case demonstrating the
actual intended real-world flow end to end — the same kind of gap
[[A006]]'s own escrow-settlement work closed for Bitcoin scripts
(isolated engine-level tests first, then a real end-to-end exercise).

Two genuinely different scopes here, staged deliberately rather than
conflated:

- **Stage 1 (this pass): a simulated template with tests.** Fronts the
  real gateway handler with a *simulated* L402 gate — no real macaroon
  minting, no real Lightning payment — proving persona's own code
  (challenge parsing, header construction, receipt signing/embedding,
  independent third-party re-verification) is wired correctly, at
  effectively no infrastructure cost.
- **Stage 2 (deferred, not started): a real Aperture + Lightning-
  regtest integration.** Standing up genuine `lnd`/`aperture`
  infrastructure and proving the same flow against it for real —
  substantially larger, staged separately on purpose (see Decisions).

## Decisions made here

- **Stage 2 is a real, standalone infrastructure undertaking, not an
  extension of the Stage 1 template — checked concretely, not
  assumed.** `lnd` and `aperture` both resolve as real Go modules, so
  the *shape* is feasible — but neither can be imported into persona's
  own `go.mod` directly: both require a newer Go toolchain than
  persona's own (checked again 2026-09-30, moved further since the
  first check: `lnd@v0.21.3-beta` now requires **Go ≥1.25.13**,
  `lnd@v0.21.4-beta.rc1` **Go ≥1.26.8**, `aperture@v0.5.0` **Go
  ≥1.26.0** — these version-to-toolchain mappings drift release to
  release, so re-check at actual implementation time rather than
  trusting this scoping pass's numbers). Same class of risk
  `hashicorp/vault/shamir` taught in [[A003]], for a dependency that's
  genuinely needed this time, not avoidable.
- **Correction (2026-09-30, checked concretely by actually attempting
  it, not assumed from the `btcd` precedent): neither binary can be
  built with a plain `go install <module>@<version>` the way `btcd`
  was.** Both `lnd`'s and `aperture`'s own `go.mod` files contain
  `replace` directives (pinning their own forked/patched
  dependencies) — Go's tooling refuses to honor `replace` directives
  for a module installed by version reference (confirmed for
  `lnd@v0.21.3-beta`, `lnd@v0.21.4-beta.rc1`, and `aperture@v0.5.0`;
  the exact error: `"The go.mod file for the module providing named
  packages contains one or more replace directives. It must not
  contain directives that would cause it to be interpreted
  differently than if it were the main module."`). This is structural
  to both projects, not a version-specific bug, and matches each
  project's own documented install method (a `git clone` + build from
  within the cloned tree, not `go install` of a remote module — `lnd`'s
  own README uses this shape). **Corrected mitigation**: `git clone`
  each repo (small footprint — confirmed ~7MB `lnd`, ~1MB `aperture`
  as shallow clones) into a scratch directory, then `go build
  ./cmd/lnd` / `go build ./cmd/aperture` from *within* that cloned
  tree (where the module's own `replace` directives resolve correctly,
  since it's then the main module) — output binaries land in an
  isolated location, never imported into persona's own `go.mod`,
  preserving the original intent (persona's own toolchain requirement
  stays untouched) even though the exact build mechanism differs from
  `btcd`'s.
- **Environment-specific operational note, found while scoping**: this
  session's own sandbox blocks building freshly-cloned external-repo
  code without explicit approval (a safety-classifier denial hit while
  testing the `git clone` + `go build` approach above) — worth knowing
  going in, not a code-level blocker. Whoever picks up Stage 2's actual
  implementation should expect to either get that approval explicitly
  or do the build step in a less restricted environment/worktree.
- **What Stage 2 actually requires, named concretely rather than left
  vague**: two `lnd` regtest nodes (pointed at the same btcd regtest
  backend [[A006]]/[[A007]] already stood up), an on-chain channel
  opened and funded between them (a single node can't pay its own
  invoice — this is where real Lightning-regtest setups typically get
  finicky), `aperture` configured against one node's macaroon/TLS and
  registered as a backend service pointing at persona's own reference
  gateway handler, and orchestration code driving the full real flow
  (persona's L402 client hits Aperture, receives a genuine macaroon+
  invoice challenge, pays it from the second node, retries with a real
  `Authorization: LSAT` header, confirms Aperture forwards through,
  confirms the gateway returns a valid `Receipt`). This is a multi-part
  undertaking closer to "stand up a second, more complex network from
  scratch" than "extend the Stage 1 template" — deliberately not
  started until Stage 1's own correctness is established first (no
  point debugging channel funding if the L402 client-side logic itself
  has a bug).
- **The simulated gate stands in for exactly one thing**: real
  cryptographic verification that a preimage hashes to the invoice's
  own payment hash and the macaroon is genuinely valid. Everything
  else in the flow — challenge parsing, header construction, receipt
  signing, embedding, and independent third-party re-verification — is
  exercised for real, against persona's own actual code, not mocked.

## Tasks

### Stage 1: simulated template (done)
- [x] Build a simulated L402 gate (`httptest`) fronting the real
      `NewGatewayHandler` — 402 on an unpaid/wrong request, forwards
      through on a request whose `Authorization` header matches the
      expected paid header — `internal/payment/template_test.go`
- [x] `TestTemplate_AttestationCostGatewayFlow`: the full real-world
      path in one test — fix a pending claim's content/timestamp,
      request a receipt, receive and parse a real L402 402 challenge,
      retry with a paid header, receive and verify a real signed
      `Receipt`, finalize the attestation event with it embedded, and
      — as a genuinely independent third-party verifier would — fetch
      the event, verify its signature from scratch, extract the
      receipt, and re-verify it against an independently recomputed
      claim hash (not reusing the original `pending`/`hash` values)

### Stage 2: real Aperture + Lightning regtest (deferred, not started)
- [ ] `git clone` `lnd` (`cmd/lnd`, `cmd/lncli`) and `aperture`
      (`cmd/aperture`) into a scratch directory each; `go build` from
      within each cloned tree (not `go install <module>@version` — see
      Decisions above for why that doesn't work for either). Re-check
      the real Go-toolchain requirement at that time, not from this
      scoping pass's numbers (both move release to release). Requires
      either explicit approval for building freshly-cloned external
      code in a sandboxed session, or doing this step in a less
      restricted environment.
- [ ] Stand up two `lnd` regtest nodes against the existing `btcd`
      regtest backend; open and fund a channel between them
- [ ] Configure `aperture` against one node's macaroon/TLS, registered
      as a backend service pointing at persona's own reference gateway
      handler (the exact D005 shape, now actually running)
- [ ] Drive the full flow for real end to end; confirm Aperture's real
      gating, not a simulation, actually protects the gateway
- [ ] Not started — explicitly deferred given the real setup cost
      (two funded nodes, a real channel, real Aperture config, a build
      step with its own environment-specific approval hurdle); same
      "defer the harder cross-domain slice" discipline [[A006]] applied
      to its own escrow-settlement follow-up

## Log

2026-09-30 — Filed and Stage 1 implemented in the same pass. Checked
Stage 2's real feasibility before deciding to defer it, rather than
assuming: `lnd`/`aperture` both resolve as real Go modules, but `lnd`
requires Go ≥1.25.13 — confirmed the same toolchain-bump risk pattern
`hashicorp/vault/shamir` taught in A003, this time for a dependency
that's actually needed rather than avoidable, mitigated the same way
`btcd` already was (standalone external binaries, never imported into
persona's own module). Named Stage 2's real requirements concretely
(two funded nodes, a real channel, real Aperture config, orchestration
code) rather than leaving it vague, and deferred it deliberately —
Stage 1's simulated template proves persona's own code is correct at
effectively no infrastructure cost, which is the actual prerequisite
before spending the much larger effort Stage 2 would need. Stage 1's
own test passed on the first run; full repo build/vet/test clean.

2026-09-30 — **Re-scoped Stage 2 with real, hands-on checks** (asked to
start scoping this while A007 waits on a reviewer). Corrected a real
mistake in the original scoping: `go install <module>@version` — the
approach that worked cleanly for `btcd` — does **not** work for `lnd`
or `aperture`. Actually attempted it (not just `go list -m`) and hit
Go's own refusal to honor `replace` directives in a remotely-installed
module's `go.mod`, for every `lnd`/`aperture` version tried. Confirmed
by cloning `lnd`'s real source (small: ~7MB shallow) that the
project's own intended build path is `git clone` + build-from-within,
not `go install` — the actual `go build` step itself was then blocked
by this session's own sandbox (external-code-execution guard), so the
full build wasn't completed in this pass, but the install-method
correction itself is confirmed. Also re-checked toolchain requirements
directly: they've moved since the first scoping pass (`lnd` now needs
Go ≥1.25.13-1.26.8 depending on exact version, `aperture` ≥1.26.0) —
recorded as drifting, re-check-at-implementation-time numbers, not
fixed facts. Task list updated to reflect the corrected build approach
and the environment-specific approval hurdle. Still deliberately not
started beyond this scoping — the real infrastructure cost (two funded
nodes, a channel, real Aperture config) is unchanged and still the
reason to defer.
