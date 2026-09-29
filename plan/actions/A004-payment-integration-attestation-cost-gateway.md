---
id: A004
title: Payment integration — attestation-cost gateway (L402/Aperture)
status: DONE
design: D001
project: P001
created: 2026-09-29
updated: 2026-09-29
---

## Context

Implements D001's Payment integration section: a per-operator
"attestation-cost gateway" — a minimal backend fronted by
`lightninglabs/aperture` (reusing `cinder`'s D005 reference
architecture rather than inventing a second L402 pattern), whose only
job is to accept an already-L402-paid request and mint a small signed
receipt. The attester embeds that receipt in the attestation event's
tags when publishing. No macaroon, invoice, or preimage code lives in
persona's own client or relay software — same as `cinder` itself
carries none, per D005's own "cinder contains no macaroon code, no
Lightning client code" result.

Builds directly on:
- [[A001]]'s `internal/identity` (a gateway operator is just another
  Nostr keypair — no new identity concept needed) and
  `internal/attestation` (`New`, which this action extends with an
  optional receipt tag; `Verify`, which gains a receipt-signature
  check as a separate, opt-in step)
- `cinder`'s D005 (`persona:D001`'s own dependency, referenced there)
  for the gateway-deployment shape: Aperture in front, discrete
  pricing tier, metered prepaid-bundle draw-down for anti-replay,
  hosted Lightning infra rather than self-hosted

**Explicitly out of scope for this action**:
- Actually executing a Lightning payment (talking to a wallet,
  handling the invoice) — the same posture D005 takes for cinder: an
  attester's own wallet pays the invoice L402 presents; this action's
  client-side code stops at "here's the paid macaroon+preimage, attach
  it to the request," never at "here's how to acquire sats"
- Running Aperture itself, or writing its config — D005 already
  specifies that shape and it transfers directly; this action only
  builds the minimal backend Aperture fronts (the receipt-minting
  service) and the client/verify code around it
- Choosing a specific hosted Lightning provider — D005 leaves this
  open too (Voltage named as a leading candidate, not formally chosen)
  and it's an operational decision independent of this action's code
- Sign-in billing's own endpoint — D001 states it "follows the
  identical shape" but as a relying site's own concern (their own
  Aperture instance, their own login-verification endpoint), not
  something persona's own codebase implements; this action builds the
  reusable *pattern* (receipt mint + verify) that a sign-in billing
  endpoint would also reuse, not a second implementation of it
- Choosing which gateway operators' receipts any given verifier trusts
  — per D001's relative-trust posture, that's a verifier-side policy
  decision, structurally identical to how [[A002]] left seed-set
  selection to the caller

## Decisions made here (not in D001 — concrete choices this action owns)

- **Receipt binds to the claim's content hash, not the final signed
  event.** The event's own Nostr `id` is a hash over the *complete*
  event including its tags — so a receipt can't reference "this
  event's id" without being circular (the receipt tag would need to
  exist before the id it references could be computed). Instead, the
  gateway signs over a deterministic hash of the claim's immutable
  content fields — `sha256(attester_pubkey | subject_pubkey |
  claim_type | claim_value | timestamp)` — computed identically by the
  attester (before requesting payment) and by any verifier (after
  fetching the published event, from fields already in `Claim`). This
  is exactly analogous to D005's anti-replay reasoning but adapted to
  a fully decentralized publish path with no central write server to
  hold state: the receipt is cryptographically tied to *this specific
  claim's content*, so it can't be replayed against a different claim,
  without needing any gateway-side request/nonce bookkeeping.
  **Refined at implementation time**: each field is length-prefixed
  before hashing (`len(field):field|...`), not simply joined with a
  bare separator — otherwise two different (attester, claim_type)
  pairs whose concatenated bytes happen to collide at a field boundary
  could hash identically (e.g. attester `"ab"`+type `"c"` vs. attester
  `"a"`+type `"bc"`). A real hash-hygiene detail the Decisions section
  glossed over, not a design change.
- **Receipt is itself a small, gateway-signed struct, not a Nostr
  event.** `{content_hash, gateway_pubkey, issued_at, signature}`,
  Schnorr-signed with the gateway operator's own Nostr private key
  (reusing `identity.Persona` and go-nostr's existing signing —
  gateway operators are just personas, no new keypair format). Kept
  as a compact struct (not a full Nostr event) since it's only ever
  embedded inside the attestation event's own tags, never published
  standalone.
- **Embedded as a single opaque tag, not decomposed into multiple
  tags.** `["payment_receipt", base64(json(receipt))]` on the
  attestation event — one tag, matching the existing `claim_type`/
  `claim_value` tag convention from A001, kept opaque (base64 JSON)
  rather than spread across several tags so a generic Nostr client
  doesn't need to understand its internal shape to pass it through
  unmodified, consistent with D001's interop-scope resolution.
