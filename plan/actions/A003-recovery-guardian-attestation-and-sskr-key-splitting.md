---
id: A003
title: Recovery — guardian attestation and SSKR key splitting
status: DONE
design: D001
project: P001
created: 2026-09-29
updated: 2026-09-29
---

## Context

Implements D001's Recovery resolution: "SSKR-based key splitting,
authorized via delegation attestation" — chosen over pure SSKR-only
guardianship specifically so guardian designation and recovery
confirmation reuse [[A001]]'s attestation primitive rather than being
a second bolted-on system. Per D001, this recovers the *original*
keypair (a persona keeps the same pubkey through recovery), not a
rotation onto a new one — SSKR reconstructs the actual private key
from a threshold of shares, the attestations are the discovery/
audit/authorization layer around that reconstruction, not a
replacement for it.

Builds directly on:
- [[A001]]'s `internal/identity` (keypair type) and
  `internal/attestation` (`New`/`Verify`, `claim_type` namespacing —
  reused verbatim for the two new claim types this action defines)
- go-nostr's `nip44` package for guardian-share encryption (already a
  transitive dependency, confirmed present in the module cache; no
  new encryption code needed)

**Explicitly out of scope for this action**:
- Any UI/config flow for a user to actually choose guardians or
  initiate a recovery session — this action implements the
  primitives (split/combine, claim construction, encrypted share
  delivery) only, given guardian pubkeys and a threshold as input
- Replay/session-freshness hardening beyond a bare per-recovery nonce
  (D001 doesn't specify this; treated as a judgment call here, see
  Decisions)
- Multi-device/FROST threshold signing — D001 explicitly defers this
  past v1 as a separate future hardening layer, not part of SSKR-based
  recovery
- Any verifier-side policy for *when* to treat a completed recovery as
  legitimate — same "relative to whoever's asking" posture as trust
  scoring in [[A002]]; this action produces the public
  `recovery_confirm` record, it doesn't dictate how a relying party
  reacts to it
- The guardian-collusion residual risk is a property of the scheme
  itself, not something this action's code can close — D001 already
  states this honestly and this action doesn't repeat or soften it

## Decisions made here (not in D001 — concrete choices this action owns)

