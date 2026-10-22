# persona signer

A NIP-07 browser-extension signer for persona identities. See
[`plan/actions/A012-nip07-browser-extension-signer.md`](../plan/actions/A012-nip07-browser-extension-signer.md)
for the full scoping and design decisions.

Private keys never leave the extension: pages only ever see
`window.nostr.getPublicKey()`/`signEvent()`, and every single call —
regardless of origin — opens a real approval popup. There is no
silent auto-approve.

## Build

```bash
npm install
npm run build      # one-shot build into dist/
npm run watch       # rebuild on change
npm run typecheck
```

## Load it in Chrome/Chromium for real testing

1. `npm install && npm run build`
2. `chrome://extensions` → enable Developer mode → "Load unpacked" →
   select this `extension/` directory.
3. Click the toolbar icon to create or import a persona (via its
   `nsec`) under a passphrase.
4. Any page calling `window.nostr.getPublicKey()`/`signEvent()` will
   trigger a real approval popup.

## Packaging a release

```bash
npm run package     # rebuilds first, then writes release/persona-signer-v<version>.zip
```

The zip contains exactly `manifest.json`, `dist/`, and the two popup
`.html` files — what "Load unpacked" needs, extracted, nothing from
the Go side or the rest of this repo. Used for the interim GitHub
Releases distribution (see A012's own Log) while a real Chrome Web
Store listing / Firefox-signed self-distribution remain unbuilt.

## What's deliberately not here yet

See A012's own Decisions for the full list — Firefox parity, store
distribution, NIP-46 remote-signer support, and the `nip04`/`nip44`/
`getRelays()` NIP-07 extensions are all named out-of-scope for Stage
1, not silently missing.
