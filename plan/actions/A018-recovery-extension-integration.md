---
id: A018
title: Guardian recovery — extension/page integration
status: PLANNING
design: D001
project: P001
created: 2026-10-23
updated: 2026-10-23
---

## Context

Follows directly from the operator's own question of how `A003`'s
guardian recovery (designation, vouching, actual share splitting and
delivery) should fit the NIP-07 extension (`A012`) versus a public
page (`A015`'s own pattern) — and whether it should at all. Checked
the real custody boundary rather than assuming one shape fits
everything: the answer is genuinely split in two, with different
architectural consequences for each half.

## Decisions made here

- **Guardian designation and recovery vouching need zero new
  extension code.** `recovery_guardian` and `recovery_confirm` are
  both just attestation events (`NewGuardianClaim`/`NewConfirmClaim`
  in `internal/recovery/claims.go`, built on `attestation.New`) —
  `window.nostr.signEvent()` already covers exactly this, the same
  real finding `A015` made for ordinary attestations. These fit a
  public page (e.g. `/recovery`), reusing `A015`'s own pattern
  directly: build the event, sign via NIP-07, publish via
  `nostr-tools`' relay pool, no extension changes.
- **Actual guardian-share generation cannot be a page, full stop** —
  not a preference, a consequence of the custody boundary `A012`
  built on purpose. `Split()` operates on the raw private key;
  `EncryptShare` needs that same raw key for NIP-44's ECDH.
  `window.nostr` never hands raw key material to a page by design.
  This has to live in the extension's own trusted popup UI, in the
  same category as create/export (`A012`'s own popup), never behind
  the generic `signEvent`/`nip44` surface any website could call.
- **The real fork this action has to pick**: once the extension has
  built N signed, NIP-44-encrypted share-delivery events (`kind:
  3301`) internally, something has to publish them — and the
  extension has never talked to a relay itself; every page so far
  (`/try-signer`, `/attest`) does that part, not the extension.
  - **Option A — give the extension its own narrow relay-publish
    capability.** Simpler to wire (reuse the `nostr-tools` dependency
    the extension already has for signing), but it's a real,
    deliberate security-boundary change: today, even a tampered
    extension update has no network permission at all and literally
    cannot exfiltrate anything over the wire — only local storage and
    page messaging. Option A gives that up, for one feature's
    convenience.
  - **Option B — keep the extension network-free; hand the signed
    events to a companion page to publish.** Preserves the invariant
    that's held since `A012` Stage 1. The real mechanism, checked
    against the actual code rather than hand-waved: `content.ts`
    already runs on `persona-web`'s own pages and already has
    `chrome.storage` access (content scripts carry extension
    privileges for `chrome.*` APIs even though they share the page's
    DOM). The popup would write the N signed events to
    `chrome.storage.session`, then open/focus the companion page;
    that page's own `content.ts` instance reads them on load and
    pushes them into the page via `postMessage` — a genuinely new,
    *unprompted* push direction `inject.ts`/`content.ts` don't have
    today (today's flow is always page-initiates, extension-responds).
  - **Leaning Option B, not fully decided** — the security property
    is real and worth keeping, but Option B is real new plumbing, not
    a free choice; worth the operator's own confirmation before
    committing engineering time to it, same as other real architecture
    forks this project has stopped to ask about rather than picking
    unilaterally.
- **Guardian-side receiving is its own real, separate piece, not
  implied by the owner-side one.** A guardian's own extension needs
  to: notice an incoming `kind: 3301` event addressed to it, decrypt
  it (needs the guardian's own raw key — another thing that can't
  happen in a page), and persist the decrypted `Share` somewhere new
  — "shares I'm holding in trust for others" is a genuinely different
  storage concern from "my own personas," not a natural extension of
  existing `storage.ts`.
- **The actual recovery moment (gathering threshold-many shares back
  and running `Combine`) is a third, separate piece again** — the
  recovering party needs real shares back from real guardians, which
  again can't transit a page in plaintext. Not designed in this pass;
  named so it isn't assumed solved by the other two.
- **Explicitly out of scope for this action**: resolving `A017`'s own
  reconstruct-vs-migrate fork. That decision changes what "recovery"
  even produces (the same key back, or a new one plus a revocation),
  which this action's own UI/delivery design depends on — this action
  should follow `A017`'s resolution, not race ahead of it.

## Tasks

### Scoping (this pass)
- [x] Split the feature by what the custody boundary actually allows,
      not by convenience
- [x] Name the real architecture fork (Option A vs. B) rather than
      picking silently
- [x] Name guardian-side receiving and the actual recovery moment as
      separate, unsolved pieces, not implied by owner-side setup

### Build (deferred, not started — and blocked on A017's own resolution for part of this)
- [ ] `/recovery` page: guardian designation + vouching, `A015`'s
      exact pattern, no extension changes
- [ ] Resolve Option A vs. B (needs the operator's own confirmation)
- [ ] Extension: "Set up recovery" popup flow — `Split()`,
      per-guardian NIP-44 encryption, build + sign N `kind: 3301`
      events
- [ ] Wire whichever publish path Option A/B resolves to
- [ ] Guardian-side: detect, decrypt, and persist an incoming share
      (new storage concern)
- [ ] The actual recovery flow: gather threshold-many real shares
      back, run `Combine` — depends on `A017`'s reconstruct-vs-migrate
      resolution for what happens next

## Log

2026-10-23 — Filed and scoped in the same pass, following the
operator's own question of whether/how this fits the extension.
Real finding: it splits cleanly along the custody boundary, not along
feature lines — designation/vouching need nothing new; share
generation and delivery genuinely can't be a page. Named the real
publish-path fork (give the extension network access, or build new
page-push plumbing to keep it network-free) without picking
unilaterally — leaning toward keeping the extension network-free, but
that's real new engineering, not a given. Nothing implemented;
explicitly follows `A017`'s own resolution for part of its own scope.
