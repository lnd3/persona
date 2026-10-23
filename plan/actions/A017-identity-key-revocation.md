---
id: A017
title: Identity key revocation (stolen-key gap)
status: PLANNING
design: D001
project: P001
created: 2026-10-23
updated: 2026-10-23
---

## Context

Surfaced while explaining `internal/recovery` to the operator
(2026-10-23): checked the actual code and `D001` directly, and found
there is **no revocation mechanism anywhere in this project** —
neither in `internal/recovery`, `internal/attestation`, nor named as
an open question in `D001`. Guardian-based recovery (`A003`)
re-establishes trust in a key after loss, but nothing ever
invalidates the *old* one. If a key was stolen rather than purely
lost (a real, different threat than the one `A003` was scoped for —
see its own Decisions, which only ever says "lost or stolen" in
passing, never designs for the "stolen" half specifically), an
attacker holding a copy can keep acting under that identity in
parallel, indefinitely, with nothing in this protocol ever flagging
it.

This is a real gap, not a hypothetical one — CLAUDE.md's own core
discipline already states "never claim the recovery mechanism
eliminates key-loss risk, it mitigates it," which is honest about the
guardian-collusion risk but doesn't cover this different gap: even a
perfectly-run recovery today does nothing about a thief's copy of the
old key still working.

## Decisions made here

- **The real fork this action has to resolve first, not a detail**:
  does "recovery" mean reconstructing the *exact same* private key
  (what `Combine` does today), or establishing a *new* key as the
  continuation of identity, with the old one marked revoked?
  These have genuinely different security properties against
  *theft* specifically:
  - Reconstructing the same key via `Combine` does nothing against
    theft — a thief's copy and the "recovered" key are
    cryptographically identical; there's no way to tell them apart,
    and no way to stop the thief from also reconstructing it via
    their own copy if they somehow also compromised enough guardian
    shares.
  - Migrating to a *new* key, with the old one explicitly revoked by
    the same guardian-threshold quorum `A003` already built, is the
    only one of the two that actually addresses theft — the thief's
    copy of the old key still cryptographically works, but every
    other persona-aware verifier would see it as revoked and stop
    trusting it.
  - **Not resolved here, named as the real first design question**:
    whether `A003`'s existing `Combine`-based recovery stays as one
    *mode* (for pure loss, no reason to suspect theft) alongside a
    new revoke-and-migrate mode (for suspected theft), or whether
    migration should just become the only path. Both are real,
    defensible designs; picking one needs more thought than this
    scoping pass, not a reflexive "just add both."
- **Revocation should reuse `A003`'s existing guardian-quorum
  machinery, not invent a second one** — same pattern this project
  already follows (recovery itself reuses the attestation primitive
  rather than being a separate system; CLAUDE.md's own stated rule).
  A new claim type (name TBD — candidates: `net.persona.core.
  key_revoked`, `net.persona.core.identity_migrated`) would carry the
  old pubkey and, for the migration mode, the new one — authorized by
  the same threshold of `recovery_guardian`-designated guardians
  `ThresholdMet` already checks for recovery_confirm.
- **A real, necessary consequence this action must also cover, not
  just the claim type itself**: `A002`'s own relative-trust
  computation has no concept of a revoked identity today — it would
  keep weighting claims from a revoked pubkey normally. Closing this
  gap half-way (adding a revocation claim nobody's trust computation
  ever checks) wouldn't actually protect anyone. `internal/trust`
  needs to treat a revoked attester's claims as untrusted (exact
  weighting — zero, or decayed from the revocation timestamp forward
  only — not decided here).
- **Self-revocation (the owner still holds the old key and chooses to
  retire it) is a different, much simpler case worth naming
  separately** — doesn't need guardian quorum at all, since the owner
  can just sign the revocation themselves. The guardian-quorum path is
  specifically for when the owner *can't* sign with the old key
  anymore (lost) but still needs to invalidate it (suspected theft) —
  the harder, actually-new case this action exists for.

## Tasks

### Scoping (this pass)
- [x] Confirm the gap is real — checked `internal/recovery`,
      `internal/attestation`, and `D001` directly, found nothing
- [x] Name the real design fork (reconstruct-same-key vs.
      revoke-and-migrate-to-new-key) rather than assuming one
- [x] Name the real, easy-to-miss consequence: `A002`'s trust
      computation must also change, not just adding a claim type

### Design (deferred, not started)
- [ ] Resolve the reconstruct-vs-migrate fork — real design work, not
      a quick pick
- [ ] Name the actual claim type and its `claim_value` shape
      (old pubkey, optional new pubkey, revocation reason/timestamp)
- [ ] Decide self-revocation's claim shape (simpler, no quorum)
      alongside the guardian-authorized one, since both are real cases
- [ ] Specify exactly how `internal/trust`'s graph walk must change to
      stop weighting a revoked attester's claims

### Build (deferred, not started)
- [ ] Implement the resolved design in `internal/recovery` (or a new
      package, depending on how Design resolves)
- [ ] Update `internal/trust` to honor revocation
- [ ] Real interop test, same discipline as every other claim type:
      publish a real revocation to a real relay, confirm a fresh
      trust computation actually stops weighting the revoked identity

## Log

2026-10-23 — Filed and scoped in the same pass. Found while explaining
`internal/recovery` to the operator, not from a reviewer or a plan
re-read — checked the actual code and `D001` directly rather than
assuming a gap this size would already be covered. Named the real
design fork (reconstruct vs. revoke-and-migrate) as the thing that
actually needs resolving first, and the easy-to-miss consequence that
`A002`'s trust computation has to change too, or a revocation claim
would exist without ever actually being honored by anything. Nothing
implemented yet — this is scoping only.