- **Gateway backend HTTP surface: one endpoint, `POST /receipt`,
  request body = the content hash (hex), fronted entirely by
  Aperture** — mirrors D005's "cinder's paid listener does no payment
  verification itself" shape exactly: this action's gateway code
  trusts that any request reaching it already passed Aperture's L402
  check, and does nothing but sign and return a receipt for the hash
  it's given. No pricing, invoicing, or macaroon logic in persona's
  own code at all, same result D005 reached for cinder.
- **L402 client-side handling scoped to parsing/retrying, not paying**:
  this action implements recognizing a 402 challenge
  (`WWW-Authenticate: LSAT macaroon="...", invoice="..."`) and retrying
  the request with an `Authorization: LSAT <macaroon>:<preimage>`
  header once a caller supplies the paid preimage — it does not
  implement invoice payment itself. A concrete Go L402-client library
  choice is deliberately left open here (none evaluated yet) rather
  than assumed, the same honesty A001 applied to the `kind`-number
  decision before checking the live registry — worth a short
  reuse-vs-build pass at actual implementation time rather than
  picking one now.
- **Receipt verification is opt-in and orthogonal to `Verify`**, not
  folded into `attestation.Verify` itself: an attestation event with
  no `payment_receipt` tag remains structurally valid (verifiable
  signature, valid claim_type) — whether a *particular verifier*
  requires a trusted gateway's receipt (or an equally-valid bonded
  stake instead, per D001) before weighting the claim is exactly the
  per-verifier trust decision D001 describes, not something baked into
  the shared `Verify` path every caller uses.
