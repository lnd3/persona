// Runs in the PAGE's own main-world JS context (injected as a real
// <script> tag by content.ts — not the isolated content-script world,
// which has no window.nostr the page could ever see). Defines the
// standard NIP-07 surface this action scopes to: getPublicKey and
// signEvent only (see A012's own Decisions for why getRelays/nip04/
// nip44 are deliberately left unimplemented, not silently missing).
//
// Never touches a secret key directly — every call round-trips
// through content.ts to the background service worker, which is the
// only place key material is ever decrypted.

export {}; // forces module scope, so this file's consts don't collide with content.ts's

const MARKER_REQUEST = "persona-nip07-request";
const MARKER_RESPONSE = "persona-nip07-response";

interface UnsignedEventTemplate {
  kind: number;
  created_at: number;
  tags: string[][];
  content: string;
}

function call(method: "getPublicKey" | "signEvent", event?: UnsignedEventTemplate): Promise<unknown> {
  const requestId = Math.random().toString(36).slice(2) + Date.now().toString(36);

  return new Promise((resolve, reject) => {
    function onMessage(ev: MessageEvent): void {
      if (ev.source !== window) return;
      const data = ev.data as { marker?: string; requestId?: string; ok?: boolean; result?: unknown; error?: string };
      if (data?.marker !== MARKER_RESPONSE || data.requestId !== requestId) return;
      window.removeEventListener("message", onMessage);
      if (data.ok) resolve(data.result);
      else reject(new Error(data.error ?? "request denied"));
    }
    window.addEventListener("message", onMessage);
    window.postMessage({ marker: MARKER_REQUEST, requestId, method, event }, window.location.origin);
  });
}

const personaNostr = {
  async getPublicKey(): Promise<string> {
    return call("getPublicKey") as Promise<string>;
  },
  async signEvent(event: UnsignedEventTemplate): Promise<unknown> {
    return call("signEvent", event);
  },
};

// Only install if nothing else already claimed window.nostr — never
// clobber another signer a user may have installed.
if (!("nostr" in window)) {
  Object.defineProperty(window, "nostr", {
    value: personaNostr,
    writable: false,
    configurable: false,
  });
}
