# persona

A persistent, keypair-anchored identity needing no institutional
countersignature — see `README.md` and `plan/theses/T001-*.md` for the
full picture.

## Validate before committing

```bash
./deps/lplan/bin/plan validate ./plan
./deps/lplan/bin/plan generate-index ./plan
```

## Relationship to cinder and EphemNet

Shares `cinder`'s architectural philosophy (capability keys, no
accounts, macaroon-authenticated ownership, L402 payment) but has no
runtime dependency on `cinder`'s code or API — not a `cinder`
extension or companion. Pairs with `EphemNet` for self-hosting a
relay behind a home NAT, but doesn't depend on it either — public
Nostr relays and self-run infrastructure both remain open options
(see D001's Open Questions).

## The core discipline this inherits

- **Sybil resistance must stay a real economic cost, never proof-of-
  personhood/biometrics.** The no-gatekeeper mission this whole
  product group is built on directly rules out centralizing,
  coercive identity-verification mechanisms, however effective they
  are at solving sybil resistance technically.
- **Trust is always relative to the viewer, never a global score.**
  Don't build a canonical reputation number anyone could capture by
  controlling it — every trust computation is relative to whoever's
  asking.
- **Recovery reuses the attestation primitive.** Don't add a second,
  separate system for guardian-based recovery — `recovery_guardian`/
  `recovery_confirm` are claim types, the same mechanism as any other
  attestation.
- **Never claim the recovery mechanism eliminates the key-loss risk.**
  It mitigates it. Any threshold-guardian scheme has its own real
  attack surface (coercing or colluding against a threshold of
  someone's guardians) — state this honestly wherever the mechanism
  is described, don't market around it.
