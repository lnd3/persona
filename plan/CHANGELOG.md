# Plan Changelog

Append-only record of all status and priority changes.

Format: `YYYY-MM-DD | ID | old_status → new_status | note`

---

2026-10-01 | A010 | NEW → DEFERRED | wisp analytics integration notes; deferred until persona has a web-facing Go service (site is static HTML) — user decision
2026-10-02 | A011 | NEW → PLANNING | Ship locally built images (docker save|load); never build on bh2 — filed from wisp after the bh2 disk incident (user request)
2026-10-02 | A011 | PLANNING → DONE | Pattern implemented and reconciled across cinder/wisp/persona
2026-10-18 | A008 | PLANNING → DONE | Stage 1 simulated L402 template plus Stage 2 real lnd/aperture/L402 regtest integration, driven end to end
2026-10-19 | A010 | DEFERRED → DONE | persona-web now reports to wisp analytics; live, zero errors since
2026-10-19 | A012 | NEW → PLANNING | Filed NIP-07 browser-extension signer (create/approve/sign, encrypted local storage)
2026-10-22 | A012 | PLANNING → DONE | Stage 1 extension built and verified end to end; later this session: build-info footer added, export-nsec feature added, two real blank-error bugs (WebCrypto OperationError has no .message) found and fixed in both the approval flow and export
2026-10-22 | A013 | NEW → DONE | Go toolchain bumped 1.24 → 1.26.9 for a security fix
2026-10-22 | A014 | NEW → DONE | Public live NIP-07 signer test page (/try-signer), real interop test against the built extension
2026-10-22 | A015 | NEW → DONE | Web-based attestation publisher (/attest) — ordinary attestations need zero extension changes, same NIP-07 + relay-pool pattern as A014
2026-10-22 | A016 | NEW → DONE | Full-history sensitive-information audit (gitleaks + manual Nostr/Bitcoin key checks); real finding (author email) fixed going forward and by rewriting all 92 affected commits via git-filter-repo, force-pushed and verified; SECURITY_AUDIT.md published
2026-10-23 | A017 | NEW → PLANNING | Filed: no identity key revocation mechanism exists anywhere in the project; guardian recovery today only addresses key loss, not theft. Real fork named (reconstruct-same-key vs. revoke-and-migrate), not yet resolved; A002's trust computation will also need to honor revocation
2026-10-23 | A018 | NEW → PLANNING | Filed: whether/how guardian designation, vouching, and share delivery fit the NIP-07 extension vs. a public page. Splits along the custody boundary, not feature lines; real publish-path fork named (give extension network access vs. new page-push plumbing), not yet resolved
2026-10-23 | A018 | PLANNING → IN_PROGRESS | Built and verified the page half (/recovery: designation + vouching + status check); two real logic bugs found and fixed (status-query missing recovery_guardian claims; guardian lookup read event signer instead of the "p" tag subject). Extension-side share generation/delivery still blocked on Option A/B and A017's resolution
