// The background service worker: the only place that ever touches a
// decrypted secret key. content.ts/inject.ts never see key material —
// they only relay a method name and (for signEvent) an event
// template, and receive back a pubkey or a signed event.
//
// Per A012's own Decision, every single request — getPublicKey or
// signEvent, regardless of origin — opens a real approval popup.
// There is no code path that signs or discloses a pubkey without a
// human clicking Approve in that popup.

import { decrypt } from "./crypto";
import { getUnlocked, listPersonas, setActivePersonaForOrigin, setUnlocked } from "./storage";
import * as nostr from "./nostr";
import type { ApprovalDecision, Nip07Request, Nip07Response, PendingApproval } from "./messages";
import { PENDING_KEY, isApprovalDecision, isNip07Request } from "./messages";

async function storePending(pending: PendingApproval): Promise<void> {
  const result = await chrome.storage.session.get(PENDING_KEY);
  const all = (result[PENDING_KEY] as Record<string, PendingApproval> | undefined) ?? {};
  all[pending.requestId] = pending;
  await chrome.storage.session.set({ [PENDING_KEY]: all });
}

async function peekPending(requestId: string): Promise<PendingApproval | undefined> {
  const result = await chrome.storage.session.get(PENDING_KEY);
  const all = (result[PENDING_KEY] as Record<string, PendingApproval> | undefined) ?? {};
  return all[requestId];
}

async function takePending(requestId: string): Promise<void> {
  const result = await chrome.storage.session.get(PENDING_KEY);
  const all = (result[PENDING_KEY] as Record<string, PendingApproval> | undefined) ?? {};
  delete all[requestId];
  await chrome.storage.session.set({ [PENDING_KEY]: all });
}

// In-memory resolvers for requests currently awaiting a popup
// decision. A popup click resolves one of these directly, as long as
// the service worker is still alive for the whole exchange — the
// same MV3 constraint every other NIP-07 extension's background
// script already accepts for a human-approval flow (see this
// action's own Stage 1 Log: not engineered around further, since
// every comparable extension ships exactly this).
const awaiting = new Map<string, (r: Nip07Response) => void>();

async function openApprovalPopup(pending: PendingApproval): Promise<void> {
  await storePending(pending);
  const url = chrome.runtime.getURL(`src/approve/approve.html?requestId=${pending.requestId}`);
  await chrome.windows.create({ url, type: "popup", width: 420, height: 460, focused: true });
}

async function handleNip07Request(req: Nip07Request): Promise<Nip07Response> {
  const requestId = crypto.randomUUID();
  const pending: PendingApproval = {
    requestId,
    method: req.method,
    origin: req.origin,
    event: req.event,
  };

  const responsePromise = new Promise<Nip07Response>((resolve) => {
    awaiting.set(requestId, resolve);
  });
  await openApprovalPopup(pending);
  return responsePromise;
}

/** Decrypts (and caches for this session) the secret key for pubkeyHex. */
async function resolveSecret(pubkeyHex: string, passphrase: string | undefined): Promise<string> {
  const already = await getUnlocked(pubkeyHex);
  if (already) return already;

  if (!passphrase) {
    throw new Error("this persona is locked; a passphrase is required");
  }
  const persona = (await listPersonas()).find((p) => p.pubkeyHex === pubkeyHex);
  if (!persona) throw new Error("unknown persona");

  const skBytes = await decrypt(passphrase, persona.encrypted); // throws on wrong passphrase
  const skHex = nostr.bytesToHex(skBytes);
  await setUnlocked(pubkeyHex, skHex);
  return skHex;
}

/**
 * Processes one decision from the approval popup. Returns whether
 * *this decision itself* was carried out (e.g. the passphrase was
 * right) — distinct from the Nip07Response's own `ok`, which is what
 * gets sent back to the page. A failed decrypt deliberately leaves
 * the pending request and its `awaiting` resolver untouched, so the
 * popup can let the user retry the passphrase against the *same*
 * still-pending request instead of silently failing the page's call.
 */
async function handleApprovalDecision(decision: ApprovalDecision): Promise<{ ok: boolean; error?: string }> {
  if (decision.decision === "deny") {
    const resolve = awaiting.get(decision.requestId);
    awaiting.delete(decision.requestId);
    await takePending(decision.requestId);
    resolve?.({ ok: false, error: "user denied the request" });
    return { ok: true };
  }

  const pending = await peekPending(decision.requestId);
  const resolve = awaiting.get(decision.requestId);
  if (!resolve || !pending || !decision.pubkeyHex) {
    return { ok: false, error: "this request is no longer pending" };
  }

  try {
    const skHex = await resolveSecret(decision.pubkeyHex, decision.passphrase);
    await setActivePersonaForOrigin(pending.origin, decision.pubkeyHex);

    const nip07Result =
      pending.method === "getPublicKey"
        ? nostr.getPublicKey(skHex)
        : pending.event
          ? nostr.signEvent(skHex, pending.event)
          : undefined;
    if (nip07Result === undefined) {
      throw new Error("missing event to sign");
    }

    awaiting.delete(decision.requestId);
    await takePending(decision.requestId);
    resolve({ ok: true, result: nip07Result });
    return { ok: true };
  } catch (err) {
    // Deliberately not consuming `pending`/`awaiting` here — the
    // request stays alive for a retry (e.g. wrong passphrase).
    return { ok: false, error: err instanceof Error ? err.message : String(err) };
  }
}

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (isNip07Request(message)) {
    handleNip07Request(message).then(sendResponse);
    return true;
  }
  if (isApprovalDecision(message)) {
    handleApprovalDecision(message).then(sendResponse);
    return true;
  }
  return false;
});
