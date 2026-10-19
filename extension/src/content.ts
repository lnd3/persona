// Runs in the content script's ISOLATED world: has chrome.* API
// access the page itself must never get, but shares the page's DOM
// and `window` object for postMessage. Two jobs: (1) inject inject.ts
// as a real <script> tag so window.nostr exists in the page's own
// main-world context, and (2) relay postMessage requests from that
// script to the background service worker and the response back.

export {}; // forces module scope, so this file's consts don't collide with inject.ts's

const MARKER_REQUEST = "persona-nip07-request";
const MARKER_RESPONSE = "persona-nip07-response";

function injectPageScript(): void {
  const script = document.createElement("script");
  script.src = chrome.runtime.getURL("dist/inject.js");
  script.type = "module";
  (document.head || document.documentElement).appendChild(script);
  script.addEventListener("load", () => script.remove());
}

window.addEventListener("message", (ev: MessageEvent) => {
  if (ev.source !== window) return;
  const data = ev.data as {
    marker?: string;
    requestId?: string;
    method?: "getPublicKey" | "signEvent";
    event?: unknown;
  };
  if (data?.marker !== MARKER_REQUEST || !data.requestId || !data.method) return;

  chrome.runtime
    .sendMessage({
      type: "nip07-request",
      method: data.method,
      origin: window.location.origin,
      event: data.event,
    })
    .then((response: { ok: boolean; result?: unknown; error?: string }) => {
      window.postMessage(
        { marker: MARKER_RESPONSE, requestId: data.requestId, ok: response.ok, result: response.result, error: response.error },
        window.location.origin,
      );
    })
    .catch((err: unknown) => {
      window.postMessage(
        {
          marker: MARKER_RESPONSE,
          requestId: data.requestId,
          ok: false,
          error: err instanceof Error ? err.message : String(err),
        },
        window.location.origin,
      );
    });
});

injectPageScript();
