# persona Plan Index

*Last updated: 2026-09-29 22:29:33 UTC*

Status: `IDEA` · `PLANNING` · `IN_PROGRESS` · `BLOCKED` · `DONE` · `DEFERRED` · `CANCELLED`

---

## Theses

| ID | Title | Status | Conviction |
| --- | --- | --- | --- |
| [T001](theses/T001-nostr-based-persona-attestation-recovery.md) | A persistent, keypair-anchored persona identity — built on Nostr's existing primitives, with economic sybil-resistance and attestation-based recovery — needs no institutional countersignature | HELD | 6 |

---

## Projects

| ID | Title | Status | Priority | Key Open Work |
| --- | --- | --- | --- | --- |
| [P001](projects/P001-nostr-based-persona-identity-attestation-and-recovery.md) | Nostr-based persona identity, attestation, and recovery | IN_PROGRESS | MEDIUM | TBD |

---

## Designs

| ID | Title | Status | Project | Doc |
| --- | --- | --- | --- | --- |
| [D001](designs/D001-persona-attestation-recovery-protocol.md) | Persona identity protocol — Nostr-based keys, peer attestation, social recovery | DONE | P001 | (link if applicable) |

---

## Actions

| ID | Title | Status | Design | Open Tasks |
| --- | --- | --- | --- | --- |
| [A001](actions/A001-core-identity-and-attestation-event-layer.md) | Core identity and attestation event layer | DONE | D001 | TBD |
| [A002](actions/A002-relative-trust-computation.md) | Relative trust computation (web-of-trust weighting) | DONE | D001 | TBD |
| [A003](actions/A003-recovery-guardian-attestation-and-sskr-key-splitting.md) | Recovery — guardian attestation and SSKR key splitting | DONE | D001 | TBD |
| [A004](actions/A004-payment-integration-attestation-cost-gateway.md) | Payment integration — attestation-cost gateway (L402/Aperture) | DONE | D001 | TBD |
| [A005](actions/A005-bonding-slashing-dispute-resolution.md) | Bonding/slashing dispute resolution (dispute type 1 only) | DONE | D001 | TBD |
| [A006](actions/A006-bitcoin-escrow-settlement.md) | Bitcoin escrow settlement (script construction, funding, on-chain verification) | PLANNING | D001 | TBD |