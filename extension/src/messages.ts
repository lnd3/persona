// Shared message shapes between the three sides of the request flow:
// the page's injected window.nostr (inject.ts), the content script
// relaying it (content.ts), and the background service worker making
// the actual decision (background.ts) — plus the approval popup
// (approve.ts) background opens for every single request, per A012's
// own Decision: no silent auto-approve, ever.

export type Nip07Method = "getPublicKey" | "signEvent";

export interface UnsignedEventTemplate {
  kind: number;
  created_at: number;
  tags: string[][];
  content: string;
}

// content.ts -> background.ts (and ultimately back to the page)
export interface Nip07Request {
  type: "nip07-request";
  method: Nip07Method;
  origin: string;
  event?: UnsignedEventTemplate; // present only for signEvent
}

export interface Nip07Response {
  ok: boolean;
  result?: unknown; // a pubkey hex string, or a SignedEvent
  error?: string;
}

// approve.ts -> background.ts
export interface ApprovalDecision {
  type: "approval-decision";
  requestId: string;
  decision: "approve" | "deny";
  pubkeyHex?: string; // which persona to act as, chosen in the popup
  passphrase?: string; // only needed if that persona isn't unlocked yet
}

// background.ts -> approve.ts (what the popup renders), persisted in
// chrome.storage.session so it survives even if the popup is opened
// in a fresh document fetching this by requestId.
export interface PendingApproval {
  requestId: string;
  method: Nip07Method;
  origin: string;
  event?: UnsignedEventTemplate;
}

/** chrome.storage.session key both background.ts and approve.ts read/write. */
export const PENDING_KEY = "pendingApprovals";

export function isNip07Request(msg: unknown): msg is Nip07Request {
  return typeof msg === "object" && msg !== null && (msg as { type?: unknown }).type === "nip07-request";
}

export function isApprovalDecision(msg: unknown): msg is ApprovalDecision {
  return typeof msg === "object" && msg !== null && (msg as { type?: unknown }).type === "approval-decision";
}