- **Implemented as a `PendingClaim` type in the new `internal/payment`
  package, not literally as an `attestation.New` extension.** `Verify`
  and the tags convention are still reused directly, but embedding a
  tag requires it to exist *before* signing, and the payment package
  already imports `attestation` (for `Kind`/`ValidateClaimType`) — so
  `attestation` importing back would be a cycle. `PendingClaim` fixes
  the claim's content and timestamp once, exposes `Hash()` for
  requesting a receipt, and `Finalize(attester, summary, *Receipt)`
  signs the actual event at that same fixed timestamp — this is what
  actually keeps the receipt's hash and the final event's timestamp
  consistent, a real correctness requirement the Decisions section
  named implicitly ("computed identically... before publishing... after
  fetching") without spelling out how a caller keeps that timestamp
  from drifting between the two steps.

## Tasks

### Receipt type and verification
- [x] Define the `Receipt` struct (`content_hash`, `gateway_pubkey`,
      `issued_at`, `signature`) and its base64-JSON tag encoding —
      `internal/payment/receipt.go`, `EncodeReceipt`/`DecodeReceipt`
- [x] `ContentHash(...)` — deterministic, length-prefixed hash over
      attester/subject/claim_type/claim_value/timestamp, computable
      both before publishing (by the attester, via `PendingClaim.Hash`)
      and after fetching (by a verifier, via `ClaimContentHash`)
- [x] `SignReceipt(gateway identity.Persona, hash) (Receipt, error)`
- [x] `VerifyReceipt(receipt, expectedHash) error` — signature check
      plus hash match; does not decide whether the caller *trusts*
      `receipt.GatewayPubkey`, only whether the receipt is genuine

### Attestation event integration
- [x] `PendingClaim`/`Finalize` (not a literal `attestation.New`
      extension — see Decisions) embeds an optional `*Receipt` as the
      `payment_receipt` tag — `internal/payment/event.go`
- [x] `ExtractReceipt` reads a `payment_receipt` tag back into a
      `Receipt`, `ok=false` (no error) when absent — receipt presence
      is opt-in, not required for `attestation.Verify` to succeed

### Reference gateway backend
- [x] Minimal `POST /receipt` HTTP handler: read a content-hash hex
      body, sign it with the gateway's own key, return the JSON
      `Receipt` — no payment logic — `internal/payment/server.go`
      `NewGatewayHandler`
- [x] Documented the Aperture registration this expects directly in
      `NewGatewayHandler`'s doc comment, referencing D005's own
      `aperturecli services create` shape rather than re-deriving it

### L402 client plumbing
- [x] Parse a `402`/`WWW-Authenticate: LSAT ...` challenge into
      macaroon + invoice — `internal/payment/l402.go`
      `ParseChallenge`/`ChallengeFromResponse`
- [x] Build the retry request's `Authorization: LSAT <macaroon>:
      <preimage>` header, given an already-obtained preimage —
      `AuthorizationHeader`
- [x] Reuse-vs-build note: no Go L402 client library was actually
      needed for this action's scope — parsing a standard HTTP
      402/`WWW-Authenticate` response and formatting an `Authorization`
      header requires nothing beyond the standard library
      (`net/http`, `regexp`); the harder half (talking to a Lightning
      wallet to actually pay an invoice) remains genuinely deferred and
      still has no library chosen, since it's still out of this
      action's scope entirely

### Tests
- [x] `ContentHash` is deterministic and order-sensitive —
      `TestContentHashDeterministicAndOrderSensitive`
- [x] Sign/verify receipt round trip; tampered signature or mismatched
      hash rejected — `TestSignVerifyReceiptRoundTrip`,
      `TestVerifyReceiptRejectsTamperedSignature`,
      `TestVerifyReceiptRejectsMismatchedHash`
- [x] Attestation event with an embedded receipt round-trips through
      `Finalize`/`Verify`/`ExtractReceipt` correctly —
      `TestFinalizeWithReceiptRoundTrip` (also checks the verifier's
      independently recomputed hash still matches the receipt)
- [x] An attestation event with *no* receipt tag still verifies
      structurally — `TestFinalizeWithoutReceiptStillVerifies`
- [x] 402-challenge parsing against a synthetic `WWW-Authenticate`
      header; malformed challenge rejected with a clear error —
      `TestParseChallenge`, `TestParseChallengeRejectsMalformed`,
      `TestChallengeFromResponse`,
      `TestChallengeFromResponseRejectsNon402`
- [x] Reference gateway handler: valid hash → signed receipt whose
      `VerifyReceipt` passes; malformed body → clear error, no partial
      receipt issued — `TestGatewayHandlerSignsValidHash`,
      `TestGatewayHandlerRejectsMalformedBody`,
      `TestGatewayHandlerRejectsWrongMethod`

## Log

2026-09-29 — Action filed against D001's Payment integration section,
reusing `cinder`'s D005 reference architecture (Aperture-fronted,
discrete pricing tier, metered prepaid-bundle anti-replay) rather than
inventing a second L402 pattern. Made the concrete choices D001 left
as architecture-only: the receipt binds to a deterministic hash of the
claim's own content fields (not the final event id, which would be
circular), travels as a single opaque base64-JSON tag, and is checked
by a verifier only if that verifier chooses to require it — receipt
verification is deliberately kept orthogonal to `attestation.Verify`
itself, the same "per-verifier decision" posture D001 already applies
everywhere else. Left one real implementation decision open rather
than guessing: which Go L402 client library to build the payment-retry
plumbing on, since none has been evaluated yet — flagged for a short
reuse-vs-build pass at implementation time, not resolved here.

2026-09-29 — Implemented. New `internal/payment` package, four files:
`receipt.go` (`Receipt` type, `ContentHash`/`SignReceipt`/
`VerifyReceipt`, base64-JSON tag encoding), `event.go` (`PendingClaim`/
`Finalize`/`ExtractReceipt` — the attestation-event integration),
`server.go` (`NewGatewayHandler`, the reference gateway backend), and
`l402.go` (challenge parsing, authorization header building). Uses
`btcec/schnorr` directly for receipt signing — the exact same
primitive `go-nostr`'s own `Event.Sign` already uses internally, so no
new signature scheme and no new dependency (it was already an indirect
one; `go mod tidy` just promoted it to direct).

Two things worth flagging as real corrections, not just style choices:
1. **Event integration couldn't literally extend `attestation.New`**
   as the Decisions section originally sketched — `internal/payment`
   already imports `internal/attestation` (for `Kind`/
   `ValidateClaimType`), so the reverse import would cycle. Built
   `PendingClaim`/`Finalize` instead: fixes a claim's content and
   timestamp once, so the hash a gateway signs and the final event's
   own timestamp can't drift apart between the two round-trip steps —
   a real correctness requirement the original Decisions text implied
   ("computed identically... before... after") without actually
   solving how a caller keeps the two in sync.
2. **`ContentHash` length-prefixes each field before hashing**, not a
   bare `|`-joined concatenation as first sketched — closes a
   field-boundary collision (two different claims whose bytes happen
   to concatenate identically) that a plain join would have left open.

The L402 client-library question flagged as open turned out to need no
library at all for this action's actual scope: parsing a 402/
`WWW-Authenticate` response and building an `Authorization` header is
plain `net/http`/`regexp`, nothing more. Paying an invoice (the
genuinely deferred half) still has no library chosen, since it's still
out of scope entirely. All tests pass (16 new tests), full repo
build/vet/test clean. Everything this action scoped is done — moved to
DONE.
