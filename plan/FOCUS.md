# Focus

*Rewritten each session. Not append-only — reflects current state only.*
*Overflow background and context → `FOCUS_context.md`. Scratch → `FOCUS_tmp.md`.*

---

## Active

Nothing actively being built — session wrapped 2026-10-23 after A018's
page half (`/recovery`) shipped to bh2 and the landing page was updated
to link it.

---

## Blocked

- **A017** (key revocation) — needs the operator's own call on the real
  fork: reconstruct the same key via `Combine` (does nothing against a
  thief holding a copy) vs. revoke-and-migrate to a new key authorized
  by the same guardian quorum. Nothing can be designed past this.
- **A018**'s remaining extension-side half (share generation/delivery,
  guardian-side receiving, the actual recovery moment) — needs (a) the
  operator's call on Option A (give the extension relay-publish access)
  vs. Option B (network-free, new page-push plumbing via `content.ts`
  + `chrome.storage.session`), and (b) follows A017's resolution, since
  that decides what "recovery" even produces.
- **A006**'s fee-bumping gap on the slashing branch (no anchor-style
  CPFP output, fee fixed at finalization) — confirmed real via a Nostr
  reviewer's reply, needs actual design work, not a quick patch.
- **A007** — one real Nostr engagement (one like, one reply promising
  to check the escrow scripts) but no confirmed reviewer yet; LinkedIn
  outreach got no signal. Waiting on real engagement, not re-pinging.

---

## Next

1. Operator decides A017's reconstruct-vs-migrate fork.
2. Operator decides A018's Option A vs. Option B publish path.
3. Once both land: build A018's extension-side "set up recovery" popup
   flow (`Split()`, per-guardian NIP-44 encryption, `kind: 3301` events)
   and whichever publish path was chosen.
4. Separately, whenever picked up: design the A006 fee-bumping fix for
   the slashing branch.
