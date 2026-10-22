---
id: A012
title: NIP-07 browser-extension signer
status: DONE
design: D001
project: P001
created: 2026-10-19
updated: 2026-10-22
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

### Stage 1: single-persona MVP (done)
- [x] Bootstrap `extension/` (sibling to `site/`, `cmd/` — not part of
      the Go module): `manifest.json` (MV3), a background service
      worker, a popup UI. **Done 2026-10-19.**
- [x] Implement passphrase-encrypted storage for one imported/
      generated persona (`PBKDF2` → `AES-GCM`, session unlock).
      **Done 2026-10-19.**
- [x] Implement `window.nostr.getPublicKey()`/`signEvent()` via
      `nostr-tools`, each gated behind a real approval popup. **Done
      2026-10-19.**
- [x] Real interop test: sign a real event in the extension, publish
      it to a real public relay, fetch and verify it independently
      with persona's own Go signature-checking code. **Done
      2026-10-19** — see Log for the exact chain (real Chromium via
      Playwright, real approval popups, a real relay round trip).

### Stage 2: multi-persona support (done, genuinely tested with N=2)
- [x] Support N stored personas with user-given labels — exercised
      for real with two real personas ("work"/"personal"), each with
      a distinct pubkey. **Done 2026-10-22** — see Log.
- [x] Per-origin active-persona mapping, with an explicit switch UI —
      never an implicit cross-site default. **Done 2026-10-22** — see
      Log for the real multi-origin test proving per-origin memory,
      per-persona unlock isolation, and that an explicit switch
      actually persists rather than being a one-off override.

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

2026-10-19 (later) — **Stage 1 built and genuinely verified, not just
compiled.** `extension/` (TypeScript, MV3, `nostr-tools` for all
crypto): `crypto.ts` (`PBKDF2`→`AES-GCM` via WebCrypto, never
hand-rolled), `storage.ts` (`chrome.storage.local` for encrypted
personas + origin mapping, `chrome.storage.session` for
decrypted-this-session key material — never written to disk),
`nostr.ts` (thin wrapper over `nostr-tools/pure`), `background.ts`
(the only code that ever touches a decrypted key), `content.ts`/
`inject.ts` (the actual `window.nostr` the page sees, via a
`postMessage` relay — content script never touches key material
either), and a toolbar `popup` (persona management) plus a separate
`approve` popup background.ts opens for *every single*
`getPublicKey`/`signEvent` call, with no code path that skips it.

Found and fixed one real UX/correctness gap while building the
approval flow: an initial version closed the approval popup
immediately on "Approve" regardless of outcome, so a wrong passphrase
would silently fail the page's call with no way to retry without
re-triggering the whole request. Fixed by having `background.ts`
leave the pending request and its resolver untouched on a failed
decrypt (only consuming it on an actual deny or success), and having
`approve.ts` await the real outcome before deciding whether to close.

**Verified for real, not just type-checked**: `tsc --noEmit` clean,
`esbuild` bundle built, then driven in an actual unpacked install in
a real Chromium (downloaded via Playwright, loaded with
`--load-extension`/`--disable-extensions-except`, `--headless=new` —
genuinely the same loading mechanism a user's own browser uses, not a
mock). Created a real persona through the real toolbar popup
(passphrase-encrypted on disk), then from a real separate test page:
`window.nostr.getPublicKey()` triggered a real approval popup
showing the real origin and persona picker, requiring the real
passphrase to unlock (confirming storage truly starts locked each
session) — returned the correct real pubkey. `window.nostr.
signEvent()` then triggered a second real approval popup (no
passphrase needed, confirming session-unlock persistence worked) and
returned a real, fully-formed signed event.

**The actual interop proof** (the point of building to a standard at
all): took that real signed event's JSON and verified it with
persona's own Go code — `(*nostr.Event).CheckSignature()` from
`go-nostr`, the exact function `internal/attestation.Verify` itself
calls internally, so this is the same check that code path already
trusts elsewhere, not a weaker one. Passed. Then published it for
real to `wss://nos.lol` (confirmed, not assumed — `relay.damus.io`
hit the same transient 503 this project has seen before, confirmed
unrelated by switching relays) and fetched it back, re-verifying
independently — the same live round-trip discipline A001 established
for persona's own Go-signed events, now proven for a JS-signed one
too. A JS-extension-signed event and a Go-signed event are
genuinely indistinguishable to a verifier.

Stage 2's multi-persona data model (N stored personas, per-origin
active-persona mapping with an explicit picker) already exists as a
direct consequence of how Stage 1 was built — not deferred narrowing
to one key as originally planned as a fallback. Not separately
exercised with more than one real persona in this pass, so marked
done on the data-model/UI level, not claimed as independently tested
at N>1.

Cleaned up: removed the throwaway Go interop-verification file,
confirmed `go build ./... && go vet ./...` clean on persona's own
module (the extension lives in its own `extension/` directory,
`node_modules`/`dist` gitignored, never touching the Go module).
Status left `IN_PROGRESS` — Stage 2's own multi-persona data model
existed but hadn't been genuinely exercised with more than one real
persona yet; see the next Log entry for that.

2026-10-22 — **Stage 2 genuinely tested, not just present in the
code.** Real multi-persona, multi-origin Playwright test against the
real unpacked extension (same real-Chromium harness as Stage 1's own
interop test): created two real personas ("work"/"personal", each
with its own passphrase and a genuinely distinct pubkey), then drove
two separate real test-page origins through the real approval popup:

1. Origin A requests `getPublicKey`, explicitly picks "work", unlocks
   it with its passphrase — returns work's real pubkey.
2. Origin B requests `getPublicKey` — no mapping yet — explicitly
   picks "personal", **requires its own separate passphrase** even
   though "work" is already unlocked this session. Confirms unlock
   state is tracked per-persona, not a single global "the extension
   is unlocked" flag.
3. Origin A asks again: the popup **defaults** to "work" with no
   passphrase prompt — confirms both the per-origin memory
   (`getActivePersonaForOrigin`) and the session-persistent unlock
   (`chrome.storage.session`) are real, not just present in the code.
4. Origin A **explicitly switches** to "personal" via the same picker
   — the actual switch UI this action's own Decisions called for,
   not an implicit default change.
5. Origin A asks a third time: the default has genuinely followed the
   switch — proving `setActivePersonaForOrigin` persists the change
   rather than only affecting the one call it was made during.
6. Origin B, untouched by any of A's activity, still defaults to its
   own "personal" assignment from step 2 — confirming origins are
   genuinely isolated from each other, not sharing one mapping.

Found one real test-harness bug while writing this (not an extension
bug): the test's own wait for "the new persona appeared" matched the
*first* `.persona` DOM element rather than one with the new label,
so creating a second persona raced against the first popup's own
initial paint. Fixed by waiting for an element containing that
specific label. Worth recording since it's the kind of race a real
multi-persona UI test has to get right, not specific to this
extension.

Cleaned up: killed the test Chromium/profile directories; nothing
from this pass touches persona's own tree beyond this plan file (the
extension's own source was already committed in the previous pass).
A012 moved to `DONE` — Stage 1 and Stage 2 (this action's actual
scope) are both genuinely built and verified; Stage 3's items
(Firefox parity, store distribution, NIP-46, `nip04`/`nip44`/
`getRelays()`) were named out of scope for this action from the
start (see this file's own Decisions), not something left unfinished.
