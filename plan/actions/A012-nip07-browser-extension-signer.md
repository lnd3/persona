---
id: A012
title: NIP-07 browser-extension signer
status: PLANNING
design: D001
project: P001
created: 2026-10-19
updated: 2026-10-19
---

## Context

Raised by `superplan` via its own steering outbox on 2026-10-05
(logged as a new, unresolved Open Question in D001 at the time, not
scoped): persona's own `internal/identity` already *is* real Nostr
key material — `go-nostr`-generated `secp256k1`, Nostr's x-only
pubkey format, real NIP-19 `Npub()`/`Nsec()` bech32 encoding. NIP-07
is Nostr's own small, widely-adopted standard for browser-extension
signers (`window.nostr.getPublicKey()`/`signEvent()`) — building one
for persona would make every existing Nostr web client already work
with a persona-custodied identity, no client-side integration work
needed anywhere else.

This action resolves that Open Question's first real question — is
this worth scoping and building at all — with "yes," and does the
actual scoping. It does not implement anything; per this project's
own established discipline (A006, A008), a dedicated scoping pass
happens before a build this size starts, especially since this is a
**new kind of surface for persona entirely**: everything built so far
(`internal/*`, `cmd/persona-web`) is server-side Go. This is the
project's first client-side, key-holding, browser-facing component —
P001's own Scope section has flagged "a client application holding
the user's keys across their different personas" as a known
requirement, "not yet designed," since the project was filed. This
action is that design, narrowed to the NIP-07 signer specifically
(the landing page/presentation/download-page half of that same
requirement is explicitly not this action's job — see Decisions).

**Why now, not blocked on anything**: unlike A006/A008's Lightning
work, this needs no payment rail, no `cinder` dependency, no mainnet
gate — it's pure client-side key management and standard-protocol
signing. Scoped now because A007 (the one thing actually blocking
further mainnet-adjacent work) is deliberately parked on external
signal, and this is real, unblocked, zero-financial-risk work that
was already sitting as an unresolved Open Question.

## Decisions made here

- **This extension *is* v1 of P001's "client application holding the
  user's keys" requirement, not a companion to a separate one.**
  Persona has never had a key-holding client before — `internal/
  identity.Persona` is a plain in-memory struct, used so far only
  inside Go tests and `cmd/persona-web`'s read-only landing page. There
  is no existing app for a browser extension to "sync" with. Treating
  the extension itself as where personas actually live (generated or
  imported, held, used to sign) resolves two open requirements with
  one build instead of designing a sync protocol to a client that
  doesn't exist yet.
- **Key import via pasting an `nsec`, not a new sync/transport
  protocol.** V1 lets a user paste a bech32 `nsec` (from anywhere —
  persona's own Go `Persona.Nsec()`, or any other NIP-19-compliant
  source) into the extension, or generate a fresh persona directly in
  the extension itself. This is exactly how `nos2x`/Alby already work,
  it's the smallest real slice, and it avoids inventing a new
  Go-app-to-browser-extension protocol this action doesn't need.
- **Tech stack: TypeScript, Manifest V3, `nostr-tools` for all
  crypto — never hand-rolled.** `nostr-tools` (built on `@noble/
  curves`/`@noble/hashes`) is the de facto reference implementation
  essentially every existing NIP-07 extension already uses for
  BIP340 Schnorr signing and NIP-01 event-id computation — using it
  is the same "don't hand-roll crypto primitives" discipline this
  project has followed since `hashicorp/vault/shamir` in A003 (and
  the dust-threshold *formula*, not primitive, hand-rolled in A006 —
  a deliberately different class of thing). Bundled with `esbuild` or
  `vite` — no framework needed, matching D001's own "deliberately
  simple, single-purpose" design ethos.
- **Chrome/Chromium first, Firefox as a stated fast-follow, not
  simultaneous scope.** Both now support MV3, and `webextension-
  polyfill` could unify the two, but almost all existing NIP-07
  adoption (`nos2x`, Alby) is Chrome-first — matching that reduces
  real unknowns for a first build. Stated as a real scope cut, not an
  oversight: Firefox parity is Stage 5 below, not this pass.
- **Storage is passphrase-encrypted, non-negotiably, from the first
  working version — not deferred as a "v2 hardening" item.**
  `chrome.storage.local` is not encrypted at rest beyond normal
  OS-user/browser-profile access control. Every persona's `nsec` held
  by this extension derives an encryption key from a user passphrase
  via WebCrypto `PBKDF2` → `AES-GCM`, unlocked once per browser
  session (same UX shape as MetaMask and most crypto-wallet
  extensions). This is the one place in the whole project where "ship
  the simple version first, harden later" doesn't apply: everything
  A006's bonding/slashing design and this project's whole sybil-
  resistance model depends on is exactly as safe as the keys
  attesting to it.
- **Multi-persona support is real v1 scope, not deferred.** P001's
  own Scope section already commits to "multiple context-specific
  personas" per person — this action keeps that real rather than
  quietly narrowing to one key. V1: the extension holds N
  independently-imported/generated personas, each with a user-given
  label; a per-origin mapping (which persona answers for which site)
  is stored locally, with an explicit popup UI to assign/switch it —
  never an implicit default across all sites.
- **NIP-07 surface is deliberately narrow: `getPublicKey()` and
  `signEvent()` only.** These are the only two calls persona's own
  attestation flow (`A001`) actually needs — publishing a signed
  event and proving who signed it. `getRelays()` and the `nip04`/
  `nip44` encrypt/decrypt extensions are real additional crypto
  surface (ECDH key agreement, symmetric cipher modes) this project's
  attestation model has no current use for. Left out explicitly, not
  silently unimplemented — a generic Nostr client calling them against
  a persona identity will get a clean "not supported" rather than a
  silent wrong answer.
- **Every `signEvent`/`getPublicKey` call shows a real approval
  popup — no silent auto-approve, ever, not even for a "trusted"
  site.** Mirrors this project's own "trust is always relative to the
  viewer, never a global score" discipline (CLAUDE.md) extended to the
  signer itself: the extension never decides on the user's behalf
  that a request is fine. A per-site "remember my choice" convenience
  is explicitly deferred (Stage 5), not assumed as a default.
- **Explicitly out of scope for this action**: Chrome Web Store /
  Firefox Add-ons store submission and review (a distribution/
  logistics concern, not a design one); Safari (WebExtensions support
  there is materially weaker); NIP-46 remote-signer support (a
  genuinely different architecture — signing happens on a different
  device over a relay, not in the browser at all); any visual/UX
  polish beyond functional; the landing-page/presentation/download-page
  half of P001's "client application" requirement (a separate,
  non-technical scoping task, not this one).

