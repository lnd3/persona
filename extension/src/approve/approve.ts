// The approval popup background.ts opens for every single
// getPublicKey/signEvent request. Reads the pending request straight
// out of chrome.storage.session (this page has full extension
// storage access, no message round-trip needed for that part) and
// sends the user's decision back to background.ts, which is the only
// place that actually touches key material.

import { PENDING_KEY, type ApprovalDecision, type PendingApproval } from "../messages";
import { getActivePersonaForOrigin, getUnlocked, listPersonas } from "../storage";
import { encodeNpub } from "../nostr";

const app = document.getElementById("app")!;

function requestIdFromUrl(): string | null {
  return new URLSearchParams(window.location.search).get("requestId");
}

async function loadPending(requestId: string): Promise<PendingApproval | undefined> {
  const result = await chrome.storage.session.get(PENDING_KEY);
  const all = (result[PENDING_KEY] as Record<string, PendingApproval> | undefined) ?? {};
  return all[requestId];
}

function sendDecision(decision: ApprovalDecision): Promise<{ ok: boolean; error?: string }> {
  return chrome.runtime.sendMessage(decision);
}

async function render(): Promise<void> {
  const requestId = requestIdFromUrl();
  if (!requestId) {
    app.innerHTML = `<p class="error">Malformed approval request.</p>`;
    return;
  }

  const pending = await loadPending(requestId);
  if (!pending) {
    app.innerHTML = `<p class="empty">This request is no longer waiting for a decision — it may have timed out or already been handled.</p>`;
    return;
  }

  const personas = await listPersonas();
  if (personas.length === 0) {
    app.innerHTML = `<p class="empty">No personas yet. Open the persona signer's toolbar icon to create or import one first.</p>
      <div class="buttons"><button class="deny" id="deny">Close</button></div>`;
    document.getElementById("deny")!.addEventListener("click", () => {
      sendDecision({ type: "approval-decision", requestId, decision: "deny" });
      window.close();
    });
    return;
  }

  const defaultPubkey = (await getActivePersonaForOrigin(pending.origin)) ?? personas[0]!.pubkeyHex;

  const eventSummary = pending.event
    ? `<div class="event">kind ${pending.event.kind}\n${JSON.stringify(pending.event.content)}</div>`
    : "";

  app.innerHTML = `
    <h1>${pending.method === "getPublicKey" ? "Share your public key?" : "Sign this event?"}</h1>
    <div class="origin">${pending.origin}</div>
    ${eventSummary}
    <label for="persona">Act as</label>
    <select id="persona">
      ${personas
        .map(
          (p) =>
            `<option value="${p.pubkeyHex}" ${p.pubkeyHex === defaultPubkey ? "selected" : ""}>${p.label} (${encodeNpub(p.pubkeyHex).slice(0, 16)}…)</option>`,
        )
        .join("")}
    </select>
    <div id="passphraseRow"></div>
    <div class="error" id="error"></div>
    <div class="buttons">
      <button class="deny" id="deny">Deny</button>
      <button class="approve" id="approve">Approve</button>
    </div>
  `;

  const select = document.getElementById("persona") as HTMLSelectElement;
  const passphraseRow = document.getElementById("passphraseRow")!;
  const errorBox = document.getElementById("error")!;

  async function refreshPassphraseField(): Promise<void> {
    const unlocked = await getUnlocked(select.value);
    passphraseRow.innerHTML = unlocked
      ? ""
      : `<label for="passphrase">Passphrase (unlocks this persona for the rest of the browser session)</label>
         <input type="password" id="passphrase" autofocus />`;
  }

  select.addEventListener("change", () => void refreshPassphraseField());
  await refreshPassphraseField();

  document.getElementById("deny")!.addEventListener("click", () => {
    void sendDecision({ type: "approval-decision", requestId, decision: "deny" });
    window.close();
  });

  document.getElementById("approve")!.addEventListener("click", () => {
    void (async () => {
      const passphraseInput = document.getElementById("passphrase") as HTMLInputElement | null;
      const passphrase = passphraseInput?.value;
      if (passphraseInput && !passphrase) {
        errorBox.textContent = "Enter the passphrase to unlock this persona.";
        return;
      }
      const outcome = await sendDecision({
        type: "approval-decision",
        requestId,
        decision: "approve",
        pubkeyHex: select.value,
        passphrase,
      });
      if (outcome.ok) {
        window.close();
        return;
      }
      // Deliberately not closing: the request is still pending (see
      // background.ts's own handleApprovalDecision), so a wrong
      // passphrase lets the user just retry rather than having to
      // re-trigger the whole request from the page.
      errorBox.textContent = outcome.error ?? "approval failed";
    })();
  });
}

void render();
