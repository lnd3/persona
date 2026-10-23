# Reflection

Append-only log of learnings, gotchas, and patterns discovered during development.
Not session notes — these are stable insights worth carrying forward indefinitely.

Format: `YYYY-MM-DD | CATEGORY | insight`
Categories: GOTCHA · PATTERN · LEARNING · WARNING · DECISION · CONSTRAINT · FINDING

*If an entry needs more than one line, put the summary here and the detail in
`REFLECTION_extension.md` with a matching anchor. This file stays one-liners only.*

---

2026-10-23 | PATTERN | Any new NIP-07-signable feature (ordinary attestations, guardian designation/vouching) needs zero extension changes and fits a public page reusing A015's pattern (build event, sign via window.nostr, publish via a fresh nostr-tools SimplePool); anything needing raw private-key material (NIP-44 ECDH for share encryption, Shamir Split) must live in the extension's trusted popup and can never cross window.nostr — split features along this custody boundary, not along feature convenience.
2026-10-23 | GOTCHA | nostr-tools' SimplePool fails unpredictably when one instance is reused across back-to-back sequential .publish() calls in the same page session; create a fresh SimplePool per publish instead of reusing a module-level one.
2026-10-23 | GOTCHA | relay.damus.io has caused real flakiness repeatedly across this project (4+ times); relay.primal.net and nos.lol have proven reliable in isolated repros — prefer them for anything doing sequential multi-publish.
2026-10-23 | GOTCHA | WebCrypto's AES-GCM auth-tag failure (OperationError) has an empty .message — code that does errorBox.textContent = err.message on a decrypt() failure silently shows a blank error on a wrong passphrase; always catch and throw an explicit message instead.
2026-10-23 | WARNING | A first successful test run can mask real logic bugs when test data happens to be convenient (e.g. reusing the owner as its own guardian collapses "signer vs. subject" bugs and "query by #p only" bugs into the same coincidentally-correct answer) — re-test with genuinely distinct identities before trusting a green run on anything involving subject/signer distinctions.
2026-10-23 | FINDING | No identity-key-revocation mechanism exists anywhere in the project (internal/recovery, internal/attestation, or D001) — guardian recovery as designed only addresses key loss, not theft, since reconstructing the same key via Combine does nothing against a thief's copy. Filed as A017.
