---
id: A016
title: Pre-public-release sensitive-information audit
status: DONE
design: D001
project: P001
created: 2026-10-22
updated: 2026-10-22
---

## Context

The operator asked for a real assessment of the whole repo for
sensitive information, given the repo is already public on GitHub and
the project is approaching its first public release push. A real
audit, not a skim: full git history (not just current `HEAD` — a
public repo's history is public forever unless rewritten), using a
real tool (`gitleaks`, downloaded as a prebuilt binary — no build,
same pattern as fetching `gh` earlier) plus targeted manual checks for
patterns `gitleaks`'s default ruleset doesn't know about (Nostr
`nsec`, Bitcoin WIF keys).

## Findings

- **Real, medium-severity**: the operator's real email
  (`linus.cumselius@gmail.com`) is the commit author/committer on 92
  of 93 commits (the 93rd uses GitHub's own noreply address),
  including every commit from this session. It's genuinely public —
  visible via `git log`, GitHub's own commit/blame views, and the
  GitHub API — on a project whose whole premise is identity
  sovereignty, which makes this a real, slightly ironic exposure, not
  a hypothetical one.
  - **Forward fix is trivial and non-destructive**: set the local git
    config's `user.email` to GitHub's noreply address
    (`<id>+<username>@users.noreply.github.com`) so every future
    commit stops carrying the real email.
  - **Fixing the existing 92 commits is a real history rewrite**
    (`git filter-repo`/BFG + force-push) — destructive, breaks any
    existing fork/clone's shared history, and GitHub may retain the
    old emails in its own API/search cache regardless of a rewrite.
    **Not done here** — this is the operator's own call, the same
    "never run a destructive operation without being asked" rule this
    project has followed all along.
- **Real, low-severity**: the server's real public IP
  (`158.174.211.245`) is written out in `deploy/README.md` and
  `plan/actions/A009`, alongside the internal `bh2` codename.
  Essentially already public regardless — `solemn.network`'s own DNS
  resolves to the same IP, so this isn't a secret, just a documented
  operational detail. Not changed; flagging only.
- **Real, trivial**: the Linux username `linus` (matching the
  operator's own first name) appears once, in a debugging note in
  `plan/actions/A008`'s own Log, as part of a local filesystem path
  (`/home/linus/.aperture/aperture.db`) quoted from a `strace` output.
  Not changed; flagging only — negligible exposure on its own.
- **Clean, checked for real, not assumed**: full git history scanned
  with `gitleaks` (0 findings) plus targeted manual greps across all
  93 commits for Nostr `nsec1…` keys, PEM private-key blocks, Bitcoin
  WIF keys, GitHub PATs, SSH private keys, and any `.env` file ever
  committed — all zero. `WISP_TOKEN` never appears with a real value
  anywhere in history (consistent with this session's own discipline
  of writing it directly to the server over SSH and shredding the
  local plaintext copy, never committing it). No key/cert/wallet/
  binary files and no unusually large blobs ever committed.
  `deploy/.env.example` holds only placeholders, with its own explicit
  "never commit this file with a real value" warning already in
  place.

## Tasks

- [x] Scan full git history with a real tool (`gitleaks`), not just
      the current working tree
- [x] Targeted manual checks for domain-specific secret formats
      `gitleaks`'s default ruleset doesn't cover (Nostr `nsec`,
      Bitcoin WIF)
- [x] Check for committed `.env` files, key/cert/binary files, and
      unusually large blobs across all history
- [x] Check for real PII (email, real name, real IP) beyond
      cryptographic secrets
- [x] Report findings with real severity, not inflated — and leave the
      one destructive remediation (historical git-history rewrite) as
      the operator's own explicit decision, not acted on unilaterally

## Log

2026-10-22 — Audited and reported. One real, actionable finding (the
real email on 92/93 commits, including every commit this session),
two low-severity/informational ones (the real server IP, already
public via DNS; a local username in one debug note), and a clean
result everywhere else that actually matters for a public repo —
checked with a real tool across full history, not a current-tree
skim. Did not rewrite history; that's a real destructive operation on
a repo others may have already cloned, left for the operator to
decide.