- **Shamir secret sharing implemented directly, not via
  `hashicorp/vault/shamir` as originally planned, and not a
  bech32-SSKR-format implementation either.** D001 cites SSKR
  (BlockchainCommons' bech32-encoded Shamir variant) as prior art, but
  the bech32 wrapping is a human-transcription convenience for
  paper-backup workflows — not needed here, since shares are never
  meant to be read or typed by a human; they move guardian-to-owner
  only as NIP-44-encrypted event payloads. **Corrected at
  implementation time** (see Log): `hashicorp/vault/shamir` turned out
  not to be an independently versioned module — importing it pulls in
  the entire `hashicorp/vault` repository and forces a Go toolchain
  bump (1.24.1 → 1.25.3), wildly disproportionate for one small,
  dependency-free, standard algorithm. Implemented the same GF(256)
  Shamir construction directly instead (~180 lines, no new
  dependencies). Revisit if a future action actually needs
  human-transcribable paper backups.
- **Two new claim types, both under this project's existing
  `net.persona.core.*` namespace** (matching A002's `trust` edge
  precedent):
  - `net.persona.core.recovery_guardian` — attester = the persona
    owner, subject = a designated guardian's pubkey, `claim_value` =
    `"<threshold>-of-<n>:<group_id>"` (e.g. `"3-of-5:a1b2c3"`), where
    `group_id` is a random hex token generated once per guardian set
    so a verifier can tell which `recovery_guardian` claims belong to
    the same designation round (a persona may redesignate guardians
    over time; old and new sets must not be conflated). The SSKR share
    itself is never embedded in this claim — it's public, discoverable
    information about *who* the guardians are and the threshold, not
    key material.
  - `net.persona.core.recovery_confirm` — attester = a guardian,
    subject = the persona's own (original, being-recovered) pubkey,
    `claim_value` = `"<group_id>:<recovery_nonce>"`, published when
    that guardian agrees to participate in a specific recovery
    attempt. `recovery_nonce` is a random token the person recovering
    generates once per attempt and communicates to guardians alongside
    their share request — this is what lets a verifier distinguish a
    real quorum of confirmations for *this* recovery attempt from
    guardians' confirmations scattered across unrelated past attempts,
    without inventing a heavier session protocol than D001 asked for.
- **Guardian shares travel only as NIP-44-encrypted payloads, never as
  plaintext or as a public claim_value.** The owner splits their
  private key into `n` shares at guardian-designation time, encrypts
  each share to its guardian's pubkey with NIP-44, and publishes each
  as a NIP-44-encrypted delivery event. **Not literally kind 4
  (NIP-04) as originally sketched**: kind 4 implies NIP-04's own
  (different, weaker) encryption scheme, so reusing it with NIP-44
  content would mislead any client that tried to interpret it
  correctly. Minted a dedicated kind, **3301**, adjacent to A001's own
  3300 in the same open gap that decision already identified — same
  "document it openly, don't wait for permission" posture, same
  live-registry-recheck caveat A001 carried. This action implements
  the split, the per-guardian encrypt/build, and the
  decrypt/extract-on-the-guardian's-side steps; it does not implement
  a DM inbox/client or any actual relay delivery, only building and
  reading the event itself.
- **Combine requires the actual threshold of decrypted shares as
  local input — no network code reconstructs a key.** Reconstruction
  happens entirely offline, given `m` shares a caller has already
  collected out-of-band (however they got returned) — this action
  will not implement any guardian-contact/request-relay workflow.
- **Corrected a pre-existing bug in A001's `claim_type` namespacing
  regex, found while implementing this action**: D001's own vocabulary
  already names these claim types with underscores
  (`recovery_guardian`/`recovery_confirm`), but A001's
  `ValidateClaimType` regex only allowed hyphens within a label, so
  the design's own established claim types would have been rejected
  by its own validator. Fixed in `internal/attestation` (not
  duplicated here) to accept both `-` and `_` in a label — a one-line
  regex fix, but worth noting since it affects every future claim_type
  choice, not just this action's two.

## Tasks

### Key splitting
- [x] Split a private key into `n` Shamir shares with threshold `m` —
      `internal/recovery/shamir.go` `Split` (direct GF(256)
      implementation, not `hashicorp/vault/shamir`; see Decisions)
- [x] Combine `m` (or more) shares back into the original private key
      and reject/error clearly on fewer than `m` — `Combine`,
      `ErrInsufficientShares`

### Guardian designation
- [x] Encrypt each share to its guardian's pubkey (NIP-44) and build
      the delivery event — `internal/recovery/share.go`
      `EncryptShare`/`BuildShareEvent` (kind 3301)
- [x] Decrypt a received share event, given the guardian's own keypair
      — `DecryptShare`/`ExtractShare`
- [x] Construct and verify `net.persona.core.recovery_guardian` claims
      (reuses A001's `New`/`Verify`/`ValidateClaimType`) —
      `internal/recovery/claims.go` `NewGuardianClaim`/
      `ParseGuardianClaim`

### Recovery confirmation
- [x] Construct and verify `net.persona.core.recovery_confirm` claims
      — `NewConfirmClaim`/`ParseConfirmClaim`
- [x] Given a fetched set of `recovery_confirm` events for a subject,
      group by `group_id:recovery_nonce` and report whether the
      designated threshold has been met — `internal/recovery/quorum.go`
      `GuardianSet`/`ThresholdMet`

### Tests
- [x] Split then combine with exactly `m` shares reconstructs the
      original private key — `TestSplitCombineExactThreshold`
- [x] Combine fails/errors with fewer than `m` shares —
      `TestCombineFailsWithFewerThanThreshold`
- [x] Combine with `m` shares from a set of `n > m` (not just the
      first `m` generated) still reconstructs correctly —
      `TestCombineArbitrarySubset`, `TestCombineWithMoreThanThreshold`
- [x] NIP-44 encrypt/decrypt round trip for a share, wrong-recipient
      decrypt fails — `TestEncryptDecryptShareRoundTrip`,
      `TestDecryptShareWrongRecipientFails`, plus the event-level
      equivalents `TestBuildAndExtractShareEventRoundTrip`,
      `TestExtractShareRejectsWrongRecipient`
- [x] `recovery_guardian`/`recovery_confirm` claim construction and
      verification round-trip — `TestGuardianClaimRoundTrip`,
      `TestConfirmClaimRoundTrip`, plus malformed-value rejection tests
- [x] Threshold-met detection: quorum reached at exactly `m` distinct
      guardians, not reached at `m-1`, and confirms for a *different*
      `group_id`/`recovery_nonce` don't count toward the current
      attempt's quorum — `TestThresholdMetAtExactThreshold`,
      `TestThresholdNotMetBelowThreshold`,
      `TestThresholdMetIgnoresDuplicateConfirmFromSameGuardian`,
      `TestThresholdMetIgnoresConfirmFromNonGuardian`,
      `TestThresholdMetIgnoresConfirmForDifferentAttempt`

## Log

2026-09-29 — Action filed against D001's Recovery resolution
(SSKR-based key splitting, authorized via delegation attestation).
Made the concrete choices D001 left unspecified: a plain Shamir
implementation (`hashicorp/vault/shamir`) rather than bech32-SSKR
framing, since shares here are transport-only and never meant for
human transcription; two new `net.persona.core.*` claim types
(`recovery_guardian`, `recovery_confirm`) with a `group_id`/
`recovery_nonce` scheme to disambiguate guardian sets and individual
recovery attempts without a heavier session protocol; NIP-44-encrypted
DM events as the only channel shares travel over. Explicitly scoped
out any guardian-contact workflow, verifier-side recovery policy, and
multi-device/FROST hardening — all separate, either out of scope
entirely or D001-deferred past v1.

2026-09-29 — Implemented. New `internal/recovery` package, four files:
`shamir.go` (GF(256) split/combine), `share.go` (NIP-44 share
encryption plus a `kind: 3301` delivery-event wrapper), `claims.go`
(`recovery_guardian`/`recovery_confirm` construction and parsing), and
`quorum.go` (guardian-set collection, distinct-guardian threshold
counting). Two corrections made along the way, both flagged rather
than silently absorbed:
1. **Shamir library swap.** `hashicorp/vault/shamir` turned out not to
   be independently modularized — `go get` pulled in the entire
   `hashicorp/vault` repository as an indirect dependency and forced a
   Go toolchain bump (1.24.1 → 1.25.3) just to reach one small,
   dependency-free algorithm. Reverted that `go.mod`/`go.sum` change
   and implemented the same GF(256) Shamir construction directly
   instead — no new dependencies, no toolchain bump, `go.sum` only
   gained the lightweight `golang.org/x/crypto` package NIP-44 itself
   already needed.
2. **`ValidateClaimType` regex bug found and fixed.** D001's own
   established claim-type vocabulary uses underscores
   (`recovery_guardian`, `recovery_confirm`), but A001's namespacing
   regex only allowed hyphens within a label — the design's own
   canonical claim types would have failed its own validator. Fixed
   `internal/attestation`'s regex to accept both `-` and `_`, since the
   underscore convention predates this action and shouldn't be renamed
   around a validator bug.

Also built one piece beyond the letter of the original Tasks list: a
concrete `kind: 3301` Nostr event wrapper (`BuildShareEvent`/
`ExtractShare`) around the raw NIP-44 ciphertext, since the Decisions
section already committed to "a NIP-44-encrypted direct message
event," not just an encrypted string — chose a dedicated kind rather
than reusing kind 4 (NIP-04), since kind 4 implies NIP-04's own,
different encryption scheme and would mislead a client that tried to
interpret it per spec. Same live-registry-recheck caveat A001's own
kind-3300 choice carries. All tests pass, full repo `go build`/`go
vet`/`go test ./...` clean. Everything this action scoped is done —
moved to DONE.
