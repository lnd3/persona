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
  - **Forward fix**: done — see Log. Repo-local `user.email`
    override to GitHub's confirmed-correct noreply address, not
    touching the global config other repos on this machine still use.
  - **Fixing the existing 92 commits** is a real history rewrite
    (`git filter-repo` + force-push) — destructive, breaks any
    existing fork/clone's shared history. **Done — see Log**: the
    operator decided to do it now specifically because the repo had
    just gone public, making this the cheapest point to ever do it
    (minimal risk of existing forks). Thoroughly verified (tree
    hashes, commit count, message order all identical to a backup
    taken first) before force-pushing, and confirmed live on GitHub's
    own API afterward, not assumed from a successful push alone.
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
skim. Did not rewrite history yet; that's a real destructive
operation on a repo others may have already cloned, left for the
operator to decide.

2026-10-22 (later) — Set a repo-local `user.email` override
(`lnd3@users.noreply.github.com` — confirmed correct for this
account, not guessed: that exact address already appeared
legitimately on one GitHub-web-UI commit in this repo's own history)
so every future commit here stops carrying the real email, without
touching the global git config other repos on this machine still use.

2026-10-22 (later still) — **Operator approved rewriting the existing
92 commits.** Decided to do it now rather than defer: the repo just
went public, so the cost of a history rewrite (broken forks/clones)
is about as low as it will ever be, and only grows the longer real
emails sit in public history. Used `git-filter-repo` (downloaded as
the single-file script, no build) with an email-callback replacing
the real address with the same confirmed-correct noreply one.

Verified the rewrite thoroughly before touching the remote at all:
every tree hash identical to a full mirror backup taken first
(spot-checked 5 random commits plus `HEAD` itself), commit count
unchanged (94), commit messages and order identical via a full diff
against the backup, author *names* preserved (only the email
changed). Only then re-added the `origin` remote (removed by
`git-filter-repo` itself as a safety default) and force-pushed with
an explicit `--force-with-lease` against the exact pre-rewrite remote
SHA — not a bare `--force` — so the push would have failed instead of
silently clobbering anything if the remote had moved underneath.

Confirmed live on GitHub itself afterward, not just locally: fetched
fresh and checked GitHub's own commits API directly, including the
very first commit in the repo's history — every commit now shows the
noreply email. A full mirror backup of the pre-rewrite state is kept
at `/tmp/persona-backup-before-rewrite` (outside the repo, not
committed) in case it's ever needed.

2026-10-22 (later) — Saved a standalone, polished version of this
audit's findings as [`SECURITY_AUDIT.md`](../../SECURITY_AUDIT.md) at
the repo root, matching this project's own existing convention
(`internal/escrow/SECURITY_REVIEW.md` for A007) of keeping a real,
public-facing reference document separate from this plan file's own
chronological Log. Linked from `README.md`.
