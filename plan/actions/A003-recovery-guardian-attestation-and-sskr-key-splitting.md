---
id: A003
title: Recovery — guardian attestation and SSKR key splitting
status: PLANNING
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

- **Shamir secret sharing via `hashicorp/vault/shamir`, not a
  from-scratch or bech32-SSKR-format implementation.** D001 cites SSKR
  (BlockchainCommons' bech32-encoded Shamir variant) as prior art, but
  the bech32 wrapping is a human-transcription convenience for
  paper-backup workflows — not needed here, since shares are never
  meant to be read or typed by a human; they move guardian-to-owner
  only as NIP-44-encrypted event payloads. `hashicorp/vault/shamir` is
  a small, dependency-light, widely-used Go implementation of the same
  underlying GF(256) Shamir scheme; reusing it avoids hand-rolling
  finite-field arithmetic for no benefit. Revisit if a future action
  actually needs human-transcribable paper backups.
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
  as a NIP-44-encrypted direct message event (kind 4-style, standard
  Nostr private-message shape) — not a new transport, reusing an
  existing NIP so generic Nostr clients could in principle also
  deliver it. This action implements the split, the per-guardian
  encrypt, and the decrypt-on-the-guardian's-side steps; it does not
  implement a DM inbox/client, only building and reading the event.
- **Combine requires the actual threshold of decrypted shares as
  local input — no network code reconstructs a key.** Reconstruction
  happens entirely offline, given `m` shares a caller has already
  collected out-of-band (however they got returned) — this action
  will not implement any guardian-contact/request-relay workflow.

## Tasks

### Key splitting
- [ ] Split a `identity.Persona` private key into `n` Shamir shares
      with threshold `m`, via `hashicorp/vault/shamir`
- [ ] Combine `m` (or more) shares back into the original private key
      and reject/error clearly on fewer than `m`

### Guardian designation
- [ ] Encrypt each share to its guardian's pubkey (NIP-44) and build
      the delivery event
- [ ] Decrypt a received share event, given the guardian's own keypair
- [ ] Construct and verify `net.persona.core.recovery_guardian` claims
      (reuses A001's `New`/`Verify`/`ValidateClaimType` — no new
      validation code needed beyond the claim_value shape)

### Recovery confirmation
- [ ] Construct and verify `net.persona.core.recovery_confirm` claims
- [ ] Given a fetched set of `recovery_confirm` events for a subject,
      group by `group_id:recovery_nonce` and report whether the
      designated threshold has been met (count of matching, uniquely-
      attesting guardians against the `recovery_guardian` set's own
      `m`)

### Tests
- [ ] Split then combine with exactly `m` shares reconstructs the
      original private key
- [ ] Combine fails/errors with fewer than `m` shares
- [ ] Combine with `m` shares from a set of `n > m` (not just the
      first `m` generated) still reconstructs correctly
- [ ] NIP-44 encrypt/decrypt round trip for a share, wrong-recipient
      decrypt fails
- [ ] `recovery_guardian`/`recovery_confirm` claim construction and
      verification round-trip (reusing A001's existing Verify path)
- [ ] Threshold-met detection: quorum reached at exactly `m` distinct
      guardians, not reached at `m-1`, and confirms for a *different*
      `group_id`/`recovery_nonce` don't count toward the current
      attempt's quorum

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
