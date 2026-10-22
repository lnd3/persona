---
id: A015
title: Web-based attestation publisher (NIP-07, not extension-specific)
status: DONE
design: D001
project: P001
created: 2026-10-22
updated: 2026-10-22
---

## Context

The operator asked which of persona's other features (attestation,
trust, recovery, payment/escrow) fit the browser extension. Checked
`internal/attestation/attestation.go` directly rather than guessing:
building an attestation is just constructing one specific event shape
(`kind: 3300`, tags `p`/`claim_type`/`claim_value`, content = a
human-readable summary) and signing it — `window.nostr.signEvent()`,
which A012's extension already implements exactly, generically, with
no attestation-specific code at all. Nothing about this needs a new
extension capability.

The real missing piece is a client-side **publisher**: somewhere that
builds the right event object, calls `signEvent`, and actually sends
it to a real relay (and, for the read side, fetches existing
attestations about a subject) — none of which the extension does
today (it only signs; it never talks to a relay itself). This action
builds that publisher as a **public page**, not extension code —
same shape as A014's `try-signer.html`, and for the same reason: NIP-07
is a generic interface, so this should work with *any* compliant
signer, not assume persona's own extension specifically. That's
D001's own interop-scope resolution applied here directly.

## Decisions made here

- **A public page (`/attest`), not an extension feature.** Confirmed
  concretely, not assumed: `attestation.New` in Go needs nothing from
  a signer beyond `Sign(privateKey)` — the exact shape `signEvent`
  already covers. Building this into the extension's own popup would
  only make it persona-extension-specific for no real reason; a page
  using `window.nostr` works with any NIP-07 signer.
- **Claim-type validation ported, not re-invented.** The reverse-domain
  regex in `attestation.go` (`claimTypeRE`) gets a direct client-side
  port for instant feedback — same rule, not a new one, and not a
  security boundary either way (relays don't enforce it; Go's own
  `Verify` is the real gate). Checked the exact pattern rather than
  approximating it.
- **Relay interaction via `nostr-tools` from `esm.sh`**, same CDN-ESM
  pattern A014 already established and verified — not a new
  dependency decision, reusing one already made.
- **Publish to the same two public relays this project already uses
  in its own Go tests** (`wss://nos.lol` primary, `wss://relay.damus.io`
  as a second target) — matches `attestation.Publish`'s own
  multi-relay-for-redundancy design, and `relay.damus.io`'s own
  documented flakiness in this project's history means a second
  target is a real need, not caution for its own sake.
- **Read side is a simple lookup, not a trust computation.** Fetching
  and listing existing attestations about a pubkey (mirroring
  `FetchForSubject`) is in scope; computing or displaying *trust*
  (A002's relative, decay-by-depth graph walk) is explicitly not —
  named in the earlier triage as its own, bigger, differently-shaped
  feature (a lookup tool across relays, not a signing action).
- **Explicitly out of scope for this action**: any extension-code
  change (none needed); a `claim_type` registry/search UI (D001
  already resolved this as an open, non-authoritative convention, not
  a lookup service); recovery-claim UI (`recovery_guardian`/
  `recovery_confirm` are claim types too, but scoping that is its own
  feature per the earlier triage, not bundled in here).

## Tasks

### Scoping (this pass)
- [x] Confirm attestation creation needs nothing new from the
      extension — checked `attestation.go` directly
- [x] Decide where this lives (public page, not extension code) and
      why
- [x] Name the real new work: relay publish/fetch from the browser,
      which nothing client-side does today

### Build (done)
- [x] `site/attest.html`: form for subject npub/pubkey, `claim_type`
      (client-side regex validation matching `attestation.go`
      exactly), `claim_value`, a human-readable summary
- [x] Build the exact unsigned event object (`kind: 3300`, the three
      tags, `content`), call `window.nostr.signEvent()`
- [x] Publish the signed event to real relays via `nostr-tools`'
      relay pool (`esm.sh` import, A014's own pattern)
- [x] A simple read side: fetch and list existing attestations about
      a given subject pubkey from the same relays
- [x] Wire a real route in `persona-web` (`servePage`, A014's own
      helper) — `/attest`
- [x] Real interop test, same discipline as A012/A014: sign via the
      page (with the real extension, in a real Chromium), publish to
      a real relay, fetch it back and verify with persona's own Go
      `attestation.Verify` — not just `CheckSignature`, since this is
      specifically the attestation shape, not a generic event

## Log

2026-10-22 — Filed and scoped in the same pass, following this
project's own precedent (A006, A008, A012) of scoping before
building. Real finding, not assumed: the extension itself needs zero
new code — `signEvent` already covers exactly what attestation
creation needs. The actual gap is a relay-talking publisher, scoped
as a public page (`/attest`) rather than extension code, for the same
interop reason A014's `try-signer.html` was built that way. Nothing
implemented yet.

2026-10-22 (later) — **Built and verified end to end.** Confirmed
`nostr-tools`' `SimplePool` API directly before writing the page
(`publish(relays, event)`, `querySync(relays, filter)`) with a live
Node round trip against a real relay first — not assumed from memory.
`site/attest.html`: builds the exact `kind: 3300` event shape,
ported `claimTypeRE` verbatim for instant client-side feedback, signs
via `window.nostr.signEvent`, publishes to `wss://nos.lol` +
`wss://relay.damus.io` (treats any relay succeeding as a pass — one
relay's own documented flakiness in this project's history means
requiring all of them would make the page needlessly fragile), and a
read side that queries the same relays for existing attestations
about a subject and lists them. Wired the real `/attest` route in
`persona-web` via A014's own `servePage` helper.

**Full real interop chain, not simulated anywhere**: created a real
persona in the real extension (real Chromium), filled in the real
page, signed through a real approval popup, published — `relay.damus.io`
hit its known 503 flakiness again during this exact run, confirming
it's a real, persistent relay issue and not a one-off; `nos.lol`
carried the publish, exactly the redundancy this Decision was made
for. Fetched the attestation back via the page's own lookup (found
it), then separately verified the exact signed JSON with persona's
own Go `attestation.Verify` (the actual claim-shape check — attester/
subject/claim_type/claim_value/summary all extracted correctly) —
not just a generic signature check. A page-built, extension-signed,
relay-published event is genuinely indistinguishable from a
Go-signed one, the same interop bar every other NIP-07 piece in this
project has been held to.

Cleaned up: removed the throwaway Go verification file, killed all
test Chromium/`persona-web` processes, `go build/vet/test` clean. No
extension changes were needed, confirming this scoping pass's own
central finding. A015 moved to `DONE`. Not yet deployed to `bh2`.
