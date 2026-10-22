// The toolbar popup: persona management (create, import, delete,
// lock). Signing/pubkey-disclosure approval is a *different* popup
// (approve.ts), opened by background.ts only in response to an
// actual page request — this one is never in that path.

import { decrypt, encrypt } from "../crypto";
import { deletePersona, getUnlocked, listPersonas, lockAll, savePersona, type StoredPersona } from "../storage";
import { decodeNsec, encodeNpub, encodeNsec, generatePersonaSecret, getPublicKey, bytesToHex, hexToBytes } from "../nostr";

const app = document.getElementById("app")!;

async function render(): Promise<void> {
  const personas = await listPersonas();
  const unlockedFlags = await Promise.all(personas.map((p) => getUnlocked(p.pubkeyHex)));

  const list = personas.length
    ? personas
        .map((p, i) => {
          const unlocked = unlockedFlags[i] !== undefined;
          return `
            <div class="persona" data-pubkey="${p.pubkeyHex}">
              <div class="meta">
                <div class="label">${p.label}</div>
                <div class="npub">${encodeNpub(p.pubkeyHex)}</div>
              </div>
              <div>
                <span class="status ${unlocked ? "unlocked" : "locked"}">${unlocked ? "unlocked" : "locked"}</span>
                <button class="export" data-pubkey="${p.pubkeyHex}">Export</button>
                <button class="delete" data-pubkey="${p.pubkeyHex}">✕</button>
              </div>
            </div>
            <div class="export-row" id="export-${p.pubkeyHex}" hidden></div>`;
        })
        .join("")
    : `<p class="empty">No personas yet.</p>`;

  app.innerHTML = `
    ${list}
    <h2>New persona</h2>
    <form id="create-form">
      <input type="text" id="label" placeholder="Label (e.g. \"work\")" required />
      <input type="text" id="nsec" placeholder="nsec to import (leave blank to generate a new one)" />
      <input type="password" id="passphrase" placeholder="Passphrase to encrypt it with" required />
      <div class="row">
        <button type="submit">Add persona</button>
        <button type="button" id="lock-all">Lock all</button>
      </div>
      <div class="error" id="form-error"></div>
    </form>
    <div class="build-info">${__BUILD_INFO__}</div>
  `;

  document.querySelectorAll<HTMLButtonElement>(".export").forEach((btn) => {
    btn.addEventListener("click", () => {
      const pubkeyHex = btn.dataset.pubkey!;
      const row = document.getElementById(`export-${pubkeyHex}`)!;
      if (!row.hidden) {
        row.hidden = true;
        row.innerHTML = "";
        return;
      }
      row.hidden = false;
      row.innerHTML = `
        <p class="warn">Anyone with this nsec can act as this persona.
        Don't paste it anywhere else; copy it straight to a password
        manager or offline backup.</p>
        <input type="password" class="export-passphrase" placeholder="Passphrase" />
        <button type="button" class="export-reveal">Reveal nsec</button>
        <div class="export-error error"></div>
        <div class="export-nsec"></div>
      `;
      const passInput = row.querySelector<HTMLInputElement>(".export-passphrase")!;
      const errorBox = row.querySelector<HTMLDivElement>(".export-error")!;
      const nsecBox = row.querySelector<HTMLDivElement>(".export-nsec")!;
      row.querySelector(".export-reveal")!.addEventListener("click", () => {
        void (async () => {
          errorBox.textContent = "";
          try {
            const persona = (await listPersonas()).find((p) => p.pubkeyHex === pubkeyHex);
            if (!persona) throw new Error("persona no longer exists");
            let skBytes: Uint8Array;
            try {
              skBytes = await decrypt(passInput.value, persona.encrypted);
            } catch {
              // AES-GCM's own auth-tag check failing is the "wrong
              // passphrase" signal (crypto.ts's own doc comment) --
              // but the resulting DOMException (name "OperationError")
              // has an EMPTY .message, so showing err.message here
              // would silently render as a blank error. Give the one
              // realistic cause directly instead.
              throw new Error("Wrong passphrase.");
            }
            const nsec = encodeNsec(bytesToHex(skBytes));
            nsecBox.innerHTML = `
              <input type="text" class="export-value" readonly value="${nsec}" />
              <button type="button" class="export-copy">Copy</button>
            `;
            nsecBox.querySelector(".export-copy")!.addEventListener("click", () => {
              void navigator.clipboard.writeText(nsec);
            });
          } catch (err) {
            errorBox.textContent = err instanceof Error ? err.message : String(err);
          }
        })();
      });
    });
  });

  document.querySelectorAll<HTMLButtonElement>(".delete").forEach((btn) => {
    btn.addEventListener("click", () => {
      void (async () => {
        await deletePersona(btn.dataset.pubkey!);
        await render();
      })();
    });
  });

  document.getElementById("lock-all")!.addEventListener("click", () => {
    void (async () => {
      await lockAll();
      await render();
    })();
  });

  document.getElementById("create-form")!.addEventListener("submit", (e) => {
    e.preventDefault();
    void (async () => {
      const label = (document.getElementById("label") as HTMLInputElement).value.trim();
      const nsec = (document.getElementById("nsec") as HTMLInputElement).value.trim();
      const passphrase = (document.getElementById("passphrase") as HTMLInputElement).value;
      const errorBox = document.getElementById("form-error")!;

      try {
        const skHex = nsec ? decodeNsec(nsec) : bytesToHex(generatePersonaSecret());
        const pubkeyHex = getPublicKey(skHex);
        const existing = await listPersonas();
        if (existing.some((p) => p.pubkeyHex === pubkeyHex)) {
          throw new Error("this persona is already stored");
        }
        const encrypted = await encrypt(passphrase, hexToBytes(skHex));
        const persona: StoredPersona = { label, pubkeyHex, encrypted };
        await savePersona(persona);
        await render();
      } catch (err) {
        errorBox.textContent = err instanceof Error ? err.message : String(err);
      }
    })();
  });
}

void render();
