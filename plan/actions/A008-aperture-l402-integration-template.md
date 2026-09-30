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
  assumed.** `lnd` and `aperture` both resolve as real Go modules
  (`go list -m` confirms `v0.21.x`/`v0.5.0` respectively), so the
  *shape* is feasible — but `lnd`'s own `go.mod` requires
  **Go ≥1.25.13**, which would force a toolchain bump if imported
  directly into persona's own module (the exact `hashicorp/vault/
  shamir` lesson from [[A003]], recurring here for a genuinely-needed
  dependency this time, not an avoidable one). The mitigation is the
  same shape [[A006]]/[[A007]] already used for `btcd`: build `lnd`,
  `lncli`, and `aperture` as **standalone external binaries** (`go
  install` into an isolated `GOBIN`) and drive them over gRPC/HTTP/
  shell from test code — never importing their Go packages into
  persona's own `go.mod`, so persona's own toolchain requirement stays
  untouched regardless of what `lnd` itself needs.
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
- [ ] Build `lnd`, `lncli`, and `aperture` as standalone external
      binaries (isolated `GOBIN`, never imported into persona's own
      `go.mod`)
- [ ] Stand up two `lnd` regtest nodes against the existing `btcd`
      regtest backend; open and fund a channel between them
- [ ] Configure `aperture` against one node's macaroon/TLS, registered
      as a backend service pointing at persona's own reference gateway
      handler (the exact D005 shape, now actually running)
- [ ] Drive the full flow for real end to end; confirm Aperture's real
      gating, not a simulation, actually protects the gateway
- [ ] Not started — explicitly deferred given the real setup cost
      (two funded nodes, a real channel, real Aperture config); same
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