## Tasks

### Scoping (this pass)
- [x] Resolve whether this is worth building — yes, scoped above
- [x] Decide the technical shape: stack, key-import model, storage/
      encryption, multi-persona model, NIP-07 surface, approval UX
- [x] Name concrete out-of-scope boundaries rather than leaving them
      implicit

### Stage 1: single-persona MVP (deferred, not started)
- [ ] Bootstrap `extension/` (sibling to `site/`, `cmd/` — not part of
      the Go module): `manifest.json` (MV3), a background service
      worker, a popup UI
- [ ] Implement passphrase-encrypted storage for one imported/
      generated persona (`PBKDF2` → `AES-GCM`, session unlock)
- [ ] Implement `window.nostr.getPublicKey()`/`signEvent()` via
      `nostr-tools`, each gated behind a real approval popup
- [ ] Real interop test: sign a real event in the extension, publish
      it to a real public relay, fetch and verify it independently
      with persona's own Go `attestation.Verify` — proving a
      JS-signed and Go-signed event are indistinguishable to a
      verifier, the actual point of standards compliance here

### Stage 2: multi-persona support (deferred, not started)
- [ ] Support N stored personas with user-given labels
- [ ] Per-origin active-persona mapping, with an explicit switch UI —
      never an implicit cross-site default

### Stage 3: out of scope for now, named not ignored (deferred)
- [ ] Firefox parity (`webextension-polyfill`)
- [ ] Store submission/distribution
- [ ] NIP-46 remote-signer support
- [ ] `nip04`/`nip44` encrypt/decrypt, `getRelays()`

## Log

2026-10-19 — Filed and scoped in the same pass, following A006/A008's
own precedent of scoping before building anything this size. Resolved
D001's 2026-10-05 Open Question ("worth building at all?") to yes.
Real decisions made rather than left vague: this extension is v1 of
P001's own long-standing "client application" requirement, not a
companion to one that doesn't exist; key import via `nsec` paste, not
a new sync protocol; `nostr-tools`/`@noble` for all crypto, never
hand-rolled, matching this project's established discipline;
passphrase-encrypted storage treated as non-negotiable v1 scope, not
deferred hardening; multi-persona support kept as real v1 scope per
P001's own existing commitment; NIP-07 surface narrowed to exactly
what persona's attestation flow uses (`getPublicKey`/`signEvent`),
with every call gated behind a real approval popup. Nothing
implemented yet — Stage 1 is the next real, unblocked, zero-financial-
risk work this project has available while A007 stays parked on
external signal.
