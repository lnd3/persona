---
id: A014
title: Public live NIP-07 signer test page
status: DONE
design: D001
project: P001
created: 2026-10-22
updated: 2026-10-22
---

## Context

The operator asked to be able to try the real extension (A012) in
their own real browser, publicly, rather than only trusting this
session's own Playwright-driven Chromium tests. `persona-web`
(A010) previously served exactly one page (`/`, `index.html`) and
404'd everything else by design — so this needed both a new static
page and a real new route, not just a static file drop.

## Decisions made here

- **A real new route (`/try-signer`), not a drive-by relaxation of
  the server's routing.** `persona-web`'s `"/"` handler explicitly
  404s anything but an exact path match — refactored into a small
  `servePage(path, file)` helper used for both `/` and `/try-signer`,
  keeping the same exact-match behavior and the same A010 rule (a
  wisp view counts only after a real 200), rather than switching to a
  general static file server that would've silently widened what's
  publicly reachable.
- **Client-side signature verification, not a trust-the-extension
  claim.** The page imports `nostr-tools`'s `verifyEvent` directly
  from `esm.sh` as an ES module and checks the signature itself,
  in the browser, before telling the visitor it's valid — the same
  "verify independently" discipline this project applies everywhere
  else (A001's relay round trips, A008's receipt verification),
  applied here to a public demo page instead of only internal tests.
  A CDN dependency for a public, non-critical demo page is an
  accepted tradeoff — this isn't a security boundary (the real
  cryptographic trust already lives in the extension itself, proven
  in A012), just confirmatory UX.
- **Scoped to exactly `getPublicKey`/`signEvent`** — the same NIP-07
  surface A012 itself implements, nothing wider.

## Tasks

- [x] Add `site/try-signer.html`: extension-presence check, a real
      `getPublicKey` button, a real `signEvent` button, independent
      client-side signature verification, a link to the extension's
      own build/load instructions
- [x] Wire a real route in `cmd/persona-web/main.go` (`servePage`
      helper, used for both `/` and `/try-signer`) rather than
      widening the server into a general static file server
- [x] Verified for real: rebuilt the extension, loaded it in a real
      Chromium (same harness as A012's own tests), created a persona,
      navigated to the actual page served by the actual `persona-web`
      binary, clicked both real buttons through the real approval
      popups, and confirmed the page's own independent verification
      reported success — not assumed from a clean compile
- [x] Confirmed routing didn't regress: `/` still 200s, `/try-signer`
      200s, anything else (including `/try-signer/` with a trailing
      slash) still 404s exactly as before

## Log

2026-10-22 — Filed and built in the same pass. Found while building
that `persona-web` only had one route ever (`/`), by design (A010's
own doc comment: "persona has exactly one real page today") — adding
a second real page meant refactoring that routing, not just dropping
a file into `site/`. Verified end to end with the same real-Chromium
harness A012's own tests use: extension detected, `getPublicKey` and
`signEvent` both worked through real approval popups against the
real `persona-web` binary serving the real file, and the page's own
independent client-side verification (`nostr-tools`'s `verifyEvent`)
confirmed the signature — not just trusted the extension's return
value. `go build/vet/test` clean. Not yet deployed to `bh2` —
left as a separate, operator-approved step, same as A013.
