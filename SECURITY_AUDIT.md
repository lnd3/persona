# Pre-public-release sensitive-information audit

Done for **A016**, `plan/actions/A016-pre-public-release-sensitive-info-audit.md`
in this repo's own planning system — a check of the whole repo and its
full git history for sensitive information, requested ahead of
persona's first wider public push (the repo was already public on
GitHub, but hadn't had this kind of pass yet).

## Scope and method

- **Full git history**, not just the current tree — a public repo's
  history is public forever unless deliberately rewritten.
- **[`gitleaks`](https://github.com/gitleaks/gitleaks)** (downloaded
  as a prebuilt binary, no build step) run against the entire history
  with its default ruleset.
- **Targeted manual checks** for formats `gitleaks`'s default ruleset
  doesn't know about: Nostr `nsec` bech32 private keys, Bitcoin WIF
  private keys, committed `.env` files, key/cert/wallet/binary files,
  and unusually large blobs anywhere in history.
- A separate check for real PII beyond cryptographic secrets: email
  addresses, real names, real server details.

As of this audit, 94 commits, full history, checked 2026-10-22.

## Findings

### Fixed

- **The operator's real email was the git commit author/committer on
  92 of 93 commits at audit time**, visible via `git log`, GitHub's
  own commit/blame views, and the GitHub API. Remediated in full:
  - Repo-local `git config user.email` now points at the account's
    GitHub noreply address, so no future commit carries it.
  - **The existing history was rewritten** (`git-filter-repo`,
    verified byte-identical file content against a full backup before
    force-pushing) — the real email no longer appears in any commit,
    confirmed live via GitHub's own commits API afterward, including
    the very first commit in the repo's history.
  - Residual note: GitHub still shows the pushing **account**
    (`lnd3`) as a contributor, which is a separate thing from the
    commit email this fix addressed — if that account's own GitHub
    profile has a real name/bio set, that's a separate, one-time
    check on the account itself, not something git history can fix.

### Noted, not changed (low severity / already effectively public)

- The server's real public IP is written in `deploy/README.md` and
  one plan action. Not a secret in practice — `solemn.network`'s own
  DNS already resolves to the same address — so left as-is.
- A local Linux username (matching the operator's own first name)
  appears once, in a debugging note, as part of a filesystem path
  quoted from tool output. Negligible exposure on its own; not worth
  the churn of editing a historical log entry for.

### Clean

- Zero `gitleaks` findings across full history.
- No Nostr `nsec`, Bitcoin WIF, PEM private key, SSH private key, or
  GitHub personal-access-token pattern anywhere in history.
- No `.env` file, key/cert/wallet file, or unusually large binary blob
  was ever committed.
- `WISP_TOKEN` (the one real external credential this project
  handles) never appears with an actual value anywhere in history —
  it's always written directly to the deployment server over SSH and
  the local plaintext copy shredded, never committed.
- `deploy/.env.example` holds only placeholders, with its own explicit
  "never commit this file with a real value" warning already in
  place.

## Re-running this audit

```bash
# gitleaks: https://github.com/gitleaks/gitleaks/releases
gitleaks git --no-banner -v .

# targeted checks this project's own secrets need beyond gitleaks' ruleset
git log --all -p | grep -oE "nsec1[a-z0-9]{20,}"               # Nostr private keys
git log --all -p | grep -oE '\b[5KL][1-9A-HJ-NP-Za-km-z]{50,51}\b'  # Bitcoin WIF keys
git log --all --diff-filter=A --name-only | grep -E "\.env$"   # committed .env files
```
