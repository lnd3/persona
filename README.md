# persona

A persistent, keypair-anchored identity that needs no institutional
countersignature — built on Nostr's existing key and event primitives,
with a peer-attestation layer for portable reputation (economic
sybil-resistance via Lightning, not biometrics) and social recovery
for lost or stolen keys (SSKR-based, authorized via the same
attestation primitive that builds reputation).

Built on the same architectural philosophy as
[cinder](https://cinderapps.org) — capability-based access, no
accounts, macaroon-authenticated ownership, L402/Lightning payment —
without any runtime dependency on cinder's own code. Pairs naturally
with [EphemNet](https://eph.network), which makes self-hosting a
persona relay behind a home NAT concretely possible.

See `plan/theses/T001-*.md` for the full reasoning and
`plan/designs/D001-*.md` for the implementation-direction spec.

Planned with [lplan](https://github.com/lnd3/lplan) — see `plan/README.md`.

## License

MIT — see [LICENSE](LICENSE).
